package main

import (
	"fmt"

	gj "github.com/JieBaiYou/groupjson"
)

type User struct {
	ID   int    `json:"id" groups:"public,admin"`
	Name string `json:"name" groups:"public,admin"`
}

func main() {
	users := []User{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}}
	// AllowSlice 已移除，默认支持
	b, _ := gj.NewEncoder().WithGroups("public").Marshal(users)
	fmt.Println("slice:", string(b))

	in := map[string]any{"u": User{ID: 3, Name: "C"}, "note": "hi"}
	// AllowMap 已移除，默认支持
	b2, _ := gj.NewEncoder().WithGroups("public").Marshal(in)
	fmt.Println("map:", string(b2))
}

// slice: [{"id":1,"name":"A"},{"id":2,"name":"B"}]
// map: {"note":"hi","u":{"id":3,"name":"C"}}
