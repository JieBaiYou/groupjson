package main

import (
	"fmt"

	"github.com/JieBaiYou/groupjson"
)

// 递归嵌套结构
type Nested struct {
	Name  string  `json:"name" groups:"public,admin"`
	Value int     `json:"value" groups:"public,admin"`
	Child *Nested `json:"child,omitempty" groups:"admin"`
}

func main() {
	// 创建一个深度嵌套的结构
	data := Nested{
		Name:  "Level 1",
		Value: 1,
		Child: &Nested{
			Name:  "Level 2",
			Value: 2,
			Child: &Nested{
				Name:  "Level 3",
				Value: 3,
				Child: &Nested{
					Name:  "Level 4",
					Value: 4,
					Child: &Nested{
						Name:  "Level 5",
						Value: 5,
					},
				},
			},
		},
	}

	fmt.Println("==== 测试递归深度限制 ====")

	// 完整输出, 无深度限制
	fullJSON, err := groupjson.New().
		WithGroups("admin").
		Marshal(data)
	if err != nil {
		fmt.Println("错误:", err)
		return
	}
	fmt.Println("完整输出(无深度限制):")
	fmt.Println(string(fullJSON))

	// 深度限制为0, 只显示基本字段
	depth0JSON, _ := groupjson.New().
		WithGroups("admin").
		WithMaxDepth(0).
		Marshal(data)
	fmt.Println("\n深度限制为0:")
	fmt.Println(string(depth0JSON))

	// 深度限制为1, 只显示第一层
	depth1JSON, _ := groupjson.New().
		WithGroups("admin").
		WithMaxDepth(1).
		Marshal(data)
	fmt.Println("\n深度限制为1:")
	fmt.Println(string(depth1JSON))

	// 深度限制为2, 显示两层
	depth2JSON, _ := groupjson.New().
		WithGroups("admin").
		WithMaxDepth(2).
		Marshal(data)
	fmt.Println("\n深度限制为2:")
	fmt.Println(string(depth2JSON))
}
