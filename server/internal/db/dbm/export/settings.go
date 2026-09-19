package export

// 分批策略默认值（SQL 格式）：行数与字节数双预算，达到任一预算即生成一批
const (
	// DefaultBatchRows 单条批量INSERT的最大行数
	DefaultBatchRows = 100

	// DefaultBatchBytes 单批累计字节预算（8MB）：避免单行大值（如大blob/text）
	// 导致单条语句超长——内存峰值暴涨，且导入时超mysql max_allowed_packet等单包上限直接失败
	DefaultBatchBytes = 8 << 20
)

// Settings 导出配置选项。
type Settings struct {
	// Format 导出格式（对应 Consumer.Format()）
	Format string

	// Encoding 字符编码（如 "utf-8"、"gbk"），默认 "utf-8"
	Encoding string

	// BatchRows 单批 INSERT 的最大行数（SQL 格式使用），默认 DefaultBatchRows
	BatchRows int

	// BatchBytes 单批累计字节预算（SQL 格式使用），默认 DefaultBatchBytes
	BatchBytes int64

	// IncludeHeader 是否包含列名行（CSV/JSON 格式使用），默认 true
	IncludeHeader bool

	// PrettyPrint 是否美化输出（JSON 格式缩进等），默认 false
	PrettyPrint bool

	// QuoteAllFields 是否对所有字段加引号（CSV 格式使用），默认 false
	QuoteAllFields bool

	// FieldSeparator 字段分隔符（CSV 格式使用），默认 ","
	FieldSeparator string

	// LineTerminator 行终止符，默认 "\n"
	LineTerminator string

	// Extra 额外格式特定选项（各格式消费者自定义）
	Extra map[string]any

	// TableCount 本次导出的表总数，由 Exporter 在开始导出前注入（调用方无需设置）。
	// 需要按表数决定产物结构的格式据此适配：JSON 单表输出纯数组、多表输出以表名为键的对象
	TableCount int
}

// DefaultSettings 默认导出配置。
func DefaultSettings(format string) *Settings {
	return &Settings{
		Format:         format,
		Encoding:       "utf-8",
		BatchRows:      DefaultBatchRows,
		BatchBytes:     DefaultBatchBytes,
		IncludeHeader:  true,
		PrettyPrint:    false,
		FieldSeparator: ",",
		LineTerminator: "\n",
	}
}
