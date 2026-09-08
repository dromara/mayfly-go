package sqlparser

import (
	"io"

	"mayfly-go/internal/pkg/utils"
)

// SQLSplitter SQL 切割器接口
// 不同方言有不同的切割规则，例如：
//   - WITH CTE 语句不能被分号切割
//   - PostgreSQL 的 DO $$ ... $$ 块
//   - MySQL 的 DELIMITER 命令
//   - 存储过程/函数中的分号
type SQLSplitter interface {
	// SplitSQL 切割 SQL 语句
	//  - r: 读取器
	//  - callback: 回调函数，每条完整的 SQL 语句调用一次
	SplitSQL(r io.Reader, callback utils.StmtCallback) error
}

// DefaultSplitter 默认 SQL 切割器
// 默认采用mysql语义（反斜杠为转义符），标准SQL方言请使用 NewStdSQLSplitter
type DefaultSplitter struct {
	delimiter rune
	opts      utils.SplitOpts
}

// NewDefaultSplitter 创建默认切割器（mysql语义：反斜杠为字符串内转义符，反引号为标识符引用符）
func NewDefaultSplitter(delimiter ...rune) *DefaultSplitter {
	delim := rune(';')
	if len(delimiter) > 0 {
		delim = delimiter[0]
	}
	return &DefaultSplitter{delimiter: delim, opts: utils.SplitOpts{BackslashEscape: true, BacktickQuote: true}}
}

// NewStdSQLSplitter 创建标准SQL方言切割器（sqlite/postgres/mssql/oracle等）
// 反斜杠为普通字符，形如 '\' 的完整字符串不会导致后续语句被误切
func NewStdSQLSplitter(delimiter ...rune) *DefaultSplitter {
	delim := rune(';')
	if len(delimiter) > 0 {
		delim = delimiter[0]
	}
	return &DefaultSplitter{delimiter: delim, opts: utils.SplitOpts{BackslashEscape: false}}
}

// NewMysqlSplitter 创建mysql切割器：反斜杠转义 + # 行注释 + 反引号标识符
func NewMysqlSplitter() *DefaultSplitter {
	return &DefaultSplitter{delimiter: ';', opts: utils.SplitOpts{BackslashEscape: true, HashComment: true, BacktickQuote: true}}
}

// SplitSQL 实现 SQLSplitter 接口
func (s *DefaultSplitter) SplitSQL(r io.Reader, callback utils.StmtCallback) error {
	return utils.SplitStmtsWithOpts(r, s.delimiter, s.opts, callback)
}
