package dto

import (
	"io"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/export"
	"mayfly-go/internal/db/domain/entity"
	tagentity "mayfly-go/internal/tag/domain/entity"
)

type SaveDbInstance struct {
	DbInstance   *entity.DbInstance
	AuthCerts    []*tagentity.ResourceAuthCert
	TagCodePaths []string
}

type DumpDb struct {
	DbId     uint64
	DbName   string
	Tables   []string
	DumpDDL  bool // 是否dump ddl
	DumpData bool // 是否dump data

	// TableFilter 表级数据过滤条件（表名→where条件），仅DumpData时生效；
	// nil/缺省=不过滤（零值兼容现有调用方）。用于大表主键分片并行迁移
	TableFilter map[string]string

	// SkipDropTable 建表前是否**不**删除同名表（true=不生成 DROP，false=生成 DROP）。
	// 仅 DumpDDL 时生效；默认 false 保持既有「建表前 DROP」行为。对应迁移任务 deleteTable 配置（1是/2否）
	SkipDropTable bool

	// NameCase 目标对象名大小写转换：1(或0)=不转换 2=转大写 3=转小写。
	// 仅影响生成的 DDL/DML 目标表名/列名，不影响源库查询与列映射
	NameCase int

	LogId uint64

	// Mask 是否对导出行数据执行查询结果脱敏（与交互式查询同一套规则与 fail-close 口径）。
	//
	// 由调用方显式声明而不开默认 true：脱敏串写进备份文件后恢复即把脱敏值当真实数据写回库，
	// 是不可逆污染，因此「迁移到库」与「备份到文件」必须保持 false；
	// 而给人看的一次性导出（api.DumpSQL）必须置 true，否则脱敏列会经导出链路明文透出
	Mask bool

	// RowMask 已装配好的行级脱敏钩子，由 application 层按 Mask 意图生成后传给导出器，
	// 调用方不要自己设置（直接给 transfer.DumpDbScript 传 Mask=true 而无 RowMask 会被拒绝）
	RowMask export.RowMaskFn

	Writer       io.Writer
	TargetDbType dbi.DbType

	// ExportFormat 导出格式（"sql"/"csv"/"json" 等），空值默认 "sql"
	ExportFormat string
	// Settings 导出配置选项，nil 时使用 DefaultSettings
	Settings *export.Settings

	Log      func(msg string)
	Progress func(currentTable string, stmtType dbi.DumpKind, stmtCount int, currentStmtTypeEnd bool) // dump进度
}
