package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"time"

	"github.com/go-sql-driver/mysql"
)

func init() {
	meta := new(Meta)
	dbi.Register(DbTypeMysql, meta)
	dbi.Register(DbTypeMariadb, meta)
}

const (
	DbTypeMysql   dbi.DbType = "mysql"
	DbTypeMariadb dbi.DbType = "mariadb"
)

type Meta struct {
}

func (mm *Meta) GetSqlDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) {
	d.Network = "tcp"
	// 使用go-sql-driver的标准DSN构建，自动对用户名、密码等特殊字符进行转义，避免注入或连接串错误
	cfg := mysql.NewConfig()
	cfg.User = d.Username
	cfg.Passwd = d.Password
	cfg.Net = d.Network
	cfg.Addr = fmt.Sprintf("%s:%d", d.Host, d.Port)
	cfg.DBName = d.Database
	cfg.Params = map[string]string{"parseTime": "true"}
	cfg.Timeout = 8 * time.Second
	// FormatDSN会自动转义用户名密码等特殊字符，自定义参数以&拼接在其后
	dsn := cfg.FormatDSN()
	if d.Params != "" {
		dsn = fmt.Sprintf("%s&%s", dsn, d.Params)
	}

	return sql.Open("mysql", dsn)
}

func (mm *Meta) GetDialect(conn *dbi.DbConn) dbi.Dialect {
	return &MysqlDialect{dc: conn}
}

func (mm *Meta) GetMetadata(conn *dbi.DbConn) dbi.Metadata {
	return &MysqlMetadata{dc: conn}
}

func (mm *Meta) GetDbDataTypes() []*dbi.DbDataType {
	return collx.AsArray(
		UnsignedBigint, UnsignedInt, UnsignedMediumint, UnsignedSmallint, Bigint, Tinyint, Smallint, Int, Bit, Float, Double, Decimal,
		Varchar, Char, Text, Longtext, Mediumtext,
		Datetime, Date, Time, Timestamp,
		Enum, JSON, Set,
		Binary, Blob, Longblob, Mediumblob, Varbinary,
	)
}

func (mm *Meta) GetCommonTypeConverter() dbi.CommonTypeConverter {
	return &commonTypeConverter{}
}
