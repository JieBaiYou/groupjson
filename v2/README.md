# GroupJSON V2

`GroupJSON V2` 是一个**轻量级**、**高性能**且**兼容标准库行为**的 Go 语言 JSON 序列化库。

它的核心功能是：通过结构体 Tag (`groups:"..."`) 对字段进行分组，在序列化时根据请求的 Context（如 `public`, `admin` 等）动态筛选字段。

> **V2 版本特点**：
> *   **单文件实现**：核心逻辑全部封装在 `groupjson.go`，无外部依赖，易于集成和审计。
> *   **零分配设计**：使用 `sync.Pool` 复用底层 Buffer，大幅降低 GC 压力。
> *   **标准库兼容**：完整支持 `omitempty`、`string` 标签、HTML 转义、Map 排序、自定义 `Marshaler` 等标准行为。
> *   **安全可靠**：内置循环引用检测和严格的层级权限过滤。

## 安装

```bash
go get github.com/JieBaiYou/groupjson/v2
```

## 快速开始

### 1. 定义结构体

在结构体字段上添加 `groups` 标签，定义该字段属于哪些分组。支持 `json` 标签的所有标准特性。

```go
type User struct {
    ID       int    `json:"id" groups:"public,admin"`       // 公开和管理员可见
    Name     string `json:"name" groups:"public,admin"`     // 公开和管理员可见
    Email    string `json:"email" groups:"admin"`           // 仅管理员可见
    Password string `json:"password" groups:"internal"`     // 仅内部可见
    Balance  int    `json:"balance,omitempty" groups:"admin"` // 仅管理员可见，且为0时省略
}
```

### 2. 根据场景序列化

#### 场景 A: 公开接口 (Public View)

只展示 `groups` 包含 `public` 的字段。

```go
import "github.com/JieBaiYou/groupjson/v2"

user := User{ID: 1, Name: "Alice", Email: "a@x.com", Password: "123"}

// 使用 Marshal 便捷函数
// 结果: {"id":1,"name":"Alice"}
data, _ := groupjson.Marshal(user, "public")
```

#### 场景 B: 管理员接口 (Admin View)

展示 `groups` 包含 `admin` 的字段。

```go
// 使用 Encoder 进行更多配置
// 结果: {"id":1,"name":"Alice","email":"a@x.com"}
data, _ := groupjson.New().WithGroups("admin").Marshal(user)
```

## 高级特性

### 分组匹配模式 (OR vs AND)

默认模式是 **OR**（只要匹配任一分组即可显示）。你可以通过 `WithMode(groupjson.ModeAnd)` 切换为 **AND** 模式（必须匹配所有指定分组）。

```go
// 仅显示同时属于 "public" 和 "admin" 的字段
// 结果: {"id":1,"name":"Alice"}  <-- Email (admin only) 被排除
groupjson.New().
    WithGroups("public", "admin").
    WithMode(groupjson.ModeAnd).
    Marshal(user)
```

### 嵌套结构体权限控制

权限控制是**逐层严格匹配**的。如果父级字段不可见，子级结构体将完全被隐藏，无论子级字段的权限如何。

```go
type Parent struct {
    Child Child `json:"child" groups:"admin"` // 父级入口仅限 admin
}
type Child struct {
    Info string `json:"info" groups:"public"` // 子级字段是 public
}

// 请求 "public" -> 输出 {}
// (因为父级 "child" 字段是 admin，public 视图无法进入)
groupjson.Marshal(Parent{...}, "public")
```

### 标准库兼容性

V2 版本完美支持以下标准库特性：

*   **`omitempty`**: 零值字段省略。
*   **`string` Tag**: 将数字/布尔值转为字符串输出（如 `json:"price,string"`）。
*   **自定义 Marshaler**: 实现了 `json.Marshaler` 或 `encoding.TextMarshaler` 的类型会优先使用其自定义逻辑。
*   **指针处理**: 自动处理 `nil` 指针。
*   **HTML 转义**: 默认开启（与 `json.Marshal` 一致）。

## 性能

V2 采用了 `sync.Pool` 复用 Buffer 和 `sync.Map` 缓存结构体元数据（Schema Cache）。在热路径上实现了**零堆内存分配**（Zero Heap Allocation），非常适合高性能 Web 服务。

## 示例代码

更多详细示例请参考 [examples/demo/main.go](examples/demo/main.go)。

