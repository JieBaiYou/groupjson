# GroupJSON

[![Go Reference](https://pkg.go.dev/badge/github.com/JieBaiYou/groupjson.svg)](https://pkg.go.dev/github.com/JieBaiYou/groupjson)
[![Go Report Card](https://goreportcard.com/badge/github.com/JieBaiYou/groupjson)](https://goreportcard.com/report/github.com/JieBaiYou/groupjson)

GroupJSON 是一个轻量级、高性能的 Go 运行时分组序列化库。它允许你通过 struct tag 定义字段分组，根据不同场景（如 API 响应的 `public`/`admin` 视图）选择性地序列化字段。

**V1 重构版本**：彻底重写了底层引擎，移除中间态 Map 分配，采用流式写入 `io.Writer`/`bytes.Buffer`，性能大幅提升。

## 核心特性

- 🚀 **高性能**：流式写入设计，零中间内存分配，自带对象池 (`sync.Pool`) 优化。
- 🔍 **分组筛选**：支持 OR (默认) 与 AND 分组逻辑，灵活控制字段可见性。
- 🔄 **标准兼容**：支持 `json` 标签的 `omitempty` 和 Go 1.24+ 的 `omitzero` 语义。
- 📦 **零依赖**：仅依赖 Go 标准库。
- 🛡️ **安全可靠**：内置递归深度限制与循环引用检测。

## 安装

```bash
go get github.com/JieBaiYou/groupjson
```

## 快速开始

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
        Name:     "Alice",
        Email:    "alice@example.com",
        Password: "secret_password",
    }

    // 场景 1: 公开视图 (仅 public 组)
    publicJSON, _ := groupjson.NewEncoder().
        WithGroups("public").
        Marshal(user)
    fmt.Println(string(publicJSON))
    // 输出: {"id":1,"name":"Alice"}

    // 场景 2: 管理员视图 (admin 组)
    adminJSON, _ := groupjson.NewEncoder().
        WithGroups("admin").
        WithTopLevelKey("data"). // 自动包装 {"data": ...}
        Marshal(user)
    fmt.Println(string(adminJSON))
    // 输出: {"data":{"id":1,"name":"Alice","email":"alice@example.com"}}
}
```

## 高级用法

### 分组逻辑

支持两种模式：

- **OR (默认)**: 字段属于任一指定分组即被包含。
- **AND**: 字段必须同时属于所有指定分组才被包含。

```go
// 仅导出同时标记为 "public" 和 "admin" 的字段
b, _ := groupjson.NewEncoder().
    WithGroups("public", "admin").
    WithGroupMode(groupjson.ModeAnd).
    Marshal(user)
```

### 性能优化

`Encoder` 是设计为不可变且轻量的，但其内部使用了 `sync.Pool` 来复用 Buffer。

对于极致性能场景，建议使用 `Encode(io.Writer, v)` 接口直接写入流：

```go
func handler(w http.ResponseWriter, r *http.Request) {
    user := getUser()
    w.Header().Set("Content-Type", "application/json")

    // 直接写入 ResponseWriter，避免字节切片拷贝
    err := groupjson.NewEncoder().
        WithGroups("public").
        Encode(w, user)

    if err != nil {
        // handle error
    }
}
```

### 顶层包装 (Top-Level Wrapper)

使用 `WithTopLevelKey` 可以方便地将结果包装在指定键下，无需手动构建 Map。

```go
groupjson.NewEncoder().
    WithGroups("public").
    WithTopLevelKey("response"). // 输出 {"response": ...}
    Marshal(user)
```

### 配置选项

```go
groupjson.NewEncoder().
    WithGroups("public").           // 必选：指定分组
    WithTagKey("access").           // 可选：自定义 Tag 名 (默认 "groups")
    WithTopLevelKey("data").        // 可选：指定顶层包装键
    WithMaxDepth(64).               // 可选：最大递归深度 (默认 32)
    WithEscapeHTML(true).           // 可选：开启 HTML 转义 (默认关闭，性能更好)
    WithSortKeys(true).             // 可选：Map 键排序 (默认关闭)
    Marshal(v)
```

## 注意事项

1.  **默认不转义 HTML**: 与标准库不同，默认情况下 `EscapeHTML` 为 `false`，这能显著提升性能。如需处理用户输入并嵌入 HTML，请显式开启。
2.  **Map/Slice 支持**: 库会自动递归处理 `map[string]any` 和切片中的结构体元素，无需额外配置。
3.  **深度限制**: 当递归深度超过 `MaxDepth` 时，会返回 `ErrMaxDepth` 错误。

## 许可证

MIT License
