package tokenizer

import "strings"

// TokenType 表示词法单元类型
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenKeyword
	TokenIdentifier
	TokenString
	TokenNumber
	TokenOperator
	TokenPunctuation
)

// Token 是一个词法单元
type Token struct {
	Type  TokenType
	Value string
	Pos   int // 在原始 SQL 中的起始位置
	End   int // 在原始 SQL 中的结束位置（不包含）
}

// IsKeyword 判断当前 token 是否匹配任意给定关键字（大小写不敏感）
func (t Token) IsKeyword(keywords ...string) bool {
	if t.Type != TokenKeyword {
		return false
	}
	upper := strings.ToUpper(t.Value)
	for _, kw := range keywords {
		if upper == strings.ToUpper(kw) {
			return true
		}
	}
	return false
}

// IsEOF 判断是否为 EOF token
func (t Token) IsEOF() bool {
	return t.Type == TokenEOF
}

// Keywords 是 SQL 标准关键字集合
var Keywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "INSERT": true, "INTO": true,
	"UPDATE": true, "SET": true, "DELETE": true, "CREATE": true, "DROP": true,
	"ALTER": true, "TABLE": true, "DATABASE": true, "INDEX": true, "VIEW": true,
	"JOIN": true, "LEFT": true, "RIGHT": true, "INNER": true, "OUTER": true,
	"NATURAL": true, "CROSS": true, "FULL": true, "ON": true, "AS": true,
	"UNION": true, "ALL": true, "DISTINCT": true, "LIMIT": true, "OFFSET": true,
	"ORDER": true, "BY": true, "GROUP": true, "HAVING": true, "WITH": true,
	"SHOW": true, "VALUES": true, "AND": true, "OR": true, "NOT": true,
	"NULL": true, "TRUE": true, "FALSE": true, "DESC": true, "ASC": true,
	"IS": true, "LIKE": true, "IN": true, "BETWEEN": true, "EXISTS": true,
	"CASE": true, "WHEN": true, "THEN": true, "ELSE": true, "END": true,
	"IF": true, "FOR": true, "PRIMARY": true, "KEY": true, "FOREIGN": true,
	"REFERENCES": true, "UNIQUE": true, "DEFAULT": true, "AUTO_INCREMENT": true,
	"COMMENT": true, "ENGINE": true, "CHARSET": true, "COLLATE": true,
	"GRANT": true, "REVOKE": true, "RENAME": true, "ANALYZE": true, "TRUNCATE": true,
	"RETURNING": true, "ONLY": true, "SIMILAR": true, "ESCAPE": true,
	"INTERVAL": true, "CAST": true,
	// ON CONFLICT 语法相关关键字（pgsql/dm等方言）
	"CONFLICT": true, "DO": true, "NOTHING": true, "CONSTRAINT": true,
	// FETCH FIRST/NEXT n ROWS ONLY 分页语法相关关键字（oracle/达梦等方言）
	"FETCH": true, "FIRST": true, "NEXT": true, "ROW": true, "ROWS": true, "TIES": true,
}
