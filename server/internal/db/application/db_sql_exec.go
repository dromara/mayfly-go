package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/db/config"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	flowapp "mayfly-go/internal/flow/application"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"strings"
)

type DbSqlExecReq struct {
	DbId      uint64
	Db        string
	Sql       string // 需要执行的sql，支持多条
	Remark    string // 执行备注
	DbConn    *dbi.DbConn
	CheckFlow bool // 是否检查存储审批流程
}

type sqlExecParam struct {
	DbConn  *dbi.DbConn
	Sql     string              // 执行的sql
	Stmt    sqlstmt.Stmt        // 解析后的sql stmt
	Procdef *flowentity.Procdef // 流程定义

	SqlExecRecord *entity.DbSqlExec // sql执行记录
}

type DbSqlExecRes struct {
	Sql      string             `json:"sql"`      // 执行的sql
	ErrorMsg string             `json:"errorMsg"` // 若执行失败，则将失败内容记录到该字段
	Columns  []*dbi.QueryColumn `json:"columns"`  // 响应的列信息
	Res      []map[string]any   `json:"res"`      // 响应结果
}

type DbSqlExec interface {
	flowapp.FlowBizHandler

	// 执行sql
	Exec(ctx context.Context, execSqlReq *DbSqlExecReq) ([]*DbSqlExecRes, error)

	// 根据条件删除sql执行记录
	DeleteBy(ctx context.Context, condition *entity.DbSqlExec) error

	// 分页获取
	GetPageList(condition *entity.DbSqlExecQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}

type dbSqlExecAppImpl struct {
	dbApp         Db                   `inject:"DbApp"`
	dbSqlExecRepo repository.DbSqlExec `inject:"DbSqlExecRepo"`

	flowProcdefApp flowapp.Procdef `inject:"ProcdefApp"`
}

func createSqlExecRecord(ctx context.Context, execSqlReq *DbSqlExecReq, sql string) *entity.DbSqlExec {
	dbSqlExecRecord := new(entity.DbSqlExec)
	dbSqlExecRecord.DbId = execSqlReq.DbId
	dbSqlExecRecord.Db = execSqlReq.Db
	dbSqlExecRecord.Sql = sql
	dbSqlExecRecord.Remark = execSqlReq.Remark
	dbSqlExecRecord.Status = entity.DbSqlExecStatusSuccess
	dbSqlExecRecord.FillBaseInfo(model.IdGenTypeNone, contextx.GetLoginAccount(ctx))
	return dbSqlExecRecord
}

func (d *dbSqlExecAppImpl) Exec(ctx context.Context, execSqlReq *DbSqlExecReq) ([]*DbSqlExecRes, error) {
	dbConn := execSqlReq.DbConn
	execSql := execSqlReq.Sql
	sp := dbConn.GetDialect().GetSQLParser()

	var flowProcdef *flowentity.Procdef
	if execSqlReq.CheckFlow {
		flowProcdef = d.flowProcdefApp.GetProcdefByCodePath(ctx, dbConn.Info.CodePath...)
	}

	allExecRes := make([]*DbSqlExecRes, 0)

	stmts, err := sp.Parse(execSql)
	// sql解析失败，则直接使用;切割进行执行
	if err != nil {
		sqlScan := sqlparser.SplitSqls(strings.NewReader(execSql))
		for sqlScan.Scan() {
			var execRes *DbSqlExecRes
			var err error

			oneSql := sqlScan.Text()
			dbSqlExecRecord := createSqlExecRecord(ctx, execSqlReq, oneSql)
			dbSqlExecRecord.Type = entity.DbSqlExecTypeOther
			sqlExec := &sqlExecParam{DbConn: dbConn, Sql: oneSql, Procdef: flowProcdef, SqlExecRecord: dbSqlExecRecord}

			if isSelect(oneSql) {
				execRes, err = d.doSelect(ctx, sqlExec)
			} else if isUpdate(oneSql) {
				execRes, err = d.doUpdate(ctx, sqlExec)
			} else if isDelete(oneSql) {
				execRes, err = d.doDelete(ctx, sqlExec)
			} else if isInsert(oneSql) {
				execRes, err = d.doInsert(ctx, sqlExec)
			} else if isOtherQuery(oneSql) {
				execRes, err = d.doOtherRead(ctx, sqlExec)
			} else {
				execRes, err = d.doExec(ctx, dbConn, oneSql)
			}
			// 执行错误
			if err != nil {
				if execRes == nil {
					execRes = &DbSqlExecRes{Sql: oneSql}
				}
				execRes.ErrorMsg = err.Error()
			} else {
				d.saveSqlExecLog(dbSqlExecRecord, dbSqlExecRecord.Res)
			}
			allExecRes = append(allExecRes, execRes)
		}

		return allExecRes, nil
	}

	for _, stmt := range stmts {
		var execRes *DbSqlExecRes
		var err error

		sql := stmt.GetText()
		dbSqlExecRecord := createSqlExecRecord(ctx, execSqlReq, sql)
		dbSqlExecRecord.Type = entity.DbSqlExecTypeOther
		sqlExec := &sqlExecParam{DbConn: dbConn, Sql: sql, Procdef: flowProcdef, Stmt: stmt, SqlExecRecord: dbSqlExecRecord}

		switch stmt.(type) {
		case *sqlstmt.SimpleSelectStmt:
			execRes, err = d.doSelect(ctx, sqlExec)
		case *sqlstmt.UnionSelectStmt:
			execRes, err = d.doSelect(ctx, sqlExec)
		case *sqlstmt.OtherReadStmt:
			execRes, err = d.doOtherRead(ctx, sqlExec)
		case *sqlstmt.UpdateStmt:
			execRes, err = d.doUpdate(ctx, sqlExec)
		case *sqlstmt.DeleteStmt:
			execRes, err = d.doDelete(ctx, sqlExec)
		case *sqlstmt.InsertStmt:
			execRes, err = d.doInsert(ctx, sqlExec)
		default:
			execRes, err = d.doExec(ctx, dbConn, sql)
		}

		if err != nil {
			if execRes == nil {
				execRes = &DbSqlExecRes{Sql: sql}
			}
			execRes.ErrorMsg = err.Error()
		} else {
			d.saveSqlExecLog(dbSqlExecRecord, execRes.Res)
		}
		allExecRes = append(allExecRes, execRes)
	}

	return allExecRes, nil
}

type FlowDbExecSqlBizForm struct {
	DbId   uint64 `json:"dbId"`   //  库id
	DbName string `json:"dbName"` // 库名
	Sql    string `json:"sql"`    // sql
}

func (d *dbSqlExecAppImpl) FlowBizHandle(ctx context.Context, bizHandleParam *flowapp.BizHandleParam) (any, error) {
	procinst := bizHandleParam.Procinst
	bizKey := procinst.BizKey
	procinstStatus := procinst.Status

	logx.Debugf("DbSqlExec FlowBizHandle -> bizKey: %s, procinstStatus: %s", bizKey, flowentity.ProcinstStatusEnum.GetDesc(procinstStatus))
	// 流程非完成状态不处理
	if procinstStatus != flowentity.ProcinstStatusCompleted {
		return nil, nil
	}

	execSqlBizForm, err := jsonx.To(procinst.BizForm, new(FlowDbExecSqlBizForm))
	if err != nil {
		return nil, errorx.NewBiz("业务表单信息解析失败: %s", err.Error())
	}

	dbConn, err := d.dbApp.GetDbConn(execSqlBizForm.DbId, execSqlBizForm.DbName)
	if err != nil {
		return nil, err
	}

	execRes, err := d.Exec(contextx.NewLoginAccount(&model.LoginAccount{Id: procinst.CreatorId, Username: procinst.Creator}), &DbSqlExecReq{
		DbId:      execSqlBizForm.DbId,
		Db:        execSqlBizForm.DbName,
		Sql:       execSqlBizForm.Sql,
		DbConn:    dbConn,
		Remark:    procinst.Remark,
		CheckFlow: false,
	})
	if err != nil {
		return nil, err
	}

	// 存在一条错误的sql，则表示业务处理失败
	for _, er := range execRes {
		if er.ErrorMsg != "" {
			return execRes, errorx.NewBiz("存在执行错误的sql")
		}
	}

	return execRes, nil
}

func (d *dbSqlExecAppImpl) DeleteBy(ctx context.Context, condition *entity.DbSqlExec) error {
	return d.dbSqlExecRepo.DeleteByCond(ctx, condition)
}

func (d *dbSqlExecAppImpl) GetPageList(condition *entity.DbSqlExecQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return d.dbSqlExecRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

// 保存sql执行记录，如果是查询类则根据系统配置判断是否保存
func (d *dbSqlExecAppImpl) saveSqlExecLog(dbSqlExecRecord *entity.DbSqlExec, res any) {
	if dbSqlExecRecord.Type != entity.DbSqlExecTypeQuery {
		dbSqlExecRecord.Res = jsonx.ToStr(res)
		d.dbSqlExecRepo.Insert(context.TODO(), dbSqlExecRecord)
		return
	}

	if config.GetDbms().QuerySqlSave {
		dbSqlExecRecord.Table = "-"
		dbSqlExecRecord.OldValue = "-"
		dbSqlExecRecord.Type = entity.DbSqlExecTypeQuery
		d.dbSqlExecRepo.Insert(context.TODO(), dbSqlExecRecord)
	}
}

func (d *dbSqlExecAppImpl) doSelect(ctx context.Context, sqlExecParam *sqlExecParam) (*DbSqlExecRes, error) {
	maxCount := config.GetDbms().MaxResultSet
	selectStmt := sqlExecParam.Stmt
	selectSql := sqlExecParam.Sql
	sqlExecParam.SqlExecRecord.Type = entity.DbSqlExecTypeQuery

	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSqlExecFlowBizType, collx.Kvs("stmtType", "select")); needStartProc {
			return nil, errorx.NewBiz("该操作需要提交工单审批执行")
		}
	}

	if selectStmt != nil {
		needCheckLimit := false
		var limit *sqlstmt.Limit
		switch stmt := selectStmt.(type) {
		case *sqlstmt.SimpleSelectStmt:
			qs := stmt.QuerySpecification
			limit = qs.Limit
			if qs.SelectElements.Star != "" || len(qs.SelectElements.Elements) > 1 {
				needCheckLimit = true
			}
		case *sqlstmt.UnionSelectStmt:
			limit = stmt.Limit
			selectSql = selectStmt.GetText()
			needCheckLimit = true
		}

		// 如果配置为0，则不校验分页参数
		if needCheckLimit && maxCount != 0 {
			if limit == nil {
				return nil, errorx.NewBiz("请完善分页信息后执行")
			}
			if limit.RowCount > maxCount {
				return nil, errorx.NewBiz("查询结果集数需小于系统配置的%d条", maxCount)
			}
		}
	} else {
		if maxCount != 0 {
			if !strings.Contains(selectSql, "limit") &&
				// 兼容oracle rownum分页
				!strings.Contains(selectSql, "rownum") &&
				// 兼容mssql offset分页
				!strings.Contains(selectSql, "offset") &&
				// 兼容mssql top 分页  with result as ({query sql}) select top 100 * from result
				!strings.Contains(selectSql, " top ") {
				// 判断是不是count语句
				if !strings.Contains(selectSql, "count(") {
					return nil, errorx.NewBiz("请完善分页信息后执行")
				}
			}
		}
	}

	return d.doQuery(ctx, sqlExecParam.DbConn, selectSql)
}

