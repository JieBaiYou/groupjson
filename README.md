# GroupJSON

[![Go Reference](https://pkg.go.dev/badge/github.com/JieBaiYou/groupjson.svg)](https://pkg.go.dev/github.com/JieBaiYou/groupjson)
[![Go Report Card](https://goreportcard.com/badge/github.com/JieBaiYou/groupjson)](https://goreportcard.com/report/github.com/JieBaiYou/groupjson)

GroupJSON 是一个运行时分组筛选的 Go 库，用于选择性地序列化结构体字段，创建针对不同视图/角色的 JSON 输出。本版本仅保留运行时实现，不包含代码生成。

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

## 快速开始（运行时分组）

### 使用反射 API（运行时）

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
    publicJSON, _ := groupjson.NewEncoder().
        WithGroups("public").
        Marshal(user)
    fmt.Println(string(publicJSON))
    // 输出: {"id":1,"name":"张三"}

    // 带选项的序列化
    adminJSON, _ := groupjson.NewEncoder().
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
    result, _ := groupjson.NewEncoder().
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
userMap, _ := groupjson.NewEncoder().
    WithGroups("public").
    MarshalToMap(user)

// 手动编辑结果
userMap["extra_field"] = "额外信息"
```

## 设计文档

完整重构设计见 `docs/DESIGN.md`（仅运行时分组筛选）。

## 贡献

欢迎提交问题报告、功能请求和 Pull Request！

## 许可证

MIT 许可证
