package dbi

import "strings"

// 库位 locator codec：`db/schema` 编码串在整个后端的唯一 parse/format 入口。
//
// 带 schema 层的方言（postgres/oracle/mssql/dm）把库位编码成 `库名/schema` 塞进连接定位串；
// 无 schema 层的方言（mysql/sqlite/clickhouse）库名本身可能含 `/`，绝不可拆。
// 因此 parse 必须方言感知：按 NamespaceHierarchy.HasSchema 决定 `/` 是否为 schema 分隔符。
// format 与方言无关：schema 非空才拼 `/`，否则就是纯库名。二者构成全段数 round-trip 保真：
//
//	FormatLocator(ParseLocator(dt, s)) == s      （对任意 dt 与 locator s）
//	ParseLocator(dt, FormatLocator(db, "")) == (db, "")

// ParseLocator 按方言类型解析库位 locator 为 (纯库名, schema)。
//
// 用 GetBackend(dt) 而非 DbInfo.Backend 取能力：parse 发生在构造 DbInfo 阶段（ToDbInfo），
// 此刻 Backend 尚未注入（它在 Conn() 建连时才赋值），而 Type 已由实例拷贝填好。
// 未注册的方言类型（GetBackend 返回 nil）保守视为无 schema 层，整串即库名。
func ParseLocator(dt DbType, s string) (db, schema string) {
	if s == "" {
		return "", ""
	}
	backend := GetBackend(dt)
	if backend == nil || !backend.GetCapabilities().NamespaceHierarchy.HasSchema {
		// 无独立 schema 层：库名本身含 `/` 也不拆，整串即库名
		return s, ""
	}
	// 有 schema 层：首个 `/` 前为库名，其后全部为 schema（保证任意段数 round-trip 保真）
	if i := strings.Index(s, "/"); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

// FormatLocator 将 (纯库名, schema) 合成为库位 locator：schema 非空则 `db/schema`，否则 `db`。
// 与方言无关（无 schema 层的方言其 schema 恒为空）。是 ParseLocator 的逆运算。
func FormatLocator(db, schema string) string {
	if schema == "" {
		return db
	}
	return db + "/" + schema
}
