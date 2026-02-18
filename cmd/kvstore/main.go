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
	err := myStore.Set("Name", "Ravi")
	err2 := myStore.Set("age", "abcd")
	err3 := myStore.Set("random", "efgh")
	if !err {
		fmt.Println("Capacity Exhausted")
	}
	if !err2 {
		value, _ := myStore.Get("age")
		fmt.Println("Value of 2 = ", value)
		fmt.Println("Capacity Exhausted inserting 2")
	}
	if !err3 {
		fmt.Println("Capacity Exhausted inserting 3")
	}
	value, newerr := myStore.Get("random")
	fmt.Println(" last item and it's error := ", value, newerr)

}