func (d *dbSqlExecAppImpl) doOtherRead(ctx context.Context, sqlExecParam *sqlExecParam) (*DbSqlExecRes, error) {
	selectSql := sqlExecParam.Sql
	sqlExecParam.SqlExecRecord.Type = entity.DbSqlExecTypeQuery

	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSqlExecFlowBizType, collx.Kvs("stmtType", "read")); needStartProc {
			return nil, errorx.NewBiz("该操作需要提交工单审批执行")
		}
	}

	return d.doQuery(ctx, sqlExecParam.DbConn, selectSql)
}

func (d *dbSqlExecAppImpl) doUpdate(ctx context.Context, sqlExecParam *sqlExecParam) (*DbSqlExecRes, error) {
	dbConn := sqlExecParam.DbConn

	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSqlExecFlowBizType, collx.Kvs("stmtType", "update")); needStartProc {
			return nil, errorx.NewBiz("该操作需要提交工单审批执行")
		}
	}

	execRecord := sqlExecParam.SqlExecRecord
	execRecord.Type = entity.DbSqlExecTypeUpdate

	stmt := sqlExecParam.Stmt
	if stmt == nil {
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}

	updatestmt, ok := stmt.(*sqlstmt.UpdateStmt)
	if !ok {
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}

	tableSources := updatestmt.TableSources.TableSources
	// 不支持多表更新记录旧值
	if len(tableSources) != 1 {
		logx.ErrorContext(ctx, "Update SQL - 记录旧值只支持单表更新")
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}

	tableName := ""
	tableAlias := ""
	if tableSourceBase, ok := tableSources[0].(*sqlstmt.TableSourceBase); ok {
		if atmoTableItem, ok := tableSourceBase.TableSourceItem.(*sqlstmt.AtomTableItem); ok {
			tableName = atmoTableItem.TableName.Identifier.Value
			tableAlias = atmoTableItem.Alias
		}
	}

	if tableName == "" {
		logx.ErrorContext(ctx, "Update SQL - 获取表名失败")
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}
	execRecord.Table = tableName

	whereStr := updatestmt.Where.GetText()
	if whereStr == "" {
		logx.ErrorContext(ctx, "Update SQL - 不存在where条件")
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}

	// 获取表主键列名,排除使用别名
	primaryKey, err := dbConn.GetMetaData().GetPrimaryKey(tableName)
	if err != nil {
		logx.ErrorfContext(ctx, "Update SQL - 获取主键列失败: %s", err.Error())
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}

	updateColumns := collx.ArrayMap[*sqlstmt.UpdatedElement, string](updatestmt.UpdatedElements, func(ue *sqlstmt.UpdatedElement) string {
		return ue.ColumnName.GetText()
	})

	primaryKeyColumn := primaryKey
	if tableAlias != "" {
		primaryKeyColumn = tableAlias + "." + primaryKey
	}
	updateColumnsAndPrimaryKey := strings.Join(updateColumns, ",") + "," + primaryKeyColumn
	// 查询要更新字段数据的旧值，以及主键值
	selectSql := fmt.Sprintf("SELECT %s FROM %s where %s", updateColumnsAndPrimaryKey, tableName+" "+tableAlias, whereStr)

	// WalkQuery查出最多200条数据
	maxRec := 200
	nowRec := 0
	res := make([]map[string]any, 0)
	_, err = dbConn.WalkQueryRows(ctx, selectSql, func(row map[string]any, columns []*dbi.QueryColumn) error {
		nowRec++
		res = append(res, row)
		if nowRec == maxRec {
			return errorx.NewBiz(fmt.Sprintf("Update SQL -超出更新最大查询条数限制: %d", maxRec))
		}
		return nil
	})
	if err != nil {
		logx.ErrorfContext(ctx, "Update SQL - 获取更新旧值失败: %s", err.Error())
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}
	execRecord.OldValue = jsonx.ToStr(res)

	return d.doExec(ctx, dbConn, sqlExecParam.Sql)
}

