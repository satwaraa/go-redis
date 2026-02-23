package main

import (
	"goredis/env"
	"goredis/internal/cli"
	"goredis/internal/store"
	"log"
	"time"
)

func main() {
	dotenvs := env.LoadEnv()

	myStore := store.NewStore(*dotenvs.Capacity)
	snapshotPath := "goredis_data.json"
	err := myStore.LoadSnapshot(snapshotPath)
	if err != nil {
		log.Printf("Load failed: %v\n", err)
	}

	// Enable auto-save every 5 minutes
	myStore.EnableAutoSave(snapshotPath, 1*time.Minute)

	// Save on shutdown
	myStore.SaveOnShutdown(snapshotPath)
	// Start TTL cleaner
	myStore.StartTTLCleaner(1 * time.Minute)
	c := cli.NewCLI(myStore)
	c.Start()

}
