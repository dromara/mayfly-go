package sqldraft

// 消费方窄接口：本包只声明「自己真正用到的那几个方法」。
//
// 为什么不 import 父包 application 的胖接口：父包要 import 本子包做 IOC 注册，
// 反向引用即成环；而 ioc 按结构匹配注入（实现完整接口的类型天然满足窄接口），
// 因此这里声明最小方法集即可，无需任何显式映射。

import (
	"context"

	"mayfly-go/internal/db/dbm/dbi"
)

// metadataReader 元数据读能力切片（方言 MetadataReader 天然满足该接口）。
//
// 只声明本包真正读的五个方法：这既让方言实现无需为 SQL 草稿做任何改动，
// 也让管线可以在没有真实数据库的情况下用替身跑完整流程（离线契约测试的前提）。
type metadataReader interface {
	GetTables(tableNames ...string) ([]dbi.Table, error)
	GetColumns(tableNames ...string) ([]dbi.Column, error)
	GetDbServer() (*dbi.DbServer, error)
	GetForeignKeys(ctx context.Context, schema, table string) ([]dbi.ForeignKey, error)
	Explain(ctx context.Context, sql string) (*dbi.ExplainResult, error)
}

// DbConnReader 取数据库连接（连接层统一做资源可达性鉴权）
type DbConnReader interface {
	GetDbConn(ctx context.Context, dbId uint64, dbName string) (*dbi.DbConn, error)
}

// SqlGate 执行闸门：只声明草稿展示真正用到的两个能力
type SqlGate interface {
	// CheckSqlsWithoutTicket 判定这批语句能否在「没有工单通道」时执行，返回提醒文案
	CheckSqlsWithoutTicket(ctx context.Context, conn *dbi.DbConn, sqls []string) ([]string, error)
	// PolicyBlockKind 把上面返回的错误归类为 "approval" / "warn_ack" / ""（无法判定）。
	// 由 sqlexec 实现（它已依赖 flow 的阻断错误码）；本包不 import flow，避免成环
	PolicyBlockKind(err error) string
}

// ColumnValueSampler 列取值采样（由 db 应用层实现：它自身已过表级/列级权限断言，
// 且限行走方言分页改写能力）。草稿侧只按上限取少量去重值用于消歧枚举语义。
type ColumnValueSampler interface {
	ColumnValues(ctx context.Context, dbId uint64, dbName, table, column string, limit int) ([]string, error)
}

// SavedSqlExamples 同库已保存脚本（作为写法示例的自有语料）
type SavedSqlExamples interface {
	Examples(ctx context.Context, dbId uint64, limit int) ([]string, error)
}
