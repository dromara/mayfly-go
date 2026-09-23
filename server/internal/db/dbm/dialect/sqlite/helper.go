package sqlite

import (
	"io"
	"mayfly-go/internal/db/dbm/dbi"
)

var _ dbi.DumpTxnWrapper = (*DumpTxnWrapper)(nil)

type DumpTxnWrapper struct {
	dbi.DefaultDumpTxnWrapper
}

func (db *DumpTxnWrapper) BeforeInsert(writer io.Writer, tableName string) error {
	return nil
}

func (db *DumpTxnWrapper) AfterInsert(writer io.Writer, tableName string, columns []dbi.Column) error {
	return nil
}
