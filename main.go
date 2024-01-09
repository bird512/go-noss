package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/nbd-wtf/go-nostr"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
)

//var sk string
//var pk string

var walletFile = "wallets.json"
var arbRpcUrl string
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
var counter Counter

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
	cookie = os.Getenv("cookie")
	numberOfWorkers, _ = strconv.Atoi(os.Getenv("numberOfWorkers"))
	interval, _ = strconv.ParseInt(os.Getenv("interval"), 10, 64)
	if interval < 1 {
		interval = 100
	}
	log.Println("interval = ", interval)
	//messageCache = expirable.NewLRU[string, string](5, nil, time.Second*10)

	counter = Counter{val: 0}
	blockClient, err = ethclient.Dial(arbRpcUrl)
	if err != nil {
		log.Fatalf("无法连接到Arbitrum节点: %v", err)
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

func main() {
	initEnv()
	blockChan := make(chan BlockInfo)

	wssAddr := "wss://report-worker-2.noscription.org"
	wssAddr = "report-worker-ng.noscription.org"
	wssAddr = "report-worker-2.noscription.org"
	// relayUrl := "wss://relay.noscription.org/"
	ctx := context.Background()

	var err error

	c, err := connectToWSS(wssAddr)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}

			var messageDecode Message
			if err := json.Unmarshal(message, &messageDecode); err != nil {
				fmt.Println(err)
				continue
			}
			messageId.Store(messageDecode.EventId)
		}

	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go syncBlockInfo(blockChan)
	go startMine(ctx, blockChan)
	select {}
}
