package sqlexec

import (
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
)

// 变更采集回查所用标识符的「目录归一」。
//
// 解析器交回的是剥掉引用符后的用户书写形态。折叠大小写的方言（PostgreSQL 把裸写的 UserName 折叠成
// username，目录里只有小写名）下，把书写形态直接交给 QuoteIdent 就是引用一个不存在的对象：
// 回查 SELECT 报错 → 旧值与执行后快照静默采空 → 这条 UPDATE/DELETE 的回滚能力整体消失，
// 而语句本身照常执行成功（失效是静默的，只有把数据读回来才发现）。反过来完全不引用，
// 又会让保留字列（order）与真正大小写敏感的对象查不到。
//
// 唯一正确的口径是引用「目录真名」：先按元数据把书写形态归一，再引用归一后的名字。
// 归一不到（元数据不可用、跨 schema、对象确实不存在）时退回裸名，交给服务端按自身折叠规则解析——
// 引用一个猜出来的名字必然报错，裸名至多与引用改造前同行为。
// dbi.Quoter.QuoteIdent 的契约本就写着「引用来自元数据的真实标识符」，此处即把该契约落实。

// catalogNames 目录真名索引：小写标识符 -> 元数据里的实际拼写
type catalogNames map[string]string

func newCatalogNames[T any](items []T, nameOf func(T) string) catalogNames {
	out := make(catalogNames, len(items))
	for _, item := range items {
		if name := nameOf(item); name != "" {
			out[strings.ToLower(name)] = name
		}
	}
	return out
}

// newColumnCatalog 由一次 GetColumns 结果同时得到列索引与该表的目录真名（免再查一次表元数据）
func newColumnCatalog(columns []dbi.Column) (catalogNames, string) {
	tableName := ""
	for _, column := range columns {
		if tableName == "" && column.TableName != "" {
			tableName = column.TableName
		}
	}
	return newCatalogNames(columns, func(c dbi.Column) string { return c.ColumnName }), tableName
}

// resolve 把书写形态归一到目录真名；索引里没有时第二个返回值为 false
func (c catalogNames) resolve(name string) (string, bool) {
	bare := lastSegment(name)
	if bare == "" {
		return "", false
	}
	if real, ok := c[strings.ToLower(bare)]; ok {
		return real, true
	}
	return "", false
}

// identQuoter 采集回查的标识符引用器：把方言 quoter 与目录索引放在一起，避免各处口径漂移
type identQuoter struct {
	quote      func(string) string
	isReserved func(string) bool
}

func newIdentQuoter(conn *dbi.DbConn) *identQuoter {
	quoter := conn.GetDialect().Quoter()
	return &identQuoter{quote: quoter.QuoteIdent, isReserved: quoter.IsReserved}
}

// catalogIdent 引用一个「目录里应当存在」的标识符（表名/列名）：归一到目录真名后引用，
// 归一不到则输出裸名（折叠语义交给服务端，与逐字回放的 WHERE 条件保持一致）
func (i *identQuoter) catalogIdent(catalog catalogNames, name string) string {
	if real, ok := catalog.resolve(name); ok {
		return i.quote(real)
	}
	return lastSegment(name)
}

// tableIdent 引用表名：元数据给出的拼写才是事实；realName 为空表示元数据没返回该表，
// 退回书写形态裸名，与逐字回放的 WHERE 条件按同样折叠规则解析
func (i *identQuoter) tableIdent(realName, writtenName string) string {
	if realName != "" {
		return i.quote(realName)
	}
	return lastSegment(writtenName)
}

// aliasIdent 引用语句内的表别名。
//
// 别名不属于目录对象，只能按「该方言是否把它列为保留字」决定要不要引用：
// 是保留字则原文必然带了引用符（否则语句自己就语法错），按同款引用回去；
// 非保留字则裸名输出，使它与逐字回放的 WHERE 条件里的同一别名按相同折叠规则解析
// （引用书写形态会让 FROM 里的别名与条件里的别名不再是同一个名字）
func (i *identQuoter) aliasIdent(alias string) string {
	bare := lastSegment(alias)
	if bare == "" {
		return ""
	}
	if i.isReserved(bare) {
		return i.quote(bare)
	}
	return bare
}
