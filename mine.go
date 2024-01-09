package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip13"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

var charset = "abcdefghijklmnopqrstuvwxyz0123456789" // 字符集

func generateRandomString(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := 0; i < length; i++ {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}

func generateRandomString2(length int) string {
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func Generate(event nostr.Event, targetDifficulty int) (nostr.Event, error) {
	tag := nostr.Tag{"nonce", "", strconv.Itoa(targetDifficulty)}
	event.Tags = append(event.Tags, tag)
	//start := time.Now()
	//for {
	nonce, err := generateRandomString(10)
	if err != nil {
		fmt.Println(err)
	}
	tag[1] = nonce
	event.CreatedAt = nostr.Now()
	if nip13.Difficulty(event.GetID()) >= targetDifficulty {
		// fmt.Print(time.Since(start))
		return event, nil
	} else {
		return event, ErrGenerateTimeout
	}
	//if time.Since(start) >= 10*time.Second {
	//	return event, ErrGenerateTimeout
	//}
	//}
}

func startMine(ctx context.Context, blockChain chan BlockInfo) {
	for {
		select {
		case info := <-blockChain:
			//log.Println("Received Block:", info)
			msgId, ok := messageId.Load().(string)
			if !ok {
				log.Println("msgId is not ready")
				time.Sleep(1 * time.Second)
				continue
			}
			go mine(info, msgId, wallets[0])
		case <-ctx.Done():
			return
		}
	}
}

func mine(blockInfo BlockInfo, messageId string, wallet Wallet) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startTime := time.Now()
	blockNumber := blockInfo.blockHeight
	blockHash := blockInfo.blockHash
	//log.Println("blockNumber: ", blockNumber, "blockHash: ", blockHash, "messageId: ", messageId)
	replayUrl := "wss://relay.noscription.org/"
	difficulty := 21

	// Create a channel to signal the finding of a valid nonce
	foundEvent := make(chan nostr.Event, 1)
	//doneEvent := make(chan nostr.Event, 1)
	notFound := make(chan nostr.Event, 1)
	// Create a channel to signal all workers to stop
	content := "{\"p\":\"nrc-20\",\"op\":\"mint\",\"tick\":\"noss\",\"amt\":\"10\"}"

	ev := nostr.Event{
		Content:   content,
		CreatedAt: nostr.Now(),
		ID:        "",
		Kind:      nostr.KindTextNote,
		PubKey:    wallet.PublicKey,
		Sig:       "",
		Tags:      nil,
	}
	ev.Tags = ev.Tags.AppendUnique(nostr.Tag{"p", "9be107b0d7218c67b4954ee3e6bd9e4dba06ef937a93f684e42f730a0c3d053c"})
	ev.Tags = ev.Tags.AppendUnique(nostr.Tag{"e", "51ed7939a984edee863bfbb2e66fdc80436b000a8ddca442d83e6a2bf1636a95", replayUrl, "root"})
	ev.Tags = ev.Tags.AppendUnique(nostr.Tag{"e", messageId, replayUrl, "reply"})
	ev.Tags = ev.Tags.AppendUnique(nostr.Tag{"seq_witness", strconv.Itoa(int(blockNumber)), blockHash})
	// Start multiple worker goroutines

	pow := func(ctx context.Context, cancel context.CancelFunc, evCopy nostr.Event) {
		//log.Println("start pow for ", blockNumber)
		counter.Inc()
		defer counter.Dec()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				evCopy, err := Generate(evCopy, difficulty)
				if err == nil {
					cancel()
					foundEvent <- evCopy
				}
				if time.Since(startTime) >= 1*time.Second {
					//notFound <- evCopy
					//log.Println("timeout1111", blockNumber)
					cancel()
				}
			}
		}

	}
	maxCount := numberOfWorkers - counter.Value()
	//fmt.Println("maxCount: ", maxCount)
	if maxCount >= numberOfWorkers*3/4 {
		for i := 0; i < maxCount; i++ {
			go pow(ctx, cancel, ev)
		}
	} else {
		//log.Println("no job to run", blockNumber)
		return
	}

	select {
	case <-notFound:
		log.Println("not found", blockNumber)
	case evNew := <-foundEvent:
		evNew.Sign(wallet.PrivateKey)
		spendTime := time.Since(startTime)
		evNewInstance := EV{
			Sig:       evNew.Sig,
			Id:        evNew.ID,
			Kind:      evNew.Kind,
			CreatedAt: evNew.CreatedAt,
			Tags:      evNew.Tags,
			Content:   evNew.Content,
			PubKey:    evNew.PubKey,
		}
		// 将ev转为Json格式
		eventJSON, err := json.Marshal(evNewInstance)
		if err != nil {
			log.Fatal(err)
		}

		wrapper := map[string]json.RawMessage{
			"event": eventJSON,
		}

		// 将包装后的对象序列化成JSON
		wrapperJSON, err := json.Marshal(wrapper)
		if err != nil {
			log.Fatalf("Error marshaling wrapper: %v", err)
		}

		url := "https://api-worker.noscription.org/inscribe/postEvent"
		// fmt.Print(bytes.NewBuffer(wrapperJSON))
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(wrapperJSON)) // 修改了弱智项目方不识别美化Json的bug
		if err != nil {
			log.Fatalf("Error creating request: %v", err)
		}

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
			log.Println("spend: [", spendTime, "]!!!!!!!!!!!!!!!!!!!!!published to:", evNew.ID, messageId, blockNumber, "res: ", bodyString)
		} else {
			log.Println("spend: [", spendTime, "]!!!!!!!!!!!!!!!!!!!!!published to:", evNew.ID, messageId, blockNumber, "error: ", resp.Status)
		}
		//case <-ctx.Done():
		//	fmt.Print("done")
	}

}
