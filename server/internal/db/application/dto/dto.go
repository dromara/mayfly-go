package dto

import (
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/utils/writer"
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

	LogId uint64

	Writer       writer.CustomWriter
	Log          func(msg string)
	TargetDbType dbi.DbType
}

func DefaultDumpLog(msg string) {

}
