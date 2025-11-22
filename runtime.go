package groupjson

import (
	"bytes"
	"encoding"
	"encoding/json"
	"io"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Encoder 为不可变的分组序列化器。
// 通过 WithXxx 方法返回新副本，确保可安全复用与并发使用。
type Encoder struct {
	// opts 编码行为配置，不随方法调用被就地修改。
	opts Options
}

// NewEncoder 创建带默认选项的 Encoder
func NewEncoder() Encoder { return Encoder{opts: DefaultOptions()} }

// 便捷函数
func Marshal(v any, groups ...string) ([]byte, error) {
	return NewEncoder().WithGroups(groups...).Marshal(v)
}

func MarshalWith(opts Options, v any, groups ...string) ([]byte, error) {
	return Encoder{opts: opts}.WithGroups(groups...).Marshal(v)
}

// WithXxx 返回复制后的新 Encoder（不可变 Builder）。
func (e Encoder) WithGroups(groups ...string) Encoder {
	e.opts.Groups = append([]string(nil), groups...)
	return e
}
func (e Encoder) WithGroupMode(mode GroupMode) Encoder { e.opts.Mode = mode; return e }
func (e Encoder) WithTagKey(key string) Encoder        { e.opts.TagKey = key; return e }
func (e Encoder) WithTopLevelKey(key string) Encoder   { e.opts.TopLevelKey = key; return e }
func (e Encoder) WithMaxDepth(n int) Encoder {
	if n < 1 {
		n = 1
	}
	e.opts.MaxDepth = n
	return e
}
func (e Encoder) WithEscapeHTML(on bool) Encoder { e.opts.EscapeHTML = on; return e }
func (e Encoder) WithSortKeys(on bool) Encoder   { e.opts.SortKeys = on; return e }

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// Marshal 输出 JSON 字节。
func (e Encoder) Marshal(v any) ([]byte, error) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	if e.opts.TopLevelKey != "" {
		buf.WriteByte('{')
		e.writeString(buf, e.opts.TopLevelKey)
		buf.WriteByte(':')
	}

	if err := e.encode(buf, reflect.ValueOf(v), newContext(e.opts)); err != nil {
		return nil, err
	}

	if e.opts.TopLevelKey != "" {
		buf.WriteByte('}')
	}

	// 复制字节以避免复用 buffer 时的数据污染
	return append([]byte(nil), buf.Bytes()...), nil
}

// Encode 直接写入 io.Writer，避免中间 []byte 拷贝。
func (e Encoder) Encode(w io.Writer, v any) error {
	// 为了复用 encode 逻辑，暂时先写入 buffer 再写入 writer
	// 真正的流式优化可以在后续版本通过直接操作 writer 实现，
	// 但考虑到很多 writer 是无缓冲的，先写入 buffer 也是一种优良实践。
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	if e.opts.TopLevelKey != "" {
		buf.WriteByte('{')
		e.writeString(buf, e.opts.TopLevelKey)
		buf.WriteByte(':')
	}

	if err := e.encode(buf, reflect.ValueOf(v), newContext(e.opts)); err != nil {
		return err
	}

	if e.opts.TopLevelKey != "" {
		buf.WriteByte('}')
	}

	_, err := w.Write(buf.Bytes())
	return err
}

// ----- 上下文与缓存 -----

// context 维护单次编码过程的状态。
type context struct {
	// opts 编码配置快照
	opts Options
	// depth 当前递归深度
	depth int
	// visited 指针身份访问集，用于循环检测
	visited map[uintptr]struct{}
}

func newContext(opts Options) *context {
	return &context{opts: opts, depth: 0, visited: make(map[uintptr]struct{})}
}

func (c *context) incDepth() error {
	c.depth++
	if c.depth > c.opts.MaxDepth {
		return ErrMaxDepth
	}
	return nil
}

func (c *context) decDepth() {
	if c.depth > 0 {
		c.depth--
	}
}

var schemaCache sync.Map // key: schemaKey

type schemaKey struct {
	t      reflect.Type
	tagKey string
}

