package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/JieBaiYou/groupjson/v2"
)

// =============================================================================
// 示例结构体定义
// =============================================================================

type User struct {
	ID       int    `json:"id" groups:"public,admin"`
	Name     string `json:"name" groups:"public,admin"`
	Email    string `json:"email" groups:"admin"`
	Password string `json:"password" groups:"internal"`
}

type Product struct {
	Name  string  `json:"name" groups:"public"`
	Price float64 `json:"price,string" groups:"public"` // 输出为字符串
	Stock int     `json:"stock,omitempty" groups:"admin"`
}

type Order struct {
	ID        string    `json:"order_id" groups:"public"`
	Items     []Product `json:"items" groups:"public"`
	CreatedAt time.Time `json:"created_at" groups:"public"`
	Note      string    `json:"note" groups:"admin"`
}

// 用于测试嵌套权限
type Parent struct {
	Child Child `json:"child" groups:"admin"`
}

type Child struct {
	PublicInfo string `json:"public_info" groups:"public"`
	Secret     string `json:"secret" groups:"admin"`
}

// 用于测试指针
type Profile struct {
	Bio     *string `json:"bio" groups:"public"`     // 可能为 nil
	Website *string `json:"website" groups:"public"` // 可能为 nil
}

// 用于测试嵌入字段
type Base struct {
	ID        int       `json:"id" groups:"public"`
	CreatedAt time.Time `json:"created_at" groups:"public"`
}

type Post struct {
	Base             // 匿名字段，字段会被提升到顶层
	Title     string `json:"title" groups:"public"`
	MyBaseStr string `json:"base_override" groups:"public"` // 避免 Struct 定义时的名字冲突
}

// 用于测试循环引用
type Node struct {
	Val  int   `json:"val" groups:"public"`
	Next *Node `json:"next" groups:"public"`
}

// 用于测试自定义 Marshaler
type CustomTime time.Time

