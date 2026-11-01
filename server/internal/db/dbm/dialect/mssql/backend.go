package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/pkg/tlsx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"net/url"
	"strings"

	mssql "github.com/microsoft/go-mssqldb"
	"github.com/microsoft/go-mssqldb/msdsn"
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
			Cursor, Rowversion, Hierarchyid, Uniqueidentifier, SQL_variant, Xml, Table,
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
	dbi.DefaultBackend
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

func (mm *Backend) GetSQLDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) {
	query := url.Values{}
	query.Add("app name", "mayfly")
	// TLS 下限取 1.2：1.0/1.1 已被主流合规基线列为不安全协议。
	// 仍允许在「其他连接参数」里显式覆盖（历史实例可能只支持 1.0，需要可见的例外而不是隐式放宽）
	query.Add("tlsmin", "1.2")
	query.Add("connection timeout", "10")
	if d.Database != "" {
		// 库位的 schema 段已由 codec 解析进 d.Schema
		query.Add("database", d.Database)
		if d.Schema != "" {
			query.Add("schema", d.Schema)
		}
	}
	params := query.Encode()
	userParams := d.Params
	tlsCfg := d.EffectiveTLS()
	if tlsCfg != nil {
		// 结构化配置优先：手写的 encrypt/certificate 等会让两处真源互相覆盖
		userParams = tlsx.StripKV(userParams, "encrypt", "trustservercertificate", "certificate",
			"servercertificate", "hostnameincertificate", "tlsmin")
	}
	if userParams != "" {
		if !strings.HasPrefix(userParams, "&") {
			params = params + "&" + userParams
		} else {
			params = params + userParams
		}
	}

	dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%d?%s", url.PathEscape(d.Username), url.PathEscape(d.Password), d.Host, d.Port, params)
	if tlsCfg == nil {
		return sql.Open("mssql", dsn)
	}

	// 启用 TLS 时改走 Connector：go-mssqldb 允许直接注入 *tls.Config，
	// 这样 CA/客户端证书无需落临时文件（DSN 的 certificate 参数只接受文件路径）
	cfg, err := msdsn.Parse(dsn)
	if err != nil {
		return nil, errorx.NewBizf("parse mssql dsn failed: %s", err.Error())
	}
	built, err := tlsCfg.Build()
	if err != nil {
		return nil, err
	}
	cfg.TLSConfig = built
	cfg.Encryption = msdsn.EncryptionRequired
	// require 档只要求加密、不校验证书；verify-full 交出校验权（链 + 主机名）
	cfg.TrustServerCertificate = tlsCfg.Mode == tlsx.ModeRequire
	// ServerName 已由 tlsx 按「原始主机名」填好（隧道场景下与 DSN host 不同），
	// 置 true 阻止驱动用 DSN host 覆盖它
	cfg.HostInCertificateProvided = true
	return sql.OpenDB(mssql.NewConnectorConfig(cfg)), nil
}

func (mm *Backend) GetDialect(di *dbi.DbInfo) dbi.Dialect {
	return &MssqlDialect{di: di}
}

// GetCapabilities SQL Server：核心能力全支持（server>database>schema 三层），并声明已实现的视图与序列内省、
// 会话内省（sys.dm_exec_sessions，MssqlMetadata 实现 SessionInspector）与执行计划分析
// （SET SHOWPLAN_XML，MssqlMetadata 实现 ExplainAnalyzer），三者与声明一致。
func (mm *Backend) GetCapabilities() dbi.MetadataCapabilities {
	// TLS：只声明能如实兑现的档位。verify-ca 未声明——go-mssqldb 一旦提供 CA
	// 就同时校验主机名（SetupTLS 的 certificate 分支），无法只校链
	return dbi.NewAllCapabilities().
		With(dbi.FeatViews, dbi.FeatSequences, dbi.FeatSessions, dbi.FeatExplain, dbi.FeatTLSStatus).
		WithTLSModes(tlsx.ModeRequire, tlsx.ModeVerifyFull).
		WithNamespace(dbi.NamespaceHierarchy{HasDatabase: true, HasSchema: true})
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
