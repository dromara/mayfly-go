package sqlite

import (
	"io"
	"mayfly-go/internal/db/dbm/dbi"
)

var _ dbi.DumpHelper = (*DumpHelper)(nil)

type DumpHelper struct {
	dbi.DefaultDumpHelper
}

func (db *DumpHelper) BeforeInsert(writer io.Writer, tableName string) error {
	return nil
}

func (db *DumpHelper) AfterInsert(writer io.Writer, tableName string, columns []dbi.Column) error {
	return nil
}