func (c CustomTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"Unix-%d"`, time.Time(c).Unix())), nil
}

type Log struct {
	Message string     `json:"msg" groups:"public"`
	Time    CustomTime `json:"time" groups:"public"`
}

// =============================================================================
// 辅助函数
// =============================================================================

func printStructInfo(v any) {
	fmt.Println("原始输入 (结构体定义与值):")
	val := reflect.ValueOf(v)
	typ := val.Type()

	// 解指针
	if typ.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		// 只是普通类型或者是 slice/map，打印值即可
		fmt.Printf("%#v\n\n", v)
		return
	}

	fmt.Printf("type %s struct {\n", typ.Name())
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// 获取 Tag
		tag := ""
		if field.Tag != "" {
			tag = fmt.Sprintf(" `%s`", field.Tag)
		}

		// 简单的值打印，截断过长的内容
		valStr := fmt.Sprintf("%v", fieldVal)
		if len(valStr) > 30 {
			valStr = valStr[:27] + "..."
		}
		if fieldVal.Kind() == reflect.String {
			valStr = fmt.Sprintf("%q", valStr)
		}

		fmt.Printf("\t%-10s %-10s%s // Value: %s\n",
			field.Name,
			field.Type.Name(),
			tag,
			valStr,
		)
	}
	fmt.Printf("}\n\n")
}

func printDemo(title string, input any, groups []string, mode groupjson.Mode) {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("测试场景: %s\n", title)
	fmt.Println(strings.Repeat("-", 60))

	// 打印原始结构体
	// 为了避免打印出巨大的循环引用结构体导致刷屏，这里做个简单判断
	if _, ok := input.(*Node); !ok {
		printStructInfo(input)
	} else {
		fmt.Printf("原始输入:\n(循环引用结构体)\n\n")
	}

	// 打印分组配置
	modeStr := "OR (默认)"
	if mode == groupjson.ModeAnd {
		modeStr = "AND"
	}
	fmt.Printf("分组配置: Groups=%v, Mode=%s\n", groups, modeStr)

	// 执行处理
	encoder := groupjson.New().WithGroups(groups...).WithMode(mode)
	output, err := encoder.Marshal(input)
	// 结果处理
	if err != nil {
		fmt.Printf("\n⚠️  程序返回错误 (符合预期):\n%v\n", err)
		fmt.Println()
		return
	}

	// 格式化输出结果
	var prettyAny any
	_ = json.Unmarshal(output, &prettyAny)
	prettyJSON, _ := json.MarshalIndent(prettyAny, "", "  ")

	fmt.Printf("处理结果:\n%s\n", string(prettyJSON))
	fmt.Println()
}

// =============================================================================
// 测试函数集
// =============================================================================

func demoBasic() {
	user := User{ID: 1, Name: "张三", Email: "zhangsan@example.com", Password: "secret"}
	printDemo("基础 - Public 视图 (常规字段筛选)", user, []string{"public"}, groupjson.ModeOr)
}

func demoTags() {
	p := Product{Name: "iPhone 15", Price: 5999.00, Stock: 0}
	printDemo("Tag特性 - string转换 & omitempty", p, []string{"public", "admin"}, groupjson.ModeOr)
}

func demoNestedPermissions() {
	data := Parent{Child: Child{PublicInfo: "公开信息", Secret: "管理员秘密"}}
	// 仅展示最典型的权限隔离场景
	printDemo("嵌套权限 - 请求 Public (父级不可见导致子级全隐藏)", data, []string{"public"}, groupjson.ModeOr)
}

func demoCollections() {
	orders := []Order{
		{ID: "A01", Items: []Product{{Name: "手机", Price: 1000}}, CreatedAt: time.Now()},
		{ID: "A02", Items: []Product{{Name: "耳机", Price: 200}}, CreatedAt: time.Now()},
	}
	printDemo("集合 - Slice 递归处理", orders, []string{"public"}, groupjson.ModeOr)
}

func demoPointers() {
	bio := "热爱编程"
	p := Profile{
		Bio:     &bio, // 非空指针
		Website: nil,  // 空指针
	}
	// 预期：Bio 显示内容，Website 显示 null
	printDemo("指针 - 处理 Nil 与 Non-Nil", p, []string{"public"}, groupjson.ModeOr)
}

func demoEmbedding() {
	p := Post{
		Base:  Base{ID: 101, CreatedAt: time.Now()},
		Title: "GroupJSON 发布",
		// Base 字段在这里是显式的 string 字段，它应该覆盖 Base 结构体中的同名匿名字段（如果JSON名冲突）
		// 但这里我们的 Base 结构体没有字段叫 "base_override"，所以都会显示。
		// 匿名字段 Base 的 ID 和 CreatedAt 应该被提升到顶层。
	}
	// 预期：id, created_at 在顶层，与 title 平级
	printDemo("嵌入 - 匿名字段提升 (Flatten)", p, []string{"public"}, groupjson.ModeOr)
}

func demoCustomMarshaler() {
	l := Log{
		Message: "系统启动",
		Time:    CustomTime(time.Now()),
	}
	// 预期：time 字段应该使用 CustomTime.MarshalJSON 的输出格式
	printDemo("接口 - 自定义 Marshaler 支持", l, []string{"public"}, groupjson.ModeOr)
}

func demoCircular() {
	n1 := &Node{Val: 1}
	n2 := &Node{Val: 2}
	n1.Next = n2
	n2.Next = n1 // 制造循环引用

	// 预期：报错 circular reference detected
	printDemo("错误处理 - 循环引用检测", n1, []string{"public"}, groupjson.ModeOr)
}

// =============================================================================
// 主入口
// =============================================================================

func main() {
	fmt.Println(">>> GroupJSON V2 全功能演示 <<<")
	fmt.Println()

	demoBasic()             // 1. 基础筛选
	demoTags()              // 2. Tag 行为 (string, omitempty)
	demoPointers()          // 3. 指针处理 (nil)
	demoEmbedding()         // 4. 结构体嵌入 (字段提升)
	demoCollections()       // 5. 切片/数组
	demoCustomMarshaler()   // 6. 自定义接口
	demoNestedPermissions() // 7. 嵌套结构体权限
	demoCircular()          // 8. 错误处理
}
