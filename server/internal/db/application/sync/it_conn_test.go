//go:build it

package sync

import (
	"testing"

	_ "mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
)

func verifyMysqlConn(t *testing.T) *dbi.DbConn {
	return syncTestConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
}

func verifyPgConn(t *testing.T) *dbi.DbConn {
	return syncTestConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
}
