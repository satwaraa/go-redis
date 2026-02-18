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
	Lru := store.NewLru()
	myStore.Set(Lru, "name", "Ravi")
	myStore.Set(Lru, "age", "Ravi")
	myStore.Set(Lru, "random", "Ravi")
	myStore.Set(Lru, "random2", "Ravi")
	fmt.Println(myStore.Get(Lru, "name"))
	fmt.Println(myStore.Get(Lru, "age"))
	fmt.Println(myStore.Get(Lru, "random"))
	fmt.Println(myStore.Get(Lru, "random2"))

}
