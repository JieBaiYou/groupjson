package groupjson

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"sync"
)

// Mode 定义分组匹配模式
// 用于控制当指定多个分组时，字段的筛选逻辑。
type Mode int

const (
	// ModeOr 表示字段只要匹配任意一个指定的分组即可（默认）。
	// 例如：Encode Groups 为 "admin", "public"，字段 Group 为 "admin" -> 包含。
	ModeOr Mode = iota
	// ModeAnd 表示字段必须匹配所有指定的分组。
	// 例如：Encode Groups 为 "admin", "public"，字段 Group 为 "admin" -> 排除。
	ModeAnd
)

// Encoder 是一个支持分组筛选的 JSON 编码器。
// 它是线程安全的（因为配置一旦设定即不可变），但通常作为临时对象使用。
type Encoder struct {
	groups []string // 需要保留的分组列表
	mode   Mode     // 分组匹配模式 (OR 或 AND)
}

// New 创建一个新的编码器，使用默认配置（ModeOr）。
func New() *Encoder {
	return &Encoder{
		mode: ModeOr,
	}
}

// WithGroups 设置需要保留的分组。
// 支持链式调用。
func (e *Encoder) WithGroups(groups ...string) *Encoder {
	// 复制切片防止外部修改
	e.groups = append([]string(nil), groups...)
	return e
}

// WithMode 设置分组匹配模式 (ModeOr 或 ModeAnd)。
// 支持链式调用。
func (e *Encoder) WithMode(mode Mode) *Encoder {
	e.mode = mode
	return e
}

// Marshal 将 v 序列化为 JSON，仅保留符合分组条件的字段。
//
// 行为说明：
// 1. 仅 struct 字段的 "groups" 标签参与筛选。
// 2. 完全遵循标准库 encoding/json 的行为（如 omitempty, string 标签, HTML 转义等）。
// 3. 遇到 map, slice, 指针会自动递归处理。
// 4. 遇到实现了 json.Marshaler 或 encoding.TextMarshaler 的类型，会直接调用其方法。
func (e *Encoder) Marshal(v any) ([]byte, error) {
	// 使用 sync.Pool 复用 buffer 优化性能，减少内存分配
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	// 创建编码上下文，用于防止循环引用和传递配置
	ctx := &encodeContext{
		encoder: e,
		visited: make(map[uintptr]struct{}),
	}

	if err := ctx.encode(buf, reflect.ValueOf(v)); err != nil {
		return nil, err
	}

	// 复制结果，因为 buf 会被放回池中，不能直接返回 buf.Bytes()
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// Marshal 是一个便捷函数，使用默认 OR 模式序列化。
func Marshal(v any, groups ...string) ([]byte, error) {
	return New().WithGroups(groups...).Marshal(v)
}

// -----------------------------------------------------------------------------
// 内部实现
// -----------------------------------------------------------------------------

// bufPool 全局缓冲池，复用 bytes.Buffer 减少 GC 压力。
var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// encodeContext 维护单次编码过程的状态。
// 包含编码器配置和循环引用检测所需的 visited map。
type encodeContext struct {
	encoder *Encoder             // 编码器配置引用
	visited map[uintptr]struct{} // 用于检测循环引用 (仅存储指针地址)
}

// encode 递归将值写入 buffer。
// 这是核心的分发函数，根据反射类型决定如何处理。
func (ctx *encodeContext) encode(buf *bytes.Buffer, v reflect.Value) error {
	if !v.IsValid() {
		buf.WriteString("null")
		return nil
	}

	// 处理接口和指针
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			buf.WriteString("null")
			return nil
		}
		// 检测循环引用 (仅针对非空指针)
		if v.Kind() == reflect.Pointer && !v.IsNil() {
			ptr := v.Pointer()
			if _, ok := ctx.visited[ptr]; ok {
				return errors.New("groupjson: circular reference detected")
			}
			// 标记当前指针已访问
			ctx.visited[ptr] = struct{}{}
			// 函数退出时移除标记
			defer delete(ctx.visited, ptr)
		}
		return ctx.encode(buf, v.Elem())
	}

	// 1. 优先支持标准库接口: json.Marshaler
	// 如果类型自定义了 JSON 序列化逻辑，直接使用它，不进行分组筛选。
	if v.CanInterface() {
		if m, ok := v.Interface().(json.Marshaler); ok {
			b, err := m.MarshalJSON()
			if err != nil {
				return err
			}
			// 标准库 MarshalJSON 返回的已经是合法的 JSON 字节，直接写入
			// 注意：这里不进行 Compact，假设实现是正确的
			buf.Write(b)
			return nil
		}
	}

	// 2. 优先支持标准库接口: encoding.TextMarshaler
	// 如果类型实现了 TextMarshaler，将其文本输出作为 JSON 字符串。
	if v.CanInterface() {
		if m, ok := v.Interface().(encoding.TextMarshaler); ok {
			text, err := m.MarshalText()
			if err != nil {
				return err
			}
			// 文本需作为 JSON 字符串输出 (带引号)
			writeString(buf, string(text))
			return nil
		}
	}

	// 3. 根据类型分发处理
	switch v.Kind() {
	case reflect.Struct:
		return ctx.encodeStruct(buf, v)
	case reflect.Map:
		return ctx.encodeMap(buf, v)
	case reflect.Slice, reflect.Array:
		return ctx.encodeSlice(buf, v)
	case reflect.String:
		writeString(buf, v.String())
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// 使用 encoding/json 相同的逻辑，不做特殊处理
		b, _ := json.Marshal(v.Interface())
		buf.Write(b)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		b, _ := json.Marshal(v.Interface())
		buf.Write(b)
		return nil
	case reflect.Float32, reflect.Float64:
		b, err := json.Marshal(v.Interface())
		if err != nil {
			return err
		}
		buf.Write(b)
		return nil
	case reflect.Bool:
		if v.Bool() {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
		return nil
	default:
		return errors.New("groupjson: unsupported type: " + v.Kind().String())
	}
}

