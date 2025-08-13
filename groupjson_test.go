package groupjson

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

type Meta struct {
	CreatedAt time.Time `json:"created_at" groups:"public,admin"`
	UpdatedAt time.Time `json:"updated_at" groups:"admin"`
}

type Address struct {
	City  string `json:"city" groups:"public,admin"`
	Line1 string `json:"line1,omitempty" groups:"public,admin"`
}

type User struct {
	ID       int      `json:"id" groups:"public,admin"`
	Name     string   `json:"name" groups:"public,admin"`
	Email    string   `json:"email" groups:"admin"`
	Password string   `json:"password" groups:"internal"`
	Tags     []string `json:"tags,omitempty" groups:"public,admin"`
	Scores   []int    `json:"scores,omitzero" groups:"public,admin"`
	Addr     Address  `json:"address" groups:"public,admin"`
	Meta
}

// 匿名字段冲突测试
type Shallow struct {
	Name string `json:"name" groups:"public"`
}
type Deep struct {
	Shallow
	Name string `json:"name" groups:"public"`
}

// 不支持类型与非字符串键 map
type Bad struct {
	C chan string    `json:"c" groups:"public"`
	M map[int]string `json:"m" groups:"public"`
}

// 循环引用
type Node struct {
	Val  int   `json:"val" groups:"public"`
	Next *Node `json:"next" groups:"public"`
}

func TestOrAndGroups(t *testing.T) {
	u := User{ID: 1, Name: "A", Email: "a@x", Password: "p", Addr: Address{City: "SZ"}}

	// OR（默认）
	b, err := NewEncoder().WithGroups("public").Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "\"id\":1") || !strings.Contains(s, "\"name\":\"A\"") {
		t.Fatalf("public fields missing: %s", s)
	}
	if strings.Contains(s, "email") || strings.Contains(s, "password") {
		t.Fatalf("private fields leaked: %s", s)
	}

	// AND：仅包含同时属于 public 与 admin 的字段
	b, err = NewEncoder().WithGroups("public", "admin").WithGroupMode(ModeAnd).Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	s2 := string(b)
	// 应包含 id/name/address.city（均属于两组）
	for _, want := range []string{"\"id\":1", "\"name\":\"A\"", "\"address\""} {
		if !strings.Contains(s2, want) {
			t.Fatalf("ModeAnd missing %s: %s", want, s2)
		}
	}
	// 不应包含仅属于 admin 或 internal 的字段
	if strings.Contains(s2, "email") || strings.Contains(s2, "password") {
		t.Fatalf("ModeAnd should not include admin-only/internal fields: %s", s2)
	}
}

func TestOmitEmptyAndOmitZero(t *testing.T) {
	u := User{ID: 1, Name: "A", Tags: nil, Scores: nil, Addr: Address{City: "SZ"}}
	// omitempty 应省略空 slice、空字符串
	b, err := NewEncoder().WithGroups("public").Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, "tags") {
		t.Fatalf("omitempty slice should be omitted: %s", s)
	}
	if strings.Contains(s, "line1") {
		t.Fatalf("omitempty string should be omitted: %s", s)
	}

	// omitzero 仅省略标量零值，应保留空集合
	u.Scores = []int{}
	b, err = NewEncoder().WithGroups("public").Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "scores") {
		t.Fatalf("omitzero should keep empty slice: %s", string(b))
	}
}

func TestAnonymousConflict(t *testing.T) {
	d := Deep{Shallow: Shallow{Name: "S"}, Name: "D"}
	b, err := NewEncoder().WithGroups("public").Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	// 浅层字段应覆盖深层（BFS 先入）
	if !strings.Contains(string(b), "\"name\":\"D\"") {
		t.Fatalf("shallow should win: %s", string(b))
	}
}

func TestMapAndSliceInput(t *testing.T) {
	u := User{ID: 2, Name: "B", Addr: Address{City: "SZ"}}
	m := map[string]any{"user": u, "note": "hi"}
	out, err := NewEncoder().AllowMap(true).WithGroups("public").MarshalToMap(m)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["note"]; !ok {
		t.Fatalf("note should present")
	}
	if um, ok := out["user"].(map[string]any); !ok || toInt(um["id"]) != 2 {
		t.Fatalf("user should be encoded with groups: %+v", out["user"])
	}

	arr := []User{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}
	amap, err := NewEncoder().AllowSlice(true).WithGroups("public").MarshalToMap(arr)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := amap["data"].([]any); !ok {
		t.Fatalf("slice should wrapped into data key")
	}
}

func TestDepthAndCircular(t *testing.T) {
	// 深度截断
	u := User{ID: 1, Name: "A", Addr: Address{City: "SZ"}}
	enc := NewEncoder().WithGroups("public").WithMaxDepth(1)
	b, err := enc.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	// depth=1 仅根层
	// depth=1：根层字段写入，下一层集合/对象被截断为 null
	if !strings.Contains(string(b), "\"address\":null") {
		t.Fatalf("address should be truncated to null: %s", string(b))
	}

	// 深度错误
	_, err = enc.WithDepthPolicy(DepthError).Marshal(u)
	if err == nil {
		t.Fatalf("expect depth error")
	}

	// 循环
	a := &Node{Val: 1}
	bnode := &Node{Val: 2}
	a.Next = bnode
	bnode.Next = a
	_, err = NewEncoder().WithGroups("public").Marshal(a)
	if err == nil {
		t.Fatalf("expect circular error")
	}
}

