package pgsql

import (
	"mayfly-go/internal/db/dbm/sqlparser/base"
	pgparser "mayfly-go/internal/db/dbm/sqlparser/pgsql/antlr4"
	"mayfly-go/pkg/logx"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"

	"github.com/antlr4-go/antlr/v4"
)

func GetPgsqlParserTree(baseLine int, statement string) (antlr.ParseTree, *antlr.CommonTokenStream, error) {
	lexer := pgparser.NewPostgreSQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := pgparser.NewPostgreSQLParser(stream)

	lexerErrorListener := &base.ParseErrorListener{
		BaseLine: baseLine,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		BaseLine: baseLine,
	}
	parser.RemoveErrorListeners()
	parser.AddErrorListener(parserErrorListener)
	parser.BuildParseTrees = true

	tree := parser.Root()

	if lexerErrorListener.Err != nil {
		return nil, nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, nil, parserErrorListener.Err
	}

	return tree, stream, nil
}

type PgsqlParser struct {
}

func (*PgsqlParser) Parse(stmt string) (stmts []sqlstmt.Stmt, err error) {
	defer func() {
		if e := recover(); e != nil {
			logx.ErrorTrace("postgres sql parser err: ", e)
			err = e.(error)
		}
	}()
	tree, _, err := GetPgsqlParserTree(1, stmt)
	if err != nil {
		return nil, err
	}

	return tree.Accept(new(PgsqlVisitor)).([]sqlstmt.Stmt), nil
}
