package groupjson

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
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
func (e Encoder) WithDepthPolicy(p DepthPolicy) Encoder { e.opts.DepthPolicy = p; return e }
func (e Encoder) WithCutoffCollection(c CutoffCollection) Encoder {
	e.opts.CutoffCollection = c
	return e
}
func (e Encoder) WithEscapeHTML(on bool) Encoder { e.opts.EscapeHTML = on; return e }
func (e Encoder) WithSortKeys(on bool) Encoder   { e.opts.SortKeys = on; return e }
func (e Encoder) AllowMap(on bool) Encoder       { e.opts.AllowMapInput = on; return e }
func (e Encoder) AllowSlice(on bool) Encoder     { e.opts.AllowSliceInput = on; return e }

// Marshal 输出 JSON 字节。
// 若设置 TopLevelKey，则以该键包裹结果对象。
func (e Encoder) Marshal(v any) ([]byte, error) {
	m, err := e.MarshalToMap(v)
	if err != nil {
		return nil, err
	}
	if e.opts.TopLevelKey != "" {
		m = map[string]any{e.opts.TopLevelKey: m}
	}
	buf := bytes.Buffer{}
	enc := json.NewEncoder(&buf)
	if !e.opts.EscapeHTML {
		enc.SetEscapeHTML(false)
	}
	if err := enc.Encode(m); err != nil {
		return nil, err
	}
	b := bytes.TrimRight(buf.Bytes(), "\n")
	return b, nil
}

// Encode 直接写入 io.Writer，避免中间 []byte 拷贝。
func (e Encoder) Encode(w io.Writer, v any) error {
	m, err := e.MarshalToMap(v)
	if err != nil {
		return err
	}
	if e.opts.TopLevelKey != "" {
		m = map[string]any{e.opts.TopLevelKey: m}
	}
	enc := json.NewEncoder(w)
	if !e.opts.EscapeHTML {
		enc.SetEscapeHTML(false)
	}
	return enc.Encode(m)
}

// MarshalToMap 仅构建 map 结果，便于调用方在序列化前做二次加工。
func (e Encoder) MarshalToMap(v any) (map[string]any, error) {
	if v == nil {
		return nil, ErrNilValue
	}
	val := reflect.ValueOf(v)
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, ErrNilValue
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		ctx := newContext(e.opts)
		return e.encodeStruct(ctx, val)
	case reflect.Map:
		if !e.opts.AllowMapInput {
			return nil, ErrInvalidType
		}
		ctx := newContext(e.opts)
		out, err := e.encodeMapTop(ctx, val)
		if err != nil {
			return nil, err
		}
		return out, nil
	case reflect.Slice, reflect.Array:
		if !e.opts.AllowSliceInput {
			return nil, ErrInvalidType
		}
		ctx := newContext(e.opts)
		arr, err := e.encodeSlice(ctx, val)
		if err != nil {
			return nil, err
		}
		return map[string]any{"data": arr}, nil
	default:
		return nil, ErrInvalidType
	}
}

// ----- 上下文与缓存 -----

// context 维护单次编码过程的状态。
type context struct {
	// opts 编码配置快照
	opts Options
	// depth 当前递归深度
	depth int
	// path 当前字段路径（以 . 连接，数组与 map 键使用 [i]/["k"]）
	path []string
	// visited 指针身份访问集，用于循环检测
	visited map[uintptr]struct{}
}

func newContext(opts Options) *context {
	return &context{opts: opts, depth: 0, path: make([]string, 0, 8), visited: make(map[uintptr]struct{})}
}

func (c *context) push(field string) { c.path = append(c.path, field) }
func (c *context) pop() {
	if len(c.path) > 0 {
		c.path = c.path[:len(c.path)-1]
	}
}

func (c *context) pathStr() string {
	var b strings.Builder
	for i, seg := range c.path {
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(seg)
	}
	return b.String()
}

