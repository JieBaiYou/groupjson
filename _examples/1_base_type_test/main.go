package main

import (
	"fmt"

	"github.com/JieBaiYou/groupjson"
)

type User struct {
	ID       int    `json:"id" groups:"public,admin"`
	Name     string `json:"name" groups:"public,admin"`
	Email    string `json:"email" groups:"admin"`
	Password string `json:"password" groups:"internal"`
}

func main() {
	user := User{
		ID:       1,
		Name:     "张三",
		Email:    "zhangsan@example.com",
		Password: "secret123",
	}

	// 使用流畅 API
	publicJSON, _ := groupjson.New().
		WithGroups("public").
		Marshal(user)
	fmt.Println(string(publicJSON))
	// 输出: {"id":1,"name":"张三"}

	// 带选项的序列化
	adminJSON, _ := groupjson.New().
		WithGroups("admin", "internal").
		WithTopLevelKey("data").
		Marshal(user)
	fmt.Println(string(adminJSON))
	// 输出: {"data":{"email":"zhangsan@example.com","id":1,"name":"张三","password":"secret123"}}

	// 使用Marshal解析
	internalJSON, _ := groupjson.Default(user, "internal")
	fmt.Println(string(internalJSON))
	// 输出: {"password":"secret123"}
}
