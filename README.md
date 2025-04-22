# GroupJSON

[![Go Reference](https://pkg.go.dev/badge/github.com/JieBaiYou/groupjson.svg)](https://pkg.go.dev/github.com/JieBaiYou/groupjson)
[![Go Report Card](https://goreportcard.com/badge/github.com/JieBaiYou/groupjson)](https://goreportcard.com/report/github.com/JieBaiYou/groupjson)

GroupJSON 是一个高性能的 Go 库, 用于按分组选择性地序列化结构体字段。它基于字段标签系统, 让开发者能够轻松创建针对不同用户角色的 JSON 视图。

## 核心特性

- 🚀 **高性能设计**：使用代码生成和内存优化技术
- 🔍 **分组筛选**：根据字段标签选择性序列化, 支持 OR/AND 逻辑
- 🔄 **兼容标准 JSON**：完全支持 Go 标准库 JSON 功能, 包括 omitempty、omitzero 标签
- 💡 **灵活配置**：支持顶层包装键、空值处理、自定义标签等
- 📦 **轻量级**：零外部依赖, 简洁的 API
- 🛡️ **类型安全**：代码生成提供类型安全保证, 减少运行时错误

## 设计原则

GroupJSON 的设计基于以下关键原则：

1. **性能优先**：通过代码生成减少反射开销
2. **灵活性**：支持多种使用方式和配置选项
3. **易用性**：提供简单直观的 API
4. **兼容性**：与标准 JSON 库行为保持一致
5. **安全性**：类型安全的 API 设计

## 安装

```bash
go get github.com/JieBaiYou/groupjson
```

## 快速开始

### 使用代码生成（推荐, 高性能）

1. 定义结构体并添加分组标签：

```go
package main

import (
    "fmt"
    "github.com/JieBaiYou/groupjson"
)

//go:generate groupjson -type=User
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

    // 生成序列化方法可直接调用
    publicJSON, _ := user.MarshalWithGroups("public")
    fmt.Println(string(publicJSON))
    // 输出: {"id":1,"name":"张三"}

    // 带选项的序列化
    opts := groupjson.Options{TopLevelKey: "data"}
    adminJSON, _ := user.MarshalWithGroupsOptions(opts, "admin")
    fmt.Println(string(adminJSON))
    // 输出: {"data":{"id":1,"name":"张三","email":"zhangsan@example.com"}}
}
```

2. 运行代码生成：

```bash
go generate ./...
```

3. 使用生成的代码

### 使用反射 API（更灵活）

```go
package main

import (
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
    fmt.Println(string(publicJSON))
    // 输出: {"id":1,"name":"张三"}

    // 带选项的序列化
    adminJSON, _ := groupjson.New().
        WithGroups("admin").
        WithTopLevelKey("data").
        Marshal(user)
    fmt.Println(string(adminJSON))
    // 输出: {"data":{"id":1,"name":"张三","email":"zhangsan@example.com"}}
}
```

## 高级用法

### 分组逻辑

GroupJSON 支持两种分组筛选逻辑：

- **OR 逻辑**（默认）：字段只要属于任一指定分组即包含在结果中
- **AND 逻辑**：字段必须同时属于所有指定分组才包含在结果中

```go
// OR 逻辑 - 默认
orJSON, _ := groupjson.New().
    WithGroups("public", "internal").
    Marshal(user)
// 包含属于 public 或 internal 组的字段

// AND 逻辑
andJSON, _ := groupjson.New().
    WithGroups("public", "admin").
    WithGroupMode(groupjson.ModeAnd).
    Marshal(user)
// 仅包含同时属于 public 和 admin 组的字段
```

### 支持 Go 1.24 的 omitzero 标签

```go
type Product struct {
    ID        int       `json:"id" groups:"public"`
    Name      string    `json:"name" groups:"public"`
    Price     float64   `json:"price,omitzero" groups:"public"`
    Tags      []string  `json:"tags,omitzero" groups:"public"`
    UpdatedAt time.Time `json:"updatedAt,omitzero" groups:"public"`
}

// 使用 omitzero 时, 零值数字、空字符串等会被省略, 但空集合会保留
```

### 自定义选项

```go
// 完整配置示例
result, _ := groupjson.New().
    WithGroups("public", "admin").       // 设置分组
    WithGroupMode(groupjson.ModeOr).     // 设置分组逻辑
    WithTopLevelKey("data").             // 添加顶层包装键
    WithTagKey("access").                // 自定义标签名 (默认 "groups")
    WithMaxDepth(10).                    // 设置最大递归深度
    Marshal(user)
```

### 映射输出

```go
// 获取 map[string]any 结果而不是 JSON 字节
userMap, _ := groupjson.New().
    WithGroups("public").
    MarshalToMap(user)

// 手动编辑结果
userMap["extra_field"] = "额外信息"
```

## 待实现

对性能影响最大的优化方案（按重要性排序）：

1. 字段信息缓存：对结构体序列化影响最大，因为反射获取字段信息的成本非常高。每次序列化如果都重新解析结构体字段，会导致大量的反射操作，特别是对于复杂结构体。缓存后只需一次反射，后续复用结果。
2. 类型编码器缓存：类似字段缓存，避免为同一类型重复创建编码器逻辑。在处理大量相同类型对象时，可减少大量反射和函数创建开销。
3. 对象池化：减少 GC 压力和内存分配，特别是在高并发环境下影响显著。每次序列化都需要的 encodeState 对象如果能复用，会大幅减少内存分配次数。
4. 字节缓冲区复用：直接操作预分配的缓冲区能显著减少内存分配和复制操作，特别是处理大对象时效果明显。
5. 专用编码器函数：相比通用 switch-case 逻辑，专用编码器减少了条件判断和分支预测失败，对高频调用路径优化效果明显。
   这五项优化关注的都是高频操作（类型判断、内存分配、反射操作），而且减少了昂贵操作（如反射和内存分配）。实施这些优化，能在保持功能不变的情况下获得最显著的性能提升。

### 缓存优化

### 内存优化

## 贡献

欢迎提交问题报告、功能请求和 Pull Request！

## 许可证

MIT 许可证
