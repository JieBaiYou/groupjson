package groupjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Marshal 根据分组过滤并序列化值为JSON。
// 先转换为map再由标准JSON库序列化，支持顶层键包装。
// 仅适用于结构体或结构体指针。
func (g *GroupJSON) Marshal(v any) ([]byte, error) {
	if v == nil {
		return nil, ErrNilValue
	}

	// 检查是否为结构体或指向结构体的指针
	val := reflect.ValueOf(v)
	for val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, ErrInvalidType
	}

	// 将值转换为map
	m, err := g.structToMap(v)
	if err != nil {
		return nil, err
	}

	// 如果指定了顶层键，则包装结果
	if g.opts.TopLevelKey != "" {
		m = map[string]any{
			g.opts.TopLevelKey: m,
		}
	}

	// 使用标准JSON库进行最终序列化
	return json.Marshal(m)
}

// structToMap 将值按分组过滤转换为map（不序列化为JSON）。
// 适用于需要在序列化前处理数据的场景。
func (g *GroupJSON) structToMap(v any) (map[string]any, error) {
	if v == nil {
		return nil, ErrNilValue
	}

	// 创建上下文用于循环引用检测和路径跟踪
	ctx := newEncodeContext(g.opts.MaxDepth)

	val := reflect.ValueOf(v)
	// 解引用指针获取实际值
	for val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	// 只处理结构体类型
	if val.Kind() != reflect.Struct {
		return nil, ErrInvalidType
	}

	return g.marshalStruct(ctx, val)
}

// marshalValue 根据值类型选择合适的序列化方法。
// 处理各种数据结构并防止递归过深。
func (g *GroupJSON) marshalValue(ctx *encodeContext, val reflect.Value) (any, error) {
	// 处理nil指针
	if val.Kind() == reflect.Ptr && val.IsNil() {
		return nil, nil
	}

	// 解引用指针获取实际值
	if val.Kind() == reflect.Ptr {
		return g.marshalValue(ctx, val.Elem())
	}

	// 递增深度计数器
	if err := ctx.IncDepth(); err != nil {
		// 循环引用错误仍然返回
		if errors.Is(err, ErrCircularReference) {
			return nil, err
		}
		// 深度限制错误被转换为nil返回，而不是抛出错误
		return nil, nil
	}
	defer ctx.DecDepth()

	// 对于结构体和集合类型，如果已经达到最大深度，返回null
	if ctx.IsMaxDepthReached() {
		switch val.Kind() {
		case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
			// 达到最大深度限制，返回nil而不是错误
			return nil, nil
		}
	}

	// 根据类型分发处理
	switch val.Kind() {
	case reflect.Struct:
		// 特殊处理time.Time
		if val.Type().String() == "time.Time" {
			return val.Interface(), nil
		}
		return g.marshalStruct(ctx, val)

	case reflect.Map:
		return g.marshalMap(ctx, val)

	case reflect.Slice, reflect.Array:
		return g.marshalSlice(ctx, val)

	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		// 不支持序列化的类型
		return nil, WrapError(ErrUnsupportedType, fmt.Sprintf("%s (%s)", ctx.Path(), val.Type().String()))

	default:
		// 基本类型直接返回值
		return val.Interface(), nil
	}
}

