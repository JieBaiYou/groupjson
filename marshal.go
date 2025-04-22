package groupjson

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

// 将值按照组过滤序列化为JSON, 这是库的主要入口点
// 先将值转换为map, 再使用标准JSON库进行最终序列化
// 如果指定了顶层键, 结果会被包装在该键下
// 只支持结构体类型或指向结构体的指针
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
		return nil, ErrInvalidValue
	}

	// 将值转换为map
	m, err := g.structToMap(v)
	if err != nil {
		return nil, err
	}

	// 如果指定了顶层键, 则包装结果
	if g.opts.TopLevelKey != "" {
		m = map[string]any{
			g.opts.TopLevelKey: m,
		}
	}

	// 使用标准JSON库进行最终序列化
	return json.Marshal(m)
}

// 将值按照组过滤序列化为map, 而不是JSON字符串
// 这对于需要在序列化前进一步处理数据的场景很有用
func (g *GroupJSON) structToMap(v any) (map[string]any, error) {
	if v == nil {
		return nil, ErrNilValue
	}

	// 创建新的上下文, 用于跟踪处理过的指针
	ctx := newEncodeContext()

	val := reflect.ValueOf(v)
	// 解引用指针, 获取实际值
	for val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	// 只处理结构体类型
	if val.Kind() != reflect.Struct {
		return nil, ErrInvalidValue
	}

	return g.marshalStruct(ctx, val, 0)
}

// 根据值的类型进行适当的序列化, 是序列化过程的核心路由函数
// 根据不同的数据类型分发到对应的处理函数
func (g *GroupJSON) marshalValue(ctx *encodeContext, val reflect.Value, depth int) (any, error) {
	// 处理nil指针
	if val.Kind() == reflect.Ptr && val.IsNil() {
		return nil, nil
	}

	// 解引用指针获取实际值
	if val.Kind() == reflect.Ptr {
		return g.marshalValue(ctx, val.Elem(), depth)
	}

	// 检查递归深度, 防止无限递归
	if depth > g.opts.MaxDepth {
		// 对于超过最大深度的复杂类型, 返回空值而不是nil
		// 这样JSON输出会更一致（如 {} 而不是 null）
		switch val.Kind() {
		case reflect.Struct:
			// 允许time.Time作为基本类型直接序列化
			if val.Type().String() == "time.Time" {
				return val.Interface(), nil
			}
			return map[string]any{}, nil
		case reflect.Map:
			return map[string]any{}, nil
		case reflect.Slice, reflect.Array:
			return []any{}, nil
		}
	}

	// 根据值类型分发到对应的处理函数
	switch val.Kind() {
	case reflect.Struct:
		// 特殊处理time.Time等内置类型
		if val.Type().String() == "time.Time" {
			return val.Interface(), nil
		}
		return g.marshalStruct(ctx, val, depth)

	case reflect.Map:
		return g.marshalMap(ctx, val, depth)

	case reflect.Slice, reflect.Array:
		return g.marshalSlice(ctx, val, depth)

	default:
		// 基本类型（数字、字符串、布尔等）直接返回
		return val.Interface(), nil
	}
}

// 处理结构体类型的序列化, 支持嵌套结构和匿名字段
// 会根据组标签过滤字段, 并处理JSON标签选项
func (g *GroupJSON) marshalStruct(ctx *encodeContext, val reflect.Value, depth int) (map[string]any, error) {
	typ := val.Type()
	result := make(map[string]any)

	// 检测循环引用 - 只对可寻址的值进行检查
	ptrAddr := g.getPointerAddress(val)
	if ptrAddr != 0 {
		if ctx.visited[ptrAddr] {
			// 发现循环引用, 返回空对象避免无限递归
			return result, nil
		}
		// 标记为已访问
		ctx.visited[ptrAddr] = true
		// 函数返回时移除标记, 允许在其他上下文中使用相同的值
		defer delete(ctx.visited, ptrAddr)
	}

	// 检查深度限制, 防止嵌套过深导致栈溢出
	if depth > g.opts.MaxDepth {
		return result, nil
	}

	// 临时存储匿名字段结果, 以便按标准库规则应用
	// 匿名字段的处理遵循encoding/json的规则
	anonymousFields := make([]map[string]any, 0)

	// 遍历结构体的所有字段
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过未导出字段（小写开头的字段）
		if !fieldType.IsExported() {
			continue
		}

		// 特殊处理匿名字段（嵌入字段）
		if fieldType.Anonymous {
			// 解引用指针类型的匿名字段
			fieldVal := field
			if fieldVal.Kind() == reflect.Ptr && !fieldVal.IsNil() {
				fieldVal = fieldVal.Elem()
			}

			// 只处理结构体类型的匿名字段
			if fieldVal.Kind() == reflect.Struct {
				// 匿名字段应视为当前层级的一部分, 不增加深度计数
				nestedResult, err := g.marshalStruct(ctx, fieldVal, depth)
				if err != nil {
					continue // 按标准库行为, 忽略错误的匿名字段
				}

				// 暂存匿名字段结果, 稍后应用
				anonymousFields = append(anonymousFields, nestedResult)
			}

			// 处理完匿名字段, 继续下一个字段
			continue
		}

		// 获取并处理json标签
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "-" {
			continue // 跳过标记为忽略的字段
		}

		// 解析json标签选项（名称和选项如omitempty）
		jsonOpts := strings.Split(jsonTag, ",")
		jsonName := fieldType.Name // 默认使用字段名
		if jsonOpts[0] != "" {
			jsonName = jsonOpts[0] // 使用标签指定的名称
		}

		// 应用各种json选项
		omitEmpty := false
		omitZero := false
		for _, opt := range jsonOpts[1:] {
			if opt == "omitempty" {
				omitEmpty = true // 空值忽略选项
			}
			if opt == "omitzero" {
				omitZero = true // 零值忽略选项（Go 1.24新特性）
			}
		}

		// 检查组标签, 确定字段是否属于指定组
		groupsTag := fieldType.Tag.Get(g.opts.TagKey)
		if groupsTag == "" {
			continue // 没有组标签的字段不包含
		}

		// 解析字段所属的组列表
		fieldGroups := strings.Split(groupsTag, ",")
		include := g.shouldIncludeField(fieldGroups)
		if !include {
			continue // 根据过滤规则排除不符合条件的字段
		}

		// 处理字段值 - 对于普通字段, 增加递归深度
		value, err := g.marshalValue(ctx, field, depth+1)
		if err != nil {
			return nil, err
		}

		// 处理omitempty和omitzero选项, 确定是否应省略该字段
		if shouldOmit(field, value, omitEmpty, omitZero) {
			continue
		}

		// 将字段添加到结果中
		result[jsonName] = value
	}

	// 应用匿名字段的结果, 遵循标准库规则
	// 只有在结果中不存在同名字段时才添加匿名字段的值
	for _, anonResult := range anonymousFields {
		for k, v := range anonResult {
			if _, exists := result[k]; !exists {
				result[k] = v
			}
		}
	}

	return result, nil
}

