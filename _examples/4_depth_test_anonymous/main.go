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
	// 指向自身的指针, 形成深层嵌套
	Parent *ComplexItem `json:"parent,omitempty" groups:"admin"`
}

func main() {
	// 创建一个有多层匿名嵌套和自引用的复杂结构
	child := ComplexItem{
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
		Parent: &child, // 形成嵌套引用
	}

	// 为了形成更复杂的循环引用, 设置child的Parent指向parent
	child.Parent = &parent

	fmt.Println("==== 测试匿名嵌套与递归深度限制 ====")

	// 完整输出, 无深度限制
	fullJSON, _ := groupjson.New().
		WithGroups("admin").
		Marshal(parent)
	fmt.Println("完整输出(无深度限制):")
	fmt.Println(string(fullJSON))

	// 深度限制为0, 只显示基本字段
	depth0JSON, _ := groupjson.New().
		WithGroups("admin").
		WithMaxDepth(0).
		Marshal(parent)
	fmt.Println("\n深度限制为0:")
	fmt.Println(string(depth0JSON))

	// 深度限制为1, 应显示匿名嵌套字段和第一层普通字段
	depth1JSON, _ := groupjson.New().
		WithGroups("admin").
		WithMaxDepth(1).
		Marshal(parent)
	fmt.Println("\n深度限制为1:")
	fmt.Println(string(depth1JSON))

	// 深度限制为2, 应显示更深层次
	depth2JSON, _ := groupjson.New().
		WithGroups("admin").
		WithMaxDepth(2).
		Marshal(parent)
	fmt.Println("\n深度限制为2:")
	fmt.Println(string(depth2JSON))
}
