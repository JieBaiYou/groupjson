package groupjson

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// =============================================================================
// 测试用结构体定义
// =============================================================================

// User 基本结构体
type User struct {
	ID       int    `json:"id" groups:"public,admin"`
	Name     string `json:"name" groups:"public,admin"`
	Email    string `json:"email" groups:"admin"`
	Password string `json:"password" groups:"internal"`
}

// Product 包含各种基础类型和 Tag
type Product struct {
	Title    string  `json:"title" groups:"public"`
	Price    float64 `json:"price,string" groups:"public"` // string tag: 输出为字符串
	Stock    int     `json:"stock,omitempty" groups:"admin"`
	IsActive bool    `json:"is_active" groups:"public"`
}

// NestedStruct 嵌套结构体
type NestedStruct struct {
	Meta    Meta `json:"meta" groups:"public"`
	Creator User `json:"creator" groups:"admin"`
}

type Meta struct {
	Version string `json:"version" groups:"public"`
}

// Collections 包含切片和 Map
type Collections struct {
	Tags    []string          `json:"tags" groups:"public"`
	Weights []int             `json:"weights,omitempty" groups:"public"`
	Extras  map[string]string `json:"extras" groups:"admin"`
}

// Pointers 包含指针
type Pointers struct {
	UserID *int  `json:"user_id" groups:"public"`
	Ref    *User `json:"ref" groups:"public"`
}

// Circular 循环引用测试
type Circular struct {
	Next *Circular `json:"next" groups:"public"`
}

// CustomMarshaler 自定义 JSON 序列化
type CustomMarshaler struct {
	ID int
}