func TestUnsupportedAndMapKeyError(t *testing.T) {
	bad := Bad{}
	_, err := NewEncoder().WithGroups("public").Marshal(bad)
	if err == nil {
		t.Fatalf("expect unsupported error")
	}

	bad = Bad{C: make(chan string)}
	_, err = NewEncoder().WithGroups("public").Marshal(bad)
	if err == nil {
		t.Fatalf("expect unsupported error")
	}
}

func TestTopLevelKeyAndEncode(t *testing.T) {
	u := User{ID: 3, Name: "C"}
	enc := NewEncoder().WithGroups("public").WithTopLevelKey("data")
	b, err := enc.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "{\"data\":") {
		t.Fatalf("should wrap with top level key: %s", string(b))
	}

	var buf bytes.Buffer
	if err := enc.Encode(&buf, u); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buf.String(), "{\"data\":") {
		t.Fatalf("encode should wrap too: %s", buf.String())
	}
}

func TestEscapeHTML(t *testing.T) {
	type T struct {
		S string `json:"s" groups:"public"`
	}
	v := T{S: "<tag> & \"quote\""}

	// 默认不转义
	b, _ := NewEncoder().WithGroups("public").Marshal(v)
	if strings.Contains(string(b), "\\u003c") {
		t.Fatalf("should not escape by default: %s", string(b))
	}

	// 开启转义
	b, _ = NewEncoder().WithGroups("public").WithEscapeHTML(true).Marshal(v)
	if !strings.Contains(string(b), "\\u003c") {
		t.Fatalf("should escape when enabled: %s", string(b))
	}
}

func TestMarshalToMapAndCompare(t *testing.T) {
	u := User{ID: 4, Name: "D", Addr: Address{City: "SZ"}, Tags: []string{"x", "y"}}
	m, err := NewEncoder().WithGroups("public").MarshalToMap(u)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(m)
	b2, _ := NewEncoder().WithGroups("public").Marshal(u)
	var a1, a2 any
	_ = json.Unmarshal(b, &a1)
	_ = json.Unmarshal(b2, &a2)
	if !reflect.DeepEqual(a1, a2) {
		t.Fatalf("map and marshal should be equivalent: %s vs %s", string(b), string(b2))
	}
}

func TestRawMessageAndBytes(t *testing.T) {
	type T struct {
		R json.RawMessage `json:"r" groups:"public"`
		B []byte          `json:"b,omitempty" groups:"public"`
	}
	raw := json.RawMessage(`{"x":1}`)
	v := T{R: raw, B: []byte("hi")}
	b, err := NewEncoder().WithGroups("public").Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "\"r\":{\"x\":1}") {
		t.Fatalf("raw message should be embedded: %s", s)
	}
	if !strings.Contains(s, "\"b\":\"") {
		t.Fatalf("bytes should be base64 string: %s", s)
	}

	// 空切片在 omitempty 下应省略
	v.B = []byte{}
	b, _ = NewEncoder().WithGroups("public").Marshal(v)
	if strings.Contains(string(b), "\"b\"") {
		t.Fatalf("empty []byte should be omitted: %s", string(b))
	}

	// 验证 base64
	v.B = []byte{1, 2, 3}
	b, _ = NewEncoder().WithGroups("public").Marshal(v)
	enc := base64.StdEncoding.EncodeToString([]byte{1, 2, 3})
	if !strings.Contains(string(b), enc) {
		t.Fatalf("bytes should be base64: %s vs %s", string(b), enc)
	}
}

// Benchmarks

func makeUsers(n int) []User {
	out := make([]User, n)
	for i := 0; i < n; i++ {
		out[i] = User{
			ID:       i + 1,
			Name:     "User" + strconvI(i),
			Email:    "u@x",
			Password: "p",
			Tags:     []string{"a", "b", "c"},
			Scores:   []int{1, 2, 3, 4, 5},
			Addr:     Address{City: "SZ", Line1: "xx"},
			Meta:     Meta{CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}
	}
	return out
}

func strconvI(i int) string {
	return strings.TrimPrefix(strings.TrimSuffix(time.Unix(int64(i), 0).Format("20060102150405"), "000000"), "2006")
}

func toInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int32:
		return int(x)
	case int64:
		return int(x)
	case float64:
		return int(x)
	default:
		return 0
	}
}

func BenchmarkMarshalSmall(b *testing.B) {
	u := User{ID: 1, Name: "A", Email: "e", Password: "p", Tags: []string{"x"}, Scores: []int{1, 2, 3}, Addr: Address{City: "SZ"}, Meta: Meta{CreatedAt: time.Now()}}
	enc := NewEncoder().WithGroups("public", "admin")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = enc.Marshal(u)
	}
}

func BenchmarkMarshalLargeSlice(b *testing.B) {
	users := makeUsers(2000)
	enc := NewEncoder().WithGroups("public")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = enc.AllowSlice(true).MarshalToMap(users)
	}
}

func BenchmarkStdlibSmall(b *testing.B) {
	u := User{ID: 1, Name: "A", Email: "e", Password: "p", Tags: []string{"x"}, Scores: []int{1, 2, 3}, Addr: Address{City: "SZ"}, Meta: Meta{CreatedAt: time.Now()}}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(u)
	}
}

func BenchmarkStdlibLargeSlice(b *testing.B) {
	users := makeUsers(2000)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(users)
	}
}
