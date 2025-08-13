package groupjson

// GroupMode 定义分组筛选逻辑。
type GroupMode int

const (
	// ModeOr 字段属于任一指定分组即包含（默认）。
	ModeOr GroupMode = iota
	// ModeAnd 字段必须同时属于所有指定分组才包含。
	ModeAnd
)

// DepthPolicy 控制超过最大深度时的行为。
type DepthPolicy int

const (
	// DepthTruncate 超深度时截断（返回 nil 或空集合）。
	DepthTruncate DepthPolicy = iota
	// DepthError 超深度时报错。
	DepthError
)

// CutoffCollection 控制截断时集合的表示。
type CutoffCollection int

const (
	// Null 集合被表示为 null。
	Null CutoffCollection = iota
	// Empty 集合被表示为空集合（[] 或 {}）。
	Empty
)

const (
	DefaultTagKey   = "groups"
	DefaultMaxDepth = 32
)

// Options 控制序列化行为。
type Options struct {
	// Groups 需要包含的分组名称列表；为空表示不输出任何分组受控字段。
	Groups []string
	// Mode 分组匹配模式：ModeOr（任一命中）或 ModeAnd（全部命中）。
	Mode GroupMode
	// TagKey 字段上用于声明分组的结构体标签键名，默认 "groups"。
	TagKey string
	// TopLevelKey 非空时，最终结果以该键包裹为顶层对象。
	TopLevelKey string
	// MaxDepth 最大递归深度（含根层，最小为 1），防止深嵌套或环导致资源耗尽。
	MaxDepth int
	// DepthPolicy 超出最大深度时的处理策略：截断或报错。
	DepthPolicy DepthPolicy
	// CutoffCollection 在截断模式下，集合类型超深度时使用 null 或空集合表示。
	CutoffCollection CutoffCollection
	// EscapeHTML 是否对 HTML 字符进行转义，保持与 encoding/json 行为一致可关闭。
	EscapeHTML bool
	// SortKeys 是否对 map 键进行排序（仅为测试/可读性，默认关闭）。
	SortKeys bool
	// AllowMapInput 顶层是否允许传入 map[string]any；值为结构体时按组编码，其它值透传。
	AllowMapInput bool
	// AllowSliceInput 顶层是否允许传入切片/数组；元素为结构体时按组编码，其它值透传。
	AllowSliceInput bool
}

// DefaultOptions 返回默认选项。
func DefaultOptions() Options {
	return Options{
		Mode:             ModeOr,
		TagKey:           DefaultTagKey,
		MaxDepth:         DefaultMaxDepth,
		DepthPolicy:      DepthTruncate,
		CutoffCollection: Null,
	}
}
