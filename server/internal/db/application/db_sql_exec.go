package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/config"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/imsg"
	flowapp "mayfly-go/internal/flow/application"
	flowentity "mayfly-go/internal/flow/domain/entity"
	msgapp "mayfly-go/internal/msg/application"
	msgdto "mayfly-go/internal/msg/application/dto"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/ws"
	"strings"
)

type sqlExecParam struct {
	DbConn  *dbi.DbConn
	Sql     string              // 执行的sql
	Stmt    sqlstmt.Stmt        // 解析后的sql stmt
	Procdef *flowentity.Procdef // 流程定义

	SqlExecRecord *entity.DbSqlExec // sql执行记录
}

// progressCategory sql文件执行进度消息类型
const progressCategory = "execSqlFileProgress"

// progressMsg sql文件执行进度消息
type progressMsg struct {
	Id                 string `json:"id"`
	Title              string `json:"title"`
	ExecutedStatements int    `json:"executedStatements"`
	Terminated         bool   `json:"terminated"`
}

type DbSqlExec interface {
	flowapp.FlowBizHandler

	// 执行sql
	Exec(ctx context.Context, execSqlReq *dto.DbSqlExecReq) ([]*dto.DbSqlExecRes, error)

	// ExecReader 从reader中读取sql并执行
	ExecReader(ctx context.Context, execReader *dto.SqlReaderExec) error

	// 根据条件删除sql执行记录
	DeleteBy(ctx context.Context, condition *entity.DbSqlExec) error

	// 分页获取
	GetPageList(condition *entity.DbSqlExecQuery, orderBy ...string) (*model.PageResult[*entity.DbSqlExec], error)
}

var _ (DbSqlExec) = (*dbSqlExecAppImpl)(nil)

type dbSqlExecAppImpl struct {
	dbApp         Db                   `inject:"T"`
	dbSqlExecRepo repository.DbSqlExec `inject:"T"`

	flowProcdefApp flowapp.Procdef `inject:"T"`
	msgApp         msgapp.Msg      `inject:"T"`
}

func createSqlExecRecord(ctx context.Context, execSqlReq *dto.DbSqlExecReq, sql string) *entity.DbSqlExec {
	dbSqlExecRecord := new(entity.DbSqlExec)
	dbSqlExecRecord.DbId = execSqlReq.DbId
	dbSqlExecRecord.Db = execSqlReq.Db
	dbSqlExecRecord.Sql = sql
	dbSqlExecRecord.Remark = execSqlReq.Remark
	dbSqlExecRecord.Status = entity.DbSqlExecStatusSuccess
	return dbSqlExecRecord
}

