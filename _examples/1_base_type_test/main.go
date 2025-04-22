package main

import (
	"encoding/json"
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
	fmt.Printf("publicJSON: %s\n\n", string(publicJSON))
	// 输出: {"id":1,"name":"张三"}

	// 带选项的序列化
	adminJSON, _ := groupjson.New().
		WithGroups("admin", "internal").
		WithTopLevelKey("data").
		Marshal(user)
	fmt.Printf("adminJSON: %s\n\n", string(adminJSON))
	// 输出: {"data":{"email":"zhangsan@example.com","id":1,"name":"张三","password":"secret123"}}

	// 使用Marshal解析
	internalJSON, _ := groupjson.Marshal(user, "internal")
	fmt.Printf("internalJSON: %s\n\n", string(internalJSON))
	// 输出: {"password":"secret123"}

	user = User{
		ID:    1,
		Name:  "张三",
		Email: "zhangsan@example.com",
		Password: `<div class="content" data-info='{"key": "value", "text": "He said, \"Hello <world> & welcome!\""}'>
  <!-- 注释：这是一个 <测试> 区域 -->
  <p title="提示 & 描述">这是一段包含 &lt;strong&gt;HTML&lt;/strong&gt; 的文本。</p>
  <script>alert("你好，世界！ & \" < >");</script>
</div>`,
	}

	// 使用标准库序列化
	json, _ := json.Marshal(user)
	fmt.Printf("json: %s\n\n", string(json))
	// 输出: {"id":1,"name":"张三","email":"zhangsan@example.com","password":"\u003cdiv class=\"content\" data-info='{\"key\": \"value\", \"text\": \"He said, \\\"Hello \u003cworld\u003e \u0026 welcome!\\\"\"}'\u003e\n  \u003c!-- 注释：这是一个 \u003c测试\u003e 区域 --\u003e\n  \u003cp title=\"提示 \u0026 描述\"\u003e这是一段包含 \u0026lt;strong\u0026gt;HTML\u0026lt;/strong\u0026gt; 的文本。\u003c/p\u003e\n  \u003cscript\u003ealert(\"你好，世界！ \u0026 \\\" \u003c \u003e\");\u003c/script\u003e\n\u003c/div\u003e"}

	// 使用Marshal解析
	internalJSON, _ = groupjson.Marshal(user, "internal")
	fmt.Printf("internalJSON: %s\n\n", string(internalJSON))
	// 输出: {"password":"\u003cdiv class=\"content\" data-info='{\"key\": \"value\", \"text\": \"He said, \\\"Hello \u003cworld\u003e \u0026 welcome!\\\"\"}'\u003e\n  \u003c!-- 注释：这是一个 \u003c测试\u003e 区域 --\u003e\n  \u003cp title=\"提示 \u0026 描述\"\u003e这是一段包含 \u0026lt;strong\u0026gt;HTML\u0026lt;/strong\u0026gt; 的文本。\u003c/p\u003e\n  \u003cscript\u003ealert(\"你好，世界！ \u0026 \\\" \u003c \u003e\");\u003c/script\u003e\n\u003c/div\u003e"}
}
