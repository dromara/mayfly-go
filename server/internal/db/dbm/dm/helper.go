package dm

import (
	"fmt"
	"io"
	"mayfly-go/internal/db/dbm/dbi"
)

type DumpHelper struct {
	dbi.DefaultDumpHelper
}

func (dh *DumpHelper) BeforeInsert(writer io.Writer, tableName string) error {
	return nil
}

func (dh *DumpHelper) BeforeInsertSql(quoteSchema string, tableName string) string {
	return fmt.Sprintf("set identity_insert %s on;", tableName)
}