func (d *dbSqlExecAppImpl) Exec(ctx context.Context, execSqlReq *dto.DbSqlExecReq) ([]*dto.DbSqlExecRes, error) {
	dbConn := execSqlReq.DbConn
	execSql := execSqlReq.Sql
	sp := dbConn.GetDialect().GetSQLParser()

	var flowProcdef *flowentity.Procdef
	if execSqlReq.CheckFlow {
		flowProcdef = d.flowProcdefApp.GetProcdefByCodePath(ctx, dbConn.Info.CodePath...)
	}

	allExecRes := make([]*dto.DbSqlExecRes, 0)

	stmts, err := sp.Parse(execSql)
	// sql解析失败，则使用默认方式切割
	if err != nil {
		sqlparser.SQLSplit(strings.NewReader(execSql), func(oneSql string) error {
			var execRes *dto.DbSqlExecRes
			var err error

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
			} else if isDDL(oneSql) {
				execRes, err = d.doExecDDL(ctx, sqlExec)
			} else {
				execRes, err = d.doExec(ctx, dbConn, oneSql)
			}
			// 执行错误
			if err != nil {
				if execRes == nil {
					execRes = &dto.DbSqlExecRes{Sql: oneSql}
				}
				execRes.ErrorMsg = err.Error()
			} else {
				d.saveSqlExecLog(ctx, dbSqlExecRecord, dbSqlExecRecord.Res)
			}
			allExecRes = append(allExecRes, execRes)
			return nil
		})
		return allExecRes, nil
	}

	// mysql parser with语句会分解析为两条，故需要特殊处理
	currentWithSql := ""
	for _, stmt := range stmts {
		var execRes *dto.DbSqlExecRes
		var err error

		sql := stmt.GetText()
		dbSqlExecRecord := createSqlExecRecord(ctx, execSqlReq, sql)
		dbSqlExecRecord.Type = entity.DbSqlExecTypeOther
		sqlExec := &sqlExecParam{DbConn: dbConn, Sql: currentWithSql + sql, Procdef: flowProcdef, Stmt: stmt, SqlExecRecord: dbSqlExecRecord}
		currentWithSql = ""

		switch stmt.(type) {
		case *sqlstmt.SimpleSelectStmt:
			execRes, err = d.doSelect(ctx, sqlExec)
		case *sqlstmt.UnionSelectStmt:
			execRes, err = d.doSelect(ctx, sqlExec)
		case *sqlstmt.OtherReadStmt:
			execRes, err = d.doOtherRead(ctx, sqlExec)
		case *sqlstmt.WithStmt:
			currentWithSql = sql
		case *sqlstmt.UpdateStmt:
			execRes, err = d.doUpdate(ctx, sqlExec)
		case *sqlstmt.DeleteStmt:
			execRes, err = d.doDelete(ctx, sqlExec)
		case *sqlstmt.InsertStmt:
			execRes, err = d.doInsert(ctx, sqlExec)
		case *sqlstmt.DdlStmt:
			execRes, err = d.doExecDDL(ctx, sqlExec)
		case *sqlstmt.CreateDatabase:
			execRes, err = d.doExecDDL(ctx, sqlExec)
		case *sqlstmt.CreateTable:
			execRes, err = d.doExecDDL(ctx, sqlExec)
		case *sqlstmt.CreateIndex:
			execRes, err = d.doExecDDL(ctx, sqlExec)
		case *sqlstmt.AlterDatabase:
			execRes, err = d.doExecDDL(ctx, sqlExec)
		case *sqlstmt.AlterTable:
			execRes, err = d.doExecDDL(ctx, sqlExec)
		default:
			execRes, err = d.doExec(ctx, dbConn, sql)
		}

		if currentWithSql != "" {
			continue
		}

		if err != nil {
			if execRes == nil {
				execRes = &dto.DbSqlExecRes{Sql: sqlExec.Sql}
			}
			execRes.ErrorMsg = err.Error()
		} else {
			d.saveSqlExecLog(ctx, dbSqlExecRecord, execRes.Res)
		}
		allExecRes = append(allExecRes, execRes)
	}

	return allExecRes, nil
}