func (d *dbSqlExecAppImpl) doDelete(ctx context.Context, sqlExecParam *sqlExecParam) (*DbSqlExecRes, error) {
	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSqlExecFlowBizType, collx.Kvs("stmtType", "delete")); needStartProc {
			return nil, errorx.NewBiz("该操作需要提交工单审批执行")
		}
	}

	dbConn := sqlExecParam.DbConn
	execRecord := sqlExecParam.SqlExecRecord
	execRecord.Type = entity.DbSqlExecTypeDelete

	stmt := sqlExecParam.Stmt
	if stmt == nil {
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}

	deletestmt, ok := stmt.(*sqlstmt.DeleteStmt)
	if !ok {
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}

	tableSources := deletestmt.TableSources.TableSources
	// 不支持多表删除记录旧值
	if len(tableSources) != 1 {
		logx.ErrorContext(ctx, "Delete SQL - 记录旧值只支持单表删除")
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}

	tableName := ""
	tableAlias := ""
	if tableSourceBase, ok := tableSources[0].(*sqlstmt.TableSourceBase); ok {
		if atmoTableItem, ok := tableSourceBase.TableSourceItem.(*sqlstmt.AtomTableItem); ok {
			tableName = atmoTableItem.TableName.Identifier.Value
			tableAlias = atmoTableItem.Alias
		}
	}

	if tableName == "" {
		logx.ErrorContext(ctx, "Delete SQL - 获取表名失败")
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}
	execRecord.Table = tableName

	whereStr := deletestmt.Where.GetText()
	if whereStr == "" {
		logx.ErrorContext(ctx, "Delete SQL - 不存在where条件")
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}

	// 查询删除数据
	selectSql := fmt.Sprintf("SELECT * FROM %s where %s LIMIT 200", tableName+" "+tableAlias, whereStr)
	_, res, _ := dbConn.QueryContext(ctx, selectSql)
	execRecord.OldValue = jsonx.ToStr(res)

	return d.doExec(ctx, dbConn, sqlExecParam.Sql)
}