// marshalStruct 序列化结构体类型，支持嵌套和匿名字段。
// 根据标签过滤字段并处理JSON特性（如omitempty）。
func (g *GroupJSON) marshalStruct(ctx *encodeContext, val reflect.Value) (map[string]any, error) {
	typ := val.Type()
	result := make(map[string]any)

	// 检测循环引用
	ptrAddr := g.getPointerAddress(val)
	if ptrAddr != 0 {
		if ctx.visited[ptrAddr] {
			// 发现循环引用，抛出错误
			return nil, WrapError(ErrCircularReference, ctx.Path())
		}
		// 标记为已访问
		ctx.visited[ptrAddr] = true
		// 函数返回时移除标记
		defer delete(ctx.visited, ptrAddr)
	}

	// 存储匿名字段结果，按encoding/json规则应用
	anonymousFields := make([]map[string]any, 0)

	// 遍历结构体所有字段
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过未导出字段
		if !fieldType.IsExported() {
			continue
		}

		// 处理匿名字段（嵌入字段）
		if fieldType.Anonymous {
			// 解引用指针
			fieldVal := field
			if fieldVal.Kind() == reflect.Ptr && !fieldVal.IsNil() {
				fieldVal = fieldVal.Elem()
			}

			// 处理结构体类型的匿名字段
			if fieldVal.Kind() == reflect.Struct {
				// 将匿名字段名添加到路径
				ctx.PushPath(fieldType.Name)

				// 对于匿名字段，我们不应检测其自身的循环引用，因为它是父结构体的一部分
				// 保存当前的访问记录，并在处理匿名字段期间使用一个新的拷贝
				visitedCopy := make(map[uintptr]bool)
				for k, v := range ctx.visited {
					visitedCopy[k] = v
				}

				// 清除匿名字段自身的标记，防止误识别为循环引用
				anonymousAddr := g.getPointerAddress(fieldVal)
				if anonymousAddr != 0 {
					delete(ctx.visited, anonymousAddr)
				}

				// 匿名字段视为当前层级
				nestedResult, err := g.marshalStruct(ctx, fieldVal)

				// 恢复原始的访问记录
				ctx.visited = visitedCopy

				// 恢复路径
				ctx.PopPath()

				if err != nil {
					// 不再静默忽略匿名字段错误
					return nil, err
				}

				// 暂存匿名字段结果
				anonymousFields = append(anonymousFields, nestedResult)
			}

			continue // 处理下一个字段
		}

		// 处理json标签
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "-" {
			continue // 跳过标记为忽略的字段
		}

		// 解析json标签选项
		jsonOpts := strings.Split(jsonTag, ",")
		jsonName := fieldType.Name // 默认使用字段名
		if jsonOpts[0] != "" {
			jsonName = jsonOpts[0] // 使用标签指定的名称
		}

		// 处理json选项
		omitEmpty := false
		omitZero := false
		for _, opt := range jsonOpts[1:] {
			if opt == "omitempty" {
				omitEmpty = true
			}
			if opt == "omitzero" {
				omitZero = true // Go 1.24特性
			}
		}

		// 检查组标签
		groupsTag := fieldType.Tag.Get(g.opts.TagKey)
		if groupsTag == "" {
			continue // 没有组标签的字段不包含
		}

		// 检查字段是否属于指定组
		fieldGroups := strings.Split(groupsTag, ",")
		include := g.shouldIncludeField(fieldGroups)
		if !include {
			continue // 排除不符合条件的字段
		}

		// 将字段名添加到当前路径
		ctx.PushPath(jsonName)

		// 处理字段值
		value, err := g.marshalValue(ctx, field)

		// 恢复路径
		ctx.PopPath()

		if err != nil {
			return nil, WrapError(err, jsonName) // 确保错误包含路径信息
		}

		// 处理omitempty和omitzero选项
		if shouldOmit(field, value, omitEmpty, omitZero) {
			continue
		}

		// 添加字段到结果
		result[jsonName] = value
	}

	// 应用匿名字段，仅添加未被当前结构体覆盖的字段
	for _, anonResult := range anonymousFields {
		for k, v := range anonResult {
			if _, exists := result[k]; !exists {
				result[k] = v
			}
		}
	}

	return result, nil
}

