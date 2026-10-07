package entity

import (
	"encoding/json"
	"strings"

	"github.com/spf13/cast"
)

// KeyType Redis 原生数据类型，取值为 TYPE 命令的返回结果
type KeyType = string

const (
	KeyTypeNone   KeyType = "none"   // key 不存在
	KeyTypeString KeyType = "string" // 字符串（含位图、HyperLogLog 等编码视角）
	KeyTypeList   KeyType = "list"   // 列表
	KeyTypeSet    KeyType = "set"    // 集合
	KeyTypeZset   KeyType = "zset"   // 有序集合（含 GEO 编码视角）
	KeyTypeHash   KeyType = "hash"   // 哈希
	KeyTypeStream KeyType = "stream" // 流
)

// GetKeyType 归一化 TYPE 命令返回值，未知类型（module 类型等）原样返回小写值
func GetKeyType(val string) KeyType {
	switch KeyType(strings.ToLower(val)) {
	case KeyTypeString, KeyTypeList, KeyTypeSet, KeyTypeZset, KeyTypeHash, KeyTypeStream:
		return KeyType(strings.ToLower(val))
	}
	return KeyType(strings.ToLower(val))
}

// MemberOp 成员级写操作
type MemberOp = string

const (
	MemberOpCreate MemberOp = "create" // 新增成员
	MemberOpUpdate MemberOp = "update" // 修改成员
	MemberOpDelete MemberOp = "delete" // 删除成员（支持批量）
)

// Member 某个数据视角下的一行数据。
//
// 各视角只填写自身语义需要的字段，未用到的字段保持零值；无法用固定字段表达的派生信息
// （如 Geo 的经纬度、Stream 的多字段内容）放 Extra。新增数据类型不需要改动该结构：
// 扩展发生在「视角处理器」上，契约本身保持稳定
type Member struct {
	Index int64   `json:"index"` // list 下标、bitmap 位偏移
	Field string  `json:"field"` // hash field、geo member、stream entry 的字段集合名
	Value string  `json:"value"` // 主值：string 内容、set/list 元素、stream 内容摘要等
	Score float64 `json:"score"` // zset/geo 分值
	Id    string  `json:"id"`    // stream entry id

	// Extra 类型专属附加列，key 与视角描述符的 Columns 一一对应
	Extra map[string]string `json:"extra,omitempty"`
}

