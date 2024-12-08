package sqlite

import (
	"io"
	"mayfly-go/internal/db/dbm/dbi"
)

type DumpHelper struct {
	dbi.DefaultDumpHelper
}

func (db *DumpHelper) BeforeInsert(writer io.Writer, tableName string) {
}
func (db *DumpHelper) AfterInsert(writer io.Writer, tableName string, columns []dbi.Column) {
}
