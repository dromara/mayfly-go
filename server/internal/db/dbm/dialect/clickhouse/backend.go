package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

func init() {
	dbi.RegisterBackend(DbTypeClickHouse, new(Backend))

	// 类型引擎注册（迁移/同步类型系统，与 Backend 解耦）
	// clickhouse的类型均不携带长度参数（如String、Int32），需清空源长度信息避免生成非法DDL
	dbi.RegisterTypeEngine(DbTypeClickHouse, func(b *dbi.TypeEngineBuilder) {
		b.RegisterTypes(
			UInt8, UInt16, UInt32, UInt64, Int8, Int16, Int32, Int64,
			Float32, Float64,
			String, FixedString,
			DateTime, DateTime64, Date, Date32,
			UUID,
			IPv4, IPv6,
			Bool,
			Decimal, Decimal32, Decimal64, Decimal128, Decimal256,
			Enum8, Enum16,
			Array, Tuple, Map, Nested, AggregateFunction, SimpleAggregateFunction,
			LowCardinality, Nullable,
		)

		// clearLength 清空列的长度与精度。clickhouse类型均不携带长度/精度参数
		cl := func(col *dbi.Column) {
			col.CharMaxLength = 0
			col.NumPrecision = 0
			col.NumScale = 0
		}

		// 字符串类 — 全部归 String
		for _, cat := range []dbi.TypeCategory{dbi.TCVarchar, dbi.TCChar, dbi.TCText, dbi.TCMediumtext, dbi.TCLongtext} {
			b.RegisterRule(cat, func(col *dbi.Column) *dbi.DbDataType {
				cl(col)
				return String
			})
		}

		// 整数/布尔/位类
		b.RegisterRule(dbi.TCBit, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return UInt8
		})
		b.RegisterRule(dbi.TCBool, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return Bool
		})
		b.RegisterRule(dbi.TCInt1, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return Int8
		})
		b.RegisterRule(dbi.TCInt2, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return Int16
		})
		b.RegisterRule(dbi.TCInt4, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return Int32
		})
		b.RegisterRule(dbi.TCInt8, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return Int64
		})

		// 数值类
		b.RegisterRule(dbi.TCNumeric, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return Float64
		})
		b.RegisterRule(dbi.TCDecimal, func(col *dbi.Column) *dbi.DbDataType {
			dbi.FillUnboundedDecimal(col, 38, 19)
			dbi.ClampDecimalPrecision(col, 76, 76)
			return Decimal
		})

		// 无符号整数类
		b.RegisterRule(dbi.TCUnsignedInt8, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return UInt64
		})
		b.RegisterRule(dbi.TCUnsignedInt4, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return UInt32
		})
		b.RegisterRule(dbi.TCUnsignedInt2, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return UInt16
		})
		b.RegisterRule(dbi.TCUnsignedInt1, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return UInt8
		})

		// 日期时间类
		b.RegisterRule(dbi.TCDate, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return Date
		})
		b.RegisterRule(dbi.TCTime, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return DateTime
		})
		b.RegisterRule(dbi.TCDateTime, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return DateTime
		})
		b.RegisterRule(dbi.TCTimestamp, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return DateTime
		})

		// 二进制类 — 以String承载
		for _, cat := range []dbi.TypeCategory{dbi.TCBinary, dbi.TCVarbinary, dbi.TCMediumblob, dbi.TCBlob, dbi.TCLongblob} {
			b.RegisterRule(cat, func(col *dbi.Column) *dbi.DbDataType {
				cl(col)
				return String
			})
		}

		// Enum/JSON
		b.RegisterRule(dbi.TCEnum, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return String
		})
		b.RegisterRule(dbi.TCJSON, func(col *dbi.Column) *dbi.DbDataType {
			cl(col)
			return String
		})
	})
}

const (
	DbTypeClickHouse dbi.DbType = "clickhouse"
)

var _ dbi.DbBackend = (*Backend)(nil)

type Backend struct {
	dbi.BaseBackend
}

func (cm *Backend) GetCapabilities() dbi.MetadataCapabilities {
	return dbi.MetadataCapabilities{
		SupportsSchemas:           false,
		SupportsIndexes:           false,
		SupportsForeignKeys:       false,
		SupportsComments:          true,
		SupportsDDLExport:         true,
		SupportsGeneratedColumns:  false,
		SupportsIdentityColumns:   false,
		SupportsExpressionDefault: false,
	}
}

func (cm *Backend) GetSqlDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) {
	dbName := d.GetDatabase()
	if dbName == "" {
		dbName = "default"
	}
	dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s", d.Username, d.Password, d.Host, d.Port, dbName)
	if d.Params != "" {
		dsn = fmt.Sprintf("%s?%s", dsn, d.Params)
	} else {
		dsn = fmt.Sprintf("%s?dial_timeout=10s&read_timeout=30s", dsn)
	}
	const driverName = "clickhouse"
	return sql.Open(driverName, dsn)
}

func (cm *Backend) GetDialect(di *dbi.DbInfo) dbi.Dialect {
	return &ClickHouseDialect{di: di}
}

func (cm *Backend) GetServerInfo(di *dbi.DbInfo) dbi.ServerInfo {
	return &ClickHouseMetadata{di: di}
}

func (cm *Backend) GetMetadataProvider(di *dbi.DbInfo) dbi.MetadataProvider {
	return &ClickHouseMetadata{di: di}
}
