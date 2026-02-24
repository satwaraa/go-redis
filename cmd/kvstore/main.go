package main

import (
	"log"
	"memstash/env"
	"memstash/internal/cli"
	"memstash/internal/server"
	"memstash/internal/store"
	"time"
)

func main() {
	dotenvs := env.LoadEnv()

	myStore := store.NewStore(*dotenvs.Capacity)
	snapshotPath := "memstash_data.json"
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

	// Start HTTP REST API server in background
	httpSrv := server.NewHTTPServer(myStore, *dotenvs.Http_port)
	go httpSrv.Start()

	c := cli.NewCLI(myStore)
	c.Start()

	// If CLI exited via Ctrl+C, wait for save to finish
	select {
	case <-done:
	default:
	}
}
