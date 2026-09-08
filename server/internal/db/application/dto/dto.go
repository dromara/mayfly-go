package dto

import (
	"io"
	"mayfly-go/internal/db/dbm/dbi"
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

	LogId uint64

	Writer       io.Writer
	TargetDbType dbi.DbType

	Log      func(msg string)
	Progress func(currentTable string, stmtType dbi.StmtType, stmtCount int, currentStmtTypeEnd bool) // dump进度
}

func DefaultDumpLog(msg string) {

}

func DefaultDumpProgress(currentTable string, stmtType dbi.StmtType, stmtCount int, currentStmtTypeEnd bool) {

}
