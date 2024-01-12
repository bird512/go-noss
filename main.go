package main

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/nbd-wtf/go-nostr"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

//var sk string
//var pk string

var walletFile = "wallets.json"
var arbRpcUrl string
var arbRpcUrls []string
var numberOfWorkers = 1
var interval int64
var cookie string
var (
	ErrDifficultyTooLow = errors.New("nip13: insufficient difficulty")
	ErrGenerateTimeout  = errors.New("nip13: generating proof of work took too long")
)
var messageId atomic.Value

// var messageCache *expirable.LRU[string, string]
var blockClient *ethclient.Client
var wallets []Wallet
var counter = Counter{val: 0}

func initEnv() {

	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime) // Add this line
	log.Println("Starting...")
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	wallets = loadWalletFromFile(walletFile)
	if len(wallets) == 0 {
		log.Println("钱包文件为空, 生成随机地址")
		generateWalletsToFile(10, walletFile)
		wallets = loadWalletFromFile(walletFile)
	}
	for _, w := range wallets {
		log.Println("加载到钱包：", w.PublicNpub)
	}

	arbRpcUrl = os.Getenv("arbRpcUrl")
	rpcs := os.Getenv("arbRpcUrls")
	// split the rpcs into array by
	arbRpcUrls = strings.Split(rpcs, ",")
	if len(arbRpcUrls) > 0 {
		// randomly select one
		arbRpcUrl = arbRpcUrls[rand.Intn(len(arbRpcUrls))]
	}
	log.Println("arbRpcUrl = ", arbRpcUrl)

	cookie = os.Getenv("cookie")
	numberOfWorkers, _ = strconv.Atoi(os.Getenv("numberOfWorkers"))
	interval, _ = strconv.ParseInt(os.Getenv("interval"), 10, 64)
	if interval < 1 {
		interval = 1000
	}
	log.Println("interval = ", interval)
	//messageCache = expirable.NewLRU[string, string](5, nil, time.Second*10)

	if blockClient != nil {
		blockClient.Close()
	}
	blockClient, err = ethclient.Dial(arbRpcUrl)

	if err != nil {
		log.Fatalf("无法连接到Arbitrum节点: %v", err)
	}
}

// refresh the var from the evn file for every 5 seconds
func refreshEnv() {
	ticker := time.NewTicker(15 * time.Minute)
	for {
		select {
		case <-ticker.C:
			initEnv()
		}
	}
}

type Message struct {
	EventId string `json:"eventId"`
}

type EV struct {
	Sig       string          `json:"sig"`
	Id        string          `json:"id"`
	Kind      int             `json:"kind"`
	CreatedAt nostr.Timestamp `json:"created_at"`
	Tags      nostr.Tags      `json:"tags"`
	Content   string          `json:"content"`
	PubKey    string          `json:"pubkey"`
}

func connectToWSS(host string) (*websocket.Conn, error) {
	url := "wss://" + host + "/"
	var conn *websocket.Conn
	var err error
	headers := http.Header{}
	headers.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0")
	headers.Add("Origin", "https://noscription.org")
	headers.Add("Host", host)

	headers.Add("Cookie", cookie)
	//headers.Add("sec-websocket-key", "U+18SHVTcfkdgYpiCIx7QA==")
	//headers.Add("sec-websocket-version", "13")

	for {
		// 使用gorilla/websocket库建立连接
		conn, _, err = websocket.DefaultDialer.Dial(url, headers)
		fmt.Println("Connecting to:", url)
		if err != nil {
			// 连接失败，打印错误并等待一段时间后重试
			fmt.Println("Error connecting to WebSocket:", err)
			// time.Sleep(1 * time.Second) // 5秒重试间隔
			continue
		}
		// 连接成功，退出循环
		break
	}
	return conn, nil
}

func ping() {
	remoteCall := func() bool {
		url := "https://api-worker.noscription.org/indexer/deployEvent?tick=noss"
		req, _ := http.NewRequest("GET", url, nil)
		// 设置HTTP Header
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0")
		req.Header.Set("Sec-ch-ua", "\"Not A(Brand\";v=\"99\", \"Microsoft Edge\";v=\"121\", \"Chromium\";v=\"121\"")
		req.Header.Set("Sec-ch-ua-mobile", "?0")
		req.Header.Set("Sec-ch-ua-platform", "\"Windows\"")
		req.Header.Set("Sec-fetch-dest", "empty")
		req.Header.Set("Sec-fetch-mode", "cors")
		req.Header.Set("Sec-fetch-site", "same-site")

		// 发送请求
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Error sending request: %v", err)
		}
		defer resp.Body.Close()
		//fmt.Println("Response Status:", resp.Status)
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)

		if resp.Status == "200 OK" {
			log.Println("ping response: ", bodyString)
			return true
		} else {
			log.Println("ping response: ", resp.Status)
			return false
		}
	}
	for remoteCall() != true {
		log.Println("ping failed, retry after 5 seconds")
		time.Sleep(5 * time.Second)
	}
	log.Println("ping success")
}

func main() {
	initEnv()
	go refreshEnv()

	ping()
	blockChan := make(chan BlockInfo)
	go getEvent()
	go syncBlockInfo(blockChan)
	go syncBlockWss()
	//go startMine(ctx, blockChan)
	select {}
}
