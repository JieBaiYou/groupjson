# GroupJSON 使用示例

这个示例展示了如何使用 GroupJSON 按分组选择性地序列化结构体字段。

## 关于示例

本例包含两个主要示例：

1. **简单产品示例 (Product)**：
   展示基本字段和标签的使用, 包含以下分组：

   - `public`: 公开可见的基本信息
   - `detail`: 产品详情信息
   - `inventory`: 库存管理相关信息
   - `internal`: 内部使用信息（如成本）
   - `admin`: 管理员可见的全部信息

2. **复杂产品示例 (ComplexProduct)**：
   展示嵌套结构体和匿名字段的处理, 包含以下分组：
   - `basic`: 基本信息
   - `base`: 来自匿名嵌套的 BaseEntity 的字段
   - `detail`: 详细信息, 包含嵌套结构体
   - `internal`: 内部信息
   - `inventory`: 库存信息
   - `admin`: 管理员视图, 包含所有字段

## 当前实现说明

虽然 GroupJSON 提供代码生成功能, 但由于当前模板解析问题, 本示例暂时使用以下方法替代：

1. 手动为 `Product` 和 `ComplexProduct` 类型添加了 `MarshalWithGroups` 和 `MarshalWithGroupsOptions` 方法
2. 这些方法内部调用 GroupJSON 的反射 API 实现功能

这种方式可以模拟代码生成的效果, 但性能不如实际生成的代码。

## 运行步骤

运行示例：

```bash
go run .
```

或者：

```bash
go run *.go
```

## 代码生成（当修复后）

当代码生成器模板问题修复后, 可以通过以下步骤使用代码生成功能：

1. **安装 GroupJSON 代码生成工具**

   ```bash
   go install github.com/JieBaiYou/groupjson/cmd/groupjson@latest
   ```

2. **修改源代码, 取消注释代码生成指令**

   将 `main.go` 和 `complex.go` 中被注释的 `//go:generate` 行取消注释。

3. **生成代码**

   ```bash
   go generate
   ```

4. **运行示例**

   ```bash
   go run .
   ```

## 参考

- 完整文档：[GroupJSON GitHub 仓库](https://github.com/JieBaiYou/groupjson)
- API 参考：[Go Package 文档](https://pkg.go.dev/github.com/JieBaiYou/groupjson)
