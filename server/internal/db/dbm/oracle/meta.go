package oracle

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/jsonx"
	"strings"

	"github.com/may-fly/cast"
	go_ora "github.com/sijms/go-ora/v2"
)

func init() {
	dbi.Register(dbi.DbTypeOracle, new(OraMeta))
}

type OraMeta struct {
}

func (md *OraMeta) GetSqlDb(d *dbi.DbInfo) (*sql.DB, error) {
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

	// 从extra获取sid或serviceName
	serviceName := ""
	if d.Extra != "" {
		extraMap := jsonx.ToMap(d.Extra)
		serviceName = cast.ToString(extraMap["serviceName"])
		if sid := cast.ToString(extraMap["sid"]); sid != "" {
			urlOptions["SID"] = sid
		}
	}

	urlOptions["TIMEOUT"] = "1000"
	connStr := go_ora.BuildUrl(d.Host, d.Port, serviceName, d.Username, d.Password, urlOptions)
	conn, err := sql.Open("oracle", connStr)
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

func (om *OraMeta) GetDialect(conn *dbi.DbConn) dbi.Dialect {
	return &OracleDialect{dc: conn}
}

func (om *OraMeta) GetMetaData(conn *dbi.DbConn) *dbi.MetaDataX {
	return dbi.NewMetaDataX(&OracleMetaData{dc: conn})
}
