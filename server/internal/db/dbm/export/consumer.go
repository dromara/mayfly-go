// Package export 数据导出能力模块（与契约层 dbi 平级的独立功能模块）。
//
// 与契约层 dbi 平级、单向依赖 dbi，模块内三层职责分离：
//   - Consumer（格式层）：每种导出格式实现此接口，负责输出序列化
//   - Exporter（编排层）：游标遍历、分批策略、列映射等共享逻辑（exporter.go）
//   - DumpHelper + SQLGenerator（方言层，在 dbi 包）：方言差异由各方言自行实现
//
// 开闭原则：
//   - 新增导出格式：实现 Consumer 接口并在 init 中 Register/RegisterFactory，
//     Descriptors() 自动暴露给上层（前端格式清单无需改动），本模块其余代码零修改
//   - 新增方言：实现 DumpHelper + SQLGenerator 后导出功能自动可用
package export

import (
	"io"

	"mayfly-go/internal/db/dbm/dbi"
)

// Consumer 数据导出消费者接口（格式层）。
// 每种导出格式（SQL/CSV/JSON 等）实现此接口，Exporter 负责编排调用。
//
// 实现可以是有状态的（按表/按次导出跟踪进度），注册中心以工厂方式每次返回新实例；
// 单表导出内 Begin/ConsumeBatch*/End 按序串行调用，无并发。
type Consumer interface {
	// Format 返回导出格式标识（如 "sql"、"csv"、"json"），注册表的键，全局唯一
	Format() string

	// SupportsScript 声明该格式的产物是否为**方言脚本形态**——true 时 Exporter 编排器启用：
	//   - SQL 头部注释段（平台/时间/方言标识）
	//   - DDL 与索引导出段（GenTableDDL/GenIndexDDL）
	//   - 方言事务前后置钩子（DumpHelper.BeforeInsert/AfterInsert/BeforeInsertSql）
	//
	// 编排器只探测本能力，禁止按 Format 名称字符串分支——新增脚本类格式（如某方言专用脚本）
	// 只需在自身实现中声明 true，编排器零修改（开闭原则）
	SupportsScript() bool

	// Name 返回人类可读的格式名称（如 "SQL 脚本"），供 Descriptors 与 UI 展示，静态字符串
	Name() string

	// ContentType 返回 MIME 类型（如 "text/csv"、"application/json"）
	ContentType() string

	// FileExtension 返回文件扩展名（如 ".csv"、".json"）
	FileExtension() string

	// Begin 表数据导出开始前调用，写入格式头部（如 CSV 表头、JSON 数组起始符、SQL 注释等）
	Begin(w io.Writer, tableName string, columns []dbi.Column, settings *Settings) error

	// ConsumeBatch 消费一批行数据，按方言/格式生成输出写入 w
	//   - helper: 方言导出辅助（SQL 格式用于事务控制，其他格式忽略）
	//   - sqlGen: 方言 SQL 生成器（SQL 格式用于生成 INSERT，其他格式忽略）
	ConsumeBatch(w io.Writer, tableName string, columns []dbi.Column, rows [][]any,
		helper dbi.DumpHelper, sqlGen dbi.SQLGenerator, settings *Settings) error

	// End 表数据导出结束后调用，写入格式尾部（如 JSON 数组闭合、CSV 缓冲落盘等）
	End(w io.Writer, tableName string, settings *Settings) error

	// Finish 全局收尾钩子：所有表导出完成后由 Exporter 调用一次（整份产物级收尾，区别于
	// End 的单表级收尾）。多表产物的外层结构闭合依赖此钩子（如 JSON 多表对象闭合右花括号）。
	// 无全局收尾需求的格式（SQL/CSV）直接返回 nil
	Finish(w io.Writer, settings *Settings) error
}
