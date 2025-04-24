package main

import (
	"fmt"

	"github.com/JieBaiYou/groupjson"
)

// BaseInfo 最基础信息
type BaseInfo struct {
	ID   int    `json:"id" groups:"public,admin"`
	Name string `json:"name" groups:"public,admin"`
}

// DetailInfo 详细信息
type DetailInfo struct {
	Description string `json:"description" groups:"admin"`
	Category    string `json:"category" groups:"admin"`
}

// MetaInfo 元信息
type MetaInfo struct {
	Version int      `json:"version" groups:"admin"`
	Tags    []string `json:"tags" groups:"admin"`
}

// SubItem 子项
type SubItem struct {
	Title    string `json:"title" groups:"admin"`
	Priority int    `json:"priority" groups:"admin"`
}

// ComplexItem 复杂嵌套结构, 包含多层匿名嵌套
type ComplexItem struct {
	// 匿名嵌套第一层
	BaseInfo
	// 匿名嵌套第二层
	DetailInfo
	// 匿名嵌套第三层
	MetaInfo

	// 普通嵌套属性
	Children []SubItem `json:"children" groups:"admin"`
	// 指向父项的指针
	Parent *ComplexItem `json:"parent,omitempty" groups:"admin"`
}

func main() {
	// 创建有层次结构但没有循环引用的对象
	child := &ComplexItem{
		BaseInfo: BaseInfo{
			ID:   101,
			Name: "子项目",
		},
		DetailInfo: DetailInfo{
			Description: "这是一个子项目",
			Category:    "子类别",
		},
		MetaInfo: MetaInfo{
			Version: 1,
			Tags:    []string{"子标签"},
		},
		Children: []SubItem{
			{Title: "子任务1", Priority: 1},
			{Title: "子任务2", Priority: 2},
		},
	}

	parent := ComplexItem{
		BaseInfo: BaseInfo{
			ID:   100,
			Name: "父项目",
		},
		DetailInfo: DetailInfo{
			Description: "这是一个父项目",
			Category:    "父类别",
		},
		MetaInfo: MetaInfo{
			Version: 2,
			Tags:    []string{"父标签1", "父标签2"},
		},
		Children: []SubItem{
			{Title: "任务1", Priority: 5},
			{Title: "任务2", Priority: 3},
		},
		Parent: nil, // 父项没有父项
	}

	// 设置子项的父项指针
	child.Parent = &parent

	fmt.Println("==== 测试匿名嵌套与递归深度限制 ====")

	// 完整输出, 无深度限制
	fullJSON, err := groupjson.New().
		WithGroups("admin").
		Marshal(parent)
	if err != nil {
		fmt.Println("完整输出错误:", err)
		return
	}
	fmt.Println("完整输出(无深度限制):")
	fmt.Println(string(fullJSON))

	// 深度限制为1, 应显示匿名嵌套字段和第一层普通字段
	depth1JSON, err := groupjson.New().
		WithGroups("admin").
		WithMaxDepth(1).
		Marshal(parent)
	if err != nil {
		fmt.Println("深度1错误:", err)
		return
	}
	fmt.Println("\n深度限制为1:")
	fmt.Println(string(depth1JSON))

	// 深度限制为2, 应显示更深层次
	depth2JSON, err := groupjson.New().
		WithGroups("admin").
		WithMaxDepth(2).
		Marshal(parent)
	if err != nil {
		fmt.Println("深度2错误:", err)
		return
	}
	fmt.Println("\n深度限制为2:")
	fmt.Println(string(depth2JSON))

	// 现在尝试序列化子项，应该也能看到父项引用
	fmt.Println("\n\n==== 测试子项序列化 ====")
	childJSON, err := groupjson.New().
		WithGroups("admin").
		Marshal(child)
	if err != nil {
		fmt.Println("子项序列化错误:", err)
		return
	}
	fmt.Println("子项序列化结果:")
	fmt.Println(string(childJSON))
}
