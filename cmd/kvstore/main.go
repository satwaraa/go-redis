package main

import (
	"goredis/env"
	"goredis/internal/cli"
	"goredis/internal/store"
	"time"
)

func main() {
	dotenvs := env.LoadEnv()

	myStore := store.NewStore(*dotenvs.Capacity)
	// Start TTL cleaner
	myStore.StartTTLCleaner(1 * time.Minute)
	c := cli.NewCLI(myStore)
	c.Start()

}
