package main

import (
	"fmt"
	"goredis/internal/store"
)

func main() {
	myStore := store.NewStore()
	myStore.Set("Name", "Ravi")
	fmt.Println(myStore.Get("Name"))
	fmt.Println(myStore.Delete("Name"))
	fmt.Println(myStore.Get("Name"))

}
