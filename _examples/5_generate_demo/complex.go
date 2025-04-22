package main

import (
	"fmt"
	"time"

	"github.com/JieBaiYou/groupjson"
)

// 基础实体类型 - 通常作为匿名字段嵌入
type BaseEntity struct {
	ID        int       `json:"id" groups:"base,admin"`
	CreatedAt time.Time `json:"createdAt" groups:"base,admin"`
	UpdatedAt time.Time `json:"updatedAt" groups:"base,admin"`
}

// 地址类型 - 常规嵌套结构体
type Address struct {
	Street     string `json:"street" groups:"detail,admin"`
	City       string `json:"city" groups:"basic,detail,admin"`
	PostalCode string `json:"postalCode" groups:"detail,admin"`
	Country    string `json:"country" groups:"basic,detail,admin"`
}

// 制造商信息 - 常规嵌套结构体
type Manufacturer struct {
	Name    string `json:"name" groups:"basic,admin"`
	Phone   string `json:"phone,omitempty" groups:"detail,admin"`
	Website string `json:"website,omitempty" groups:"basic,admin"`
}

// 如果能使用代码生成, 可以添加下面的注释:
// //go:generate groupjson -type=ComplexProduct -source=complex.go
type ComplexProduct struct {
	// 嵌入基础实体字段 (匿名嵌套)
	BaseEntity

	// 基本产品信息
	Name        string   `json:"name" groups:"basic,admin"`
	Description string   `json:"description" groups:"detail,admin"`
	Price       float64  `json:"price" groups:"basic,admin"`
	Tags        []string `json:"tags,omitempty" groups:"basic,admin"`

	// 常规嵌套结构体
	Manufacturer Manufacturer `json:"manufacturer" groups:"detail,admin"`
	Origin       Address      `json:"origin" groups:"detail,admin"`

	// 非公开信息
	Cost       float64 `json:"cost" groups:"internal,admin"`
	StockLevel int     `json:"stockLevel,omitzero" groups:"inventory,admin"`
	SKU        string  `json:"sku" groups:"inventory,admin"`
}

// 为ComplexProduct类型添加方便的方法, 模拟代码生成的效果
func (p ComplexProduct) MarshalWithGroups(groups ...string) ([]byte, error) {
	return groupjson.MarshalWithGroups(p, groups...)
}

func (p ComplexProduct) MarshalWithGroupsOptions(opts groupjson.Options, groups ...string) ([]byte, error) {
	return groupjson.MarshalWithGroupsOptions(opts, p, groups...)
}

// 该函数展示如何使用ComplexProduct的序列化方法
func demoComplexProduct() {
	// 创建示例数据
	now := time.Now()
	product := ComplexProduct{
		BaseEntity: BaseEntity{
			ID:        2001,
			CreatedAt: now.Add(-30 * 24 * time.Hour), // 30天前创建
			UpdatedAt: now.Add(-2 * 24 * time.Hour),  // 2天前更新
		},
		Name:        "高级无人机",
		Description: "专业级航拍无人机, 4K高清摄像, 30分钟续航",
		Price:       5999.00,
		Tags:        []string{"电子产品", "航拍设备", "无人机"},
		Manufacturer: Manufacturer{
			Name:    "科技未来有限公司",
			Phone:   "400-123-4567",
			Website: "https://example.com/future-tech",
		},
		Origin: Address{
			Street:     "创新路888号",
			City:       "深圳",
			PostalCode: "518000",
			Country:    "中国",
		},
		Cost:       3200.50,
		StockLevel: 15,
		SKU:        "DRONE-PRO-4K",
	}

	// 基本信息视图 (包含基本字段和匿名嵌套的BaseEntity字段)
	basicJSON, err := product.MarshalWithGroups("basic", "base")
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	fmt.Println("基本信息视图 (basic + base):")
	fmt.Println(string(basicJSON))

	// 详细信息视图 (包含详细信息和嵌套结构体)
	detailJSON, err := product.MarshalWithGroups("basic", "detail")
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	fmt.Println("\n详细信息视图 (basic + detail):")
	fmt.Println(string(detailJSON))

	// 管理员视图 (包含所有字段)
	// 对比下面两种方式:

	// 1. 使用我们的方法
	adminJSON1, err := product.MarshalWithGroups("admin")
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	fmt.Println("\n管理员视图 (使用MarshalWithGroups方法):")
	fmt.Println(string(adminJSON1))

	// 2. 使用反射API
	adminJSON2, err := groupjson.New().
		WithGroups("admin").
		Marshal(product)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	fmt.Println("\n管理员视图 (使用反射API):")
	fmt.Println(string(adminJSON2))

	// 提示: 生成的代码通常比反射API更快, 特别是对于频繁序列化的场景
}