func (c *context) incDepth() error {
	c.depth++
	if c.depth > c.opts.MaxDepth {
		// 始终返回错误信号，由调用方依据策略决定报错或截断
		return WrapError(ErrMaxDepth, c.pathStr())
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
			fi := fieldInfo{name: sf.Name, jsonName: jname, index: idx, omitEmpty: omitEmpty, omitZero: omitZero, groups: groups, anonymous: sf.Anonymous}
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

func (e Encoder) encodeStruct(ctx *context, v reflect.Value) (map[string]any, error) {
	if err := ctx.incDepth(); err != nil {
		if e.opts.DepthPolicy == DepthError {
			return nil, err
		}
		// 截断：容器被置为 nil
		return nil, nil
	}
	defer ctx.decDepth()

	// 循环检测（仅指针身份）
	if v.CanAddr() {
		addr := v.Addr().Pointer()
		if _, ok := ctx.visited[addr]; ok {
			return nil, WrapError(ErrCircularReference, ctx.pathStr())
		}
		ctx.visited[addr] = struct{}{}
		defer delete(ctx.visited, addr)
	}

	t := v.Type()
	sch := getSchema(t, e.opts.TagKey)
	out := make(map[string]any, len(sch.fields))

	for _, f := range sch.fields {
		if len(e.opts.Groups) == 0 {
			continue
		}
		if !e.includeField(f.groups) {
			continue
		}

		fv := fieldByIndex(v, f.index)
		name := f.jsonName
		ctx.push(name)
		val, omit, err := e.encodeValue(ctx, fv, f.omitEmpty, f.omitZero)
		ctx.pop()
		if err != nil {
			return nil, WrapError(err, name)
		}
		if omit {
			continue
		}
		out[name] = val
	}

	if e.opts.SortKeys {
		// 无需处理，json.Encoder 不保证顺序；排序由外层决定是否需要
	}
	return out, nil
}

func (e Encoder) encodeMapTop(ctx *context, v reflect.Value) (map[string]any, error) {
	if v.IsNil() {
		return nil, nil
	}
	if v.Type().Key().Kind() != reflect.String {
		return nil, WrapError(ErrNonStringMapKey, ctx.pathStr())
	}
	out := make(map[string]any, v.Len())
	iter := v.MapRange()
	for iter.Next() {
		k := iter.Key().String()
		ctx.push(fmt.Sprintf("[\"%s\"]", k))
		item := iter.Value()
		val, _, err := e.encodeValue(ctx, item, false, false)
		ctx.pop()
		if err != nil {
			return nil, WrapError(err, fmt.Sprintf("[\"%s\"]", k))
		}
		out[k] = val
	}
	return out, nil
}

func (e Encoder) encodeSlice(ctx *context, v reflect.Value) ([]any, error) {
	if v.Kind() == reflect.Slice && v.IsNil() {
		return nil, nil
	}
	n := v.Len()
	out := make([]any, 0, n)
	for i := 0; i < n; i++ {
		ctx.push(fmt.Sprintf("[%d]", i))
		val, _, err := e.encodeValue(ctx, v.Index(i), false, false)
		ctx.pop()
		if err != nil {
			return nil, WrapError(err, fmt.Sprintf("[%d]", i))
		}
		out = append(out, val)
	}
	return out, nil
}

func (e Encoder) encodeValue(ctx *context, v reflect.Value, omitEmpty, omitZero bool) (any, bool, error) {
	// 处理 nil 指针
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, omitEmpty, nil
		}
		return e.encodeValue(ctx, v.Elem(), omitEmpty, omitZero)
	}
	// 解包 interface 值
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil, omitEmpty, nil
		}
		return e.encodeValue(ctx, v.Elem(), omitEmpty, omitZero)
	}

	// 通用接口：优先使用 json.Marshaler / encoding.TextMarshaler
	if m, ok := asJSONMarshaler(v); ok {
		b, err := m.MarshalJSON()
		if err != nil {
			return nil, false, err
		}
		var anyVal any
		if err := json.Unmarshal(b, &anyVal); err != nil {
			return nil, false, err
		}
		return anyVal, false, nil
	}
	if tm, ok := asTextMarshaler(v); ok {
		txt, err := tm.MarshalText()
		if err != nil {
			return nil, false, err
		}
		if omitEmpty && len(txt) == 0 {
			return nil, true, nil
		}
		return string(txt), false, nil
	}

	// 特殊：[]byte 遵循标准库编码为 base64 字符串
	if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
		if omitEmpty && v.Len() == 0 {
			return nil, true, nil
		}
		return v.Interface(), false, nil
	}

	switch v.Kind() {
	case reflect.Struct:
		m, err := e.encodeStruct(ctx, v)
		return m, false, err
	case reflect.Map:
		if err := ctx.incDepth(); err != nil {
			if e.opts.DepthPolicy == DepthError {
				return nil, false, err
			}
			if e.opts.CutoffCollection == Empty {
				return map[string]any{}, false, nil
			}
			return nil, false, nil
		}
		defer ctx.decDepth()
		if v.Type().Key().Kind() != reflect.String {
			return nil, false, WrapError(ErrNonStringMapKey, ctx.pathStr())
		}
		// omitempty 针对空 map 省略（nil 或 len==0）
		if omitEmpty {
			if v.IsNil() || v.Len() == 0 {
				return nil, true, nil
			}
		}
		// 普通 map：递归编码值
		out := make(map[string]any, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			k := iter.Key().String()
			ctx.push(fmt.Sprintf("[\"%s\"]", k))
			val, _, err := e.encodeValue(ctx, iter.Value(), false, false)
			ctx.pop()
			if err != nil {
				return nil, false, WrapError(err, fmt.Sprintf("[\"%s\"]", k))
			}
			out[k] = val
		}
		return out, false, nil
	case reflect.Slice, reflect.Array:
		if err := ctx.incDepth(); err != nil {
			if e.opts.DepthPolicy == DepthError {
				return nil, false, err
			}
			if e.opts.CutoffCollection == Empty {
				return []any{}, false, nil
			}
			return nil, false, nil
		}
		defer ctx.decDepth()
		// omitempty 针对空集合省略
		if omitEmpty {
			if (v.Kind() == reflect.Slice && v.IsNil()) || v.Len() == 0 {
				return nil, true, nil
			}
		}
		arr, err := e.encodeSlice(ctx, v)
		return arr, false, err
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return nil, false, WrapError(ErrUnsupportedType, ctx.pathStr())
	default:
		// 标量 + 省略策略
		if omitEmpty && v.IsZero() {
			return nil, true, nil
		}
		if omitZero && isZeroScalar(v) {
			return nil, true, nil
		}
		return v.Interface(), false, nil
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
