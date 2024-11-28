package pgsql

import (
	"strings"

	pgparser "mayfly-go/internal/db/dbm/sqlparser/pgsql/antlr4"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"

	"github.com/may-fly/cast"
)

type PgsqlVisitor struct {
	*pgparser.BasePostgreSQLParserVisitor
}

func (v *PgsqlVisitor) VisitRoot(ctx *pgparser.RootContext) interface{} {
	if sbc := ctx.Stmtblock(); sbc != nil {
		return sbc.Accept(v)
	}
	return sqlstmt.NewNode(ctx.GetParser(), ctx)
}

func (v *PgsqlVisitor) VisitPlsqlroot(ctx *pgparser.PlsqlrootContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *PgsqlVisitor) VisitStmtblock(ctx *pgparser.StmtblockContext) interface{} {
	if smc := ctx.Stmtmulti(); smc != nil {
		return smc.Accept(v)
	}
	return sqlstmt.NewNode(ctx.GetParser(), ctx)
}

func (v *PgsqlVisitor) VisitStmtmulti(ctx *pgparser.StmtmultiContext) interface{} {
	allSqlStatement := ctx.AllStmt()
	stmts := make([]sqlstmt.Stmt, 0)
	for _, sqlStatement := range allSqlStatement {
		stmts = append(stmts, sqlStatement.Accept(v).(sqlstmt.Stmt))
	}
	return stmts
}

func (v *PgsqlVisitor) VisitStmt(ctx *pgparser.StmtContext) interface{} {
	if selectstmtCtx := ctx.Selectstmt(); selectstmtCtx != nil {
		return selectstmtCtx.Accept(v)
	}
	if updatestmtCtx := ctx.Updatestmt(); updatestmtCtx != nil {
		return updatestmtCtx.Accept(v)
	}
	if deletestmtCtx := ctx.Deletestmt(); deletestmtCtx != nil {
		return deletestmtCtx.Accept(v)
	}
	if insertstmtC := ctx.Insertstmt(); insertstmtC != nil {
		return insertstmtC.Accept(v)
	}
	if c := ctx.Createdbstmt(); c != nil {
		cds := new(sqlstmt.CreateDatabase)
		cds.Node = sqlstmt.NewNode(c.GetParser(), c)
		return cds
	}
	if c := ctx.Createtablespacestmt(); c != nil {
		cds := new(sqlstmt.CreateTable)
		cds.Node = sqlstmt.NewNode(c.GetParser(), c)
		return cds
	}
	if c := ctx.Altertablestmt(); c != nil {
		cds := new(sqlstmt.AlterTable)
		cds.Node = sqlstmt.NewNode(c.GetParser(), c)
		return cds
	}
	if c := ctx.Dropdbstmt(); c != nil {
		cds := new(sqlstmt.DropDatabase)
		cds.Node = sqlstmt.NewNode(c.GetParser(), c)
		return cds
	}
	if c := ctx.Droptablespacestmt(); c != nil {
		cds := new(sqlstmt.DropTable)
		cds.Node = sqlstmt.NewNode(c.GetParser(), c)
		return cds
	}
	if explain := ctx.Explainstmt(); explain != nil {
		otherRead := new(sqlstmt.OtherReadStmt)
		otherRead.Node = sqlstmt.NewNode(explain.GetParser(), explain)
		return otherRead
	}
	if c := ctx.Variableshowstmt(); c != nil {
		otherRead := new(sqlstmt.OtherReadStmt)
		otherRead.Node = sqlstmt.NewNode(c.GetParser(), c)
		return otherRead
	}
	return sqlstmt.NewNode(ctx.GetParser(), ctx)
}

func (v *PgsqlVisitor) VisitSelectstmt(ctx *pgparser.SelectstmtContext) interface{} {
	if spnc := ctx.Select_no_parens(); spnc != nil {
		return spnc.Accept(v)
	}
	selectstmt := new(sqlstmt.SelectStmt)
	selectstmt.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return selectstmt
}

