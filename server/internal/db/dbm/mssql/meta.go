package mssql

import (
	"database/sql"
	"fmt"
	_ "github.com/microsoft/go-mssqldb"
	"mayfly-go/internal/db/dbm/dbi"
	"net/url"
	"strings"
)

func init() {
	meta := new(Meta)
	dbi.Register(dbi.DbTypeMssql, meta)
}

type Meta struct {
}

func (md *Meta) GetSqlDb(d *dbi.DbInfo) (*sql.DB, error) {
	err := d.IfUseSshTunnelChangeIpPort()
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

func (md *Meta) GetDialect(conn *dbi.DbConn) dbi.Dialect {
	return &MssqlDialect{conn}
}