type fieldInfo struct {
	// name Go 字段名（未导出不入表）
	name string
	// jsonName 输出使用的 JSON 键名
	jsonName string
	// keyBytes 预计算的键名 JSON 字节，包含引号和冒号，如 "key":
	keyBytes []byte
	// index 反射字段索引路径（支持匿名提升）
	index []int
	// omitEmpty 是否应用 omitempty 省略规则
	omitEmpty bool
	// omitZero 是否应用 omitzero 省略规则（仅标量零值）
	omitZero bool
	// groups 从 TagKey 标签解析出的分组列表
	groups []string
	// anonymous 是否为匿名字段（仅用于构建期判断）
	anonymous bool
}

type schema struct {
	// fields 该类型在当前 TagKey 下可见且可导出的字段信息
	fields []fieldInfo
}

func getSchema(t reflect.Type, tagKey string) *schema {
	key := schemaKey{t: t, tagKey: tagKey}
	if v, ok := schemaCache.Load(key); ok {
		return v.(*schema)
	}
	s := buildSchema(t, tagKey)
	schemaCache.Store(key, s)
	return s
}

func buildSchema(t reflect.Type, tagKey string) *schema {
	// BFS 按标准库规则收集导出字段，处理匿名嵌入与冲突
	type queueItem struct {
		t     reflect.Type
		index []int
		depth int
	}
	q := []queueItem{{t: t, index: nil, depth: 0}}
	out := make([]fieldInfo, 0, t.NumField())
	seen := map[string]int{} // jsonName -> idx in out (浅层优先)

	for len(q) > 0 {
		it := q[0]
		q = q[1:]
		if it.t.Kind() != reflect.Struct {
			continue
		}
		n := it.t.NumField()
		for i := 0; i < n; i++ {
			sf := it.t.Field(i)
			if sf.PkgPath != "" { // 未导出
				continue
			}
			tag := sf.Tag.Get("json")
			if tag == "-" {
				continue
			}
			parts := strings.Split(tag, ",")
			jname := sf.Name
			if len(parts[0]) > 0 {
				jname = parts[0]
			}
			omitEmpty := false
			omitZero := false
			for _, p := range parts[1:] {
				if p == "omitempty" {
					omitEmpty = true
				}
				if p == "omitzero" {
					omitZero = true
				}
			}

			if sf.Anonymous && (sf.Type.Kind() == reflect.Struct || (sf.Type.Kind() == reflect.Ptr && sf.Type.Elem().Kind() == reflect.Struct)) && (len(parts[0]) == 0) {
				// 匿名嵌入，按标准库进行字段提升
				st := sf.Type
				if st.Kind() == reflect.Ptr {
					st = st.Elem()
				}
				base := append(append([]int(nil), it.index...), i)
				q = append(q, queueItem{t: st, index: base, depth: it.depth + 1})
				continue
			}

			groups := strings.Split(sf.Tag.Get(tagKey), ",")
			idx := append(append([]int(nil), it.index...), i)

			// 预计算 keyBytes: "jsonName":
			kb, _ := json.Marshal(jname)
			kb = append(kb, ':')

			fi := fieldInfo{
				name:      sf.Name,
				jsonName:  jname,
				keyBytes:  kb,
				index:     idx,
				omitEmpty: omitEmpty,
				omitZero:  omitZero,
				groups:    groups,
				anonymous: sf.Anonymous,
			}
			if prev, ok := seen[jname]; ok {
				// 冲突：保留更浅层（先入队的），与 encoding/json 一致
				_ = prev
				continue
			}
			seen[jname] = len(out)
			out = append(out, fi)
		}
	}

	return &schema{fields: out}
}

// ----- 编码实现 -----

