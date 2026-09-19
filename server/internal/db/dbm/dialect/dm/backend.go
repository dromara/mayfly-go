package dm

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"strings"

	_ "gitee.com/chunanyong/dm"
)

func init() {
	dbi.RegisterBackend(DbTypeDM, new(Backend))

	// 类型引擎注册（迁移/同步类型系统，与 Backend 解耦）
	dbi.RegisterTypeEngine(DbTypeDM, func(b *dbi.TypeEngineBuilder) {
		b.RegisterTypes(
			CHAR, VARCHAR, TEXT, LONG, LONGVARCHAR, IMAGE, LONGVARBINARY, CLOB,
			BLOB,
			NUMERIC, DECIMAL, NUMBER, INTEGER, INT, BIGINT, TINYINT, BYTE, SMALLINT, BIT, DOUBLE, FLOAT,
			TIME, DATE, TIMESTAMP, DATETIME,
			ST_CURVE, ST_LINESTRING, ST_GEOMCOLLECTION, ST_GEOMETRY, ST_MULTICURVE, ST_MULTILINESTRING,
			ST_MULTIPOINT, ST_MULTIPOLYGON, ST_MULTISURFACE, ST_POINT, ST_POLYGON, ST_SURFACE,
			TABLES,
		)

		clearLen := func(col *dbi.Column) {
			col.CharMaxLength = 0
			col.NumPrecision = 0
		}

		// 字符串类
		b.RegisterRule(dbi.TCVarchar, func(col *dbi.Column) *dbi.DbDataType {
			if col.CharMaxLength > 32767 {
				col.CharMaxLength = 0
				return TEXT
			}
			return VARCHAR
		})
		b.RegisterRule(dbi.TCChar, func(col *dbi.Column) *dbi.DbDataType { return CHAR })
		b.RegisterRule(dbi.TCText, func(col *dbi.Column) *dbi.DbDataType {
			col.CharMaxLength = 0
			return TEXT
		})
		b.RegisterRule(dbi.TCMediumtext, func(col *dbi.Column) *dbi.DbDataType {
			col.CharMaxLength = 0
			return TEXT
		})
		b.RegisterRule(dbi.TCLongtext, func(col *dbi.Column) *dbi.DbDataType {
			col.CharMaxLength = 0
			return LONGVARCHAR
		})

		// 整数/布尔/位类
		b.RegisterRule(dbi.TCBit, func(col *dbi.Column) *dbi.DbDataType {
			clearLen(col)
			return BIT
		})
		b.RegisterRule(dbi.TCBool, func(col *dbi.Column) *dbi.DbDataType { return BIT })
		b.RegisterRule(dbi.TCInt1, func(col *dbi.Column) *dbi.DbDataType {
			clearLen(col)
			return TINYINT
		})
		b.RegisterRule(dbi.TCInt2, func(col *dbi.Column) *dbi.DbDataType {
			clearLen(col)
			return SMALLINT
		})
		b.RegisterRule(dbi.TCInt4, func(col *dbi.Column) *dbi.DbDataType {
			clearLen(col)
			return INTEGER
		})
		b.RegisterRule(dbi.TCInt8, func(col *dbi.Column) *dbi.DbDataType {
			clearLen(col)
			return BIGINT
		})

		// 数值类
		b.RegisterRule(dbi.TCNumeric, func(col *dbi.Column) *dbi.DbDataType {
			clearLen(col)
			col.NumScale = 0
			return NUMBER
		})
		b.RegisterRule(dbi.TCDecimal, func(col *dbi.Column) *dbi.DbDataType {
			dbi.FillUnboundedDecimal(col, 38, 19)
			dbi.ClampDecimalPrecision(col, 38, 38)
			return DECIMAL
		})

		// 无符号整数类
		b.RegisterRule(dbi.TCUnsignedInt8, func(col *dbi.Column) *dbi.DbDataType {
			clearLen(col)
			return BIGINT
		})
		b.RegisterRule(dbi.TCUnsignedInt4, func(col *dbi.Column) *dbi.DbDataType {
			clearLen(col)
			return INT
		})
		b.RegisterRule(dbi.TCUnsignedInt2, func(col *dbi.Column) *dbi.DbDataType {
			clearLen(col)
			return INT
		})
		b.RegisterRule(dbi.TCUnsignedInt1, func(col *dbi.Column) *dbi.DbDataType {
			clearLen(col)
			return INT
		})

		// 日期时间类
		b.RegisterRule(dbi.TCDate, func(col *dbi.Column) *dbi.DbDataType {
			clearLen(col)
			return DATE
		})
		b.RegisterRule(dbi.TCTime, func(col *dbi.Column) *dbi.DbDataType {
			clearLen(col)
			return TIME
		})
		b.RegisterRule(dbi.TCDateTime, func(col *dbi.Column) *dbi.DbDataType {
			clearLen(col)
			return DATETIME
		})
		b.RegisterRule(dbi.TCTimestamp, func(col *dbi.Column) *dbi.DbDataType {
			clearLen(col)
			return TIMESTAMP
		})

		// 二进制类
		for _, cat := range []dbi.TypeCategory{dbi.TCBinary, dbi.TCVarbinary, dbi.TCMediumblob, dbi.TCBlob, dbi.TCLongblob} {
			b.RegisterRule(cat, func(col *dbi.Column) *dbi.DbDataType { return BLOB })
		}

		// Enum/JSON
		b.RegisterRule(dbi.TCEnum, func(col *dbi.Column) *dbi.DbDataType { return VARCHAR })
		b.RegisterRule(dbi.TCJSON, func(col *dbi.Column) *dbi.DbDataType { return VARCHAR })
	})
}

const (
	DbTypeDM dbi.DbType = "dm"
)

var _ dbi.DbBackend = (*Backend)(nil)

type Backend struct {
	dbi.BaseBackend
}

func (dm *Backend) GetSqlDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) {
	driverName := "dm"
	db := d.Database
	dbParam := "?escapeProcess=true"
	if db != "" {
		ss := strings.Split(db, "/")
		if len(ss) > 1 {
			dbParam = fmt.Sprintf("%s&schema=\"%s\"", dbParam, ss[1])
		}
	}
	if d.Params != "" {
		dbParam += "&" + d.Params
	}
	dsn := fmt.Sprintf("dm://%s:%s@%s:%d%s&appName=mayfly-go", d.Username, d.Password, d.Host, d.Port, dbParam)
	return sql.Open(driverName, dsn)
}

func (dm *Backend) GetDialect(di *dbi.DbInfo) dbi.Dialect {
	return &DMDialect{di: di}
}

func (dm *Backend) GetServerInfo(di *dbi.DbInfo) dbi.ServerInfo {
	return &DMMetadata{
		di: di,
	}
}

func (dm *Backend) GetMetadataProvider(di *dbi.DbInfo) dbi.MetadataProvider {
	return &DMMetadata{
		di: di,
	}
}
