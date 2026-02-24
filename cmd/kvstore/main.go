package main

import (
	"goredis/env"
	"goredis/internal/cli"
	"goredis/internal/server"
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

	// Save on shutdown (Ctrl+C)
	done := myStore.SaveOnShutdown(snapshotPath)
	// Start TTL cleaner
	myStore.StartTTLCleaner(1 * time.Minute)

	// Start TCP server in background (shares the same store)
	srv := server.NewServer(myStore, *dotenvs.Tcp_port)
	go srv.Start()

	c := cli.NewCLI(myStore)
	c.Start()

	// If CLI exited via Ctrl+C, wait for save to finish
	select {
	case <-done:
	default:
	}
}
