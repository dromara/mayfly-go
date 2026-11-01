package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/pkg/tlsx"
	"mayfly-go/pkg/logx"
	"strings"

	go_ora "github.com/sijms/go-ora/v2"
	"github.com/spf13/cast"
)

func init() {
	dbi.RegisterBackend(DbTypeOracle, new(Backend))

	// 类型引擎注册（迁移/同步类型系统，与 Backend 解耦）
	dbi.RegisterTypeEngine(DbTypeOracle, func(b *dbi.TypeEngineBuilder) {
		b.RegisterTypes(
			CHAR, NCHAR, VARCHAR2, NVARCHAR2, TEXT, LONG, LONGVARCHAR, IMAGE,
			LONGVARBINARY, CLOB, BLOB,
			DECIMAL, NUMBER, INTEGER, INT, BIGINT, TINYINT, BYTE, SMALLINT, BIT, DOUBLE, FLOAT,
			TIME, DATE, TIMESTAMP,
		)

		// 字符串类
		b.RegisterRule(dbi.TCVarchar, func(col *dbi.Column) *dbi.DbDataType {
			if col.CharMaxLength > 4000 {
				col.CharMaxLength = 0
				return CLOB
			}
			return VARCHAR2
		})
		b.RegisterRule(dbi.TCChar, func(col *dbi.Column) *dbi.DbDataType { return CHAR })
		// lobType: 大文本统一归 CLOB
		lobRule := func(col *dbi.Column) *dbi.DbDataType {
			col.CharMaxLength = 0
			return CLOB
		}
		b.RegisterRule(dbi.TCText, lobRule)
		b.RegisterRule(dbi.TCMediumtext, lobRule)
		b.RegisterRule(dbi.TCLongtext, lobRule)

		// 整数/布尔/位类
		b.RegisterRules(BIT, dbi.TCBit, dbi.TCBool)
		b.RegisterRules(TINYINT, dbi.TCInt1)
		b.RegisterRules(SMALLINT, dbi.TCInt2)
		b.RegisterRules(INTEGER, dbi.TCInt4)
		b.RegisterRules(BIGINT, dbi.TCInt8)

		// 数值类
		b.RegisterRule(dbi.TCNumeric, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClearNumPrecision(col)
			return NUMBER
		})
		b.RegisterRule(dbi.TCDecimal, func(col *dbi.Column) *dbi.DbDataType {
			if col.NumPrecision <= 0 {
				dbi.ClearNumPrecision(col)
				return NUMBER
			}
			dbi.ClampDecimalPrecision(col, 38, 38)
			return DECIMAL
		})

		// 无符号整数类
		b.RegisterRules(BIGINT, dbi.TCUnsignedInt8)
		b.RegisterRules(INT, dbi.TCUnsignedInt4, dbi.TCUnsignedInt2, dbi.TCUnsignedInt1)

		// 日期时间类
		b.RegisterRule(dbi.TCDate, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClearNumPrecision(col)
			return DATE
		})
		b.RegisterRule(dbi.TCTime, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClearNumPrecision(col)
			return TIME
		})
		b.RegisterRule(dbi.TCDateTime, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClampTimeFsp(col, 9)
			return TIMESTAMP
		})
		b.RegisterRule(dbi.TCTimestamp, func(col *dbi.Column) *dbi.DbDataType {
			dbi.ClampTimeFsp(col, 9)
			return TIMESTAMP
		})

		// 二进制类
		b.RegisterRules(BLOB, dbi.TCBinary, dbi.TCVarbinary, dbi.TCMediumblob, dbi.TCBlob, dbi.TCLongblob)

		// Enum/JSON
		b.RegisterRules(NVARCHAR2, dbi.TCEnum, dbi.TCJSON)
	})
}

const (
	DbVersionOracle11 dbi.DbVersion = "11"
	DbTypeOracle      dbi.DbType    = "oracle"
)

var _ dbi.DbBackend = (*Backend)(nil)

type Backend struct {
	dbi.DefaultBackend
}