func (d *dbSqlExecAppImpl) ExecReader(ctx context.Context, execReader *dto.SqlReaderExec) error {
	dbConn := execReader.DbConn

	clientId := execReader.ClientId
	filename := stringx.Truncate(execReader.Filename, 20, 10, "...")
	la := contextx.GetLoginAccount(ctx)
	needSendMsg := la != nil && clientId != ""

	defer func() {
		if err := recover(); err != nil {
			errInfo := anyx.ToString(err)
			logx.Errorf("exec sql reader error: %s", errInfo)
			if needSendMsg {
				errInfo = stringx.Truncate(errInfo, 300, 10, "...")
				d.msgApp.CreateAndSend(la, msgdto.ErrSysMsg(i18n.T(imsg.SqlScriptRunFail), fmt.Sprintf("[%s][%s] execution failure: [%s]", filename, dbConn.Info.GetLogDesc(), errInfo)).WithClientId(clientId))
			}
		}
	}()

	executedStatements := 0
	progressId := stringx.Rand(32)
	if needSendMsg {
		defer ws.SendJsonMsg(ws.UserId(la.Id), clientId, msgdto.InfoSysMsg(i18n.T(imsg.SqlScripRunProgress), &progressMsg{
			Id:                 progressId,
			Title:              filename,
			ExecutedStatements: executedStatements,
			Terminated:         true,
		}).WithCategory(progressCategory))
	}

	tx, _ := dbConn.Begin()
	err := sqlparser.SQLSplit(execReader.Reader, func(sql string) error {
		if executedStatements%50 == 0 {
			if needSendMsg {
				ws.SendJsonMsg(ws.UserId(la.Id), clientId, msgdto.InfoSysMsg(i18n.T(imsg.SqlScripRunProgress), &progressMsg{
					Id:                 progressId,
					Title:              filename,
					ExecutedStatements: executedStatements,
					Terminated:         false,
				}).WithCategory(progressCategory))
			}
		}

		executedStatements++
		if _, err := dbConn.TxExec(tx, sql); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	_ = tx.Commit()

	if needSendMsg {
		d.msgApp.CreateAndSend(la, msgdto.SuccessSysMsg(i18n.T(imsg.SqlScriptRunSuccess), "execution success").WithClientId(clientId))
	}
	return nil
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

	execSqlBizForm, err := jsonx.To[*FlowDbExecSqlBizForm](procinst.BizForm)
	if err != nil {
		return nil, errorx.NewBiz("failed to parse the business form information: %s", err.Error())
	}

	dbConn, err := d.dbApp.GetDbConn(ctx, execSqlBizForm.DbId, execSqlBizForm.DbName)
	if err != nil {
		return nil, err
	}

	execRes, err := d.Exec(contextx.NewLoginAccount(&model.LoginAccount{Id: procinst.CreatorId, Username: procinst.Creator}), &dto.DbSqlExecReq{
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
			return execRes, errorx.NewBizI(ctx, imsg.ErrExistRunFailSql)
		}
	}

	return execRes, nil
}

func (d *dbSqlExecAppImpl) DeleteBy(ctx context.Context, condition *entity.DbSqlExec) error {
	return d.dbSqlExecRepo.DeleteByCond(ctx, condition)
}

func (d *dbSqlExecAppImpl) GetPageList(condition *entity.DbSqlExecQuery, orderBy ...string) (*model.PageResult[*entity.DbSqlExec], error) {
	return d.dbSqlExecRepo.GetPageList(condition, orderBy...)
}

// 保存sql执行记录，如果是查询类则根据系统配置判断是否保存
func (d *dbSqlExecAppImpl) saveSqlExecLog(ctx context.Context, dbSqlExecRecord *entity.DbSqlExec, res any) {
	if dbSqlExecRecord.Type != entity.DbSqlExecTypeQuery {
		dbSqlExecRecord.Res = jsonx.ToStr(res)
		d.dbSqlExecRepo.Insert(ctx, dbSqlExecRecord)
		return
	}

	if config.GetDbms().QuerySqlSave {
		dbSqlExecRecord.Table = "-"
		dbSqlExecRecord.OldValue = "-"
		dbSqlExecRecord.Type = entity.DbSqlExecTypeQuery
		d.dbSqlExecRepo.Insert(ctx, dbSqlExecRecord)
	}
}

func (d *dbSqlExecAppImpl) doSelect(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSqlExecRes, error) {
	maxCount := config.GetDbms().MaxResultSet
	selectSql := sqlExecParam.Sql
	sqlExecParam.SqlExecRecord.Type = entity.DbSqlExecTypeQuery

	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSqlExecFlowBizType, collx.Kvs("stmtType", "select")); needStartProc {
			return nil, errorx.NewBizI(ctx, imsg.ErrNeedSubmitWorkTicket)
		}
	}

	return d.doQuery(ctx, sqlExecParam.DbConn, selectSql, maxCount)
}

func (d *dbSqlExecAppImpl) doOtherRead(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSqlExecRes, error) {
	selectSql := sqlExecParam.Sql
	sqlExecParam.SqlExecRecord.Type = entity.DbSqlExecTypeQuery

	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSqlExecFlowBizType, collx.Kvs("stmtType", "read")); needStartProc {
			return nil, errorx.NewBizI(ctx, imsg.ErrNeedSubmitWorkTicket)
		}
	}

	return d.doQuery(ctx, sqlExecParam.DbConn, selectSql, 0)
}

func (d *dbSqlExecAppImpl) doExecDDL(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSqlExecRes, error) {
	selectSql := sqlExecParam.Sql
	sqlExecParam.SqlExecRecord.Type = entity.DbSqlExecTypeDDL

	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSqlExecFlowBizType, collx.Kvs("stmtType", "ddl")); needStartProc {
			return nil, errorx.NewBizI(ctx, imsg.ErrNeedSubmitWorkTicket)
		}
	}

	return d.doExec(ctx, sqlExecParam.DbConn, selectSql)
}