func (e Encoder) encode(buf *bytes.Buffer, v reflect.Value, ctx *context) error {
	if !v.IsValid() {
		buf.WriteString("null")
		return nil
	}

	// 处理 nil 指针/接口
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			buf.WriteString("null")
			return nil
		}
		return e.encode(buf, v.Elem(), ctx)
	}

	// 优先使用 json.Marshaler / encoding.TextMarshaler
	if m, ok := asJSONMarshaler(v); ok {
		b, err := m.MarshalJSON()
		if err != nil {
			return err
		}
		buf.Write(b)
		return nil
	}
	if tm, ok := asTextMarshaler(v); ok {
		txt, err := tm.MarshalText()
		if err != nil {
			return err
		}
		e.writeString(buf, string(txt))
		return nil
	}

	// 特殊：[]byte 遵循标准库编码为 base64 字符串
	if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
		b, err := json.Marshal(v.Interface())
		if err != nil {
			return err
		}
		buf.Write(b)
		return nil
	}

	switch v.Kind() {
	case reflect.Struct:
		return e.encodeStruct(buf, v, ctx)
	case reflect.Map:
		return e.encodeMap(buf, v, ctx)
	case reflect.Slice, reflect.Array:
		return e.encodeSlice(buf, v, ctx)
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return ErrUnsupportedType
	default:
		// 标量
		return e.encodeScalar(buf, v)
	}
}

func (e Encoder) encodeStruct(buf *bytes.Buffer, v reflect.Value, ctx *context) error {
	if err := ctx.incDepth(); err != nil {
		return err
	}
	defer ctx.decDepth()

	// 循环检测（仅指针身份）
	if v.CanAddr() {
		addr := v.Addr().Pointer()
		if _, ok := ctx.visited[addr]; ok {
			return ErrCircularReference
		}
		ctx.visited[addr] = struct{}{}
		defer delete(ctx.visited, addr)
	}

	t := v.Type()
	sch := getSchema(t, e.opts.TagKey)

	buf.WriteByte('{')
	first := true

	for _, f := range sch.fields {
		if len(e.opts.Groups) > 0 && !e.includeField(f.groups) {
			continue
		}

		fv := fieldByIndex(v, f.index)

		// 检查 omit 规则
		if f.omitEmpty && isEmptyValue(fv) {
			continue
		}
		if f.omitZero && isZeroScalar(fv) {
			continue
		}

		if !first {
			buf.WriteByte(',')
		}
		first = false

		buf.Write(f.keyBytes)
		if err := e.encode(buf, fv, ctx); err != nil {
			return err
		}
	}

	buf.WriteByte('}')
	return nil
}

func (e Encoder) encodeMap(buf *bytes.Buffer, v reflect.Value, ctx *context) error {
	if v.IsNil() {
		buf.WriteString("null")
		return nil
	}
	if err := ctx.incDepth(); err != nil {
		return err
	}
	defer ctx.decDepth()

	if v.Type().Key().Kind() != reflect.String {
		return ErrNonStringMapKey
	}

	buf.WriteByte('{')

	// 获取所有 key 并排序（如果需要）
	keys := v.MapKeys()
	if e.opts.SortKeys {
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		})
	}

	first := true
	for _, key := range keys {
		val := v.MapIndex(key)

		if !first {
			buf.WriteByte(',')
		}
		first = false

		// 写入 key
		e.writeString(buf, key.String())
		buf.WriteByte(':')

		// 写入 value
		if err := e.encode(buf, val, ctx); err != nil {
			return err
		}
	}

	buf.WriteByte('}')
	return nil
}

func (e Encoder) encodeSlice(buf *bytes.Buffer, v reflect.Value, ctx *context) error {
	if v.Kind() == reflect.Slice && v.IsNil() {
		buf.WriteString("null")
		return nil
	}
	if err := ctx.incDepth(); err != nil {
		return err
	}
	defer ctx.decDepth()

	buf.WriteByte('[')
	n := v.Len()
	for i := 0; i < n; i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		if err := e.encode(buf, v.Index(i), ctx); err != nil {
			return err
		}
	}
	buf.WriteByte(']')
	return nil
}