// encodeStruct 处理结构体：核心分组逻辑在此。
func (ctx *encodeContext) encodeStruct(buf *bytes.Buffer, v reflect.Value) error {
	t := v.Type()
	// 获取缓存的结构体元数据 (Schema Cache)
	// 这一步通过缓存避免了重复的反射解析开销
	fields := getCachedFields(t)

	buf.WriteByte('{')
	first := true

	for _, f := range fields {
		// 1. 分组筛选 (Group Filter)
		// 检查当前字段的 groups 标签是否匹配用户请求的 groups
		if len(ctx.encoder.groups) > 0 {
			if !matchGroups(f.groups, ctx.encoder.groups, ctx.encoder.mode) {
				continue // 不匹配则跳过该字段
			}
		}

		// 获取字段值 (支持匿名字段路径)
		fv := v
		for _, i := range f.index {
			if fv.Kind() == reflect.Pointer {
				if fv.IsNil() {
					goto NEXT_FIELD // 指针为空，无法获取深层字段，跳过整个字段
				}
				fv = fv.Elem()
			}
			fv = fv.Field(i)
		}

		// 2. omitempty 处理
		// 如果标记了 omitempty 且值为零值，则跳过
		if f.omitEmpty && isEmptyValue(fv) {
			continue
		}

		if !first {
			buf.WriteByte(',')
		}
		first = false

		// 写入键名
		// 这里的 quotedName 已经预先计算好了 (包含引号和冒号，如 "key":)
		buf.WriteString(f.quotedName)

		// 3. string 标签处理 (将字段值再次转为 JSON 字符串)
		// 对应 `json:",string"` 选项
		if f.asString {
			// 这是一个特殊情况：需要先将值编码到临时 buffer (得到原始 JSON)，
			// 然后再将这个 JSON 文本作为字符串写入主 buffer。
			var tmp bytes.Buffer
			// 复用 ctx 进行递归编码，以便正确处理深层逻辑
			if err := ctx.encode(&tmp, fv); err != nil {
				return err
			}
			writeString(buf, tmp.String())
		} else {
			// 正常递归编码
			if err := ctx.encode(buf, fv); err != nil {
				return err
			}
		}

	NEXT_FIELD:
	}
	buf.WriteByte('}')
	return nil
}

// encodeMap 处理 Map。
// JSON 要求 Map 的 Key 必须是字符串。
func (ctx *encodeContext) encodeMap(buf *bytes.Buffer, v reflect.Value) error {
	if v.IsNil() {
		buf.WriteString("null")
		return nil
	}
	// 检查 Key 是否为字符串
	if v.Type().Key().Kind() != reflect.String {
		return errors.New("groupjson: map key must be string")
	}

	buf.WriteByte('{')

	// 提取所有 Key 并排序 (标准库行为：Key 必须排序以保证确定性)
	keys := v.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})

	first := true
	for _, k := range keys {
		val := v.MapIndex(k)

		if !first {
			buf.WriteByte(',')
		}
		first = false

		// 写入 Key
		writeString(buf, k.String())
		buf.WriteByte(':')

		// 递归写入 Value
		if err := ctx.encode(buf, val); err != nil {
			return err
		}
	}
	buf.WriteByte('}')
	return nil
}

// encodeSlice 处理切片和数组。
func (ctx *encodeContext) encodeSlice(buf *bytes.Buffer, v reflect.Value) error {
	if v.Kind() == reflect.Slice && v.IsNil() {
		buf.WriteString("null")
		return nil
	}
	// 特殊处理：[]byte 转 Base64 (标准库行为)
	if v.Type().Elem().Kind() == reflect.Uint8 {
		b, err := json.Marshal(v.Interface())
		if err != nil {
			return err
		}
		buf.Write(b)
		return nil
	}

	buf.WriteByte('[')
	n := v.Len()
	for i := 0; i < n; i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		if err := ctx.encode(buf, v.Index(i)); err != nil {
			return err
		}
	}
	buf.WriteByte(']')
	return nil
}

