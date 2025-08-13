package main

import (
	"fmt"

	gj "github.com/JieBaiYou/groupjson"
)

type User struct {
	ID    int    `json:"id" groups:"public,admin"`
	Name  string `json:"name" groups:"public,admin"`
	Email string `json:"email" groups:"admin"`
}

func main() {
	u := User{ID: 1, Name: "Alice", Email: "a@example.com"}

	b, _ := gj.NewEncoder().WithGroups("public").Marshal(u)
	fmt.Println("public:", string(b))

	b, _ = gj.NewEncoder().WithGroups("admin").WithTopLevelKey("data").Marshal(u)
	fmt.Println("admin:", string(b))
}

// public: {"id":1,"name":"Alice"}
// admin: {"data":{"email":"a@example.com","id":1,"name":"Alice"}}