func (v *PgsqlVisitor) VisitSelect_with_parens(ctx *pgparser.Select_with_parensContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *PgsqlVisitor) VisitSelect_no_parens(ctx *pgparser.Select_no_parensContext) interface{} {
	if c := ctx.Select_clause(); c == nil {
		return sqlstmt.NewNode(ctx.GetParser(), ctx)
	}

	var limit *sqlstmt.Limit
	if limitC := ctx.Select_limit(); limitC != nil {
		limit = limitC.Accept(v).(*sqlstmt.Limit)
	}
	if limitC := ctx.Opt_select_limit(); limitC != nil {
		limit = limitC.Accept(v).(*sqlstmt.Limit)
	}

	selectClause := ctx.Select_clause()
	asis := selectClause.AllSimple_select_intersect()
	// 简单查询
	if len(asis) == 1 {
		sss := new(sqlstmt.SimpleSelectStmt)
		sss.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
		sss.QuerySpecification = ctx.Select_clause().Accept(v).([]*sqlstmt.QuerySpecification)[0]
		sss.QuerySpecification.Limit = limit
		return sss
	}

	uss := new(sqlstmt.UnionSelectStmt)
	uss.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	allUnion := selectClause.AllUNION()
	// todo 赋值union信息
	for _, union := range allUnion {
		uss.UnionType = union.GetText()
	}
	// uss.QuerySpecifications = ctx.Select_clause().Accept(v).([]*sqlstmt.QuerySpecification)
	uss.Limit = limit
	return uss
}

func (v *PgsqlVisitor) VisitSelect_clause(ctx *pgparser.Select_clauseContext) interface{} {
	qs := make([]*sqlstmt.QuerySpecification, 0)
	for _, ssi := range ctx.AllSimple_select_intersect() {
		qs = append(qs, ssi.Accept(v).(*sqlstmt.QuerySpecification))
	}

	return qs
}

func (v *PgsqlVisitor) VisitSimple_select_intersect(ctx *pgparser.Simple_select_intersectContext) interface{} {
	// 只返回一个查询，INTERSECT（交集）暂不支持
	if spsc := ctx.AllSimple_select_pramary(); spsc != nil {
		return spsc[0].Accept(v)
	}
	return sqlstmt.NewNode(ctx.GetParser(), ctx)
}

func (v *PgsqlVisitor) VisitSimple_select_pramary(ctx *pgparser.Simple_select_pramaryContext) interface{} {
	qs := new(sqlstmt.QuerySpecification)
	qs.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	if c := ctx.From_clause(); c != nil {
		qs.From = c.Accept(v).(*sqlstmt.TableSources)
	}

	if c := ctx.Opt_target_list(); c != nil {
		qs.SelectElements = c.Accept(v).(*sqlstmt.SelectElements)
	}

	if c := ctx.Target_list(); c != nil {
		qs.SelectElements = c.Accept(v).(*sqlstmt.SelectElements)
	}

	if c := ctx.Where_clause(); c != nil && c.A_expr() != nil {
		qs.Where = c.A_expr().Accept(v).(sqlstmt.IExpr)
	}

	return qs
}

func (v *PgsqlVisitor) VisitSelect_limit(ctx *pgparser.Select_limitContext) interface{} {
	limit := new(sqlstmt.Limit)
	limit.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	if lc := ctx.Limit_clause(); lc != nil {
		if lv := lc.Select_limit_value(); lv != nil {
			limit.RowCount = cast.ToInt(lv.GetText())
		}
	}
	if oc := ctx.Offset_clause(); oc != nil {
		if ov := oc.Select_offset_value(); ov != nil {
			limit.Offset = cast.ToInt(ov.GetText())
		}
	}
	return limit
}

func (v *PgsqlVisitor) VisitOpt_select_limit(ctx *pgparser.Opt_select_limitContext) interface{} {
	if slc := ctx.Select_limit(); slc != nil {
		return slc.Accept(v)
	}
	return nil
}

func (v *PgsqlVisitor) VisitFrom_clause(ctx *pgparser.From_clauseContext) interface{} {
	if c := ctx.From_list(); c != nil {
		return c.Accept(v)
	}
	return sqlstmt.NewNode(ctx.GetParser(), ctx)
}

func (v *PgsqlVisitor) VisitFrom_list(ctx *pgparser.From_listContext) interface{} {
	ts := new(sqlstmt.TableSources)
	ts.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	// ts.StartIndex = ctx.GetStart().GetStart()
	// ts.StopIndex = ctx.GetStop().GetStop()

	tableSources := make([]sqlstmt.ITableSource, 0)
	allTableRefCtx := ctx.AllTable_ref()
	for _, trc := range allTableRefCtx {
		tableSources = append(tableSources, trc.Accept(v).(sqlstmt.ITableSource))
	}

	ts.TableSources = tableSources
	return ts
}