func (d *dbSqlExecAppImpl) doInsert(ctx context.Context, sqlExecParam *sqlExecParam) (*DbSqlExecRes, error) {
	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSqlExecFlowBizType, collx.Kvs("stmtType", "insert")); needStartProc {
			return nil, errorx.NewBiz("该操作需要提交工单审批执行")
		}
	}

	dbConn := sqlExecParam.DbConn
	execRecord := sqlExecParam.SqlExecRecord
	execRecord.Type = entity.DbSqlExecTypeInsert

	stmt := sqlExecParam.Stmt
	if stmt == nil {
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}

	insertstmt, ok := stmt.(*sqlstmt.InsertStmt)
	if !ok {
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}

	execRecord.Table = insertstmt.TableName.Identifier.Value

	return d.doExec(ctx, sqlExecParam.DbConn, sqlExecParam.Sql)
}

func (d *dbSqlExecAppImpl) doQuery(ctx context.Context, dbConn *dbi.DbConn, sql string) (*DbSqlExecRes, error) {
	cols, res, err := dbConn.QueryContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	return &DbSqlExecRes{
		Sql:     sql,
		Columns: cols,
		Res:     res,
	}, nil
}

func (d *dbSqlExecAppImpl) doExec(ctx context.Context, dbConn *dbi.DbConn, sql string) (*DbSqlExecRes, error) {
	rowsAffected, err := dbConn.ExecContext(ctx, sql)
	if err != nil {
		return nil, err
	}

	res := make([]map[string]any, 0)
	res = append(res, collx.Kvs("rowsAffected", rowsAffected))

	return &DbSqlExecRes{
		Columns: []*dbi.QueryColumn{
			{Name: "rowsAffected", Type: "number"},
		},
		Res: res,
		Sql: sql,
	}, err
}

func isSelect(sql string) bool {
	return strings.Contains(strings.ToLower(sql[:10]), "select")
}

func isUpdate(sql string) bool {
	return strings.Contains(strings.ToLower(sql[:10]), "update")
}

func isDelete(sql string) bool {
	return strings.Contains(strings.ToLower(sql[:10]), "delete")
}

func isInsert(sql string) bool {
	return strings.Contains(strings.ToLower(sql[:10]), "insert")
}

func isOtherQuery(sql string) bool {
	sqlPrefix := strings.ToLower(sql[:10])
	return strings.Contains(sqlPrefix, "explain") || strings.Contains(sqlPrefix, "show")
}
