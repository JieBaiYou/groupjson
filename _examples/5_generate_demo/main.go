package main

import (
	"fmt"

	"github.com/JieBaiYou/groupjson"
)

//go:generate groupjson -type=Product -source=main.go
type Product struct {
	ID          int      `json:"id" groups:"public,admin"`
	Name        string   `json:"name" groups:"public,admin"`
	Price       float64  `json:"price" groups:"public,admin"`
	Description string   `json:"description" groups:"detail,admin"`
	Cost        float64  `json:"cost" groups:"internal,admin"`
	SKU         string   `json:"sku" groups:"inventory,admin"`
	Tags        []string `json:"tags,omitempty" groups:"public,admin"`
	StockLevel  int      `json:"stockLevel,omitzero" groups:"inventory,admin"`
}

// 为Product类型添加方便的方法, 模拟代码生成的效果
func (p Product) MarshalWithGroups(groups ...string) ([]byte, error) {
	return groupjson.Marshal(p, groups...)
}

func (p Product) MarshalWithOptions(opts groupjson.Options, groups ...string) ([]byte, error) {
	return groupjson.MarshalWithOptions(opts, p, groups...)
}

func demoSimpleProduct() {
	// 创建示例产品数据
	product := Product{
		ID:          1001,
		Name:        "智能手表",
		Price:       1299.99,
		Description: "高性能智能手表, 支持多种运动模式和健康监测",
		Cost:        699.50,
		SKU:         "SW-1001-BLK",
		Tags:        []string{"电子产品", "可穿戴设备", "智能手表"},
		StockLevel:  42,
	}

	// 使用我们手动添加的方法 - 公开视图 (仅包含 public 组的字段)
	publicJSON, err := product.MarshalWithGroups("public")
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	fmt.Println("公开视图 (public):")
	fmt.Println(string(publicJSON))

	// 使用我们手动添加的方法 - 详情视图 (包含 public 和 detail 组的字段)
	detailJSON, err := product.MarshalWithGroups("public", "detail")
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	fmt.Println("\n详情视图 (public + detail):")
	fmt.Println(string(detailJSON))

	// 使用我们手动添加的方法 - 库存管理视图 (包含 inventory 组的字段)
	// 使用自定义选项包装结果
	opts := groupjson.Options{
		TopLevelKey: "data",
		GroupMode:   groupjson.ModeOr, // 默认是OR模式, 这里显式指定
	}
	inventoryJSON, err := product.MarshalWithOptions(opts, "inventory")
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	fmt.Println("\n库存视图 (inventory, 带顶层键):")
	fmt.Println(string(inventoryJSON))

	// 使用我们手动添加的方法 - 管理员视图 (包含 admin 组的所有字段)
	adminJSON, err := product.MarshalWithGroups("admin")
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	fmt.Println("\n管理员视图 (admin):")
	fmt.Println(string(adminJSON))

	// 使用反射 API - 内部信息视图 (包含 internal 组的字段)
	internalJSON, err := groupjson.New().
		WithGroups("internal").
		Marshal(product)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	fmt.Println("\n内部视图 (internal, 使用反射 API):")
	fmt.Println(string(internalJSON))
}

func main() {
	fmt.Println("=============== 简单产品示例 ===============")
	demoSimpleProduct()

	fmt.Println("\n\n=============== 复杂产品示例 ===============")
	demoComplexProduct()

	fmt.Println("\n\n提示: 这个示例展示了如何使用 GroupJSON 库来按组序列化结构体。")
	fmt.Println("实际项目中可以通过代码生成获得更高的性能。")
}
