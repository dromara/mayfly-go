package sqlparser

import (
	"bufio"
	"github.com/kanzihuang/vitess/go/vt/sqlparser"
	"io"
	"regexp"
)

type Parser interface {
	Next() error
	Current() string
}

var _ Parser = &MysqlParser{}
var _ Parser = &PostgresParser{}

type MysqlParser struct {
	tokenizer *sqlparser.Tokenizer
	statement string
}

func NewMysqlParser(reader io.Reader) *MysqlParser {
	return &MysqlParser{
		tokenizer: sqlparser.NewReaderTokenizer(reader),
	}
}

func (parser *MysqlParser) Next() error {
	statement, err := sqlparser.ParseNext(parser.tokenizer)
	if err != nil {
		parser.statement = ""
		return err
	}
	parser.statement = sqlparser.String(statement)
	return nil
}

func (parser *MysqlParser) Current() string {
	return parser.statement
}

type PostgresParser struct {
	scanner   *bufio.Scanner
	statement string
}

func NewPostgresParser(reader io.Reader) *PostgresParser {
	return &PostgresParser{
		scanner: splitSqls(reader),
	}
}

func (parser *PostgresParser) Next() error {
	if !parser.scanner.Scan() {
		return io.EOF
	}
	return nil
}

func (parser *PostgresParser) Current() string {
	return parser.scanner.Text()
}

// 根据;\n切割sql
func splitSqls(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	re := regexp.MustCompile(`\s*;\s*\n`)

	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, io.EOF
		}

		match := re.FindIndex(data)

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

func SplitStatementToPieces(sql string) ([]string, error) {
	return sqlparser.SplitStatementToPieces(sql)
}