func (e Encoder) encodeScalar(buf *bytes.Buffer, v reflect.Value) error {
	switch v.Kind() {
	case reflect.String:
		e.writeString(buf, v.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		buf.WriteString(strconv.FormatInt(v.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		buf.WriteString(strconv.FormatUint(v.Uint(), 10))
	case reflect.Float32, reflect.Float64:
		// 模仿 json 标准库的 float 格式化
		f := v.Float()
		if json.Valid([]byte(strconv.FormatFloat(f, 'f', -1, 64))) {
			// 简单的校验不够，标准库有更复杂的逻辑处理 NaN/Inf
			// 直接用 strconv 即可，但 NaN/Inf 会生成无效 JSON。
			// 标准库 json 会报错：UnsupportedValueError
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return &json.UnsupportedValueError{Value: v, Str: strconv.FormatFloat(f, 'g', -1, 64)}
			}
		}
		// 使用 -1 让 strconv 自动选择最简格式
		// 标准 json 库对 float64 使用 'g', -1, 64，对 float32 使用 32
		bitSize := 64
		if v.Kind() == reflect.Float32 {
			bitSize = 32
		}
		buf.WriteString(strconv.FormatFloat(f, 'g', -1, bitSize))
	case reflect.Bool:
		if v.Bool() {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	default:
		return ErrUnsupportedType // 支持的标量类型不应到达此处
	}
	return nil
}

// writeString 写入字符串，根据 EscapeHTML 选项决定转义策略
func (e Encoder) writeString(buf *bytes.Buffer, s string) {
	if e.opts.EscapeHTML {
		b, _ := json.Marshal(s)
		buf.Write(b)
	} else {
		// 使用 Encoder 关闭 HTML 转义
		// 这种方式略慢，但为了正确性。
		// 可以考虑优化：手动检查是否含有 HTML 字符，没有则直接 json.Marshal
		// 既然是 debloat，先用正确的方法。
		start := buf.Len()
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		enc.Encode(s)
		// Encode 增加了一个换行符，需要移除
		if buf.Len() > start {
			buf.Truncate(buf.Len() - 1)
		}
	}
}

func fieldByIndex(v reflect.Value, index []int) reflect.Value {
	for _, i := range index {
		v = v.Field(i)
		if v.Kind() == reflect.Ptr && !v.IsNil() {
			v = v.Elem()
		}
	}
	return v
}

func (e Encoder) includeField(fieldGroups []string) bool {
	if len(e.opts.Groups) == 0 {
		return false
	}
	switch e.opts.Mode {
	case ModeAnd:
		for _, g := range e.opts.Groups {
			found := false
			for _, fg := range fieldGroups {
				if fg == g {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	default: // OR
		for _, g := range e.opts.Groups {
			for _, fg := range fieldGroups {
				if fg == g {
					return true
				}
			}
		}
		return false
	}
}

func isZeroScalar(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.String:
		return v.Len() == 0
	default:
		return false
	}
}

// isEmptyValue 模仿 encoding/json 的实现，用于 omitempty
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
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

// asJSONMarshaler 尝试提取 json.Marshaler 接口
func asJSONMarshaler(v reflect.Value) (json.Marshaler, bool) {
	if !v.IsValid() {
		return nil, false
	}
	if v.CanInterface() {
		if m, ok := v.Interface().(json.Marshaler); ok {
			return m, true
		}
	}
	if v.CanAddr() {
		pv := v.Addr()
		if pv.CanInterface() {
			if m, ok := pv.Interface().(json.Marshaler); ok {
				return m, true
			}
		}
	}
	return nil, false
}

// asTextMarshaler 尝试提取 encoding.TextMarshaler 接口
func asTextMarshaler(v reflect.Value) (encoding.TextMarshaler, bool) {
	if !v.IsValid() {
		return nil, false
	}
	if v.CanInterface() {
		if m, ok := v.Interface().(encoding.TextMarshaler); ok {
			return m, true
		}
	}
	if v.CanAddr() {
		pv := v.Addr()
		if pv.CanInterface() {
			if m, ok := pv.Interface().(encoding.TextMarshaler); ok {
				return m, true
			}
		}
	}
	return nil, false
}
