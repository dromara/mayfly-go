package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/logx"
	"net/url"
	"strings"

	_ "github.com/microsoft/go-mssqldb"
)

func init() {
	backend := new(Backend)
	dbi.RegisterBackend(DbTypeMssql, backend)

	// 类型引擎注册（迁移/同步类型系统，与 Backend 解耦）
	dbi.RegisterTypeEngine(DbTypeMssql, func(b *dbi.TypeEngineBuilder) {
		b.RegisterTypes(
			Bigint, Numeric, Bit, Smallint, Decimal, Smallmoney, Int, Tinyint, Money,
			Float, Real, Date, Datetimeoffset, Datetime2, Smalldatetime, Datetime, Time,
			Char, Varchar, VarcharMax, NvarcharMax, VarbinaryMax,
			Text, Nchar, Nvarchar, Ntext, Binary, Varbinary,
			Cursor, Rowversion, Hierarchyid, Uniqueidentifier, Sql_variant, Xml, Table,
			Geometry, Geography,
		)

		// 字符串类 — 字符类异构目标列统一落Unicode类型（nvarchar）
		b.RegisterRule(dbi.TCVarchar, func(col *dbi.Column) *dbi.DbDataType { return varcharType(col) })
		b.RegisterRule(dbi.TCChar, func(col *dbi.Column) *dbi.DbDataType {
			if col.CharMaxLength <= 0 || col.CharMaxLength > 4000 {
				col.CharMaxLength = 0
				return NvarcharMax
			}
			return Nchar
		})
		// textType: 大文本归nvarchar(max)
		textRule := func(col *dbi.Column) *dbi.DbDataType {
			col.CharMaxLength = 0
			return NvarcharMax
		}
		b.RegisterRule(dbi.TCText, textRule)
		b.RegisterRule(dbi.TCMediumtext, textRule)
		b.RegisterRule(dbi.TCLongtext, textRule)

		// 整数/布尔/位类
		b.RegisterRules(Bit, dbi.TCBit, dbi.TCBool)
		b.RegisterRules(Tinyint, dbi.TCInt1)
		b.RegisterRules(Smallint, dbi.TCInt2)
		b.RegisterRules(Int, dbi.TCInt4)
		b.RegisterRules(Bigint, dbi.TCInt8)

		// 数值类
		b.RegisterRule(dbi.TCNumeric, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClearNumPrecision(col)
			return Float
		})
		b.RegisterRule(dbi.TCDecimal, func(col *dbi.Column) *dbi.DbDataType {
			dbi.FillUnboundedDecimal(col, 38, 19)
			dbi.ClampDecimalPrecision(col, 38, 19)
			return Decimal
		})

		// 无符号整数类
		b.RegisterRules(Bigint, dbi.TCUnsignedInt8)
		b.RegisterRules(Int, dbi.TCUnsignedInt4)
		b.RegisterRules(Smallint, dbi.TCUnsignedInt2)
		b.RegisterRules(Tinyint, dbi.TCUnsignedInt1)

		// 日期时间类
		b.RegisterRule(dbi.TCDate, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClearNumPrecision(col)
			return Date
		})
		b.RegisterRule(dbi.TCTime, func(col *dbi.Column) *dbi.DbDataType {
			dbi.NormalizeTimeFsp(col, 7)
			return Time
		})
		b.RegisterRule(dbi.TCDateTime, func(col *dbi.Column) *dbi.DbDataType {
			dbi.NormalizeTimeFsp(col, 7)
			return Datetime2
		})
		b.RegisterRule(dbi.TCTimestamp, func(col *dbi.Column) *dbi.DbDataType {
			dbi.NormalizeTimeFsp(col, 7)
			return Datetime2
		})

		// 二进制类
		for _, cat := range []dbi.TypeCategory{dbi.TCBinary, dbi.TCVarbinary, dbi.TCMediumblob, dbi.TCBlob, dbi.TCLongblob} {
			b.RegisterRule(cat, func(col *dbi.Column) *dbi.DbDataType { return bytesType(col) })
		}

		// Enum/JSON
		b.RegisterRule(dbi.TCEnum, func(col *dbi.Column) *dbi.DbDataType { return varcharType(col) })
		b.RegisterRule(dbi.TCJSON, textRule)
	})
}

const (
	DbTypeMssql dbi.DbType = "mssql"
)

var _ dbi.DbBackend = (*Backend)(nil)

type Backend struct {
	dbi.BaseBackend
}

// CommitTargetTx mssql驱动存在已知怪癖：连接池复用的conn在特定时序下commit会返回
// "no corresponding begin transaction"，但事务实际已提交；对该怪癖做精确兼容（忽略错误），
// 其余提交失败一律上抛，不得静默吞错
func (mm *Backend) CommitTargetTx(conn *dbi.DbConn, tx *sql.Tx) error {
	if tx == nil {
		return nil
	}
	if err := tx.Commit(); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no corresponding begin transaction") {
			logx.Warnf("mssql target tx commit returned driver quirk error (ignored, data committed): %s", err.Error())
			return nil
		}
		return err
	}
	return nil
}

func (mm *Backend) GetSqlDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) {
	query := url.Values{}
	query.Add("app name", "mayfly")
	query.Add("tlsmin", "1.0")
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

func (mm *Backend) GetDialect(di *dbi.DbInfo) dbi.Dialect {
	return &MssqlDialect{di: di}
}

func (mm *Backend) GetServerInfo(di *dbi.DbInfo) dbi.ServerInfo {
	return &MssqlMetadata{di: di}
}

func (mm *Backend) GetMetadataProvider(di *dbi.DbInfo) dbi.MetadataProvider {
	return &MssqlMetadata{di: di}
}

// varcharType 字符类异构目标列统一落Unicode类型
func varcharType(col *dbi.Column) *dbi.DbDataType {
	if col.CharMaxLength <= 0 || col.CharMaxLength > 4000 {
		col.CharMaxLength = 0
		return NvarcharMax
	}
	return Nvarchar
}

// bytesType 二进制类型统一归一化
func bytesType(col *dbi.Column) *dbi.DbDataType {
	if col.CharMaxLength <= 0 || col.CharMaxLength > 8000 {
		col.CharMaxLength = 0
		return VarbinaryMax
	}
	return Varbinary
}
