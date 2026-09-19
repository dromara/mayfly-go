package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"time"

	"github.com/go-sql-driver/mysql"
)

func init() {
	backend := new(Backend)
	dbi.RegisterBackend(DbTypeMysql, backend)
	dbi.RegisterBackend(DbTypeMariadb, backend)

	// 类型引擎注册（迁移/同步类型系统，与 Backend 解耦）
	dbi.RegisterTypeEngine(DbTypeMysql, func(b *dbi.TypeEngineBuilder) {
		b.RegisterTypes(
			UnsignedBigint, UnsignedInt, UnsignedMediumint, UnsignedSmallint, UnsignedTinyint,
			Bigint, Tinyint, Smallint, Int, Bit, Float, Double, Decimal,
			Varchar, Char, Text, Longtext, Mediumtext,
			Datetime, Date, Time, Timestamp,
			Enum, JSON, Set,
			Binary, Blob, Longblob, Mediumblob, Tinyblob, Varbinary,
		)

		// 字符串类
		b.RegisterRule(dbi.TCVarchar, func(col *dbi.Column) *dbi.DbDataType {
			if col.CharMaxLength > 16383 {
				col.CharMaxLength = 0
				return Text
			}
			return Varchar
		})
		b.RegisterRules(Char, dbi.TCChar)
		b.RegisterRule(dbi.TCText, func(col *dbi.Column) *dbi.DbDataType {
			col.CharMaxLength = 0
			col.NumPrecision = 0
			return Text
		})
		b.RegisterRule(dbi.TCMediumtext, func(col *dbi.Column) *dbi.DbDataType {
			col.CharMaxLength = 0
			col.NumPrecision = 0
			return Mediumtext
		})
		b.RegisterRule(dbi.TCLongtext, func(col *dbi.Column) *dbi.DbDataType {
			col.CharMaxLength = 0
			col.NumPrecision = 0
			return Longtext
		})

		// 整数/布尔/位类
		b.RegisterRules(Bit, dbi.TCBit)
		b.RegisterRules(Tinyint, dbi.TCBool, dbi.TCInt1)
		b.RegisterRules(Smallint, dbi.TCInt2)
		b.RegisterRules(Int, dbi.TCInt4)
		b.RegisterRules(Bigint, dbi.TCInt8)

		// 数值类
		b.RegisterRule(dbi.TCNumeric, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClearNumPrecision(col)
			return Double
		})
		b.RegisterRule(dbi.TCDecimal, func(col *dbi.Column) *dbi.DbDataType {
			dbi.FillUnboundedDecimal(col, 65, 30)
			dbi.ClampDecimalPrecision(col, 65, 30)
			return Decimal
		})

		// 无符号整数类
		b.RegisterRules(UnsignedBigint, dbi.TCUnsignedInt8)
		b.RegisterRules(UnsignedInt, dbi.TCUnsignedInt4)
		b.RegisterRules(UnsignedMediumint, dbi.TCUnsignedInt2)
		b.RegisterRules(UnsignedSmallint, dbi.TCUnsignedInt1)

		// 日期时间类
		b.RegisterRule(dbi.TCDate, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClearNumPrecision(col)
			return Date
		})
		b.RegisterRule(dbi.TCTime, func(col *dbi.Column) *dbi.DbDataType {
			dbi.NormalizeTimeFsp(col, 6)
			return Time
		})
		b.RegisterRule(dbi.TCDateTime, func(col *dbi.Column) *dbi.DbDataType {
			dbi.NormalizeTimeFsp(col, 6)
			return Datetime
		})
		b.RegisterRule(dbi.TCTimestamp, func(col *dbi.Column) *dbi.DbDataType {
			dbi.NormalizeTimeFsp(col, 6)
			return Timestamp
		})

		// 二进制类
		b.RegisterRule(dbi.TCBinary, func(col *dbi.Column) *dbi.DbDataType {
			if col.CharMaxLength <= 0 {
				return Blob
			}
			return Binary
		})
		b.RegisterRule(dbi.TCVarbinary, func(col *dbi.Column) *dbi.DbDataType { return Varbinary })
		b.RegisterRule(dbi.TCMediumblob, func(col *dbi.Column) *dbi.DbDataType { return Mediumblob })
		b.RegisterRule(dbi.TCBlob, func(col *dbi.Column) *dbi.DbDataType { return Blob })
		b.RegisterRule(dbi.TCLongblob, func(col *dbi.Column) *dbi.DbDataType {
			col.CharMaxLength = 0
			return Longblob
		})

		// Enum/JSON
		b.RegisterRules(Enum, dbi.TCEnum)
		b.RegisterRules(JSON, dbi.TCJSON)
	})

	// 别名方言复用主方言的类型引擎
	dbi.RegisterTypeEngineAlias(DbTypeMariadb, DbTypeMysql)
}

const (
	DbTypeMysql   dbi.DbType = "mysql"
	DbTypeMariadb dbi.DbType = "mariadb"
)

var _ dbi.DbBackend = (*Backend)(nil)

type Backend struct {
	dbi.BaseBackend
}

func (mm *Backend) GetCapabilities() dbi.MetadataCapabilities {
	return dbi.MetadataCapabilities{
		SupportsSchemas:           false,
		SupportsIndexes:           true,
		SupportsForeignKeys:       true,
		SupportsComments:          true,
		SupportsDDLExport:         true,
		SupportsGeneratedColumns:  true,
		SupportsIdentityColumns:   true,
		SupportsExpressionDefault: true,
	}
}

func (mm *Backend) GetSqlDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) {
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

func (mm *Backend) GetDialect(di *dbi.DbInfo) dbi.Dialect {
	return &MysqlDialect{di: di}
}

func (mm *Backend) GetServerInfo(di *dbi.DbInfo) dbi.ServerInfo {
	return &MysqlMetadata{di: di}
}

func (mm *Backend) GetMetadataProvider(di *dbi.DbInfo) dbi.MetadataProvider {
	return &MysqlMetadata{di: di}
}