// 处理map类型的序列化, 支持任意值类型但键必须是字符串
func (g *GroupJSON) marshalMap(ctx *encodeContext, val reflect.Value, depth int) (any, error) {
	if val.IsNil() {
		return nil, nil
	}

	// 检查深度限制
	if depth > g.opts.MaxDepth {
		return map[string]any{}, nil
	}

	// 检测循环引用 - 对于map类型
	ptrAddr := g.getPointerAddress(val)
	if ptrAddr != 0 {
		if ctx.visited[ptrAddr] {
			// 发现循环引用, 返回空对象避免无限递归
			return map[string]any{}, nil
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
			// 只支持字符串键的map, 非字符串键会被跳过
			continue
		}

		keyStr := k.String()
		// 递归处理map的值部分
		itemVal, err := g.marshalValue(ctx, iter.Value(), depth+1)
		if err != nil {
			// 忽略达到递归深度的元素, 继续处理其他元素
			if errors.Is(err, ErrMaxDepth) {
				continue
			}
			return nil, err
		}

		result[keyStr] = itemVal
	}

	return result, nil
}

// 处理切片和数组类型的序列化, 支持任意元素类型
func (g *GroupJSON) marshalSlice(ctx *encodeContext, val reflect.Value, depth int) (any, error) {
	if val.IsNil() {
		return nil, nil
	}

	// 检查深度限制
	if depth > g.opts.MaxDepth {
		return []any{}, nil
	}

	// 检测循环引用 - 对于可寻址的slice/array
	ptrAddr := g.getPointerAddress(val)
	if ptrAddr != 0 {
		if ctx.visited[ptrAddr] {
			// 发现循环引用, 返回空数组避免无限递归
			return []any{}, nil
		}
		// 标记为已访问
		ctx.visited[ptrAddr] = true
		// 函数返回时移除标记
		defer delete(ctx.visited, ptrAddr)
	}

	// 预分配适当容量的切片以提高性能
	result := make([]any, 0, val.Len())
	for i := 0; i < val.Len(); i++ {
		// 递归处理每个元素
		itemVal, err := g.marshalValue(ctx, val.Index(i), depth+1)
		if err != nil {
			// 忽略达到递归深度的元素, 继续处理其他元素
			if errors.Is(err, ErrMaxDepth) {
				continue
			}
			return nil, err
		}
		result = append(result, itemVal)
	}

	return result, nil
}

// 根据分组逻辑确定是否应包含字段
// 支持AND和OR两种逻辑模式
func (g *GroupJSON) shouldIncludeField(fieldGroups []string) bool {
	// 如果没有指定过滤组, 不包含任何字段
	if len(g.opts.Groups) == 0 {
		return false
	}

	if g.opts.GroupMode == ModeAnd {
		// AND逻辑：字段必须同时属于所有指定组
		for _, group := range g.opts.Groups {
			found := false
			for _, fieldGroup := range fieldGroups {
				if fieldGroup == group {
					found = true
					break
				}
			}
			if !found {
				return false // 如有任一指定组不匹配, 则排除该字段
			}
		}
		return true // 所有指定组都匹配, 包含该字段
	} else {
		// OR逻辑（默认）：字段只要属于任一指定组即可
		for _, group := range g.opts.Groups {
			for _, fieldGroup := range fieldGroups {
				if fieldGroup == group {
					return true // 只要有一个组匹配即包含
				}
			}
		}
		return false // 没有任何组匹配, 排除该字段
	}
}

// 根据omitempty和omitzero标签确定是否应该省略字段
// 检查各种类型的空值和零值条件
func shouldOmit(field reflect.Value, value any, omitEmpty, omitZero bool) bool {
	// 处理nil值
	if value == nil {
		return omitEmpty
	}

	// 对于集合类型, 检查长度是否为0
	switch field.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		if field.Len() == 0 && omitEmpty {
			return true
		}
	}

	// 处理omitzero选项（Go 1.24新特性）
	// 对各种基本类型检查是否为零值
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

// 获取值的指针地址(如果是可寻址的)
// 用于循环引用检测
func (g *GroupJSON) getPointerAddress(v reflect.Value) uintptr {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return 0
		}
		return v.Pointer()
	}

	// 对于非指针值, 如果是可寻址的, 获取其指针地址
	if v.CanAddr() {
		return v.Addr().Pointer()
	}

	// 不可寻址的值不能检测循环引用
	return 0
}
