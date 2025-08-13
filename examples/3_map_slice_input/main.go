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
	m, _ := gj.NewEncoder().AllowSlice(true).WithGroups("public").MarshalToMap(users)
	fmt.Println("slice->map:", m)

	in := map[string]any{"u": User{ID: 3, Name: "C"}, "note": "hi"}
	m2, _ := gj.NewEncoder().AllowMap(true).WithGroups("public").MarshalToMap(in)
	fmt.Println("map->map:", m2)
}

// slice->map: map[data:[map[id:1 name:A] map[id:2 name:B]]]
// map->map: map[note:hi u:map[id:3 name:C]]
