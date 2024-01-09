package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/joho/godotenv"
	"github.com/nbd-wtf/go-nostr"
)

var sk string
var pk string

var arbRpcUrl string

var (
	ErrDifficultyTooLow = errors.New("nip13: insufficient difficulty")
	ErrGenerateTimeout  = errors.New("nip13: generating proof of work took too long")
)

var messageCache *expirable.LRU[string, string]
var blockClient *ethclient.Client

func init1() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime) // Add this line
	log.Println("Starting...")
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	sk = os.Getenv("sk")
	pk = os.Getenv("pk")
	err = checkWallet(sk, pk)
	if err != nil {
		log.Fatalf("私钥公钥不匹配: %v", err)
		return
	}
	newPrivateKey()

	arbRpcUrl = os.Getenv("arbRpcUrl")

	messageCache = expirable.NewLRU[string, string](5, nil, time.Second*10)
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

func getBlockInfo() (uint64, string) {
	header, err := blockClient.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Fatalf("无法获取最新区块号: %v", err)
	}
	return header.Number.Uint64(), header.Hash().Hex()
}

func connectToWSS(url string) (*websocket.Conn, error) {
	var conn *websocket.Conn
	var err error
	headers := http.Header{}
	headers.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0")
	headers.Add("Origin", "https://noscription.org")
	headers.Add("Host", "report-worker-2.noscription.org")
	for {
		// 使用gorilla/websocket库建立连接
		conn, _, err = websocket.DefaultDialer.Dial(url, headers)
		fmt.Println("Connecting to wss")
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
	init1()
	wssAddr := "wss://report-worker-2.noscription.org"
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

			_, ok := messageCache.Get(messageDecode.EventId)
			// check for OK value
			if ok {
				//fmt.Println("message already saved: ", messageDecode.EventId)
			} else {
				//log.Println("recv: ", messageDecode.EventId)
				messageCache.Add(messageDecode.EventId, messageDecode.EventId)
				//chLimit <- messageDecode.EventId
				go mine(ctx, messageDecode.EventId)
			}
		}

	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	select {}
}
