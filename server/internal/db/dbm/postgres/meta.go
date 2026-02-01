package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"

	_ "gitee.com/liuzongyang/libpq"
)

func init() {
	meta := new(Meta)
	dbi.Register(DbTypePostgres, meta)
	dbi.Register(DbTypeKingbaseEs, meta)
	dbi.Register(DbTypeVastbase, meta)

	gauss := &Meta{
		Param: "dbtype=gauss",
	}
	dbi.Register(DbTypeGauss, gauss)
}

const (
	DbTypePostgres   dbi.DbType = "postgres"
	DbTypeGauss      dbi.DbType = "gauss"
	DbTypeKingbaseEs dbi.DbType = "kingbaseEs"
	DbTypeVastbase   dbi.DbType = "vastbase"
)

type Meta struct {
	Param string
}

func (pm *Meta) GetSqlDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) {
	db := d.Database
	var dbParam string
	existSchema := false
	// postgres database可以使用db/schema表示，方便连接指定schema, 若不存在schema则使用默认schema
	ss := strings.Split(db, "/")
	if len(ss) > 1 {
		existSchema = true
		dbParam = fmt.Sprintf("dbname=%s search_path=%s", ss[0], ss[len(ss)-1])
	} else {
		dbParam = "dbname=" + db
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s %s sslmode=disable connect_timeout=8", d.Host, d.Port, d.Username, d.Password, dbParam)
	// 存在额外指定参数，则拼接该连接参数
	if d.Params != "" {
		// 存在指定的db，则需要将dbInstance配置中的parmas排除掉dbname和search_path
		if db != "" {
			paramArr := strings.Split(d.Params, "&")
			paramArr = collx.ArrayRemoveFunc(paramArr, func(param string) bool {
				if strings.HasPrefix(param, "dbname=") {
					return true
				}
				if existSchema && strings.HasPrefix(param, "search_path") {
					return true
				}
				return false
			})
			d.Params = strings.Join(paramArr, " ")
		}
		dsn = fmt.Sprintf("%s %s", dsn, strings.Join(strings.Split(d.Params, "&"), " "))
	}

	if pm.Param != "" && !strings.Contains(dsn, "dbtype") {
		dsn = fmt.Sprintf("%s %s", dsn, pm.Param)
	}

	return sql.Open("postgres", dsn)
}

func (pm *Meta) GetDialect(conn *dbi.DbConn) dbi.Dialect {
	return &PgsqlDialect{dc: conn}
}

func (pm *Meta) GetMetadata(conn *dbi.DbConn) dbi.Metadata {
	return &PgsqlMetadata{dc: conn}
}

func (pm *Meta) GetDbDataTypes() []*dbi.DbDataType {
	return collx.AsArray(
		Bool, Int2, Int4, Int8, Numeric, Decimal, Smallserial, Serial, Bigserial, Largeserial,
		Money,
		Char, Nchar, Varchar, Text, Json, Jsonb,
		Date, Time, Timestamp,
		Bytea,
	)
}

func (pm *Meta) GetCommonTypeConverter() dbi.CommonTypeConverter {
	return &commonTypeConverter{}
}
