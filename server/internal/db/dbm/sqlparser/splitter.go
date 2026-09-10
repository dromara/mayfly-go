package sqlparser

import (
	"bufio"
	"context"
	"errors"
	"io"

	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/internal/pkg/utils"
	"mayfly-go/pkg/errorx"
)

// SQLSplitter SQL 切割器接口
//
// 各方言的切割规则差异（字符串与标识符引用符、注释风格、复合语句块）统一由
// tokenizer.DialectConfig 能力表描述，具体实现见 DialectSplitter。
type SQLSplitter interface {
	// SplitSQL 切割 SQL 语句
	//  - r: 读取器
	//  - callback: 回调函数，每条完整的 SQL 语句调用一次
	SplitSQL(r io.Reader, callback utils.StmtCallback) error

	// LeadingKeyword 返回语句的首个关键字（小写，已跳过前导空白与注释），
	// 供解析失败时判定语句类型使用（切割保留注释原文后，不能再按前缀字符匹配）
	LeadingKeyword(sql string) string

	// MaskComments 将文本中的普通注释替换为等长空白（字面量保留原文，可执行注释解壳保留内容），
	// 供「按语句整体文本判定语句类型」使用（如导入侧识别 dump 事务包装语句）
	MaskComments(sql string) string
}

// DialectSplitter 基于方言能力表的统一 SQL 切割器（替代此前各自为政的默认/std/pgsql 三套实现）
//
// 语义与词法器共用 tokenizer.DialectConfig，各方言只需注册一次能力位：
//   - 注释与字面量原文保留（Oracle /*+ hint */、MySQL /*!40101 ... */ 等可执行注释不得丢弃）
//   - BEGIN..END / CASE..END 等复合块内的分号不切割（档位由能力表 BlockMode 决定）
//   - 未闭合的引号/注释返回 *tokenizer.UnterminatedError（可用 SplitError 转为国际化业务错误），不再静默吞并后续脚本
//
// 已知限制：不识别客户端指令型分隔符（mysql 的 `DELIMITER xx` 与 sqlcmd/SSMS 的 `GO` 批处理分隔符），
// 分隔符由调用方通过 NewSplitter 显式指定；脚本内出现的 `DELIMITER` 行会被当作普通语句交给服务端执行。
type DialectSplitter struct {
	cfg       tokenizer.DialectConfig
	delimiter rune
}

var _ SQLSplitter = (*DialectSplitter)(nil)

// NewSplitter 创建使用指定方言能力表的切割器，delimiter 为语句分隔符（默认 ';'）
func NewSplitter(cfg tokenizer.DialectConfig, delimiter ...rune) *DialectSplitter {
	delim := rune(';')
	if len(delimiter) > 0 {
		delim = delimiter[0]
	}
	return &DialectSplitter{cfg: cfg, delimiter: delim}
}

// SplitSQL 实现 SQLSplitter 接口：按行流式读取（支持 GB 级导入脚本），逐条回调完整语句
func (s *DialectSplitter) SplitSQL(r io.Reader, callback utils.StmtCallback) error {
	reader := bufio.NewReaderSize(r, 512*1024)
	scanner := tokenizer.NewStatementScanner(s.cfg, s.delimiter)
	for {
		line, readErr := reader.ReadString('\n')
		if line != "" {
			stmts, scanErr := scanner.Feed(line)
			if err := callbackStmts(stmts, scanErr, callback); err != nil {
				return err
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	stmts, scanErr := scanner.Finish()
	return callbackStmts(stmts, scanErr, callback)
}

// LeadingKeyword 返回该方言下的语句首个关键字
func (s *DialectSplitter) LeadingKeyword(sql string) string {
	return tokenizer.LeadingKeyword(sql, s.cfg)
}

// MaskComments 按本方言语义掩码注释（等长空白，保留换行与字面量原文，可执行注释解壳）
func (s *DialectSplitter) MaskComments(sql string) string {
	return tokenizer.MaskSqlComments(sql, s.cfg)
}

// callbackStmts 逐条回调已切割出的语句，并优先透传扫描错误
func callbackStmts(stmts []string, scanErr error, callback utils.StmtCallback) error {
	if scanErr != nil {
		return scanErr
	}
	for _, stmt := range stmts {
		if err := callback(stmt); err != nil {
			return err
		}
	}
	return nil
}

// SplitSQLText 切割完整 SQL 文本为语句列表（同 SplitSQL，入参为字符串），
// 未闭合的引号/注释返回 *tokenizer.UnterminatedError
func SplitSQLText(sql string, cfg tokenizer.DialectConfig, delimiter ...rune) ([]string, error) {
	delim := rune(';')
	if len(delimiter) > 0 {
		delim = delimiter[0]
	}
	return tokenizer.SplitStatements(sql, cfg, delim)
}

// SplitError 归一化语句切割错误：未闭合的引号/注释（*tokenizer.UnterminatedError）转为带行号定位的国际化业务错误；
// 其余错误（如读取上传文件失败、回调执行失败）原样返回，不改变其类型与错误码
func SplitError(ctx context.Context, err error) error {
	var ue *tokenizer.UnterminatedError
	if errors.As(err, &ue) {
		return errorx.NewBizI(ctx, imsg.ErrSqlSplitUnterminated, "kind", ue.Kind, "line", ue.Line)
	}
	return err
}
