package mysql

import (
	"strings"

	mysqlparser "mayfly-go/internal/db/dbm/sqlparser/mysql/antlr4"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"

	"github.com/may-fly/cast"
)

type MysqlVisitor struct {
	*mysqlparser.BaseMySqlParserVisitor
}

func (v *MysqlVisitor) VisitRoot(ctx *mysqlparser.RootContext) interface{} {
	stms := ctx.SqlStatements()
	if stms != nil {
		return stms.Accept(v)
	}

	return nil
}

func (v *MysqlVisitor) VisitSqlStatements(ctx *mysqlparser.SqlStatementsContext) interface{} {
	allSqlStatement := ctx.AllSqlStatement()
	stmts := make([]sqlstmt.Stmt, 0)
	for _, sqlStatement := range allSqlStatement {
		stmts = append(stmts, sqlStatement.Accept(v).(sqlstmt.Stmt))
	}
	return stmts
}

func (v *MysqlVisitor) VisitSqlStatement(ctx *mysqlparser.SqlStatementContext) interface{} {
	if c := ctx.DmlStatement(); c != nil {
		return ctx.DmlStatement().Accept(v)
	}
	if c := ctx.DdlStatement(); c != nil {
		return ctx.DdlStatement().Accept(v)
	}
	if c := ctx.AdministrationStatement(); c != nil {
		return c.Accept(v)
	}
	if c := ctx.UtilityStatement(); c != nil {
		return c.Accept(v)
	}

	return sqlstmt.NewNode(ctx.GetParser(), ctx)
}

func (v *MysqlVisitor) VisitEmptyStatement_(ctx *mysqlparser.EmptyStatement_Context) interface{} {
	return ""
}

func (v *MysqlVisitor) VisitDdlStatement(ctx *mysqlparser.DdlStatementContext) interface{} {
	ddlStmt := &sqlstmt.DdlStmt{}
	ddlStmt.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ddlStmt
}

func (v *MysqlVisitor) VisitDmlStatement(ctx *mysqlparser.DmlStatementContext) interface{} {
	if ssc := ctx.SelectStatement(); ssc != nil {
		return ssc.Accept(v)
	}
	if withStmt := ctx.WithStatement(); withStmt != nil {
		return withStmt.Accept(v)
	}
	if usc := ctx.UpdateStatement(); usc != nil {
		return usc.Accept(v)
	}
	if dsc := ctx.DeleteStatement(); dsc != nil {
		return dsc.Accept(v)
	}
	if isc := ctx.InsertStatement(); isc != nil {
		return isc.Accept(v)
	}

	dmlStmt := sqlstmt.DmlStmt{}
	dmlStmt.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return dmlStmt
}

func (v *MysqlVisitor) VisitAdministrationStatement(ctx *mysqlparser.AdministrationStatementContext) interface{} {
	if ssc := ctx.ShowStatement(); ssc != nil {
		return ssc.Accept(v)
	}
	return sqlstmt.NewNode(ctx.GetParser(), ctx)
}

func (v *MysqlVisitor) VisitUtilityStatement(ctx *mysqlparser.UtilityStatementContext) interface{} {
	if c := ctx.SimpleDescribeStatement(); c != nil {
		return c.Accept(v)
	}
	if c := ctx.FullDescribeStatement(); c != nil {
		return c.Accept(v)
	}
	return sqlstmt.NewNode(ctx.GetParser(), ctx)
}

