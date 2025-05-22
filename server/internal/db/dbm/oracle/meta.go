package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/may-fly/cast"
	go_ora "github.com/sijms/go-ora/v2"
)

func init() {
	dbi.Register(DbTypeOracle, new(Meta))
}

const (
	DbVersionOracle11 dbi.DbVersion = "11"
	DbTypeOracle      dbi.DbType    = "oracle"
)

type Meta struct {
}

func (om *Meta) GetSqlDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) {
	err := d.IfUseSshTunnelChangeIpPort(ctx)
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
	serviceName := d.GetExtraString("serviceName")
	if sid := d.GetExtraString("sid"); sid != "" {
		urlOptions["SID"] = sid
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

func (om *Meta) GetDialect(conn *dbi.DbConn) dbi.Dialect {
	return &OracleDialect{dc: conn}
}

func (om *Meta) GetMetadata(conn *dbi.DbConn) dbi.Metadata {

	// 查询数据库版本信息，以做兼容性处理
	if conn.Info.Version == "" && !conn.Info.DefaultVersion {
		if conn.GetDb() != nil {
			_, res, _ := conn.Query("select VERSION from v$instance")
			if len(res) > 0 {
				version := cast.ToString(res[0]["VERSION"])
				// 11开头为11g版本
				if strings.HasPrefix(version, "11") {
					conn.Info.Version = DbVersionOracle11
					conn.Info.DefaultVersion = false
				} else {
					conn.Info.DefaultVersion = true
				}
			}
		}
	}

	if conn.Info.Version == DbVersionOracle11 {
		md := &OracleMetadata11{}
		md.dc = conn
		md.version = DbVersionOracle11
		return md
	}
	return &OracleMetadata{dc: conn}
}

func (sm *Meta) GetDbDataTypes() []*dbi.DbDataType {
	return collx.AsArray[*dbi.DbDataType](
		CHAR,
		NCHAR,
		VARCHAR2,
		NVARCHAR2,
		TEXT,
		LONG,
		LONGVARCHAR,
		IMAGE,
		LONGVARBINARY,
		CLOB,
		BLOB,
		DECIMAL,
		NUMBER,
		INTEGER,
		INT,
		BIGINT,
		TINYINT,
		BYTE,
		SMALLINT,
		BIT,
		DOUBLE,
		FLOAT,
		TIME,
		DATE,
		TIMESTAMP,
	)
}

func (mm *Meta) GetCommonTypeConverter() dbi.CommonTypeConverter {
	return &commonTypeConverter{}
}
