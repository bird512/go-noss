package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
)

var lastBlockInfo atomic.Value

type BlockInfo struct {
	BlockNumber uint64 `json:"BlockNumber"`
	BlockHash   string `json:"BlockHash"`
}

//

func getBlockInfo() *BlockInfo {
	last, ok := lastBlockInfo.Load().(BlockInfo)
	if !ok {
		return nil
	}
	return &last
}

//	{
//	   "BlockNumber": 169084060,
//	   "BlockHash": "0xcc38c7414dc18683d083388786ca0aaeb7bdff858a097767b0bd9ce3dbcb287e"
//	}
func syncBlockWss() {
	wssAddr := "report-worker-arbstate.noscription.org"
	c, err := connectToWSS(wssAddr)
	if err != nil {
		panic(err)
	}
	defer c.Close()
	fmt.Println("connect to wss success", wssAddr)
	func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read syncBlockWss:", err)
				break
			}

			var info BlockInfo
			if err := json.Unmarshal(message, &info); err != nil {
				fmt.Println(err)
				continue
			}
			last := lastBlockInfo.Load() //.(BlockInfo)
			if last == nil || last.(BlockInfo).BlockNumber < info.BlockNumber {

				lastBlockInfo.Store(info)
			}

		}

	}()
}
func syncBlockInfo(blockChain chan BlockInfo) {
	getBlock := func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println("SyncBlockInfo Recovered. Error:\n", r)
			}
		}()
		header, err := blockClient.HeaderByNumber(context.Background(), nil)
		if err != nil {
			log.Println("无法获取最新区块号: %v", err)
		}
		info := BlockInfo{
			BlockNumber: header.Number.Uint64(),
			BlockHash:   header.Hash().Hex(),
		}
		//log.Println(info.BlockNumber)
		last := lastBlockInfo.Load() //.(BlockInfo)
		if last == nil || last.(BlockInfo).BlockNumber < info.BlockNumber {

			lastBlockInfo.Store(info)
		}
	}
	for {
		getBlock()
		//header, err := blockClient.HeaderByNumber(context.Background(), nil)
		//if err != nil {
		//	log.Fatalf("无法获取最新区块号: %v", err)
		//}
		//info := BlockInfo{
		//	BlockNumber: header.Number.Uint64(),
		//	BlockHash:   header.Hash().Hex(),
		//}
		////log.Println(info.BlockNumber)
		//lastBlockInfo.Store(info)
		//////blockChain <- info
		//////
		////last := lastBlockInfo.Load() //.(BlockInfo)
		////if last == nil || last.(BlockInfo).BlockNumber != info.BlockNumber {
		////	lastBlockInfo.Store(info)
		////	//blockChain <- info
		////}
		//////time.Sleep(time.Duration(interval) * time.Millisecond)
	}
}
