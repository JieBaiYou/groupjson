package main

import (
	"fmt"
	"time"

	"github.com/JieBaiYou/groupjson"
)

// 基础元数据结构, 用于匿名嵌入
type Metadata struct {
	CreatedAt  time.Time `json:"created_at" groups:"public,admin"`
	UpdatedAt  time.Time `json:"updated_at" groups:"admin"`
	VisitCount int       `json:"visit_count" groups:"stats,admin"`
}

// 用户地址信息, 展示嵌套结构
type Address struct {
	Province   string `json:"province" groups:"admin,shipping"`
	City       string `json:"city" groups:"admin,shipping"`
	Detail     string `json:"detail" groups:"shipping"`
	PostalCode string `json:"postal_code" groups:"shipping"`
}

// 用户基本信息
type User struct {
	ID       int     `json:"id" groups:"public,admin"`
	Name     string  `json:"name" groups:"public,admin"`
	Email    string  `json:"email" groups:"admin,contact"`
	Password string  `json:"password" groups:"internal"`
	Address  Address `json:"address" groups:"admin,shipping"` // 嵌套结构
	Metadata         // 匿名嵌套
	Verified bool    `json:"verified" groups:"admin,public"`
	VIP      bool    `json:"vip" groups:"admin,marketing"`
}

// 文章内容, 展示多种数据类型
type Article struct {
	ID         int       `json:"id" groups:"public,admin"`
	Title      string    `json:"title" groups:"public,admin"`
	Content    string    `json:"content" groups:"public,admin"`
	Draft      bool      `json:"draft" groups:"admin,editor"`
	Tags       []string  `json:"tags" groups:"public,admin"`
	ViewCount  int       `json:"view_count" groups:"stats,admin"`
	Metadata             // 匿名嵌套
	AuthorID   int       `json:"author_id" groups:"admin"`
	Likes      int       `json:"likes" groups:"public,stats"`
	RelatedIDs []int     `json:"related_ids" groups:"admin,recommendation"`
	Comments   []Comment `json:"comments" groups:"public,admin"` // 嵌套切片
}

// 评论内容
type Comment struct {
	ID        int       `json:"id" groups:"public,admin"`
	Content   string    `json:"content" groups:"public,admin"`
	CreatedAt time.Time `json:"created_at" groups:"public,admin"`
	UserID    int       `json:"user_id" groups:"admin"`
	Approved  bool      `json:"approved" groups:"admin,moderator"`
}

// 创建一个包装结构体
type APIResponse struct {
	User    User    `json:"user" groups:"public,admin,stats"`
	Article Article `json:"article" groups:"public,admin,stats"`
}

func main() {
	// 创建测试数据
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	user := User{
		ID:       1,
		Name:     "张三",
		Email:    "zhangsan@example.com",
		Password: "secret123",
		Address: Address{
			Province:   "广东",
			City:       "深圳",
			Detail:     "南山区科技园",
			PostalCode: "518000",
		},
		Metadata: Metadata{
			CreatedAt:  yesterday,
			UpdatedAt:  now,
			VisitCount: 42,
		},
		Verified: true,
		VIP:      true,
	}

	article := Article{
		ID:        101,
		Title:     "使用 GroupJSON 优化 API 响应",
		Content:   "这篇文章介绍如何使用 GroupJSON 库来过滤 API 的 JSON 响应...",
		Draft:     false,
		Tags:      []string{"Go", "JSON", "API", "教程"},
		ViewCount: 1527,
		Metadata: Metadata{
			CreatedAt:  yesterday,
			UpdatedAt:  now,
			VisitCount: 1527,
		},
		AuthorID:   1,
		Likes:      76,
		RelatedIDs: []int{102, 105, 108},
		Comments: []Comment{
			{
				ID:        201,
				Content:   "非常实用的库, 已经用到项目中了！",
				CreatedAt: now.Add(-12 * time.Hour),
				UserID:    2,
				Approved:  true,
			},
			{
				ID:        202,
				Content:   "如何处理嵌套结构？",
				CreatedAt: now.Add(-6 * time.Hour),
				UserID:    3,
				Approved:  true,
			},
			{
				ID:        203,
				Content:   "垃圾文章, 毫无价值",
				CreatedAt: now.Add(-2 * time.Hour),
				UserID:    4,
				Approved:  false, // 未审核通过的评论
			},
		},
	}

	// 1. 公开API响应
	// 使用包装结构体, 序列化多个结构体
	resp := APIResponse{User: user, Article: article}
	// publicJSON, err := groupjson.New().WithGroups("public").Marshal(resp)
	publicJSON, err := groupjson.New().WithGroups("public").Marshal(
		map[string]interface{}{
			"user":    user,
			"article": article,
		},
	)
	if err != nil {
		fmt.Println("1. 公开API响应:", err)
	} else {
		fmt.Println("1. 公开API响应:")
		fmt.Println(string(publicJSON))
	}
	fmt.Println()

	// 2. 管理员视图 - 展示所有信息
	adminJSON, err := groupjson.New().
		WithGroups("admin").
		Marshal(user)
	if err != nil {
		fmt.Println("2. 管理员API响应:", err)
	} else {
		fmt.Println("2. 管理员API响应:")
		fmt.Println(string(adminJSON))
	}
	fmt.Println()

	// 3. 统计分析视图 - 仅展示统计数据
	statsJSON, err := groupjson.New().
		WithGroups("stats").
		WithTopLevelKey("data").
		Marshal(resp)
	if err != nil {
		fmt.Println("3. 统计分析API响应:", err)
	} else {
		fmt.Println("3. 统计分析API响应:")
		fmt.Println(string(statsJSON))
	}
	fmt.Println()

	// 4. 物流系统视图 - 仅需要地址信息
	shippingJSON, err := groupjson.New().
		WithGroups("shipping").
		Marshal(user)
	if err != nil {
		fmt.Println("4. 物流系统API响应:", err)
	} else {
		fmt.Println("4. 物流系统API响应:")
		fmt.Println(string(shippingJSON))
	}
	fmt.Println()

	// 5. 内部系统视图 - 包含敏感信息
	internalJSON, err := groupjson.New().
		WithGroups("internal", "admin").
		Marshal(user)
	if err != nil {
		fmt.Println("5. 内部系统API响应:", err)
	} else {
		fmt.Println("5. 内部系统API响应:")
		fmt.Println(string(internalJSON))
	}
	fmt.Println()

	// 6. 使用AND模式 - 同时满足multiple组的字段
	contactAndPublicJSON, err := groupjson.New().
		WithGroups("contact", "public").
		WithGroupMode(groupjson.ModeAnd).
		Marshal(user)
	if err != nil {
		fmt.Println("6. AND模式 (同时满足contact和public的字段):", err)
	} else {
		fmt.Println("6. AND模式 (同时满足contact和public的字段):")
		fmt.Println(string(contactAndPublicJSON))
	}
	fmt.Println()

	// 7. 编辑器视图 - 文章编辑所需信息
	editorJSON, err := groupjson.New().
		WithGroups("editor", "public").
		Marshal(article)
	if err != nil {
		fmt.Println("7. 编辑器视图:", err)
	} else {
		fmt.Println("7. 编辑器视图:")
		fmt.Println(string(editorJSON))
	}
}
