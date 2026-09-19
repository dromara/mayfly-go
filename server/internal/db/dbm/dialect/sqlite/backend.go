package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"mayfly-go/internal/db/dbm/dbi"
	"os"
)

func init() {
	dbi.RegisterBackend(DbTypeSqlite, new(Backend))

	// 类型引擎注册（迁移/同步类型系统，与 Backend 解耦）
	dbi.RegisterTypeEngine(DbTypeSqlite, func(b *dbi.TypeEngineBuilder) {
		b.RegisterTypes(
			Integer, Real,
			Text,
			Blob,
			DateTime, Date, Time,
			Int, Int2, Int4, Int8, Tinyint, Smallint, Mediumint, Bigint, UnsignedBigint,
			Char, NChar, Varchar, NVarchar, Clob,
			Float, Float4, Float8, Double, DoublePrecision,
			Numeric, Decimal, Dec, Fixed,
			Timestamp,
		)

		// 字符串类 — SQLite 是动态类型库，所有字符串归 Text
		b.RegisterRules(Text, dbi.TCVarchar, dbi.TCChar, dbi.TCText, dbi.TCMediumtext, dbi.TCLongtext,
			dbi.TCEnum, dbi.TCJSON)

		// 整数/布尔/位类 — 统一归 Integer
		b.RegisterRules(Integer, dbi.TCBit, dbi.TCBool, dbi.TCInt1, dbi.TCInt2, dbi.TCInt4, dbi.TCInt8,
			dbi.TCUnsignedInt8, dbi.TCUnsignedInt4, dbi.TCUnsignedInt2, dbi.TCUnsignedInt1)

		// 数值类 — 必须落 NUMERIC 亲和，不能降级为 real
		b.RegisterRules(Numeric, dbi.TCNumeric)
		b.RegisterRules(Decimal, dbi.TCDecimal)

		// 日期时间类
		b.RegisterRule(dbi.TCDate, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClearNumPrecision(col)
			return Date
		})
		b.RegisterRule(dbi.TCTime, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClearNumPrecision(col)
			return Time
		})
		b.RegisterRule(dbi.TCDateTime, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClearNumPrecision(col)
			return DateTime
		})
		b.RegisterRule(dbi.TCTimestamp, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClearNumPrecision(col)
			return DateTime
		})

		// 二进制类
		b.RegisterRules(Blob, dbi.TCBinary, dbi.TCVarbinary, dbi.TCMediumblob, dbi.TCBlob, dbi.TCLongblob)
	})
}

const (
	DbTypeSqlite dbi.DbType = "sqlite"
)

var _ dbi.DbBackend = (*Backend)(nil)

type Backend struct {
	dbi.BaseBackend
}

func (sm *Backend) GetCapabilities() dbi.MetadataCapabilities {
	return dbi.MetadataCapabilities{
		SupportsSchemas:           false,
		SupportsIndexes:           true,
		SupportsForeignKeys:       true,
		SupportsComments:          false,
		SupportsDDLExport:         true,
		SupportsGeneratedColumns:  true,
		SupportsIdentityColumns:   false,
		SupportsExpressionDefault: true,
	}
}

func (md *Backend) GetSqlDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) {
	// 用host字段来存sqlite的文件路径
	// 检查文件是否存在,否则报错，基于sqlite会自动创建文件，为了服务器文件安全，所以先确定文件存在再连接，不自动创建
	if _, err := os.Stat(d.Host); err != nil {
		return nil, errors.New("数据库文件不存在")
	}

	db, err := sql.Open("sqlite", d.Host)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("PRAGMA busy_timeout = 50000;")
	return db, err
}

func (sm *Backend) GetDialect(di *dbi.DbInfo) dbi.Dialect {
	return &SqliteDialect{di: di}
}

func (sm *Backend) GetServerInfo(di *dbi.DbInfo) dbi.ServerInfo {
	return &SqliteMetadata{di: di}
}

func (sm *Backend) GetMetadataProvider(di *dbi.DbInfo) dbi.MetadataProvider {
	return &SqliteMetadata{di: di}
}