// marshalMap 序列化map类型，键类型必须是字符串。
func (g *GroupJSON) marshalMap(ctx *encodeContext, val reflect.Value) (any, error) {
	if val.IsNil() {
		return nil, nil
	}

	// 检测循环引用
	ptrAddr := g.getPointerAddress(val)
	if ptrAddr != 0 {
		if ctx.visited[ptrAddr] {
			// 发现循环引用，抛出错误
			return nil, WrapError(ErrCircularReference, ctx.Path())
		}
		// 标记为已访问
		ctx.visited[ptrAddr] = true
		// 函数返回时移除标记
		defer delete(ctx.visited, ptrAddr)
	}

	result := make(map[string]any)
	iter := val.MapRange()
	for iter.Next() {
		k := iter.Key()
		if k.Kind() != reflect.String {
			// 只支持字符串键
			return nil, WrapError(ErrNonStringMapKey, ctx.Path())
		}

		keyStr := k.String()

		// 将map键添加到路径
		ctx.PushPath(fmt.Sprintf("[%s]", keyStr))

		// 递归处理值
		itemVal, err := g.marshalValue(ctx, iter.Value())

		// 恢复路径
		ctx.PopPath()

		if err != nil {
			return nil, WrapError(err, fmt.Sprintf("[%s]", keyStr))
		}

		result[keyStr] = itemVal
	}

	return result, nil
}

// marshalSlice 序列化切片或数组类型。
func (g *GroupJSON) marshalSlice(ctx *encodeContext, val reflect.Value) (any, error) {
	if val.IsNil() {
		return nil, nil
	}

	// 检测循环引用
	ptrAddr := g.getPointerAddress(val)
	if ptrAddr != 0 {
		if ctx.visited[ptrAddr] {
			// 发现循环引用，抛出错误
			return nil, WrapError(ErrCircularReference, ctx.Path())
		}
		// 标记为已访问
		ctx.visited[ptrAddr] = true
		// 函数返回时移除标记
		defer delete(ctx.visited, ptrAddr)
	}

	// 预分配适当容量
	result := make([]any, 0, val.Len())
	for i := 0; i < val.Len(); i++ {
		// 将数组索引添加到路径
		ctx.PushPath(fmt.Sprintf("[%d]", i))

		// 递归处理每个元素
		itemVal, err := g.marshalValue(ctx, val.Index(i))

		// 恢复路径
		ctx.PopPath()

		if err != nil {
			return nil, WrapError(err, fmt.Sprintf("[%d]", i))
		}

		result = append(result, itemVal)
	}

	return result, nil
}

// shouldIncludeField 确定字段是否应包含在输出中。
// 根据配置的GroupMode决定使用AND或OR逻辑。
func (g *GroupJSON) shouldIncludeField(fieldGroups []string) bool {
	// 无分组过滤则不包含任何字段
	if len(g.opts.Groups) == 0 {
		return false
	}

	if g.opts.GroupMode == ModeAnd {
		// AND逻辑：必须属于所有指定分组
		for _, group := range g.opts.Groups {
			found := false
			for _, fieldGroup := range fieldGroups {
				if fieldGroup == group {
					found = true
					break
				}
			}
			if !found {
				return false // 有任一指定分组不匹配则排除
			}
		}
		return true // 所有分组都匹配则包含
	} else {
		// OR逻辑：属于任一指定分组即可
		for _, group := range g.opts.Groups {
			for _, fieldGroup := range fieldGroups {
				if fieldGroup == group {
					return true // 匹配任一分组则包含
				}
			}
		}
		return false // 无匹配则排除
	}
}

// shouldOmit 判断是否应省略字段。
// 根据字段值和json标签选项（omitempty、omitzero）决定。
func shouldOmit(field reflect.Value, value any, omitEmpty, omitZero bool) bool {
	// 处理nil值
	if value == nil {
		return omitEmpty
	}

	// 检查空集合类型
	switch field.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		if field.Len() == 0 && omitEmpty {
			return true
		}
	}

	// 处理omitzero选项（Go 1.24特性）
	if omitZero {
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return field.Int() == 0
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return field.Uint() == 0
		case reflect.Float32, reflect.Float64:
			return field.Float() == 0
		case reflect.Bool:
			return !field.Bool()
		case reflect.String:
			return field.String() == ""
		}
	}

	return false
}

// getPointerAddress 获取值的指针地址用于循环引用检测。
func (g *GroupJSON) getPointerAddress(v reflect.Value) uintptr {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return 0
		}
		return v.Pointer()
	}

	// 对于可寻址的非指针值获取其地址
	if v.CanAddr() {
		return v.Addr().Pointer()
	}

	// 不可寻址的值无法检测循环引用
	return 0
}