func (d *dbSqlExecAppImpl) doUpdate(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSqlExecRes, error) {
	dbConn := sqlExecParam.DbConn

	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSqlExecFlowBizType, collx.Kvs("stmtType", "update")); needStartProc {
			return nil, errorx.NewBizI(ctx, imsg.ErrNeedSubmitWorkTicket)
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
		logx.ErrorContext(ctx, "update SQL - logging old values only supports single-table updates")
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
		logx.ErrorContext(ctx, "update SQL - failed to get table name")
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}
	execRecord.Table = tableName

	if updatestmt.Where == nil {
		logx.ErrorContext(ctx, "update SQL - there is no where condition")
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}
	whereStr := updatestmt.Where.GetText()

	// 获取表主键列名,排除使用别名
	primaryKey, err := dbConn.GetMetadata().GetPrimaryKey(tableName)
	if err != nil {
		logx.ErrorfContext(ctx, "update SQL - failed to get primary key column: %s", err.Error())
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
			return errorx.NewBiz("update SQL - the maximum number of updated queries is exceeded: %d", maxRec)
		}
		return nil
	})
	if err != nil {
		logx.ErrorfContext(ctx, "update SQL - failed to get the updated old value: %s", err.Error())
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}
	execRecord.OldValue = jsonx.ToStr(res)

	return d.doExec(ctx, dbConn, sqlExecParam.Sql)
}

func (d *dbSqlExecAppImpl) doDelete(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSqlExecRes, error) {
	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSqlExecFlowBizType, collx.Kvs("stmtType", "delete")); needStartProc {
			return nil, errorx.NewBizI(ctx, imsg.ErrNeedSubmitWorkTicket)
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
		logx.ErrorContext(ctx, "delete SQL - logging old values only supports single-table deletion")
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
		logx.ErrorContext(ctx, "delete SQL - failed to get table name")
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}
	execRecord.Table = tableName

	deleteWhere := deletestmt.Where
	if deleteWhere == nil {
		logx.ErrorContext(ctx, "delete SQL - there is no where condition")
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}

	whereStr := deleteWhere.GetText()
	// 查询删除数据
	selectSql := fmt.Sprintf("SELECT * FROM %s where %s LIMIT 200", tableName+" "+tableAlias, whereStr)
	_, res, _ := dbConn.QueryContext(ctx, selectSql)
	execRecord.OldValue = jsonx.ToStr(res)

	return d.doExec(ctx, dbConn, sqlExecParam.Sql)
}

func (d *dbSqlExecAppImpl) doInsert(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSqlExecRes, error) {
	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSqlExecFlowBizType, collx.Kvs("stmtType", "insert")); needStartProc {
			return nil, errorx.NewBizI(ctx, imsg.ErrNeedSubmitWorkTicket)
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

func (d *dbSqlExecAppImpl) doQuery(ctx context.Context, dbConn *dbi.DbConn, sql string, maxRows int) (*dto.DbSqlExecRes, error) {
	res := make([]map[string]any, 0, 16)
	nowRows := 0
	cols, err := dbConn.WalkQueryRows(ctx, sql, func(row map[string]any, columns []*dbi.QueryColumn) error {
		nowRows++
		// 超过指定的最大查询记录数，则停止查询
		if maxRows != 0 && nowRows > maxRows {
			return dbi.NewStopWalkQueryError(fmt.Sprintf("exceed the maximum number of query records %d", maxRows))
		}
		res = append(res, row)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &dto.DbSqlExecRes{
		Sql:     sql,
		Columns: cols,
		Res:     res,
	}, nil
}

func (d *dbSqlExecAppImpl) doExec(ctx context.Context, dbConn *dbi.DbConn, sql string) (*dto.DbSqlExecRes, error) {
	rowsAffected, err := dbConn.ExecContext(ctx, sql)
	if err != nil {
		return nil, err
	}

	res := make([]map[string]any, 0)
	res = append(res, collx.Kvs("rowsAffected", rowsAffected))

	return &dto.DbSqlExecRes{
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
	return strings.Contains(sqlPrefix, "explain") || strings.Contains(sqlPrefix, "show") || strings.Contains(sqlPrefix, "with")
}

func isDDL(sql string) bool {
	sqlPrefix := strings.ToLower(sql[:10])
	return strings.Contains(sqlPrefix, "create") || strings.Contains(sqlPrefix, "alter") ||
		strings.Contains(sqlPrefix, "drop") || strings.Contains(sqlPrefix, "truncate") || strings.Contains(sqlPrefix, "rename")
}
