package main

import (
	"encoding/json"
	"fmt"
	"log"
)

func getEvent() {
	loopEvent := func() {
		defer func() {
			if err := recover(); err != nil {
				log.Println("getEvent", err)
			}
		}()
		wssAddr := "wss://report-worker-2.noscription.org"
		wssAddr = "report-worker-ng.noscription.org"
		wssAddr = "report-worker-2.noscription.org"
		// relayUrl := "wss://relay.noscription.org/"
		var err error
		c, err := connectToWSS(wssAddr)
		if err != nil {
			panic(err)
		}
		defer c.Close()

		func() {
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
				last := messageId.Load()
				if last != nil && last.(string) == messageDecode.EventId {
					continue
				}
				messageId.Store(messageDecode.EventId)
				MineOneEvent(messageDecode.EventId)
			}

		}()
	}
	for {
		loopEvent()
	}
}
