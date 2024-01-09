package main

import (
	"context"
	"log"
	"sync/atomic"
)

var lastBlockInfo atomic.Value

type BlockInfo struct {
	blockHeight uint64
	blockHash   string
}

//func getBlockInfo() (uint64, string) {
//	info := lastBlockInfo.Load().(BlockInfo)
//	return info.blockHeight, info.blockHash
//}

func syncBlockInfo(blockChain chan BlockInfo) {
	for {
		header, err := blockClient.HeaderByNumber(context.Background(), nil)
		if err != nil {
			log.Fatalf("无法获取最新区块号: %v", err)
		}
		info := BlockInfo{
			blockHeight: header.Number.Uint64(),
			blockHash:   header.Hash().Hex(),
		}
		//blockChain <- info
		//
		last := lastBlockInfo.Load() //.(BlockInfo)
		if last == nil || last.(BlockInfo).blockHeight != info.blockHeight {
			lastBlockInfo.Store(info)
			blockChain <- info
		}
	}
}