func (v *MysqlVisitor) VisitWithStatement(ctx *mysqlparser.WithStatementContext) interface{} {
	ort := new(sqlstmt.WithStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitSimpleSelect(ctx *mysqlparser.SimpleSelectContext) interface{} {
	sss := new(sqlstmt.SimpleSelectStmt)
	sss.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	sss.QuerySpecification = ctx.QuerySpecification().Accept(v).(*sqlstmt.QuerySpecification)
	return sss
}

func (v *MysqlVisitor) VisitUnionSelect(ctx *mysqlparser.UnionSelectContext) interface{} {
	uss := new(sqlstmt.UnionSelectStmt)
	uss.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	if lc := ctx.LimitClause(); lc != nil {
		uss.Limit = lc.Accept(v).(*sqlstmt.Limit)
	}

	if ausc := ctx.AllUnionStatement(); ausc != nil {
		unionStmts := make([]*sqlstmt.UnionStmt, 0)
		for _, usc := range ausc {
			unionStmts = append(unionStmts, usc.Accept(v).(*sqlstmt.UnionStmt))
		}
		uss.UnionStmts = unionStmts
	}

	if qsc := ctx.QuerySpecification(); qsc != nil {
		uss.QuerySpecification = qsc.Accept(v).(*sqlstmt.QuerySpecification)
	}
	if qscn := ctx.QuerySpecificationNointo(); qscn != nil {
		uss.QuerySpecification = qscn.Accept(v).(*sqlstmt.QuerySpecification)
	}
	if qec := ctx.QueryExpression(); qec != nil {
		uss.QueryExpr = qec.Accept(v).(*sqlstmt.QueryExpr)
	}
	if qenc := ctx.QueryExpressionNointo(); qenc != nil {
		uss.QueryExpr = qenc.Accept(v).(*sqlstmt.QueryExpr)
	}

	if ui := ctx.UNION(); ui != nil {
		uss.UnionType = ui.GetText()
	}

	return uss
}

func (v *MysqlVisitor) VisitParenthesisSelect(ctx *mysqlparser.ParenthesisSelectContext) interface{} {
	ps := new(sqlstmt.ParenthesisSelect)
	ps.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	if qec := ctx.QueryExpression(); qec != nil {
		ps.QueryExpr = qec.Accept(v).(*sqlstmt.QueryExpr)
	}

	return ps
}

func (v *MysqlVisitor) VisitUnionParenthesisSelect(ctx *mysqlparser.UnionParenthesisSelectContext) interface{} {
	ss := new(sqlstmt.SelectStmt)
	ss.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ss
}

func (v *MysqlVisitor) VisitWithLateralStatement(ctx *mysqlparser.WithLateralStatementContext) interface{} {
	ss := new(sqlstmt.SelectStmt)
	ss.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ss
}

func (v *MysqlVisitor) VisitUnionStatement(ctx *mysqlparser.UnionStatementContext) interface{} {
	us := new(sqlstmt.UnionStmt)
	us.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	if qs := ctx.QuerySpecificationNointo(); qs != nil {
		us.QuerySpecification = qs.Accept(v).(*sqlstmt.QuerySpecification)
	}
	if qec := ctx.QueryExpressionNointo(); qec != nil {
		us.QueryExpr = qec.Accept(v).(*sqlstmt.QueryExpr)
	}

	if ui := ctx.UNION(); ui != nil {
		us.UnionType = ui.GetText()
	}

	return us
}

func (v *MysqlVisitor) VisitQuerySpecification(ctx *mysqlparser.QuerySpecificationContext) interface{} {
	qs := new(sqlstmt.QuerySpecification)
	qs.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	qs.SelectElements = ctx.SelectElements().Accept(v).(*sqlstmt.SelectElements)

	if fromClause := ctx.FromClause(); fromClause != nil {
		where := fromClause.GetWhereExpr()
		if where != nil {
			qs.Where = v.GetExpr(where)
		}

		tableSourcesCtx := fromClause.TableSources()
		if tableSourcesCtx != nil {
			tss := new(sqlstmt.TableSources)
			tss.Node = sqlstmt.NewNode(tableSourcesCtx.GetParser(), tableSourcesCtx)
			tss.TableSources = tableSourcesCtx.Accept(v).([]sqlstmt.ITableSource)
			qs.From = tss
		}
	}

	if limitClause := ctx.LimitClause(); limitClause != nil {
		qs.Limit = limitClause.Accept(v).(*sqlstmt.Limit)
	}

	return qs
}

func (v *MysqlVisitor) VisitQuerySpecificationNointo(ctx *mysqlparser.QuerySpecificationNointoContext) interface{} {
	qs := new(sqlstmt.QuerySpecification)
	qs.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	qs.SelectElements = ctx.SelectElements().Accept(v).(*sqlstmt.SelectElements)

	if fromClause := ctx.FromClause(); fromClause != nil {
		where := fromClause.GetWhereExpr()
		if where != nil {
			qs.Where = v.GetExpr(where)
		}

		tableSourcesCtx := fromClause.TableSources()
		if tableSourcesCtx != nil {
			tss := new(sqlstmt.TableSources)
			tss.Node = sqlstmt.NewNode(tableSourcesCtx.GetParser(), tableSourcesCtx)
			tss.TableSources = tableSourcesCtx.Accept(v).([]sqlstmt.ITableSource)
			qs.From = tss
		}
	}

	if limitClause := ctx.LimitClause(); limitClause != nil {
		qs.Limit = limitClause.Accept(v).(*sqlstmt.Limit)
	}

	return qs
}

func (v *MysqlVisitor) VisitQueryExpression(ctx *mysqlparser.QueryExpressionContext) interface{} {
	qe := new(sqlstmt.QueryExpr)
	qe.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	if qec := ctx.QueryExpression(); qec != nil {
		qe.QueryExpr = qec.Accept(v).(*sqlstmt.QueryExpr)
	}

	if qsc := ctx.QuerySpecification(); qsc != nil {
		qe.QuerySpecification = qsc.Accept(v).(*sqlstmt.QuerySpecification)
	}

	return qe
}

func (v *MysqlVisitor) VisitQueryExpressionNointo(ctx *mysqlparser.QueryExpressionNointoContext) interface{} {
	qe := new(sqlstmt.QueryExpr)
	qe.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	if qec := ctx.QueryExpressionNointo(); qec != nil {
		qe.QueryExpr = qec.Accept(v).(*sqlstmt.QueryExpr)
	}

	if qsc := ctx.QuerySpecificationNointo(); qsc != nil {
		qe.QuerySpecification = qsc.Accept(v).(*sqlstmt.QuerySpecification)
	}

	return qe
}

func (v *MysqlVisitor) VisitSelectElements(ctx *mysqlparser.SelectElementsContext) interface{} {
	ses := new(sqlstmt.SelectElements)
	ses.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	if ctx.STAR() != nil {
		ses.Star = ctx.STAR().GetText()
	}

	eles := make([]sqlstmt.ISelectElement, 0)
	ase := ctx.AllSelectElement()
	for _, selectElement := range ase {
		eles = append(eles, selectElement.Accept(v).(sqlstmt.ISelectElement))
	}
	ses.Elements = eles

	return ses
}

func (v *MysqlVisitor) VisitSelectStarElement(ctx *mysqlparser.SelectStarElementContext) interface{} {
	sse := new(sqlstmt.SelectStarElement)
	sse.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	sse.FullId = ctx.FullId().GetText()
	return sse
}

func (v *MysqlVisitor) VisitSelectColumnElement(ctx *mysqlparser.SelectColumnElementContext) interface{} {
	sce := new(sqlstmt.SelectColumnElement)
	sce.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	sce.ColumnName = ctx.FullColumnName().Accept(v).(*sqlstmt.ColumnName)
	if uid := ctx.Uid(); uid != nil {
		sce.Alias = uid.GetText()
	}
	return sce
}

func (v *MysqlVisitor) VisitSelectFunctionElement(ctx *mysqlparser.SelectFunctionElementContext) interface{} {
	node := sqlstmt.NewNode(ctx.GetParser(), ctx)
	return node
}

func (v *MysqlVisitor) VisitSelectExpressionElement(ctx *mysqlparser.SelectExpressionElementContext) interface{} {
	node := sqlstmt.NewNode(ctx.GetParser(), ctx)
	return node
}

func (v *MysqlVisitor) VisitTableSources(ctx *mysqlparser.TableSourcesContext) interface{} {
	tableSourcesCtx := ctx.AllTableSource()
	tableSources := make([]sqlstmt.ITableSource, 0)
	for _, tableSourceCtx := range tableSourcesCtx {
		tableSources = append(tableSources, tableSourceCtx.Accept(v).(sqlstmt.ITableSource))
	}
	return tableSources
}

func (v *MysqlVisitor) VisitTableSourceBase(ctx *mysqlparser.TableSourceBaseContext) interface{} {
	tsb := new(sqlstmt.TableSourceBase)
	tsb.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	tsb.TableSourceItem = v.GetTableSourceItem(ctx.TableSourceItem())
	tsb.JoinParts = v.GetJoinParts(ctx.AllJoinPart())
	return tsb
}

func (v *MysqlVisitor) VisitAtomTableItem(ctx *mysqlparser.AtomTableItemContext) interface{} {
	tableSourceItem := new(sqlstmt.AtomTableItem)
	tableSourceItem.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	tableSourceItem.TableName = ctx.TableName().Accept(v).(*sqlstmt.TableName)

	if alias := ctx.GetAlias(); alias != nil {
		tableSourceItem.Alias = alias.GetText()
	}

	return tableSourceItem
}

func (v *MysqlVisitor) VisitInnerJoin(ctx *mysqlparser.InnerJoinContext) interface{} {
	ij := new(sqlstmt.InnerJoin)
	ij.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	ij.TableSourceItem = v.GetTableSourceItem(ctx.TableSourceItem())
	return ij
}

func (v *MysqlVisitor) VisitStraightJoin(ctx *mysqlparser.StraightJoinContext) interface{} {
	jp := new(sqlstmt.JoinPart)
	jp.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	jp.TableSourceItem = v.GetTableSourceItem(ctx.TableSourceItem())
	return jp
}

func (v *MysqlVisitor) VisitOuterJoin(ctx *mysqlparser.OuterJoinContext) interface{} {
	oj := new(sqlstmt.OuterJoin)
	oj.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	oj.TableSourceItem = v.GetTableSourceItem(ctx.TableSourceItem())
	return oj
}

func (v *MysqlVisitor) VisitNaturalJoin(ctx *mysqlparser.NaturalJoinContext) interface{} {
	nj := new(sqlstmt.NaturalJoin)
	nj.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	nj.TableSourceItem = v.GetTableSourceItem(ctx.TableSourceItem())
	return nj
}

func (v *MysqlVisitor) VisitJoinSpec(ctx *mysqlparser.JoinSpecContext) interface{} {
	node := sqlstmt.NewNode(ctx.GetParser(), ctx)
	return node
}

func (v *MysqlVisitor) VisitIsExpression(ctx *mysqlparser.IsExpressionContext) interface{} {
	e := new(sqlstmt.Expr)
	e.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return e
}

func (v *MysqlVisitor) VisitNotExpression(ctx *mysqlparser.NotExpressionContext) interface{} {
	e := new(sqlstmt.Expr)
	e.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return e
}

func (v *MysqlVisitor) VisitLogicalExpression(ctx *mysqlparser.LogicalExpressionContext) interface{} {
	le := new(sqlstmt.ExprLogical)
	le.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	le.Operator = ctx.LogicalOperator().GetText()
	le.Exprs = v.GetExprs(ctx.AllExpression())
	return le
}

func (v *MysqlVisitor) VisitPredicateExpression(ctx *mysqlparser.PredicateExpressionContext) interface{} {
	ep := new(sqlstmt.ExprPredicate)
	ep.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	ep.Predicate = ctx.Predicate().Accept(v).(sqlstmt.IPredicate)
	return ep
}

func (v *MysqlVisitor) VisitSoundsLikePredicate(ctx *mysqlparser.SoundsLikePredicateContext) interface{} {
	e := new(sqlstmt.Expr)
	e.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return e
}

func (v *MysqlVisitor) VisitExpressionAtomPredicate(ctx *mysqlparser.ExpressionAtomPredicateContext) interface{} {
	pea := new(sqlstmt.PredicateExprAtom)
	pea.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	pea.ExprAtom = ctx.ExpressionAtom().Accept(v).(sqlstmt.IExprAtom)
	return pea
}

func (v *MysqlVisitor) VisitLogicalOperator(ctx *mysqlparser.LogicalOperatorContext) interface{} {
	return ctx.GetText()
}

func (v *MysqlVisitor) VisitBinaryComparisonPredicate(ctx *mysqlparser.BinaryComparisonPredicateContext) interface{} {
	bcp := new(sqlstmt.PredicateBinaryComparison)
	bcp.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	bcp.Left = ctx.GetLeft().Accept(v).(sqlstmt.IPredicate)
	bcp.Right = ctx.GetRight().Accept(v).(sqlstmt.IPredicate)
	bcp.ComparisonOperator = ctx.ComparisonOperator().Accept(v).(string)
	return bcp
}

func (v *MysqlVisitor) VisitInPredicate(ctx *mysqlparser.InPredicateContext) interface{} {
	inPredicate := new(sqlstmt.PredicateIn)
	inPredicate.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	if pc := ctx.Predicate(); pc != nil {
		inPredicate.InPredicate = pc.Accept(v).(sqlstmt.IPredicate)
	}
	if ssc := ctx.SelectStatement(); ssc != nil {
		inPredicate.SelectStmt = ssc.Accept(v).(sqlstmt.ISelectStmt)
	}
	if ec := ctx.Expressions(); ec != nil {
		inPredicate.Exprs = v.GetExprs(ec.AllExpression())
	}

	return inPredicate
}

func (v *MysqlVisitor) VisitBetweenPredicate(ctx *mysqlparser.BetweenPredicateContext) interface{} {
	predicate := new(sqlstmt.PredicateLike)
	predicate.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return predicate
}

func (v *MysqlVisitor) VisitIsNullPredicate(ctx *mysqlparser.IsNullPredicateContext) interface{} {
	predicate := new(sqlstmt.PredicateLike)
	predicate.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return predicate
}

func (v *MysqlVisitor) VisitLikePredicate(ctx *mysqlparser.LikePredicateContext) interface{} {
	likePredicate := new(sqlstmt.PredicateLike)
	likePredicate.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return likePredicate
}

func (v *MysqlVisitor) VisitRegexpPredicate(ctx *mysqlparser.RegexpPredicateContext) interface{} {
	predicate := new(sqlstmt.PredicateLike)
	predicate.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return predicate
}

func (v *MysqlVisitor) VisitUnaryExpressionAtom(ctx *mysqlparser.UnaryExpressionAtomContext) interface{} {
	ea := new(sqlstmt.ExprAtom)
	ea.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ea
}

func (v *MysqlVisitor) VisitCollateExpressionAtom(ctx *mysqlparser.CollateExpressionAtomContext) interface{} {
	ea := new(sqlstmt.ExprAtom)
	ea.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ea
}

func (v *MysqlVisitor) VisitVariableAssignExpressionAtom(ctx *mysqlparser.VariableAssignExpressionAtomContext) interface{} {
	ea := new(sqlstmt.ExprAtom)
	ea.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ea
}

func (v *MysqlVisitor) VisitMysqlVariableExpressionAtom(ctx *mysqlparser.MysqlVariableExpressionAtomContext) interface{} {
	ea := new(sqlstmt.ExprAtom)
	ea.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ea
}

func (v *MysqlVisitor) VisitNestedExpressionAtom(ctx *mysqlparser.NestedExpressionAtomContext) interface{} {
	ea := new(sqlstmt.ExprAtom)
	ea.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ea
}

func (v *MysqlVisitor) VisitNestedRowExpressionAtom(ctx *mysqlparser.NestedRowExpressionAtomContext) interface{} {
	ea := new(sqlstmt.ExprAtom)
	ea.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ea
}

func (v *MysqlVisitor) VisitMathExpressionAtom(ctx *mysqlparser.MathExpressionAtomContext) interface{} {
	ea := new(sqlstmt.ExprAtom)
	ea.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ea
}

func (v *MysqlVisitor) VisitExistsExpressionAtom(ctx *mysqlparser.ExistsExpressionAtomContext) interface{} {
	ea := new(sqlstmt.ExprAtom)
	ea.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ea
}

func (v *MysqlVisitor) VisitIntervalExpressionAtom(ctx *mysqlparser.IntervalExpressionAtomContext) interface{} {
	ea := new(sqlstmt.ExprAtom)
	ea.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ea
}

func (v *MysqlVisitor) VisitJsonExpressionAtom(ctx *mysqlparser.JsonExpressionAtomContext) interface{} {
	ea := new(sqlstmt.ExprAtom)
	ea.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ea
}

func (v *MysqlVisitor) VisitSubqueryExpressionAtom(ctx *mysqlparser.SubqueryExpressionAtomContext) interface{} {
	ea := new(sqlstmt.ExprAtom)
	ea.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ea
}

func (v *MysqlVisitor) VisitConstantExpressionAtom(ctx *mysqlparser.ConstantExpressionAtomContext) interface{} {
	constExprAtom := new(sqlstmt.ExprAtomConstant)
	constExprAtom.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	constExprAtom.Constant = ctx.Constant().Accept(v).(*sqlstmt.Constant)
	return constExprAtom
}

func (v *MysqlVisitor) VisitFunctionCallExpressionAtom(ctx *mysqlparser.FunctionCallExpressionAtomContext) interface{} {
	ea := new(sqlstmt.ExprAtom)
	ea.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ea
}

func (v *MysqlVisitor) VisitBinaryExpressionAtom(ctx *mysqlparser.BinaryExpressionAtomContext) interface{} {
	ea := new(sqlstmt.ExprAtom)
	ea.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ea
}

func (v *MysqlVisitor) VisitFullColumnNameExpressionAtom(ctx *mysqlparser.FullColumnNameExpressionAtomContext) interface{} {
	eacn := new(sqlstmt.ExprAtomColumnName)
	eacn.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	eacn.ColumnName = ctx.FullColumnName().Accept(v).(*sqlstmt.ColumnName)
	return eacn
}

func (v *MysqlVisitor) VisitBitExpressionAtom(ctx *mysqlparser.BitExpressionAtomContext) interface{} {
	ea := new(sqlstmt.ExprAtom)
	ea.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ea
}

func (v *MysqlVisitor) VisitComparisonOperator(ctx *mysqlparser.ComparisonOperatorContext) interface{} {
	return ctx.GetText()
}

func (v *MysqlVisitor) VisitTableName(ctx *mysqlparser.TableNameContext) interface{} {
	tableName := new(sqlstmt.TableName)
	tableName.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	fullId := ctx.FullId().Accept(v).(*sqlstmt.FullId)

	if uids := fullId.Uids; len(uids) == 1 {
		tableName.Identifier = sqlstmt.NewIdentifierValue(uids[0])
	} else {
		tableName.Owner = uids[0]
		tableName.Identifier = sqlstmt.NewIdentifierValue(uids[1])
	}

	return tableName
}

func (v *MysqlVisitor) VisitFullId(ctx *mysqlparser.FullIdContext) interface{} {
	fid := new(sqlstmt.FullId)
	fid.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	uids := make([]string, 0)
	for _, uid := range ctx.AllUid() {
		uids = append(uids, uid.GetText())
	}

	if did := ctx.DOT_ID(); did != nil {
		uids = append(uids, strings.TrimPrefix(did.GetText(), "."))
	}

	fid.Uids = uids

	return fid
}

func (v *MysqlVisitor) VisitRoleName(ctx *mysqlparser.RoleNameContext) interface{} {
	node := sqlstmt.NewNode(ctx.GetParser(), ctx)
	return node
}

func (v *MysqlVisitor) VisitFullColumnName(ctx *mysqlparser.FullColumnNameContext) interface{} {
	fullColumnName := new(sqlstmt.ColumnName)
	fullColumnName.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	adis := ctx.AllDottedId()
	// 不存在.则直接取标识符
	if len(adis) == 0 {
		fullColumnName.Identifier = sqlstmt.NewIdentifierValue(ctx.Uid().GetText())
	} else {
		fullColumnName.Owner = ctx.Uid().GetText()
		fullColumnName.Identifier = sqlstmt.NewIdentifierValue(adis[0].GetText())
	}

	return fullColumnName
}

func (v *MysqlVisitor) VisitIndexColumnName(ctx *mysqlparser.IndexColumnNameContext) interface{} {
	node := sqlstmt.NewNode(ctx.GetParser(), ctx)
	return node
}

func (v *MysqlVisitor) VisitConstant(ctx *mysqlparser.ConstantContext) interface{} {
	constant := new(sqlstmt.Constant)
	constant.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	constant.Value = ctx.GetText()
	return constant
}

func (v *MysqlVisitor) VisitLimitClause(ctx *mysqlparser.LimitClauseContext) interface{} {
	limit := new(sqlstmt.Limit)
	limit.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	if lc := ctx.GetLimit(); lc != nil {
		limit.RowCount = cast.ToInt(lc.GetText())
	}
	if oc := ctx.GetOffset(); oc != nil {
		limit.Offset = cast.ToInt(oc.GetText())
	}

	return limit
}

func (v *MysqlVisitor) VisitInsertStatement(ctx *mysqlparser.InsertStatementContext) interface{} {
	is := new(sqlstmt.InsertStmt)
	is.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	is.TableName = ctx.TableName().Accept(v).(*sqlstmt.TableName)
	return is
}

func (v *MysqlVisitor) VisitUpdateStatement(ctx *mysqlparser.UpdateStatementContext) interface{} {
	if sus := ctx.SingleUpdateStatement(); sus != nil {
		return sus.Accept(v)
	}
	if mus := ctx.MultipleUpdateStatement(); mus != nil {
		return mus.Accept(v)
	}

	us := new(sqlstmt.UpdateStmt)
	us.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return us
}

func (v *MysqlVisitor) VisitSingleUpdateStatement(ctx *mysqlparser.SingleUpdateStatementContext) interface{} {
	sus := new(sqlstmt.UpdateStmt)
	sus.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	tss := new(sqlstmt.TableSources)
	tss.Node = sqlstmt.NewNode(ctx.TableName().GetParser(), ctx.TableName())
	atomTable := new(sqlstmt.AtomTableItem)
	atomTable.TableName = ctx.TableName().Accept(v).(*sqlstmt.TableName)
	if uid := ctx.Uid(); uid != nil {
		atomTable.Alias = uid.GetText()
	}

	tableSourceBase := new(sqlstmt.TableSourceBase)
	tableSourceBase.Node = tss.Node
	tableSourceBase.TableSourceItem = atomTable
	tss.TableSources = []sqlstmt.ITableSource{tableSourceBase}

	sus.TableSources = tss

	if aucs := ctx.AllUpdatedElement(); aucs != nil {
		ues := make([]*sqlstmt.UpdatedElement, 0)
		for _, auc := range aucs {
			ues = append(ues, auc.Accept(v).(*sqlstmt.UpdatedElement))
		}
		sus.UpdatedElements = ues
	}

	if ec := ctx.Expression(); ec != nil {
		sus.Where = v.GetExpr(ec)
	}

	return sus
}

func (v *MysqlVisitor) VisitMultipleUpdateStatement(ctx *mysqlparser.MultipleUpdateStatementContext) interface{} {
	mus := new(sqlstmt.UpdateStmt)
	mus.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	if tssc := ctx.TableSources(); tssc != nil {
		tss := new(sqlstmt.TableSources)
		tss.Node = sqlstmt.NewNode(tssc.GetParser(), tssc)
		tss.TableSources = tssc.Accept(v).([]sqlstmt.ITableSource)
		mus.TableSources = tss
	}

	if aucs := ctx.AllUpdatedElement(); aucs != nil {
		ues := make([]*sqlstmt.UpdatedElement, 0)
		for _, auc := range aucs {
			ues = append(ues, auc.Accept(v).(*sqlstmt.UpdatedElement))
		}
		mus.UpdatedElements = ues
	}

	if ec := ctx.Expression(); ec != nil {
		mus.Where = v.GetExpr(ec)
	}

	return mus
}

func (v *MysqlVisitor) VisitUpdatedElement(ctx *mysqlparser.UpdatedElementContext) interface{} {
	ue := new(sqlstmt.UpdatedElement)
	ue.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	ue.ColumnName = ctx.FullColumnName().Accept(v).(*sqlstmt.ColumnName)
	ue.Value = v.GetExpr(ctx.Expression())
	return ue
}

func (v *MysqlVisitor) VisitDeleteStatement(ctx *mysqlparser.DeleteStatementContext) interface{} {
	if sus := ctx.SingleDeleteStatement(); sus != nil {
		return sus.Accept(v)
	}
	if mus := ctx.MultipleDeleteStatement(); mus != nil {
		return mus.Accept(v)
	}

	ds := new(sqlstmt.DeleteStmt)
	ds.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ds
}

func (v *MysqlVisitor) VisitSingleDeleteStatement(ctx *mysqlparser.SingleDeleteStatementContext) interface{} {
	ds := new(sqlstmt.DeleteStmt)
	ds.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	tss := new(sqlstmt.TableSources)
	tss.Node = sqlstmt.NewNode(ctx.TableName().GetParser(), ctx.TableName())
	atomTable := new(sqlstmt.AtomTableItem)
	atomTable.TableName = ctx.TableName().Accept(v).(*sqlstmt.TableName)
	if uid := ctx.Uid(); uid != nil {
		atomTable.Alias = uid.GetText()
	}

	tableSourceBase := new(sqlstmt.TableSourceBase)
	tableSourceBase.Node = tss.Node
	tableSourceBase.TableSourceItem = atomTable
	tss.TableSources = []sqlstmt.ITableSource{tableSourceBase}

	ds.TableSources = tss

	if ec := ctx.Expression(); ec != nil {
		ds.Where = v.GetExpr(ec)
	}

	return ds
}

func (v *MysqlVisitor) VisitMultipleDeleteStatement(ctx *mysqlparser.MultipleDeleteStatementContext) interface{} {
	ds := new(sqlstmt.DeleteStmt)
	ds.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	if tssc := ctx.TableSources(); tssc != nil {
		tss := new(sqlstmt.TableSources)
		tss.Node = sqlstmt.NewNode(tssc.GetParser(), tssc)
		tss.TableSources = tssc.Accept(v).([]sqlstmt.ITableSource)
		ds.TableSources = tss
	}

	if ec := ctx.Expression(); ec != nil {
		ds.Where = v.GetExpr(ec)
	}

	return ds
}

func (v *MysqlVisitor) VisitSimpleDescribeStatement(ctx *mysqlparser.SimpleDescribeStatementContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitFullDescribeStatement(ctx *mysqlparser.FullDescribeStatementContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowMasterLogs(ctx *mysqlparser.ShowMasterLogsContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowLogEvents(ctx *mysqlparser.ShowLogEventsContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowObjectFilter(ctx *mysqlparser.ShowObjectFilterContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowColumns(ctx *mysqlparser.ShowColumnsContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowCreateDb(ctx *mysqlparser.ShowCreateDbContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowCreateFullIdObject(ctx *mysqlparser.ShowCreateFullIdObjectContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowCreateUser(ctx *mysqlparser.ShowCreateUserContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowEngine(ctx *mysqlparser.ShowEngineContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowGlobalInfo(ctx *mysqlparser.ShowGlobalInfoContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowErrors(ctx *mysqlparser.ShowErrorsContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowCountErrors(ctx *mysqlparser.ShowCountErrorsContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowSchemaFilter(ctx *mysqlparser.ShowSchemaFilterContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowRoutine(ctx *mysqlparser.ShowRoutineContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowGrants(ctx *mysqlparser.ShowGrantsContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowIndexes(ctx *mysqlparser.ShowIndexesContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowOpenTables(ctx *mysqlparser.ShowOpenTablesContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowProfile(ctx *mysqlparser.ShowProfileContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitShowSlaveStatus(ctx *mysqlparser.ShowSlaveStatusContext) interface{} {
	ort := new(sqlstmt.OtherReadStmt)
	ort.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ort
}

func (v *MysqlVisitor) VisitCreateDatabase(ctx *mysqlparser.CreateDatabaseContext) interface{} {
	cds := new(sqlstmt.CreateDatabase)
	cds.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return cds
}

func (v *MysqlVisitor) GetTableSourceItem(ctx mysqlparser.ITableSourceItemContext) sqlstmt.ITableSourceItem {
	if ctx == nil {
		return nil
	}
	return ctx.Accept(v).(sqlstmt.ITableSourceItem)
}

func (v *MysqlVisitor) GetExpr(ctx mysqlparser.IExpressionContext) sqlstmt.IExpr {
	if ctx == nil {
		return nil
	}

	return ctx.Accept(v).(sqlstmt.IExpr)
}

func (v *MysqlVisitor) GetExprs(ctxs []mysqlparser.IExpressionContext) []sqlstmt.IExpr {
	if ctxs == nil {
		return nil
	}

	exprs := make([]sqlstmt.IExpr, 0)
	for _, exprCtx := range ctxs {
		exprs = append(exprs, exprCtx.Accept(v).(sqlstmt.IExpr))
	}
	return exprs
}

func (v *MysqlVisitor) GetJoinParts(ctxs []mysqlparser.IJoinPartContext) []sqlstmt.IJoinPart {
	if ctxs == nil {
		return nil
	}

	joinPorts := make([]sqlstmt.IJoinPart, 0)
	for _, joinPartCtx := range ctxs {
		joinPorts = append(joinPorts, joinPartCtx.Accept(v).(sqlstmt.IJoinPart))
	}
	return joinPorts
}
