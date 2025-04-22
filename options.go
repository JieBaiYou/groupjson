package groupjson

// GroupMode 定义分组筛选的逻辑模式。
type GroupMode int

const (
	// ModeOr 字段属于任一指定分组即包含（默认）。
	ModeOr GroupMode = iota
	// ModeAnd 字段必须同时属于所有指定分组才包含。
	ModeAnd
)

const (
	// DefaultMaxDepth 默认最大递归深度。
	DefaultMaxDepth = 32
	// DefaultTagKey 默认分组标签键名。
	DefaultTagKey = "groups"
)

// Options 配置序列化行为的选项。
type Options struct {
	// Groups 要包含的分组名称列表。
	Groups []string
	// GroupMode 分组筛选逻辑（OR或AND）。
	GroupMode GroupMode
	// TagKey 自定义分组标签的键名。
	TagKey string
	// TopLevelKey 输出包装键，非空时JSON结果被包装在此键下。
	TopLevelKey string
	// MaxDepth 最大递归深度，防止循环引用导致栈溢出。
	MaxDepth int
}

// DefaultOptions 返回默认选项配置。
func DefaultOptions() Options {
	return Options{
		GroupMode: ModeOr,
		TagKey:    DefaultTagKey,
		MaxDepth:  DefaultMaxDepth,
	}
}

// GroupJSON 分组JSON序列化器。
type GroupJSON struct {
	opts Options
}

// New 创建序列化器实例，使用默认选项。
// 示例: groupjson.New().WithGroups("admin", "internal").Marshal(user)
func New() *GroupJSON {
	return &GroupJSON{
		opts: DefaultOptions(),
	}
}

// Marshal 使用指定分组序列化值。
func Marshal(v any, groups ...string) ([]byte, error) {
	return New().WithGroups(groups...).Marshal(v)
}

// MarshalWithOptions 使用自定义选项序列化值。
func MarshalWithOptions(opts Options, v any, groups ...string) ([]byte, error) {
	return New().WithGroups(groups...).Marshal(v)
}

// WithGroups 设置要包含的分组名称。
func (g *GroupJSON) WithGroups(groups ...string) *GroupJSON {
	g.opts.Groups = groups
	return g
}

// WithGroupMode 设置分组筛选逻辑模式。
func (g *GroupJSON) WithGroupMode(mode GroupMode) *GroupJSON {
	g.opts.GroupMode = mode
	return g
}

// WithTagKey 设置自定义标签键名。
func (g *GroupJSON) WithTagKey(tagKey string) *GroupJSON {
	g.opts.TagKey = tagKey
	return g
}

// WithTopLevelKey 设置结果的顶层包装键。
func (g *GroupJSON) WithTopLevelKey(key string) *GroupJSON {
	g.opts.TopLevelKey = key
	return g
}

// WithMaxDepth 设置递归深度上限。
func (g *GroupJSON) WithMaxDepth(depth int) *GroupJSON {
	g.opts.MaxDepth = depth
	return g
}

// encodeContext 序列化上下文，跟踪单次序列化过程状态。
type encodeContext struct {
	// visited 已处理的指针地址集合，用于检测循环引用。
	visited map[uintptr]bool
}

// newEncodeContext 创建新的序列化上下文。
func newEncodeContext() *encodeContext {
	return &encodeContext{
		visited: make(map[uintptr]bool),
	}
}
