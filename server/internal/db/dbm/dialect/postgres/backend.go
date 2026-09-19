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
	backend := new(Backend)
	dbi.RegisterBackend(DbTypePostgres, backend)
	dbi.RegisterBackend(DbTypeKingbaseEs, backend)
	dbi.RegisterBackend(DbTypeVastbase, backend)

	gauss := &Backend{
		Param: "dbtype=gauss",
	}
	dbi.RegisterBackend(DbTypeGauss, gauss)

	// 类型引擎注册（迁移/同步类型系统，与 Backend 解耦）
	dbi.RegisterTypeEngine(DbTypePostgres, func(b *dbi.TypeEngineBuilder) {
		b.RegisterTypes(
			Bool, Int2, Int4, Int8, Numeric, Decimal, Smallserial, Serial, Bigserial, Largeserial,
			Float4, Float8,
			Money,
			Char, Nchar, Bpchar, Varchar, Text, Json, Jsonb,
			Date, Time, Timetz, Timestamp, Timestamptz,
			Bytea,
			Bit, Varbit, BitVarying,
		)

		// 字符串类
		b.RegisterRules(Varchar, dbi.TCVarchar)
		b.RegisterRules(Char, dbi.TCChar)
		b.RegisterRules(Text, dbi.TCText, dbi.TCMediumtext, dbi.TCLongtext)

		// 整数/布尔/位类
		b.RegisterRules(Int2, dbi.TCBit, dbi.TCInt1, dbi.TCInt2)
		b.RegisterRule(dbi.TCBool, func(col *dbi.Column) *dbi.DbDataType { return Bool })
		b.RegisterRules(Int4, dbi.TCInt4)
		b.RegisterRules(Int8, dbi.TCInt8)

		// 数值类
		b.RegisterRule(dbi.TCNumeric, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClearNumPrecision(col)
			return Numeric
		})
		b.RegisterRule(dbi.TCDecimal, func(col *dbi.Column) *dbi.DbDataType {
			if col.NumPrecision <= 0 {
				dbi.ClearNumPrecision(col)
			} else {
				dbi.ClampDecimalPrecision(col, 1000, 1000)
			}
			return Decimal
		})

		// 无符号整数类
		b.RegisterRules(Numeric, dbi.TCUnsignedInt8)
		b.RegisterRules(Int8, dbi.TCUnsignedInt4)
		b.RegisterRules(Int4, dbi.TCUnsignedInt2)
		b.RegisterRules(Int2, dbi.TCUnsignedInt1)

		// 日期时间类
		b.RegisterRule(dbi.TCDate, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClearNumPrecision(col)
			return Date
		})
		b.RegisterRule(dbi.TCTime, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClampTimeFsp(col, 6)
			return Time
		})
		b.RegisterRule(dbi.TCDateTime, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClampTimeFsp(col, 6)
			return Timestamp
		})
		b.RegisterRule(dbi.TCTimestamp, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClampTimeFsp(col, 6)
			return Timestamp
		})

		// 二进制类
		b.RegisterRules(Bytea, dbi.TCBinary, dbi.TCVarbinary, dbi.TCMediumblob, dbi.TCBlob, dbi.TCLongblob)

		// Enum/JSON
		b.RegisterRules(Varchar, dbi.TCEnum)
		b.RegisterRules(Json, dbi.TCJSON)
	})

	// 别名方言复用主方言的类型引擎
	dbi.RegisterTypeEngineAlias(DbTypeGauss, DbTypePostgres)
	dbi.RegisterTypeEngineAlias(DbTypeKingbaseEs, DbTypePostgres)
	dbi.RegisterTypeEngineAlias(DbTypeVastbase, DbTypePostgres)
}

const (
	DbTypePostgres   dbi.DbType = "postgres"
	DbTypeGauss      dbi.DbType = "gauss"
	DbTypeKingbaseEs dbi.DbType = "kingbaseEs"
	DbTypeVastbase   dbi.DbType = "vastbase"
)

var _ dbi.DbBackend = (*Backend)(nil)

type Backend struct {
	dbi.BaseBackend
	Param string
}

func (pm *Backend) GetSqlDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) {
	db := d.Database
	var dbParam string
	existSchema := false
	// postgres database可以使用db/schema表示，方便连接指定schema, 若不存在schema则使用默认schema
	ss := strings.Split(db, "/")
	if len(ss) > 1 {
		existSchema = true
		dbParam = fmt.Sprintf("dbname=%s search_path=%s", escapeLibpqParam(ss[0]), escapeLibpqParam(ss[len(ss)-1]))
	} else {
		dbParam = "dbname=" + escapeLibpqParam(db)
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s %s sslmode=disable connect_timeout=8",
		escapeLibpqParam(d.Host), d.Port, escapeLibpqParam(d.Username), escapeLibpqParam(d.Password), dbParam)
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

func (pm *Backend) GetDialect(di *dbi.DbInfo) dbi.Dialect {
	return &PgsqlDialect{di: di}
}

func (pm *Backend) GetServerInfo(di *dbi.DbInfo) dbi.ServerInfo {
	return &PgsqlMetadata{di: di}
}

func (pm *Backend) GetMetadataProvider(di *dbi.DbInfo) dbi.MetadataProvider {
	return &PgsqlMetadata{di: di}
}

// escapeLibpqParam libpq的keyword/value格式中，值含空白字符时必须用单引号包裹，
// 值中的单引号与反斜杠需反斜杠转义；否则密码含空格/特殊字符时会导致DSN解析错误或连接参数被截断
func escapeLibpqParam(val string) string {
	if !strings.ContainsAny(val, " \t\n'\\") {
		return val
	}
	val = strings.ReplaceAll(val, "\\", "\\\\")
	val = strings.ReplaceAll(val, "'", "\\'")
	return "'" + val + "'"
}
