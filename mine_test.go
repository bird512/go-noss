package main

import (
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip13"
	"testing"
	"time"
)

func BenchmarkGenerateRandomString(b *testing.B) {

	for i := 0; i < b.N; i++ {
		generateRandomString(10)
	}
}

func BenchmarkGenerateRandomString2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateRandomString(10)
	}

}

func BenchmarkDifficulty(b *testing.B) {
	event := nostr.Event{
		ID:        "aaa",
		PubKey:    "asdfasdfasdf",
		CreatedAt: nostr.Now(),
		Kind:      0,
		Tags:      nil,
		Content:   "ssss",
		Sig:       "ffasasdf",
	}
	tag := nostr.Tag{"nonce", "", "strconv.Itoa(targetDifficulty)"}
	event.Tags = append(event.Tags, tag)
	for i := 0; i < b.N; i++ {
		generate1(event)
	}
}
func generate1(event nostr.Event) (int, error) {
	//event.CreatedAt = nostr.Now()
	return nip13.Difficulty(event.GetID()), nil
}

func BenchmarkDifficulty2(b *testing.B) {
	event := nostr.Event{
		ID:        "aaa",
		PubKey:    "asdfasdfasdf",
		CreatedAt: nostr.Now(),
		Kind:      0,
		Tags:      nil,
		Content:   "ssss",
		Sig:       "ffasasdf",
	}
	tag := nostr.Tag{"nonce", "", "strconv.Itoa(targetDifficulty)"}
	event.Tags = append(event.Tags, tag)
	start := time.Now()
	for i := 0; i < b.N; i++ {
		generate2(event, start)
	}
}
func generate2(event nostr.Event, start time.Time) (int, bool) {
	//event.CreatedAt = nostr.Now()
	out := time.Since(start) >= 10*time.Second
	return nip13.Difficulty(event.GetID()), out
}
