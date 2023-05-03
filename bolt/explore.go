package main

import (
	"fmt"
	"time"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/boltdb"
)

func init() {
	boltdb.Register()
}

func main() {
	db := "/data/eni-ipam.db"
	bucket := "IPAM"

	kv, err := libkv.NewStore(
		store.BOLTDB,
		[]string{db},
		&store.Config{
			Bucket:            bucket,
			ConnectionTimeout: 10 * time.Second,
		},
	)
	if err != nil {
		fmt.Printf("Creating db failed: %v\n", err)
	}

	entries, err := kv.List("1")
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, pair := range entries {
		fmt.Printf("key=%v - value=%v\n", pair.Key, string(pair.Value))
	}
}
