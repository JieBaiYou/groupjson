package groupjson

// 定义组过滤的逻辑模式
type GroupMode int

const (
	// 表示字段只要属于任一指定分组即包含在结果中（默认）
	ModeOr GroupMode = iota
	// 表示字段必须同时属于所有指定组才包含在结果中
	ModeAnd
)

const (
	// 默认最大递归深度
	DefaultMaxDepth = 32
	// 默认分组标签键名
	DefaultTagKey = "groups"
)

// 配置GroupJSON的序列化行为
type Options struct {
	// 要包含的组名称列表
	Groups []string
	// 定义组筛选逻辑（OR或AND）
	GroupMode GroupMode
	// 自定义分组标签的键名（默认为"groups"）
	TagKey string
	// 如果非空, 输出会被包装在此键下
	TopLevelKey string
	// 最大递归深度, 用于防止循环引用
	MaxDepth int
}

// 返回默认选项
func DefaultOptions() Options {
	return Options{
		GroupMode: ModeOr,
		TagKey:    DefaultTagKey,
		MaxDepth:  DefaultMaxDepth,
	}
}

// 分组JSON序列化的主要入口点
type GroupJSON struct {
	opts Options
}

// 创建一个新的GroupJSON实例, 使用默认选项
// 示例: groupjson.New().WithGroups("admin", "internal").WithTopLevelKey("data").Marshal(user)
func New() *GroupJSON {
	return &GroupJSON{
		opts: DefaultOptions(),
	}
}

// 使用默认选项序列化带有组的值
func Marshal(v any, groups ...string) ([]byte, error) {
	return New().WithGroups(groups...).Marshal(v)
}

// 使用自定义选项序列化带有组的值
func MarshalWithOptions(opts Options, v any, groups ...string) ([]byte, error) {
	return New().WithGroups(groups...).Marshal(v)
}

// 设置要包含的组名
func (g *GroupJSON) WithGroups(groups ...string) *GroupJSON {
	g.opts.Groups = groups
	return g
}

// 设置组筛选逻辑
func (g *GroupJSON) WithGroupMode(mode GroupMode) *GroupJSON {
	g.opts.GroupMode = mode
	return g
}

// 设置自定义标签键
func (g *GroupJSON) WithTagKey(tagKey string) *GroupJSON {
	g.opts.TagKey = tagKey
	return g
}

// 设置顶层包装键
func (g *GroupJSON) WithTopLevelKey(key string) *GroupJSON {
	g.opts.TopLevelKey = key
	return g
}

// 设置最大递归深度
func (g *GroupJSON) WithMaxDepth(depth int) *GroupJSON {
	g.opts.MaxDepth = depth
	return g
}

// 序列化上下文，包含单次序列化过程的状态
type encodeContext struct {
	// 用于跟踪处理过的指针地址, 防止循环引用
	visited map[uintptr]bool
}

// 创建新的序列化上下文
func newEncodeContext() *encodeContext {
	return &encodeContext{
		visited: make(map[uintptr]bool),
	}
}
