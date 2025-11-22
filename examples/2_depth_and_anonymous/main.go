package main

import (
	"fmt"

	gj "github.com/JieBaiYou/groupjson"
)

type Meta struct {
	Version int `json:"version" groups:"public,admin"`
}

type Base struct {
	ID   int    `json:"id" groups:"public,admin"`
	Name string `json:"name" groups:"public,admin"`
}

type Item struct {
	Base
	Meta
	Child *Item `json:"child,omitempty" groups:"admin"`
}

func main() {
	it := Item{Base: Base{ID: 1, Name: "root"}, Meta: Meta{Version: 1}, Child: &Item{Base: Base{ID: 2, Name: "child"}}}

	// 无深度限制
	b, _ := gj.NewEncoder().WithGroups("admin").Marshal(it)
	fmt.Println("full:", string(b))

	// 深度限制 - 超过深度会报错
	_, err := gj.NewEncoder().WithGroups("admin").WithMaxDepth(1).Marshal(it)
	if err != nil {
		fmt.Println("depth=1 error:", err)
	}
}

// full: {"child":{"id":2,"name":"child","version":0},"id":1,"name":"root","version":1}
// depth=1 error: groupjson: exceeded maximum recursion depth
