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

	// 设置一个较小的最大深度
	g := groupjson.New().
		WithGroups("base").
		WithMaxDepth(3)

	_, err := g.Marshal(root)
	if err != nil {
		fmt.Printf("预期错误: %v\n", err)
		fmt.Println("检查错误信息是否包含深度信息: ", contains(err.Error(), "depth"))
		fmt.Println("检查错误信息是否包含路径信息: ", contains(err.Error(), "path") || contains(err.Error(), "NextLevel"))
	} else {
		fmt.Println("错误: 预期应该返回深度超限错误，但没有返回错误")
	}

	// 测试场景2: 循环引用检测
	fmt.Println("\n2. 测试循环引用检测:")
	a := &CircularA{Name: "A"}
	b := &CircularB{Value: 1}
	a.B = b
	b.A = a

	g = groupjson.New().
		WithGroups("base")

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
		WithGroups("base")

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

	// 测试场景4: 使用新的Marshal函数测试深度超限
	fmt.Println("\n4. 使用不同API测试深度超限:")
	root = createDeepNestedStructure(5) // 创建5层嵌套

	// 使用较小的最大深度
	// _, err = groupjson.MarshalWithOptions(root, 2) // 原错误代码

	// 使用已知可用的API
	g = groupjson.New().
		WithGroups("base").
		WithMaxDepth(2)

	_, err = g.Marshal(root)
	if err != nil {
		fmt.Printf("预期错误: %v\n", err)
		fmt.Println("检查错误信息是否包含深度信息: ", contains(err.Error(), "depth"))
		fmt.Println("检查错误信息是否包含路径信息: ", contains(err.Error(), "path") || contains(err.Error(), "Level"))
	} else {
		fmt.Println("错误: 预期应该返回深度超限错误，但没有返回错误")
	}
}

// 辅助函数：检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return strings.Contains(
		strings.ToLower(s),
		strings.ToLower(substr),
	)
}
