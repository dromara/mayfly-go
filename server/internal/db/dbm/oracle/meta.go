package oracle

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"strings"

	go_ora "github.com/sijms/go-ora/v2"
)

func init() {
	dbi.Register(dbi.DbTypeOracle, new(OraMeta))
}

type OraMeta struct {
}

func (md *OraMeta) GetSqlDb(d *dbi.DbInfo) (*sql.DB, error) {
	driverName := "oracle"

	err := d.IfUseSshTunnelChangeIpPort()
	if err != nil {
		return nil, err
	}
	// 参数参考 https://github.com/sijms/go-ora?tab=readme-ov-file#other-connection-options
	urlOptions := make(map[string]string)

	db := d.Database
	schema := ""
	if db != "" {
		// oracle database可以使用db/schema表示，方便连接指定schema, 若不存在schema则使用默认schema
		ss := strings.Split(db, "/")
		if len(ss) > 1 {
			// user=hr&defaultSchema=hr
			schema = ss[1]
		}
	}
	// 解析参数
	if d.Params != "" {
		paramArr := strings.Split(d.Params, "&")
		for _, param := range paramArr {
			ps := strings.Split(param, "=")
			if len(ps) > 1 {
				if ps[0] == "clientCharset" {
					urlOptions["client charset"] = ps[1]
				} else {
					urlOptions[ps[0]] = ps[1]
				}
			}
		}
	}
	urlOptions["TIMEOUT"] = "10"
	connStr := go_ora.BuildUrl(d.Host, d.Port, d.Sid, d.Username, d.Password, urlOptions)
	conn, err := sql.Open(driverName, connStr)
	if err != nil {
		return nil, err
	}
	// 目前没找到如何连接的时候就获取schema的方法，只能连接后再设置
	if schema != "" {
		_, err := conn.Exec(fmt.Sprintf("ALTER SESSION SET CURRENT_SCHEMA=%s", schema))
		if err != nil {
			return nil, err
		}
	}
	return conn, err
}

func (md *OraMeta) GetDialect(conn *dbi.DbConn) dbi.Dialect {
	return &OracleDialect{conn}
}
