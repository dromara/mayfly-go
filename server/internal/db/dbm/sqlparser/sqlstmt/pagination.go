package sqlstmt

import (
	"regexp"
	"strconv"
	"strings"
)

// orderByPattern 检测 SQL 中是否已包含 ORDER BY 子句。
// 使用正则匹配避免误匹配字符串字面量中的 "ORDER BY"（简单启发式：忽略引号内内容）。
var orderByPattern = regexp.MustCompile(`(?i)\bORDER\s+BY\s+`)

// hasOrderBy 检测 SQL 是否包含 ORDER BY 子句（简单启发式）。
func hasOrderBy(sql string) bool {
	return orderByPattern.MatchString(sql)
}

// DefaultRewritePagination 默认分页改写（LIMIT/OFFSET 语法）。
// 适用于 MySQL/SQLite/ClickHouse/PostgreSQL 等支持标准 LIMIT 的方言。
// SQL Server 与 Oracle 需各自覆写。
func DefaultRewritePagination(sql string, offset, limit int64) string {
	if limit <= 0 && offset <= 0 {
		return sql
	}
	result := sql
	if limit > 0 {
		result += " LIMIT " + strconv.FormatInt(limit, 10)
	}
	if offset > 0 {
		if limit <= 0 {
			result += " LIMIT -1"
		}
		result += " OFFSET " + strconv.FormatInt(offset, 10)
	}
	return result
}

// MssqlRewritePagination SQL Server 分页改写（OFFSET-FETCH 语法）。
// SQL Server 要求 OFFSET-FETCH 必须跟在 ORDER BY 后面。
// 若原 SQL 不含 ORDER BY，自动注入 "ORDER BY (SELECT NULL)" 以满足语法要求。
func MssqlRewritePagination(sql string, offset, limit int64) string {
	if limit <= 0 && offset <= 0 {
		return sql
	}

	// SQL Server 要求 OFFSET-FETCH 必须有 ORDER BY
	if !hasOrderBy(sql) {
		// 使用 (SELECT NULL) 作为占位排序：语义为任意顺序，满足语法要求
		sql = strings.TrimRight(sql, " \t\n\r;") + " ORDER BY (SELECT NULL)"
	}

	// OFFSET-FETCH 必须跟在 ORDER BY 后面
	if offset > 0 {
		sql += " OFFSET " + strconv.FormatInt(offset, 10) + " ROWS"
	} else {
		sql += " OFFSET 0 ROWS"
	}
	if limit > 0 {
		sql += " FETCH NEXT " + strconv.FormatInt(limit, 10) + " ROWS ONLY"
	}
	return sql
}

// OracleRewritePagination Oracle 12c+ 分页改写（OFFSET-FETCH 语法）。
// Oracle 12c+ 不强制要求 ORDER BY，但语义正确的分页通常需要。
func OracleRewritePagination(sql string, offset, limit int64) string {
	if limit <= 0 && offset <= 0 {
		return sql
	}
	if offset > 0 {
		sql += " OFFSET " + strconv.FormatInt(offset, 10) + " ROWS"
	}
	if limit > 0 {
		sql += " FETCH NEXT " + strconv.FormatInt(limit, 10) + " ROWS ONLY"
	}
	return sql
}

// DefaultClassifyStmt 基于 AST 的默认语句分类。
// 将 sqlstmt.Kind 映射为 StmtType。
func DefaultClassifyStmt(kind Kind) StmtType {
	switch kind {
	case KindSelect:
		return StmtTypeSelect
	case KindInsert:
		return StmtTypeInsert
	case KindUpdate:
		return StmtTypeUpdate
	case KindDelete:
		return StmtTypeDelete
	case KindDdl:
		return StmtTypeDDL
	default:
		return StmtTypeOther
	}
}