// UnmarshalJSON 兼容数字字段的「字符串形态」。
//
// 前端响应解析用 json-bigint（storeAsString）以保住 64 位整数精度，16 位以上的整数
// （如 geo 成员的 geohash 分值 4054134072858107）会被转成字符串；成员行原样回传做改/删时
// 就成了 `"score":"4054134072858107"`，按默认规则反序列化到 float64 会让整个请求报 500，
// 表现为「geo 成员永远删不掉」。这里统一收数字面量与字符串两种入参
func (m *Member) UnmarshalJSON(data []byte) error {
	raw := struct {
		Index any               `json:"index"`
		Field string            `json:"field"`
		Value string            `json:"value"`
		Score any               `json:"score"`
		Id    string            `json:"id"`
		Extra map[string]string `json:"extra"`
	}{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	m.Index = cast.ToInt64(raw.Index)
	m.Field = raw.Field
	m.Value = raw.Value
	m.Score = cast.ToFloat64(raw.Score)
	m.Id = raw.Id
	m.Extra = raw.Extra
	return nil
}

// MemberQuery 成员读取条件，各视角按自身能力选用其中的分页方式
type MemberQuery struct {
	Key     string `json:"key"`
	View    string `json:"view"`
	Cursor  string `json:"cursor"`  // 游标分页起始游标，空串表示从头开始
	Offset  int64  `json:"offset"`  // 排名分页起始下标
	Size    int64  `json:"size"`    // 单页成员数
	Keyword string `json:"keyword"` // 关键字过滤，是否生效由各视角决定（不支持过滤的视角忽略）
}

// MemberPage 成员分页结果
type MemberPage struct {
	Total   int64     `json:"total"`  // 成员总数（视角自定义语义，如 bitmap 为总位数）
	Cursor  string    `json:"cursor"` // 下一页游标，空串表示已到末尾
	Members []*Member `json:"members"`
}

// MemberWrite 成员写请求
type MemberWrite struct {
	Key     string            `json:"key"`
	View    string            `json:"view"`
	Op      MemberOp          `json:"op"`
	Member  *Member           `json:"member"`  // 新增/修改的单行数据
	Members []*Member         `json:"members"` // 删除的批量数据
	Args    map[string]string `json:"args"`    // 成员表单值，字段名由各视角的 Form 声明
	Ttl     int64             `json:"ttl"`     // 仅新建 key 时生效，随成员写入一起设置过期时间
}

// OpRequest 视角扩展操作请求，Args 由该操作的操作表单收集
type OpRequest struct {
	Key  string            `json:"key"`
	View string            `json:"view"`
	Op   string            `json:"op"`
	Args map[string]string `json:"args"`
}

// Capabilities 视角能力位。
// 前端按能力位显隐操作入口，而不是按类型名写 if 分支：新增视角只需声明能力位
type Capabilities struct {
	Create       bool `json:"create"`
	Update       bool `json:"update"`
	Delete       bool `json:"delete"`
	BatchDelete  bool `json:"batchDelete"`
	Keyword      bool `json:"keyword"`      // 是否支持关键字过滤
	RankPaging   bool `json:"rankPaging"`   // 是否按下标分页（list/zset/stream）
	CursorPaging bool `json:"cursorPaging"` // 是否按游标分页（hash/set/zset）
	Ops          bool `json:"ops"`          // 是否支持扩展操作
}

// Column 成员表格列描述，Field 对应 Member 的字段名或 Extra 的 key
type Column struct {
	Field    string `json:"field"`
	Label    string `json:"label"` // 前端 i18n key
	Width    int    `json:"width"` // 建议列宽，0 表示自适应
	Value    string `json:"value"` // 渲染语义：text 文本 | number 数字 | code 可折叠代码 | tag 标签
	Sortable bool   `json:"sortable"`
}

// OpSpec 视角扩展操作描述
type OpSpec struct {
	Name  string      `json:"name"`  // 操作标识，请求时回传
	Label string      `json:"label"` // 前端 i18n key
	Form  *FormSchema `json:"form"`  // 操作入参表单，为空表示无需入参直接执行
	Write bool        `json:"write"` // 是否为写操作（写操作受保存权限与审批流约束）
}

// ViewDescriptor 数据视角自描述信息。
//
// 前端渲染完全由该描述符驱动（列结构、能力位、表单结构、操作按钮），
// 因此新增数据类型时前端无需新增分支；label 类字段一律是前端 i18n key
type ViewDescriptor struct {
	View    KeyType   `json:"view"`    // 视角标识，注册键
	Label   string    `json:"label"`   // 展示名 i18n key
	Types   []KeyType `json:"types"`   // 可服务的 Redis 原生类型
	Default bool      `json:"default"` // 是否为所属原生类型的默认视角
	// Layout 成员区的布局：table 表格（多行成员）| value 单值面板（整键只有一个值）
	Layout  string       `json:"layout"`
	Caps    Capabilities `json:"caps"`
	Columns []Column     `json:"columns"`
	Form    *FormSchema  `json:"form"` // 成员新增表单结构
	// UpdateForm 成员修改表单结构；为空表示与新增同构（前端用行数据回填，prop 命中 Member 字段名或 Extra key）
	UpdateForm *FormSchema `json:"updateForm,omitempty"`
	Ops        []OpSpec    `json:"ops"` // 扩展操作
	// ConsoleHints 命令控制台的快捷命令模板，{key} 为占位符、执行时换成当前 key 名。
	// 由各视角自行声明（哪些命令对该类型有意义只有该处理器知道），前端只负责渲染
	ConsoleHints []string `json:"consoleHints"`
	// ReadCmd 该视角「读内容」等价的命令名（如 hash 为 HGETALL），必须出现在本视角的 ConsoleHints 里。
	//
	// 面板读取内容按这个名字过触发策略，而不是按底层实际发出的命令：分页时 hash 走的是 HSCAN，
	// 拿实现细节当治理口径会让管理员必须猜命令才能配对规则，也会出现「命令台拦得住、面板放过」。
	// 「申请查看」提单也用它拼命令，保证判定、提单、命令台三处同一个口径
	ReadCmd string `json:"readCmd"`
}

// CommandSpec 实例命令目录的一条记录，供命令控制台做输入提示与执行前确认。
//
// 目录取自实例自身的 COMMAND 回复而不是平台内置表：Redis 版本与加载的模块都会影响可用命令，
// 只有实例自描述才能保证「提示里出现的命令真能执行」
type CommandSpec struct {
	Name     string   `json:"name"`
	Arity    int      `json:"arity"`    // 参数个数，负数表示「至少 |arity| 个」
	Flags    []string `json:"flags"`    // Redis 命令标志（write/readonly/loading 等）
	FirstKey int      `json:"firstKey"` // 第一个键参数的位置，0 表示该命令无键参数
	LastKey  int      `json:"lastKey"`  // 最后一个键参数位置，-1 表示直到末尾
	Step     int      `json:"step"`     // 键参数的步长
	// NeedConfirm 执行前需要二次确认，与执行侧的高危命令判定同源，前端不再另立一份名单
	NeedConfirm bool `json:"needConfirm"`
}

// FormSchema AutoForm v1 表单结构（可序列端子集），由前端编译层渲染。
// 放在后端定义意味着「某个类型需要哪些入参」这一知识与该类型的处理器同处一地
type FormSchema struct {
	Version int         `json:"version"`
	Cols    int         `json:"cols,omitempty"`
	Fields  []FormField `json:"fields"`
}

// FormField 表单字段，Type 取值需在前端 CONTROL_REGISTRY 白名单内
type FormField struct {
	Prop         string       `json:"prop"`
	Label        string       `json:"label"`
	Type         string       `json:"type,omitempty"`
	Placeholder  string       `json:"placeholder,omitempty"`
	Description  string       `json:"description,omitempty"`
	DefaultValue any          `json:"defaultValue,omitempty"`
	Rows         int          `json:"rows,omitempty"`
	Span         int          `json:"span,omitempty"`
	Min          *float64     `json:"min,omitempty"`
	Max          *float64     `json:"max,omitempty"`
	Options      []FormOption `json:"options,omitempty"`
	Rules        *FormRules   `json:"rules,omitempty"`
}

// FormOption 选项类控件的选项
type FormOption struct {
	Value any    `json:"value"`
	Label string `json:"label"`
}

// FormRules 字段校验规则（结构化子集）
type FormRules struct {
	Required  bool   `json:"required,omitempty"`
	MinLength int    `json:"minLength,omitempty"`
	MaxLength int    `json:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
	Message   string `json:"message,omitempty"`
}

// KeyMeta key 元信息，与具体视角无关的部分 + 当前视角下的统计
type KeyMeta struct {
	Key      string        `json:"key"`
	Type     KeyType       `json:"type"`
	View     string        `json:"view"`
	Views    []*ViewOption `json:"views"`
	Encoding string        `json:"encoding"`
	TTL      int64         `json:"ttl"`    // 剩余过期秒数，-1 永久，-2 key 不存在
	MemUse   int64         `json:"memuse"` // MEMORY USAGE 字节数，不支持时为 0
	Size     int64         `json:"size"`   // 当前视角下的成员总数
	Caps     Capabilities  `json:"caps"`   // 当前视角能力位
	Exists   bool          `json:"exists"`
}

// KeySummary key 列表的批量摘要，用于树的类型角标与按类型筛选
type KeySummary struct {
	Key  string  `json:"key"`
	Type KeyType `json:"type"`
	TTL  int64   `json:"ttl"` // 剩余秒数，-1 永久
}

// ViewOption key 可选的数据视角
type ViewOption struct {
	View  string `json:"view"`
	Label string `json:"label"`
}
