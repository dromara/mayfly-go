package sqlite

import (
	"database/sql"
	"errors"
	"mayfly-go/internal/db/dbm/dbi"
	"os"
)

func init() {
	dbi.Register(dbi.DbTypeSqlite, new(SqliteMeta))
}

type SqliteMeta struct {
}

func (md *SqliteMeta) GetSqlDb(d *dbi.DbInfo) (*sql.DB, error) {
	// 用host字段来存sqlite的文件路径
	// 检查文件是否存在,否则报错，基于sqlite会自动创建文件，为了服务器文件安全，所以先确定文件存在再连接，不自动创建
	if _, err := os.Stat(d.Host); err != nil {
		return nil, errors.New("数据库文件不存在")
	}
	return sql.Open("sqlite", d.Host)
}

func (md *SqliteMeta) GetDialect(conn *dbi.DbConn) dbi.Dialect {
	return &SqliteDialect{conn}
}
