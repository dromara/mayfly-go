package dm

import (
	"fmt"
	"io"
	"mayfly-go/internal/db/dbm/dbi"
)

type DumpHelper struct {
}

func (dh *DumpHelper) BeforeInsert(writer io.Writer, tableName string) {

}

func (dh *DumpHelper) BeforeInsertSql(quoteSchema string, tableName string) string {
	return fmt.Sprintf("set identity_insert %s on;", tableName)
}

func (dh *DumpHelper) AfterInsert(writer io.Writer, tableName string, columns []dbi.Column) {
	writer.Write([]byte("COMMIT;\n"))
}
