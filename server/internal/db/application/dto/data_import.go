package dto

import (
	"io"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/importer"
)

// ImportPreviewReq 文件预览入参：解析文件表头与前若干数据行，供前端构建列映射界面。
type ImportPreviewReq struct {
	Reader   io.Reader
	Filename string
	Options  *importer.Options
}

// ImportPreviewRes 文件预览结果。
type ImportPreviewRes struct {
	Headers    []string   `json:"headers"`    // 文件表头列名（无表头时为空）
	SampleRows [][]string `json:"sampleRows"` // 预览数据行（受 Options.PreviewLimit 限制）
	TotalRows  int        `json:"totalRows"`  // 已解析的数据行总数
	Sheets     []string   `json:"sheets"`     // Excel 的全部工作表名（CSV 等为 nil），供前端下拉选择
}

// ImportColumn 目标列映射：一个文件列 → 一个数据库列。Target 为空表示该文件列不导入。
type ImportColumn struct {
	// Source 文件列的位置下标（字符串，如 "0"）。
	// 预览与导入对同一文件用相同选项解析，列序稳定，故按位置而非列名映射，避免同名表头被解析到同一列导致错位。
	Source string `json:"source"`
	Target string `json:"target"` // 目标数据库列名；空=跳过该文件列
}

// DataImportReq 数据导入执行入参：把已解析文件的行数据按列映射写入目标表。
type DataImportReq struct {
	DbConn    *dbi.DbConn
	TableName string
	Reader    io.Reader
	Filename  string
	Options   *importer.Options

	// Columns 有序列映射，决定导入哪些列及其顺序；未列出的文件列不导入
	Columns []ImportColumn

	// EmptyAsNull 单元格为空字符串时写入 NULL（true）还是写入空串（false）
	EmptyAsNull bool

	// DuplicateStrategy 主键/唯一键冲突处理策略，复用 dbi.DuplicateStrategy*；None=直接插入
	DuplicateStrategy int

	// BatchSize 单条批量 INSERT 的最大行数，<=0 时用 DefaultImportBatchSize
	BatchSize int

	ClientId string // 客户端 id，用于回传导入进度
	UploadId string // 上传 id，前端据此匹配进度通知（与 ClientId 同时存在时才会回传进度）
}

// DefaultImportBatchSize 数据导入默认批量行数：与导出默认批量同量级，兼顾语句数与单包体积
const DefaultImportBatchSize = 500

// DataImportRes 数据导入执行结果。
type DataImportRes struct {
	TotalRows   int   `json:"totalRows"`   // 文件解析出的待导入数据行总数
	Imported    int   `json:"imported"`    // 成功写入的行数
	BatchCount  int   `json:"batchCount"`  // 生成的批量插入语句条数
	AffectedRow int64 `json:"affectedRow"` // 数据库返回的累计影响行数
}
