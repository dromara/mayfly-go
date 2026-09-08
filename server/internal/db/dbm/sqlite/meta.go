package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"os"
)

func init() {
	dbi.Register(DbTypeSqlite, new(Meta))
}

const (
	DbTypeSqlite dbi.DbType = "sqlite"
)

var _ dbi.Meta = (*Meta)(nil)

type Meta struct {
}

func (md *Meta) GetSqlDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) {
	// 用host字段来存sqlite的文件路径
	// 检查文件是否存在,否则报错，基于sqlite会自动创建文件，为了服务器文件安全，所以先确定文件存在再连接，不自动创建
	if _, err := os.Stat(d.Host); err != nil {
		return nil, errors.New("数据库文件不存在")
	}

	db, err := sql.Open("sqlite", d.Host)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("PRAGMA busy_timeout = 50000;")
	return db, err
}

func (sm *Meta) GetDialect(conn *dbi.DbConn) dbi.Dialect {
	return &SqliteDialect{dc: conn}
}

func (sm *Meta) GetMetadata(conn *dbi.DbConn) dbi.Metadata {
	return &SqliteMetadata{dc: conn}
}

func (sm *Meta) GetDbDataTypes() []*dbi.DbDataType {
	return collx.AsArray(
		Integer, Real,
		Text,
		Blob,
		DateTime, Date, Time,
		// SQLite弱类型下常见的外库风格类型名别名（否则整型/数值/日期列会被静默转为varchar）
		Int, Int2, Int4, Int8, Tinyint, Smallint, Mediumint, Bigint, UnsignedBigint,
		Char, NChar, Varchar, NVarchar, Clob,
		Float, Float4, Float8, Double, DoublePrecision,
		Numeric, Decimal, Dec, Fixed,
		Timestamp,
	)
}

func (sm *Meta) GetCommonTypeConverter() dbi.CommonTypeConverter {
	return &commonTypeConverter{}
}
