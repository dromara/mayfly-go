package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"net/url"
	"strings"

	_ "github.com/microsoft/go-mssqldb"
)

func init() {
	meta := new(Meta)
	dbi.Register(DbTypeMssql, meta)
}

const (
	DbTypeMssql dbi.DbType = "mssql"
)

type Meta struct {
}

func (mm *Meta) GetSqlDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) {
	err := d.IfUseSshTunnelChangeIpPort(ctx)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	// The application name (default is go-mssqldb)
	query.Add("app name", "mayfly")
	// 指定与服务器协商加密的最低TLS版本
	query.Add("tlsmin", "1.0")
	// 连接超时时间10秒
	query.Add("connection timeout", "10")
	if d.Database != "" {
		ss := strings.Split(d.Database, "/")
		if len(ss) > 1 {
			query.Add("database", ss[0])
			query.Add("schema", ss[1])
		} else {
			query.Add("database", d.Database)
		}
	}
	params := query.Encode()
	if d.Params != "" {
		if !strings.HasPrefix(d.Params, "&") {
			params = params + "&" + d.Params
		} else {
			params = params + d.Params
		}
	}

	const driverName = "mssql"
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%d?%s", url.PathEscape(d.Username), url.PathEscape(d.Password), d.Host, d.Port, params)
	return sql.Open(driverName, dsn)
}

func (mm *Meta) GetDialect(conn *dbi.DbConn) dbi.Dialect {
	return &MssqlDialect{dc: conn}
}

func (mm *Meta) GetMetadata(conn *dbi.DbConn) dbi.Metadata {
	return &MssqlMetadata{dc: conn}
}

func (sm *Meta) GetDbDataTypes() []*dbi.DbDataType {
	return collx.AsArray[*dbi.DbDataType](Bigint,
		Numeric,
		Bit,
		Smallint,
		Decimal,
		Smallmoney,
		Int,
		Tinyint,
		Money,
		Float,
		Real,
		Date,
		Datetimeoffset,
		Datetime2,
		Smalldatetime,
		Datetime,
		Time,
		Char,
		Varchar,
		Text,
		Nchar,
		Nvarchar,
		Ntext,
		Binary,
		Varbinary,
		Cursor,
		Rowversion,
		Hierarchyid,
		Uniqueidentifier,
		Sql_variant,
		Xml,
		Table,
		Geometry,
		Geography,
	)
}

func (sm *Meta) GetCommonTypeConverter() dbi.CommonTypeConverter {
	return &commonTypeConverter{}
}