func (v *PgsqlVisitor) VisitNon_ansi_join(ctx *pgparser.Non_ansi_joinContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *PgsqlVisitor) VisitTable_ref(ctx *pgparser.Table_refContext) interface{} {
	tableSourceBase := new(sqlstmt.TableSourceBase)
	tableSourceBase.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	atomTable := new(sqlstmt.AtomTableItem)

	if c := ctx.Relation_expr(); c != nil {
		tableName := new(sqlstmt.TableName)
		if qn := c.Qualified_name(); qn != nil {
			if qc := qn.Colid(); qc != nil {
				if c := qn.Indirection(); c != nil {
					tableName.Owner = qc.GetText()
					tableName.Identifier = sqlstmt.NewIdentifierValue(c.GetText())
				} else {
					tableName.Identifier = sqlstmt.NewIdentifierValue(qc.Identifier().GetText())
				}
			}
		}
		atomTable.TableName = tableName
	}
	if c := ctx.Opt_alias_clause(); c != nil {
		if aliasC := c.Table_alias_clause(); aliasC != nil {
			atomTable.Alias = aliasC.Table_alias().GetText()
		}
	}

	tableSourceBase.TableSourceItem = atomTable
	return tableSourceBase
}

func (v *PgsqlVisitor) VisitOpt_target_list(ctx *pgparser.Opt_target_listContext) interface{} {
	if c := ctx.Target_list(); c != nil {
		return c.Accept(v)
	}

	ses := new(sqlstmt.SelectElements)
	ses.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return ses
}

func (v *PgsqlVisitor) VisitTarget_list(ctx *pgparser.Target_listContext) interface{} {
	ses := new(sqlstmt.SelectElements)
	ses.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	if tecs := ctx.AllTarget_el(); tecs != nil {
		eles := make([]sqlstmt.ISelectElement, 0)
		for _, tec := range tecs {
			eles = append(eles, tec.Accept(v).(sqlstmt.ISelectElement))
		}
		ses.Elements = eles
	}
	if len(ses.Elements) == 1 && ses.Elements[0].GetText() == "*" {
		ses.Star = "*"
	}

	return ses
}

// Visit a parse tree produced by PostgreSQLParser#target_label.
func (v *PgsqlVisitor) VisitTarget_label(ctx *pgparser.Target_labelContext) interface{} {
	sce := new(sqlstmt.SelectColumnElement)
	sce.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	columnName := new(sqlstmt.ColumnName)

	if c := ctx.Collabel(); c != nil {
		sce.Alias = c.GetText()
	}
	if c := ctx.Identifier(); c != nil {
		sce.Alias = c.GetText()
	}
	if exprCtx := ctx.A_expr(); exprCtx != nil {
		columnName.Node = sqlstmt.NewNode(ctx.GetParser(), exprCtx)
		if aextrCtx := exprCtx.A_expr_qual(); aextrCtx != nil {
			col := aextrCtx.GetText()
			ownerAndColname := strings.Split(col, ".")
			if len(ownerAndColname) == 2 {
				columnName.Owner = ownerAndColname[0]
				columnName.Identifier = sqlstmt.NewIdentifierValue(ownerAndColname[1])
			} else {
				columnName.Identifier = sqlstmt.NewIdentifierValue(col)
			}
		}
	} else {
		columnName.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	}
	sce.ColumnName = columnName
	return sce
}

// Visit a parse tree produced by PostgreSQLParser#target_star.
func (v *PgsqlVisitor) VisitTarget_star(ctx *pgparser.Target_starContext) interface{} {
	sse := new(sqlstmt.SelectStarElement)
	sse.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	sse.FullId = ctx.STAR().GetText()
	return sse
}

func (v *PgsqlVisitor) VisitAlias_clause(ctx *pgparser.Alias_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *PgsqlVisitor) VisitOpt_alias_clause(ctx *pgparser.Opt_alias_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *PgsqlVisitor) VisitTable_alias_clause(ctx *pgparser.Table_alias_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *PgsqlVisitor) VisitFunc_alias_clause(ctx *pgparser.Func_alias_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *PgsqlVisitor) VisitRelation_expr(ctx *pgparser.Relation_exprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *PgsqlVisitor) VisitRelation_expr_list(ctx *pgparser.Relation_expr_listContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *PgsqlVisitor) VisitUpdatestmt(ctx *pgparser.UpdatestmtContext) interface{} {
	updateStmt := new(sqlstmt.UpdateStmt)
	updateStmt.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	updateStmt.TableSources = v.GetTableSourcesByrelation_expr_opt_alias(ctx.Relation_expr_opt_alias())
	updateStmt.UpdatedElements = ctx.Set_clause_list().Accept(v).([]*sqlstmt.UpdatedElement)
	if ec := ctx.Where_or_current_clause().A_expr(); ec != nil {
		updateStmt.Where = ec.Accept(v).(sqlstmt.IExpr)
	}

	return updateStmt
}

func (v *PgsqlVisitor) VisitSet_clause_list(ctx *pgparser.Set_clause_listContext) interface{} {
	ues := make([]*sqlstmt.UpdatedElement, 0)
	aucs := ctx.AllSet_clause()
	for _, auc := range aucs {
		ues = append(ues, auc.Accept(v).(*sqlstmt.UpdatedElement))
	}
	return ues
}