// matchGroups 检查字段分组是否满足条件。
func matchGroups(fieldGroups []string, targetGroups []string, mode Mode) bool {
	// 字段未定义分组时，视为不属于任何分组。
	// 如果目标分组不为空，则该字段默认被排除。
	if len(fieldGroups) == 0 {
		return false
	}

	if mode == ModeOr {
		// OR 模式：有一个匹配即可
		for _, fg := range fieldGroups {
			for _, tg := range targetGroups {
				if fg == tg {
					return true
				}
			}
		}
		return false
	}

	// AND 模式：所有目标分组都必须存在于字段分组中。
	// 注意：这里是 "Requested Groups" SUBSET OF "Field Groups"。
	// 也就是说，如果我请求 "admin, public"，字段必须同时有 "admin" 和 "public" 标签。
	for _, tg := range targetGroups {
		found := false
		for _, fg := range fieldGroups {
			if fg == tg {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// writeString 将 s 转义并写入 buf (标准库行为，包括 HTML 转义)。
func writeString(buf *bytes.Buffer, s string) {
	// 直接利用 json.Marshal 处理字符串转义，简单且正确
	b, _ := json.Marshal(s)
	buf.Write(b)
}

// -----------------------------------------------------------------------------
// 结构体元数据缓存 (Schema Cache)
// -----------------------------------------------------------------------------

// fieldInfo 存储单个字段的预处理信息。
// 这些信息在首次反射解析时计算，后续复用。
type fieldInfo struct {
	index      []int    // 字段索引路径 (支持嵌入字段)
	name       string   // 字段名 (Go Struct Field Name)
	quotedName string   // 预处理后的键名，包含冒号，如 "name":
	omitEmpty  bool     // 是否有 omitempty 标签
	asString   bool     // 是否有 string 标签
	groups     []string // 所属分组列表
}

var fieldCache sync.Map // 全局缓存: map[reflect.Type][]fieldInfo

// getCachedFields 获取或构建结构体字段信息。
func getCachedFields(t reflect.Type) []fieldInfo {
	if v, ok := fieldCache.Load(t); ok {
		return v.([]fieldInfo)
	}

	// 构建字段信息，遵循标准库规则：
	// 1. 导出字段。
	// 2. json:"-" 忽略。
	// 3. 匿名字段提升。
	// 4. 优先使用 json 标签名。
	// 5. 冲突处理 (这里简化实现，暂不处理复杂的层级冲突，按 BFS 顺序优先)。

	var fields []fieldInfo
	// 使用 BFS 遍历字段以支持匿名字段
	type item struct {
		typ   reflect.Type
		index []int
	}
	queue := []item{{typ: t, index: nil}}

	// 简单的去重 map，key 为最终的 JSON 字段名
	seen := make(map[string]bool)

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		// 必须是 Struct
		if curr.typ.Kind() != reflect.Struct {
			continue
		}

		for i := 0; i < curr.typ.NumField(); i++ {
			sf := curr.typ.Field(i)

			// 忽略未导出字段 (PkgPath 不为空)
			if sf.PkgPath != "" {
				// 除非是嵌入的结构体，标准库允许嵌入未导出结构体中的导出字段
				if !sf.Anonymous || sf.Type.Kind() != reflect.Struct {
					continue
				}
			}

			tag := sf.Tag.Get("json")
			if tag == "-" {
				continue
			}

			// 解析 json 标签
			parts := strings.Split(tag, ",")
			name := sf.Name
			if len(parts) > 0 && parts[0] != "" {
				name = parts[0]
			}

			omitEmpty := false
			asString := false
			for _, opt := range parts[1:] {
				if opt == "omitempty" {
					omitEmpty = true
				}
				if opt == "string" {
					asString = true
				}
			}

			// 处理索引
			currIndex := make([]int, len(curr.index)+1)
			copy(currIndex, curr.index)
			currIndex[len(curr.index)] = i

			// 如果是匿名字段，且没有指定 JSON 名称，则将其字段提升 (Flatten)
			if sf.Anonymous && (len(parts) == 0 || parts[0] == "") {
				t := sf.Type
				if t.Kind() == reflect.Ptr {
					t = t.Elem()
				}
				if t.Kind() == reflect.Struct {
					queue = append(queue, item{typ: t, index: currIndex})
					continue
				}
			}

			if seen[name] {
				continue
			}
			seen[name] = true

			// 解析 groups 标签
			var groups []string
			if gTag := sf.Tag.Get("groups"); gTag != "" {
				groups = strings.Split(gTag, ",")
			}

			// 预先生成带引号和冒号的键名，避免运行时拼接
			qName, _ := json.Marshal(name)
			quotedName := string(qName) + ":"

			fields = append(fields, fieldInfo{
				index:      currIndex,
				name:       name,
				quotedName: quotedName,
				omitEmpty:  omitEmpty,
				asString:   asString,
				groups:     groups,
			})
		}
	}

	fieldCache.Store(t, fields)
	return fields
}

// isEmptyValue 判断值是否为空 (用于 omitempty 逻辑)。
// 遵循 encoding/json 的定义。
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	}
	return false
}