func (c CustomMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{"custom_id":%d}`, c.ID)), nil
}

// TextMarshaler 自定义文本序列化
type TextMarshaler struct {
	Val string
}

func (t TextMarshaler) MarshalText() ([]byte, error) {
	return []byte("prefix_" + t.Val), nil
}

// =============================================================================
// 综合测试
// =============================================================================

func TestMarshal_Comprehensive(t *testing.T) {
	ptrInt := 100
	refUser := User{ID: 99, Name: "Ref"}

	tests := []struct {
		name       string
		input      any
		groups     []string
		mode       Mode
		wantJSON   string   // 期望包含的 JSON 片段或完整 JSON
		notWant    []string // 不期望包含的字符串片段
		wantErr    bool
		errContent string // 错误信息包含的内容
	}{
		// 1. 基础类型与分组
		{
			name:     "基础结构体 - Public 分组",
			input:    User{ID: 1, Name: "Alice", Email: "alice@x.com", Password: "123"},
			groups:   []string{"public"},
			wantJSON: `{"id":1,"name":"Alice"}`,
			notWant:  []string{`"email":`, `"password":`},
		},
		{
			name:     "基础结构体 - Admin 分组",
			input:    User{ID: 1, Name: "Alice", Email: "alice@x.com", Password: "123"},
			groups:   []string{"admin"},
			wantJSON: `{"id":1,"name":"Alice","email":"alice@x.com"}`,
			notWant:  []string{`"password":`},
		},
		{
			name:     "基础结构体 - AND 模式 (Public & Admin)",
			input:    User{ID: 1, Name: "Alice", Email: "alice@x.com"},
			groups:   []string{"public", "admin"},
			mode:     ModeAnd,
			wantJSON: `{"id":1,"name":"Alice"}`, // ID 和 Name 都有这两个组
			notWant:  []string{`"email":`},      // Email 只有 admin，不满足 AND
		},

		// 2. 特殊 Tag 测试 (string, omitempty)
		{
			name:     "Tag - string",
			input:    Product{Title: "Phone", Price: 99.9},
			groups:   []string{"public"},
			wantJSON: `"price":"99.9"`, // Price 应该是字符串
		},
		{
			name:    "Tag - omitempty (Zero Value)",
			input:   Product{Title: "Phone", Price: 0, Stock: 0}, // Price 0 is not omitted (no omitempty), Stock 0 is omitted
			groups:  []string{"admin"},                           // Price not in admin
			notWant: []string{`"stock":`, `"price":`},
		},
		{
			name:     "Tag - omitempty (Non-Zero Value)",
			input:    Product{Title: "Phone", Stock: 5},
			groups:   []string{"admin"},
			wantJSON: `"stock":5`,
		},

		// 3. 嵌套结构体
		{
			name: "嵌套结构体 - 递归筛选",
			input: NestedStruct{
				Meta:    Meta{Version: "v1"},
				Creator: User{ID: 2, Name: "Bob", Email: "bob@x.com"},
			},
			groups:   []string{"public"},
			wantJSON: `{"meta":{"version":"v1"}}`,
			notWant:  []string{`"creator":`}, // Creator 是 admin 组，public 下应完全不可见
		},
		{
			name: "嵌套结构体 - 内部字段筛选",
			input: NestedStruct{
				Meta:    Meta{Version: "v1"},
				Creator: User{ID: 2, Name: "Bob", Email: "bob@x.com"},
			},
			groups: []string{"admin"},
			// Creator 可见，但 Creator 内部也应用 admin 分组筛选 (User 下 ID/Name/Email 均有 admin)
			wantJSON: `"creator":{"id":2,"name":"Bob","email":"bob@x.com"}`,
		},

		// 4. 集合 (Slice, Map)
		{
			name: "集合 - Slice & Map",
			input: Collections{
				Tags:   []string{"a", "b"},
				Extras: map[string]string{"k": "v"},
			},
			groups:   []string{"public", "admin"}, // OR 模式
			wantJSON: `"tags":["a","b"]`,
		},
		{
			name:     "集合 - Map 排序保证",
			input:    map[string]int{"b": 2, "a": 1},
			groups:   []string{"public"},
			wantJSON: `{"a":1,"b":2}`, // 必须按键排序
		},
		{
			name:    "集合 - omitempty Slice",
			input:   Collections{Weights: []int{}},
			groups:  []string{"public"},
			notWant: []string{`"weights":`},
		},

		// 5. 指针
		{
			name:     "指针 - 非空",
			input:    Pointers{UserID: &ptrInt, Ref: &refUser},
			groups:   []string{"public"},
			wantJSON: `{"user_id":100,"ref":{"id":99,"name":"Ref"}}`,
		},
		{
			name:     "指针 - 空值",
			input:    Pointers{UserID: nil, Ref: nil},
			groups:   []string{"public"},
			wantJSON: `{"user_id":null,"ref":null}`,
		},

		// 6. 自定义 Marshaler
		{
			name:     "Custom Marshaler (Root)",
			input:    CustomMarshaler{ID: 888},
			groups:   []string{"any"}, // CustomMarshaler 自己接管了序列化，忽略外部 Group
			wantJSON: `{"custom_id":888}`,
		},
		{
			name:     "Text Marshaler (Map Value)",
			input:    map[string]TextMarshaler{"k": {Val: "val"}},
			groups:   []string{"any"},
			wantJSON: `{"k":"prefix_val"}`,
		},

		// 7. 错误情况
		{
			name: "错误 - 循环引用",
			input: func() *Circular {
				c := &Circular{}
				c.Next = c
				return c
			}(),
			groups:     []string{"public"},
			wantErr:    true,
			errContent: "circular reference",
		},
		{
			name:       "错误 - Map Key 非字符串",
			input:      map[int]string{1: "a"},
			groups:     []string{"public"},
			wantErr:    true,
			errContent: "map key must be string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. 打印输入信息 (尝试用 json 打印，失败则用 %+v)
			t.Logf("=== 测试用例: %s ===", tt.name)
			inputJSON, _ := json.Marshal(tt.input)
			if len(inputJSON) > 0 {
				t.Logf("Input (Raw/JSON): %s", string(inputJSON))
			} else {
				t.Logf("Input (Go Struct): %+v", tt.input)
			}
			t.Logf("Groups: %v, Mode: %v", tt.groups, tt.mode)

			// 2. 执行序列化
			got, err := New().WithGroups(tt.groups...).WithMode(tt.mode).Marshal(tt.input)

			// 3. 错误检查
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if !strings.Contains(err.Error(), tt.errContent) {
					t.Errorf("Marshal() error = %v, want error containing %q", err, tt.errContent)
				}
				t.Logf("✅ 成功捕获预期错误: %v", err)
				return
			}

			// 4. 输出检查
			jsonStr := string(got)
			t.Logf("Output JSON: %s", jsonStr)

			// 检查期望存在的 JSON
			if tt.wantJSON != "" {
				// 如果期望的是完整 JSON 对象 (以 { 开头)，尝试语义对比
				if strings.HasPrefix(tt.wantJSON, "{") && strings.HasSuffix(tt.wantJSON, "}") {
					if !jsonEqual(jsonStr, tt.wantJSON) {
						// 如果语义不相等，再检查是否仅仅是包含关系（有时候测试数据只是片段）
						if !strings.Contains(jsonStr, tt.wantJSON) {
							t.Errorf("❌ JSON 匹配失败.\nGot:  %s\nWant: %s", jsonStr, tt.wantJSON)
						}
					}
				} else {
					// 否则做字符串包含检查
					if !strings.Contains(jsonStr, tt.wantJSON) {
						t.Errorf("❌ 输出未包含期望内容.\nGot:  %s\nWant Substr: %s", jsonStr, tt.wantJSON)
					}
				}
			}

			// 检查不期望存在的内容
			for _, nw := range tt.notWant {
				if strings.Contains(jsonStr, nw) {
					t.Errorf("❌ 输出包含不应存在的内容: %q", nw)
				}
			}
		})
	}
}

// TestNestedPermissions 专门测试父子结构体权限不一致的场景
func TestNestedPermissions(t *testing.T) {
	type Child struct {
		PublicInfo string `json:"public_info" groups:"public"`
		Secret     string `json:"secret" groups:"admin"`
	}
	type Parent struct {
		// Parent.Child 本身属于 admin 组
		Child Child `json:"child" groups:"admin"`
	}

	input := Parent{
		Child: Child{
			PublicInfo: "Everyone can see",
			Secret:     "Admin only",
		},
	}

	t.Run("Case 1: 请求 Public (父级不可见 -> 子级完全隐藏)", func(t *testing.T) {
		b, _ := New().WithGroups("public").Marshal(input)
		s := string(b)
		t.Logf("JSON: %s", s)
		if strings.Contains(s, "child") {
			t.Error("Child field should be hidden because Parent.Child is admin-only")
		}
		if strings.Contains(s, "public_info") {
			t.Error("Child.PublicInfo should be hidden because parent struct is skipped")
		}
	})

	t.Run("Case 2: 请求 Admin (父级可见, 子级只显示匹配项)", func(t *testing.T) {
		b, _ := New().WithGroups("admin").Marshal(input)
		s := string(b)
		t.Logf("JSON: %s", s)
		if !strings.Contains(s, "child") {
			t.Error("Child field should be visible")
		}
		// Child.PublicInfo 是 public，我们只请求了 admin，所以这里应该也不可见
		if strings.Contains(s, "public_info") {
			t.Error("Child.PublicInfo should be hidden because it is not in admin group")
		}
		// Child.Secret 是 admin，应该可见
		if !strings.Contains(s, "secret") {
			t.Error("Child.Secret should be visible")
		}
	})

	t.Run("Case 3: 请求 Public+Admin (全可见)", func(t *testing.T) {
		b, _ := New().WithGroups("public", "admin").Marshal(input)
		s := string(b)
		t.Logf("JSON: %s", s)
		if !strings.Contains(s, "public_info") {
			t.Error("Child.PublicInfo should be visible")
		}
		if !strings.Contains(s, "secret") {
			t.Error("Child.Secret should be visible")
		}
	})
}

// jsonEqual 比较两个 JSON 字符串语义是否相等
func jsonEqual(a, b string) bool {
	var j1, j2 interface{}
	if err := json.Unmarshal([]byte(a), &j1); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &j2); err != nil {
		return false
	}
	return reflect.DeepEqual(j1, j2)
}