func (v *PgsqlVisitor) VisitSet_clause(ctx *pgparser.Set_clauseContext) interface{} {
	updateEle := new(sqlstmt.UpdatedElement)
	updateEle.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	updateEle.ColumnName = ctx.Set_target().Accept(v).(*sqlstmt.ColumnName)
	if ac := ctx.A_expr(); ac != nil {
		updateEle.Value = ac.Accept(v).(sqlstmt.IExpr)
	}
	return updateEle
}

func (v *PgsqlVisitor) VisitSet_target(ctx *pgparser.Set_targetContext) interface{} {
	columnName := new(sqlstmt.ColumnName)
	columnName.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	if ic := ctx.Opt_indirection(); ic != nil {
		if ic.GetText() == "" {
			columnName.Identifier = sqlstmt.NewIdentifierValue(ctx.Colid().GetText())
		} else {
			columnName.Owner = ctx.Colid().GetText()
			columnName.Identifier = sqlstmt.NewIdentifierValue(ic.GetText())
		}
	} else {
		columnName.Identifier = sqlstmt.NewIdentifierValue(ctx.Colid().GetText())
	}
	return columnName
}

func (v *PgsqlVisitor) VisitSet_target_list(ctx *pgparser.Set_target_listContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *PgsqlVisitor) VisitA_expr(ctx *pgparser.A_exprContext) interface{} {
	expr := new(sqlstmt.Expr)
	expr.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return expr
}

func (v *PgsqlVisitor) VisitA_expr_qual(ctx *pgparser.A_expr_qualContext) interface{} {
	expr := new(sqlstmt.Expr)
	expr.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	return expr
}

func (v *PgsqlVisitor) VisitDeletestmt(ctx *pgparser.DeletestmtContext) interface{} {
	deletestmt := new(sqlstmt.DeleteStmt)
	deletestmt.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	deletestmt.TableSources = v.GetTableSourcesByrelation_expr_opt_alias(ctx.Relation_expr_opt_alias())

	if ec := ctx.Where_or_current_clause().A_expr(); ec != nil {
		deletestmt.Where = ec.Accept(v).(sqlstmt.IExpr)
	}
	return deletestmt
}

func (v *PgsqlVisitor) VisitInsertstmt(ctx *pgparser.InsertstmtContext) interface{} {
	insertstmt := new(sqlstmt.InsertStmt)
	insertstmt.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	insertstmt.TableName = ctx.Insert_target().Accept(v).(*sqlstmt.TableName)
	return insertstmt
}

func (v *PgsqlVisitor) VisitInsert_target(ctx *pgparser.Insert_targetContext) interface{} {
	tableName := new(sqlstmt.TableName)
	tableName.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	table := ctx.GetText()
	if strings.Contains(table, ".") {
		tableAndOwner := strings.Split(table, ".")
		tableName.Identifier = sqlstmt.NewIdentifierValue(tableAndOwner[1])
		tableName.Owner = tableAndOwner[0]
	} else {
		tableName.Identifier = sqlstmt.NewIdentifierValue(table)
	}
	return tableName
}

func (v *PgsqlVisitor) VisitInsert_rest(ctx *pgparser.Insert_restContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *PgsqlVisitor) GetTableSourcesByrelation_expr_opt_alias(ctx pgparser.IRelation_expr_opt_aliasContext) *sqlstmt.TableSources {
	tableSources := new(sqlstmt.TableSources)
	tableSources.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	atomTable := new(sqlstmt.AtomTableItem)
	atomTable.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)

	if c := ctx.Relation_expr(); c != nil {
		tableName := new(sqlstmt.TableName)
		if qn := c.Qualified_name(); qn != nil {
			if qc := qn.Colid(); qc != nil {
				if c := qn.Indirection(); c != nil {
					tableName.Owner = qc.GetText()
					tableName.Identifier = sqlstmt.NewIdentifierValue(c.GetText())
				} else {
					tableName.Identifier = sqlstmt.NewIdentifierValue(qc.Identifier().GetText())
				}
			}
		}
		atomTable.TableName = tableName
	}

	if c := ctx.Colid(); c != nil {
		atomTable.Alias = c.GetText()
	}

	tableSourceBase := new(sqlstmt.TableSourceBase)
	tableSourceBase.Node = sqlstmt.NewNode(ctx.GetParser(), ctx)
	tableSourceBase.TableSourceItem = atomTable

	tableSources.TableSources = []sqlstmt.ITableSource{tableSourceBase}
	return tableSources
}
