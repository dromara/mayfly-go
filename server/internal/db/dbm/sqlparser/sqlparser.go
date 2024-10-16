package sqlparser

import (
	"bufio"
	"io"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"regexp"
)

type DbDialect string

// const (
// 	mysql DbDialect = "mysql"
// 	pgsql DbDialect = "pgsql"
// )

type SqlParser interface {

	// sql解析
	// @param stmt sql语句
	Parse(stmt string) ([]sqlstmt.Stmt, error)
}

// var (
// 	parsers = make(map[string]SqlParser)
// )

// // 注册数据库类型与dbmeta
// func Register(dialect string, parser SqlParser) {
// 	parsers[dialect] = parser
// }

// func getParser(dialect string) (SqlParser, error) {
// 	parser, ok := parsers[dialect]
// 	if !ok {
// 		return nil, errors.New("不存在该parser")
// 	}
// 	return parser, nil
// }

// 解析sql
// @param dialect 方言
// @param stmt sql语句
// func Parse(dialect string, stmt string) ([]sqlstmt.Stmt, error) {
// 	if parser, err := getParser(dialect); err != nil {
// 		return nil, err
// 	} else {
// 		return parser.Parse(stmt)
// 	}
// }

var sqlSplitRegexp = regexp.MustCompile(`\s*;\s*\n`)

// SplitSqls 根据;\n切割sql
func SplitSqls(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)

	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, io.EOF
		}

		match := sqlSplitRegexp.FindIndex(data)

		if match != nil {
			// 如果找到了";\n"，判断是否为最后一行
			if match[1] == len(data) {
				// 如果是最后一行，则返回完整的切片
				return len(data), data, nil
			}
			// 否则，返回到";\n"之后，并且包括";\n"本身
			return match[1], data[:match[1]], nil
		}

		if atEOF {
			return len(data), data, nil
		}

		return 0, nil, nil
	})

	return scanner
}
