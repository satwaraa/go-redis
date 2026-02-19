package main

import (
	"fmt"
	"goredis/env"
	"goredis/internal/store"
)

func main() {
	dotenvs := env.LoadEnv()
	fmt.Println("This is something ", *dotenvs.Capacity, dotenvs.Memory)

	myStore := store.NewStore(*dotenvs.Capacity)
	myStore.Set("name", "Ravi")
	myStore.Set("age", "Ravi")
	myStore.Set("random", "Ravi")
	myStore.Set("random2", "Ravi")
	fmt.Println(myStore.Get("name"))
	fmt.Println(myStore.Get("age"))
	fmt.Println(myStore.Get("random"))
	fmt.Println(myStore.Get("random2"))

}
