package groupjson

// GroupMode 定义分组筛选逻辑。
type GroupMode int

const (
	// ModeOr 字段属于任一指定分组即包含（默认）。
	ModeOr GroupMode = iota
	// ModeAnd 字段必须同时属于所有指定分组才包含。
	ModeAnd
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
	// EscapeHTML 是否对 HTML 字符进行转义，保持与 encoding/json 行为一致可关闭。
	EscapeHTML bool
	// SortKeys 是否对 map 键进行排序（仅为测试/可读性，默认关闭）。
	SortKeys bool
}

// DefaultOptions 返回默认选项。
func DefaultOptions() Options {
	return Options{
		Mode:     ModeOr,
		TagKey:   DefaultTagKey,
		MaxDepth: DefaultMaxDepth,
	}
}
