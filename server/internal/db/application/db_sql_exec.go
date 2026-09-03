package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/config"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/imsg"
	flowapp "mayfly-go/internal/flow/application"
	flowentity "mayfly-go/internal/flow/domain/entity"
	msgdto "mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/pkg/event"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/stringx"
	"strings"
	"time"
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
}

var _ DbSqlExec = (*dbSqlExecAppImpl)(nil)

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

	var flowProcdef *flowentity.Procdef
	if execSqlReq.CheckFlow {
		flowProcdef = d.flowProcdefApp.GetProcdefByCodePath(ctx, dbConn.Info.CodePath...)
	}

	allExecRes := make([]*dto.DbSqlExecRes, 0)

	// 先使用方言切割器切割 SQL
	splitter := dbConn.GetDialect().GetSQLSplitter()
	var sqlList []string
	err := splitter.SplitSQL(strings.NewReader(execSql), func(oneSql string) error {
		sqlList = append(sqlList, oneSql)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("SQL 切割失败: %v", err)
	}

	// 获取解析器
	sp := dbConn.GetDialect().GetSQLParser()

	// 逐条解析并执行
	for _, sql := range sqlList {
		var execRes *dto.DbSqlExecRes
		var err error

		stmt, parseErr := sp.Parse(sql)
		dbSqlExecRecord := createSqlExecRecord(ctx, execSqlReq, sql)
		dbSqlExecRecord.Type = entity.DbSqlExecTypeOther
		sqlExec := &sqlExecParam{
			DbConn:        dbConn,
			Sql:           sql,
			Stmt:          stmt,
			Procdef:       flowProcdef,
			SqlExecRecord: dbSqlExecRecord,
		}

		// 优先使用 Stmt 类型判断，解析失败时使用字符串匹配兜底
		if parseErr != nil || stmt == nil {
			// 解析失败，使用字符串匹配兜底
			if isSelect(sql) {
				execRes, err = d.doSelect(ctx, sqlExec)
			} else if isUpdate(sql) {
				execRes, err = d.doUpdate(ctx, sqlExec)
			} else if isDelete(sql) {
				execRes, err = d.doDelete(ctx, sqlExec)
			} else if isInsert(sql) {
				execRes, err = d.doInsert(ctx, sqlExec)
			} else if isOtherQuery(sql) {
				execRes, err = d.doOtherRead(ctx, sqlExec)
			} else if isDDL(sql) {
				execRes, err = d.doExecDDL(ctx, sqlExec)
			} else {
				execRes, err = d.doExec(ctx, dbConn, sql)
			}
		} else {
			// 解析成功，使用 Stmt 类型判断
			switch stmt.(type) {
			case *sqlstmt.WithStmt:
				execRes, err = d.doSelect(ctx, sqlExec)
			case *sqlstmt.SelectStmt:
				execRes, err = d.doSelect(ctx, sqlExec)
			case *sqlstmt.UpdateStmt:
				execRes, err = d.doUpdate(ctx, sqlExec)
			case *sqlstmt.DeleteStmt:
				execRes, err = d.doDelete(ctx, sqlExec)
			case *sqlstmt.InsertStmt:
				execRes, err = d.doInsert(ctx, sqlExec)
			case *sqlstmt.DdlStmt:
				execRes, err = d.doExecDDL(ctx, sqlExec)
			case *sqlstmt.OtherStmt:
				execRes, err = d.doOtherRead(ctx, sqlExec)
			default:
				execRes, err = d.doExec(ctx, dbConn, sql)
			}
		}

		// 执行错误
		if err != nil {
			if execRes == nil {
				execRes = &dto.DbSqlExecRes{Sql: sql}
			}
			execRes.ErrorMsg = err.Error()
		} else {
			d.saveSqlExecLog(ctx, dbSqlExecRecord, dbSqlExecRecord.Res)
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
	needSendMsg := la != nil && clientId != "" && execReader.UploadId != ""

	startTime := time.Now()
	executedStatements := 0

	dbInfo := dbConn.Info
	msgEvent := &msgdto.MsgTmplSendEvent{
		TmplChannel: msgdto.MsgTmplSqlScriptRunSuccess,
		Params:      collx.M{"filename": filename, "dbId": dbInfo.Id, "dbName": dbInfo.Name},
	}

	progressMsgEvent := &msgdto.MsgTmplSendEvent{
		TmplChannel: msgdto.MsgTmplSqlScriptRunProgress,
		Params: collx.M{
			"id":                 execReader.UploadId,
			"dbCode":             dbInfo.DbCode,
			"dbName":             dbInfo.Name,
			"title":              filename,
			"executedStatements": executedStatements,
			"terminated":         false,
			"clientId":           clientId,
		},
	}

	if needSendMsg {
		msgEvent.ReceiverIds = []uint64{la.Id}
		progressMsgEvent.ReceiverIds = []uint64{la.Id}
	}

	defer func() {
		if needSendMsg {
			progressMsgEvent.Params["terminated"] = true
			global.EventBus.Publish(ctx, event.EventTopicMsgTmplSend, progressMsgEvent)
		}

		if err := recover(); err != nil {
			errInfo := anyx.ToString(err)
			logx.Errorf("exec sql reader error: %s", errInfo)
			if needSendMsg {
				errInfo = stringx.Truncate(errInfo, 300, 10, "...")
				msgEvent.TmplChannel = msgdto.MsgTmplSqlScriptRunFail
				msgEvent.Params["error"] = errInfo
				global.EventBus.Publish(ctx, event.EventTopicMsgTmplSend, msgEvent)
			}
		}
	}()

	tx, err := dbConn.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	// 使用方言切割器进行 SQL 切割
	splitter := dbConn.GetDialect().GetSQLSplitter()
	err = splitter.SplitSQL(execReader.Reader, func(sql string) error {
		// 检查context是否已取消
		if ctx.Err() != nil {
			if needSendMsg {
				progressMsgEvent.Params["executedStatements"] = executedStatements
				progressMsgEvent.Params["status"] = "cancelled"
				global.EventBus.Publish(ctx, event.EventTopicMsgTmplSend, progressMsgEvent)
			}
			return errorx.NewBizI(ctx, imsg.ErrSqlExecCancelled)
		}

		if executedStatements%50 == 0 {
			if needSendMsg {
				progressMsgEvent.Params["executedStatements"] = executedStatements
				global.EventBus.Publish(ctx, event.EventTopicMsgTmplSend, progressMsgEvent)
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
		if needSendMsg {
			msgEvent.TmplChannel = msgdto.MsgTmplSqlScriptRunFail
			msgEvent.Params["error"] = err.Error()
			global.EventBus.Publish(ctx, event.EventTopicMsgTmplSend, msgEvent)
		}
		return err
	}
	_ = tx.Commit()

	if needSendMsg {
		msgEvent.Params["cost"] = fmt.Sprintf("%dms", time.Since(startTime).Milliseconds())
		global.EventBus.Publish(ctx, event.EventTopicMsgTmplSend, msgEvent)
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

	execSqlBizForm, err := jsonx.ToByStr[FlowDbExecSqlBizForm](procinst.BizForm)
	if err != nil {
		return nil, errorx.NewBizf("failed to parse the business form information: %s", err.Error())
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

	// 不支持多表更新记录旧值
	if len(updatestmt.Tables) != 1 {
		logx.ErrorContext(ctx, "update SQL - logging old values only supports single-table updates")
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}

	tableName := updatestmt.Tables[0].Name
	tableAlias := updatestmt.Tables[0].Alias

	if tableName == "" {
		logx.ErrorContext(ctx, "update SQL - failed to get table name")
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}
	execRecord.Table = tableName

	if updatestmt.Where == nil {
		logx.ErrorContext(ctx, "update SQL - there is no where condition")
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}
	whereStr := updatestmt.Where.Text

	// 获取表主键列名,排除使用别名
	primaryKey, err := dbConn.GetMetadata().GetPrimaryKey(tableName)
	if err != nil {
		logx.ErrorfContext(ctx, "update SQL - failed to get primary key column: %s", err.Error())
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}

	updateColumns := collx.ArrayMap[sqlstmt.Assignment, string](updatestmt.Set, func(a sqlstmt.Assignment) string {
		return a.Column
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
			return errorx.NewBizf("update SQL - the maximum number of updated queries is exceeded: %d", maxRec)
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

	// 不支持多表删除记录旧值
	if len(deletestmt.Tables) != 1 {
		logx.ErrorContext(ctx, "delete SQL - logging old values only supports single-table deletion")
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}

	tableName := deletestmt.Tables[0].Name
	tableAlias := deletestmt.Tables[0].Alias

	if tableName == "" {
		logx.ErrorContext(ctx, "delete SQL - failed to get table name")
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}
	execRecord.Table = tableName

	if deletestmt.Where == nil {
		logx.ErrorContext(ctx, "delete SQL - there is no where condition")
		return d.doExec(ctx, dbConn, sqlExecParam.Sql)
	}

	whereStr := deletestmt.Where.Text
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

	execRecord.Table = insertstmt.Table.Name

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
			{Name: "rowsAffected", Key: "rowsAffected", Type: "number"},
		},
		Res: res,
		Sql: sql,
	}, err
}

func isSelect(sql string) bool {
	return strings.Contains(getSqlPrefix(sql), "select")
}

func isUpdate(sql string) bool {
	return strings.Contains(getSqlPrefix(sql), "update")
}

func isDelete(sql string) bool {
	return strings.Contains(getSqlPrefix(sql), "delete")
}

func isInsert(sql string) bool {
	return strings.Contains(getSqlPrefix(sql), "insert")
}

func isOtherQuery(sql string) bool {
	sqlPrefix := getSqlPrefix(sql)
	return strings.Contains(sqlPrefix, "explain") || strings.Contains(sqlPrefix, "show") || strings.Contains(sqlPrefix, "with")
}

func isDDL(sql string) bool {
	sqlPrefix := getSqlPrefix(sql)
	return strings.Contains(sqlPrefix, "create") || strings.Contains(sqlPrefix, "alter") ||
		strings.Contains(sqlPrefix, "drop") || strings.Contains(sqlPrefix, "truncate") || strings.Contains(sqlPrefix, "rename")
}

func getSqlPrefix(sql string) string {
	if len(sql) < 10 {
		return strings.ToLower(sql)
	}
	return strings.ToLower(sql[:10])
}