func (om *Backend) GetSQLDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) {
	urlOptions := make(map[string]string)

	// 库位的 schema 段已在 ToDbInfo 阶段由 codec 解析进 d.Schema（无 schema 层时恒为空）
	schema := d.Schema
	tlsCfg := d.EffectiveTLS()
	if d.Params != "" {
		paramArr := strings.Split(d.Params, "&")
		for _, param := range paramArr {
			ps := strings.Split(param, "=")
			if len(ps) > 1 {
				// 结构化 TLS 生效时忽略手写的 ssl 键：否则同一条连接上的加密开关有两处真源
				if tlsCfg != nil && isOracleTLSKey(ps[0]) {
					continue
				}
				if ps[0] == "clientCharset" {
					urlOptions["client charset"] = ps[1]
				} else {
					urlOptions[ps[0]] = ps[1]
				}
			}
		}
	}

	if tlsCfg != nil {
		// go-ora 的 ServerName 固定取拨号地址（network/session.go），隧道场景下会是 127.0.0.1，
		// 因此本方言只声明 require（加密不校验证书）；证书校验需 wallet/服务端 DN，属后续能力
		urlOptions["SSL"] = "true"
		urlOptions["SSL VERIFY"] = "false"
	}

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
	if schema != "" {
		_, err := conn.Exec(fmt.Sprintf("ALTER SESSION SET CURRENT_SCHEMA=%s", schema))
		if err != nil {
			return nil, err
		}
	}

	return conn, err
}

// GetCapabilities oracle：核心能力全支持（含独立 schema 层），并声明已实现的视图与序列内省。
// OracleMetadata11 内嵌 OracleMetadata 继承其 MetadataNavigator 实现，故 11/12 两版本声明均真实。
func (om *Backend) GetCapabilities() dbi.MetadataCapabilities {
	// TLS：只声明 require。go-ora 的证书校验主机名固定取拨号地址，
	// 无法与 SSH 隧道共存，也没有自定义 CA 入口（校验走系统根 + 服务端 DN）
	return dbi.NewAllCapabilities().
		With(dbi.FeatViews, dbi.FeatSequences, dbi.FeatSessions, dbi.FeatExplain).
		WithTLSModes(tlsx.ModeRequire).
		WithNamespace(dbi.NamespaceHierarchy{HasDatabase: true, HasSchema: true})
}

// isOracleTLSKey 手写参数中与结构化 TLS 冲突的键（go-ora 解析时大小写不敏感）
func isOracleTLSKey(key string) bool {
	switch strings.ToUpper(strings.TrimSpace(key)) {
	case "SSL", "SSL VERIFY", "SSL SERVER DN", "WALLET":
		return true
	}
	return false
}

func (om *Backend) GetDialect(di *dbi.DbInfo) dbi.Dialect {
	return &OracleDialect{di: di}
}

func (om *Backend) GetServerInfo(di *dbi.DbInfo) dbi.ServerInfo {
	return om.buildMetadata(di)
}

func (om *Backend) GetMetadataProvider(di *dbi.DbInfo) dbi.MetadataProvider {
	return om.buildMetadata(di)
}

// oracleMetadataProvider 同时满足 ServerInfo + MetadataProvider 的复合接口
type oracleMetadataProvider interface {
	dbi.ServerInfo
	dbi.MetadataProvider
}

// buildMetadata 内部统一构建元数据实例（含 Oracle 版本检测逻辑）
func (om *Backend) buildMetadata(di *dbi.DbInfo) oracleMetadataProvider {
	if di.Version == "" && !di.DefaultVersion {
		if di.GetDb() != nil {
			_, res, err := di.Query("select VERSION from v$instance")
			if err != nil {
				logx.Errorf("failed to query oracle version, use default version: %s", err.Error())
				di.DefaultVersion = true
			} else if len(res) > 0 {
				version := cast.ToString(res[0]["VERSION"])
				if strings.HasPrefix(version, "11") {
					di.Version = DbVersionOracle11
					di.DefaultVersion = false
				} else {
					di.DefaultVersion = true
				}
			} else {
				di.DefaultVersion = true
			}
		}
	}

	if di.Version == DbVersionOracle11 {
		md := &OracleMetadata11{}
		md.di = di
		md.version = DbVersionOracle11
		return md
	}
	return &OracleMetadata{di: di}
}
