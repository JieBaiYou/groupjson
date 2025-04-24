package main

import (
	"fmt"
	"strings"

	"github.com/JieBaiYou/groupjson"
)

// 创建一个嵌套结构体来测试路径错误
type Level struct {
	Value     int    `json:"value" groupjson:"base"`
	Name      string `json:"name" groupjson:"base"`
	NextLevel *Level `json:"next_level,omitempty" groupjson:"base"`
}

// 创建循环引用结构体
type CircularA struct {
	Name string     `json:"name" groupjson:"base"`
	B    *CircularB `json:"b,omitempty" groupjson:"base"`
}

type CircularB struct {
	Value int        `json:"value" groupjson:"base"`
	A     *CircularA `json:"a,omitempty" groupjson:"base"`
}

// 包含不支持类型的结构体
type UnsupportedType struct {
	Name string      `json:"name" groupjson:"base"`
	Ch   chan string `json:"channel" groupjson:"base"` // JSON不支持channel类型
}

// 创建嵌套结构
func createDeepNestedStructure(depth int) *Level {
	if depth <= 0 {
		return nil
	}

	current := &Level{
		Value: depth,
		Name:  "Level" + string(rune('A'+depth-1)),
	}

	if depth > 1 {
		current.NextLevel = createDeepNestedStructure(depth - 1)
	}

	return current
}

func main() {
	fmt.Println("=== GroupJSON 错误路径测试 ===")

	// 测试场景1: 超出最大递归深度
	fmt.Println("\n1. 测试超出最大递归深度:")
	root := createDeepNestedStructure(10) // 创建10层嵌套
	fmt.Printf("原始结构深度: %d\n", 10)

	// 设置一个较小的最大深度
	maxDepth := 3
	g := groupjson.New().
		WithGroups("base").
		WithTagKey("groupjson"). // 设置正确的标签名称
		WithMaxDepth(maxDepth)

	result1, err := g.Marshal(root)
	if err != nil {
		fmt.Printf("意外错误: %v\n", err)
	} else {
		fmt.Println("成功截断深度超限结构")
		resultStr := string(result1)
		fmt.Printf("结果: %s\n", resultStr)

		// 检查嵌套深度 - 注意第一层不包含"next_level"
		levels := countNestedLevels(resultStr)
		fmt.Printf("实际嵌套层数: %d (最大深度设置为%d)\n", levels, maxDepth)
		fmt.Printf("深度限制是否生效: %v\n", levels < 10)
	}

	// 测试场景2: 循环引用检测
	fmt.Println("\n2. 测试循环引用检测:")
	a := &CircularA{Name: "A"}
	b := &CircularB{Value: 1}
	a.B = b
	b.A = a

	g = groupjson.New().
		WithGroups("base").
		WithTagKey("groupjson")

	result, err := g.Marshal(a)
	if err != nil {
		fmt.Printf("意外错误: %v\n", err)
	} else {
		fmt.Println("序列化循环引用结构成功")
		fmt.Printf("结果: %s\n", string(result))
		// 验证输出是否包含期望的内容
		fmt.Println("输出包含'name': ", contains(string(result), `"name"`))
		fmt.Println("输出包含'value': ", contains(string(result), `"value"`))
	}

	// 测试场景3: 包含不支持的值类型
	fmt.Println("\n3. 测试不支持的值类型:")
	obj := UnsupportedType{
		Name: "Test",
		Ch:   make(chan string),
	}

	g = groupjson.New().
		WithGroups("base").
		WithTagKey("groupjson")

	_, err = g.Marshal(obj)
	if err != nil {
		fmt.Printf("预期错误: %v\n", err)
		fmt.Println("错误信息包含'channel'或'Ch': ",
			contains(err.Error(), "channel") || contains(err.Error(), "Ch"))
		fmt.Println("错误信息包含字段路径: ",
			contains(err.Error(), "path") || contains(err.Error(), "UnsupportedType"))
	} else {
		fmt.Println("错误: 预期应该返回类型错误，但没有返回错误")
	}

	// 测试场景4: 使用不同API测试深度超限
	fmt.Println("\n4. 使用不同API测试深度超限:")
	root = createDeepNestedStructure(5) // 创建5层嵌套
	fmt.Printf("原始结构深度: %d\n", 5)

	// 使用已知可用的API
	maxDepth = 2
	g = groupjson.New().
		WithGroups("base").
		WithTagKey("groupjson"). // 设置正确的标签名称
		WithMaxDepth(maxDepth)

	result4, err := g.Marshal(root)
	if err != nil {
		fmt.Printf("意外错误: %v\n", err)
	} else {
		fmt.Println("成功截断深度超限结构")
		resultStr := string(result4)
		fmt.Printf("结果: %s\n", resultStr)

		// 检查嵌套深度
		levels := countNestedLevels(resultStr)
		fmt.Printf("实际嵌套层数: %d (最大深度设置为%d)\n", levels, maxDepth)
		fmt.Printf("深度限制是否生效: %v\n", levels < 5)
	}
}

// 辅助函数：检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return strings.Contains(
		strings.ToLower(s),
		strings.ToLower(substr),
	)
}

// 辅助函数：计算嵌套层数
func countNestedLevels(s string) int {
	// 计算嵌套的"next_level"出现次数 + 1 (根层级)
	return strings.Count(s, "next_level") + 1
}
