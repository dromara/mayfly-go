package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/pkg/tlsx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/collx"
	"net"
	"strconv"
	"strings"
	"time"

	pq "gitee.com/liuzongyang/libpq"
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
			Uuid,
		)

		// pg 元数据里数组列的 typname 是 `_` + 元素类型名（bigint[] → _int8），逐元素类型登记既
		// 是第二套真相也必然漏项；声明前缀后，凡元素类型已注册的数组都能按 TCArray 解析
		b.RegisterArrayType("_", dbi.NewDbDataType("array", dbi.DTString).WithCategory(dbi.TCArray).WithFixColumn(dbi.ClearCharLength))

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

		// UUID：pg 有原生 uuid 类型，异构源（如 clickhouse UUID、mssql uniqueidentifier）转入时
		// 落回 uuid 而非 varchar，避免主键/索引列退化成宽字符串
		b.RegisterRules(Uuid, dbi.TCUUID)
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
	dbi.DefaultBackend
	Param string
}

func (pm *Backend) GetSQLDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) {
	db := d.Database
	var dbParam string
	// 库位的 schema 段已在 ToDbInfo 阶段由 codec 解析进 d.Schema，此处不再拆串
	existSchema := d.Schema != ""
	if existSchema {
		dbParam = fmt.Sprintf("dbname=%s search_path=%s", escapeLibpqParam(db), escapeLibpqParam(d.Schema))
	} else {
		dbParam = "dbname=" + escapeLibpqParam(db)
	}

	// host/port 在 libpq 里同时承担两个职责：拨号目标 + TLS 校验用的主机名
	// （sslmode=verify-full 时 ServerName 直接取 host）。经中转通道时 di.Host/Port
	// 已被改写成通道本地地址，若照抄进 DSN，verify-full 会拿 127.0.0.1 去校验证书，
	// 必然失败。因此 DSN 保留原始主机名，拨号改由 fixedDialer 指向通道本地地址。
	host, port := d.Host, d.Port
	dialAddr := ""
	if d.RemoteAddr != "" {
		dialAddr = fmt.Sprintf("%s:%d", d.Host, d.Port)
		if h, p, splitErr := net.SplitHostPort(d.RemoteAddr); splitErr == nil {
			host = h
			if parsedPort, convErr := strconv.Atoi(p); convErr == nil {
				port = parsedPort
			}
		}
	}

	// 显式配置的 ServerName 必须真的生效：libpq 在 sslmode=verify-full 时把 DSN 的 host
	// 直接当校验主机名，没有独立的 servername 参数。因此把 host 换成待校验的名字，
	// 拨号仍走真实地址（自定义 Dialer）——否则界面上的「主机名」输入框是个摆设
	if tlsCfgForName := d.EffectiveTLS(); tlsCfgForName != nil && tlsCfgForName.ServerName != "" && tlsCfgForName.ServerName != host {
		if dialAddr == "" {
			dialAddr = fmt.Sprintf("%s:%d", d.Host, d.Port)
		}
		host = tlsCfgForName.ServerName
		port = d.Port
	}

	// 默认明文：libpq 不指定 sslmode 时是 prefer（会尝试加密再回落），
	// 与「未配置即明文」的显式语义不一致，故恒显式下发
	sslmode := "disable"
	params := d.Params
	var tlsParts []string
	if tlsCfg := d.EffectiveTLS(); tlsCfg != nil {
		sslmode = string(tlsCfg.Mode)
		files, err := d.PrepareTLSFiles()
		if err != nil {
			return nil, err
		}
		// require 档刻意不传 sslrootcert：libpq 在 require 下若发现根证书文件存在，
		// 会把行为升级为 verify-ca（见其 ssl.go 的向后兼容分支），
		// 那等于把「只要加密」偷偷变成「必须校验链」
		if files != nil {
			if tlsCfg.Mode != tlsx.ModeRequire && files.CAPath != "" {
				tlsParts = append(tlsParts, "sslrootcert="+escapeLibpqParam(files.CAPath))
			}
			if files.ClientCertPath != "" {
				tlsParts = append(tlsParts, "sslcert="+escapeLibpqParam(files.ClientCertPath),
					"sslkey="+escapeLibpqParam(files.ClientKeyPath))
			}
		}
		// 结构化配置优先：剔除用户手写的 ssl* 参数，避免两处真源
		params = tlsx.StripKV(params, "sslmode", "sslrootcert", "sslcert", "sslkey", "sslinline")
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s %s connect_timeout=8 sslmode=%s",
		escapeLibpqParam(host), port, escapeLibpqParam(d.Username), escapeLibpqParam(d.Password), dbParam, sslmode)
	if len(tlsParts) > 0 {
		dsn = dsn + " " + strings.Join(tlsParts, " ")
	}

	// 存在额外指定参数，则拼接该连接参数
	if params != "" {
		// 存在指定的db，则需要将dbInstance配置中的parmas排除掉dbname和search_path
		if db != "" {
			paramArr := strings.Split(params, "&")
			paramArr = collx.ArrayRemoveFunc(paramArr, func(param string) bool {
				if strings.HasPrefix(param, "dbname=") {
					return true
				}
				if existSchema && strings.HasPrefix(param, "search_path") {
					return true
				}
				return false
			})
			params = strings.Join(paramArr, " ")
		}
		dsn = fmt.Sprintf("%s %s", dsn, strings.Join(strings.Split(params, "&"), " "))
	}

	if pm.Param != "" && !strings.Contains(dsn, "dbtype") {
		dsn = fmt.Sprintf("%s %s", dsn, pm.Param)
	}

	if dialAddr == "" {
		return sql.Open("postgres", dsn)
	}
	connector, err := pq.NewConnector(dsn)
	if err != nil {
		return nil, errorx.NewBizf("build postgres connector failed: %s", err.Error())
	}
	connector.Dialer(fixedDialer{addr: dialAddr})
	return sql.OpenDB(connector), nil
}

// fixedDialer 把 libpq 的拨号目标固定为通道本地地址。
//
// 为什么需要它而不是直接把本地地址写进 DSN：DSN 的 host 同时是 TLS 校验主机名，
// 两者必须拆开才能让「经 SSH 隧道 + verify-full」成立。
type fixedDialer struct{ addr string }

func (d fixedDialer) Dial(network, _ string) (net.Conn, error) {
	return net.Dial(network, d.addr)
}

func (d fixedDialer) DialTimeout(network, _ string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(network, d.addr, timeout)
}

// GetCapabilities pg 系方言：核心能力全支持（含独立 schema 层），并声明已实现的视图、序列与会话内省。
// 视图/序列/会话查询都基于 pg 9.x 通用系统表（pg_stat_activity 自 9.2 起即有 state/query 列），
// 对 gauss/kingbase/vastbase 亦适用。
// TLS：libpq 原生支持 sslmode 四档，证书以文件路径下发（由 tlsx 落临时目录）
func (pm *Backend) GetCapabilities() dbi.MetadataCapabilities {
	return dbi.NewAllCapabilities().
		With(dbi.FeatViews, dbi.FeatSequences, dbi.FeatSessions, dbi.FeatExplain, dbi.FeatTLSStatus, dbi.FeatSlowQuery, dbi.FeatLockWait, dbi.FeatRecovery, dbi.FeatAccounts).
		WithTLSModes(tlsx.ModeRequire, tlsx.ModeVerifyCA, tlsx.ModeVerifyFull).
		WithNamespace(dbi.NamespaceHierarchy{HasDatabase: true, HasSchema: true})
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
