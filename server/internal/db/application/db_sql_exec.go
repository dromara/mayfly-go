package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/application/mask"
	"mayfly-go/internal/db/config"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"
	masksvc "mayfly-go/internal/db/domain/mask"
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
	SQL     string              // 执行的sql
	Stmt    sqlstmt.Stmt        // 解析后的sql stmt
	Procdef *flowentity.Procdef // 流程定义

	SQLExecRecord *entity.DbSQLExec // sql执行记录
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

type DbSQLExec interface {
	flowapp.FlowBizHandler

	// 执行sql
	Exec(ctx context.Context, execSQLReq *dto.DbSQLExecReq) ([]*dto.DbSQLExecRes, error)

	// ExecReader 从reader中读取sql并执行
	ExecReader(ctx context.Context, execReader *dto.SQLReaderExec) error

	// 根据条件删除sql执行记录
	DeleteBy(ctx context.Context, condition *entity.DbSQLExec) error

	// 分页获取
	GetPageList(condition *entity.DbSQLExecQuery, orderBy ...string) (*model.PageResult[*entity.DbSQLExec], error)
}

var _ (DbSQLExec) = (*dbSQLExecAppImpl)(nil)

type dbSQLExecAppImpl struct {
	dbApp         Db                   `inject:"T"`
	dbSQLExecRepo repository.DbSQLExec `inject:"T"`
	maskEngine    mask.MaskEngine      `inject:"T"`

	flowProcdefApp flowapp.Procdef `inject:"T"`
}

var _ DbSQLExec = (*dbSQLExecAppImpl)(nil)

func createSQLExecRecord(ctx context.Context, execSQLReq *dto.DbSQLExecReq, sql string) *entity.DbSQLExec {
	dbSQLExecRecord := new(entity.DbSQLExec)
	dbSQLExecRecord.DbId = execSQLReq.DbId
	dbSQLExecRecord.Db = execSQLReq.Db
	dbSQLExecRecord.SQL = sql
	dbSQLExecRecord.Remark = execSQLReq.Remark
	dbSQLExecRecord.Status = entity.DbSQLExecStatusSuccess
	return dbSQLExecRecord
}

func (d *dbSQLExecAppImpl) Exec(ctx context.Context, execSQLReq *dto.DbSQLExecReq) ([]*dto.DbSQLExecRes, error) {
	dbConn := execSQLReq.DbConn
	execSQL := execSQLReq.SQL

	var flowProcdef *flowentity.Procdef
	if execSQLReq.CheckFlow {
		flowProcdef = d.flowProcdefApp.GetProcdefByCodePath(ctx, dbConn.Info.CodePath...)
	}

	allExecRes := make([]*dto.DbSQLExecRes, 0)

	// 先使用方言切割器切割 SQL
	splitter := dbConn.GetDialect().GetSQLSplitter()
	var sqlList []string
	err := splitter.SplitSQL(strings.NewReader(execSQL), func(oneSQL string) error {
		sqlList = append(sqlList, oneSQL)
		return nil
	})
	if err != nil {
		return nil, imsg.SplitError(ctx, err)
	}

	// 获取解析器
	sp := dbConn.GetDialect().GetSQLParser()

	// 逐条解析并执行
	for _, sql := range sqlList {
		var execRes *dto.DbSQLExecRes
		var err error

		stmt, parseErr := sp.Parse(sql)
		dbSQLExecRecord := createSQLExecRecord(ctx, execSQLReq, sql)
		dbSQLExecRecord.Type = entity.DbSQLExecTypeOther
		sqlExec := &sqlExecParam{
			DbConn:        dbConn,
			SQL:           sql,
			Stmt:          stmt,
			Procdef:       flowProcdef,
			SQLExecRecord: dbSQLExecRecord,
		}

		// 语句分类单点收敛至 sqlparser.Classify：AST 优先，其次方言分类能力，最后整词关键字兜底
		switch sqlparser.Classify(sp, splitter, sql, stmt, parseErr) {
		case sqlstmt.StmtTypeSelect:
			execRes, err = d.doSelect(ctx, sqlExec)
		case sqlstmt.StmtTypeUpdate:
			execRes, err = d.doUpdate(ctx, sqlExec)
		case sqlstmt.StmtTypeDelete:
			execRes, err = d.doDelete(ctx, sqlExec)
		case sqlstmt.StmtTypeInsert:
			execRes, err = d.doInsert(ctx, sqlExec)
		case sqlstmt.StmtTypeDDL:
			// 改结构后的元数据缓存失效由 dbi 执行层按语句类型统一负责（见 DbInfo.invalidateIfDDL）
			execRes, err = d.doExecDDL(ctx, sqlExec)
		case sqlstmt.StmtTypeOther:
			execRes, err = d.doOtherRead(ctx, sqlExec)
		default:
			execRes, err = d.doExec(ctx, dbConn, sql)
		}

		// 执行错误
		if err != nil {
			if execRes == nil {
				execRes = &dto.DbSQLExecRes{SQL: sql}
			}
			execRes.ErrorMsg = err.Error()
		} else {
			// 保存执行结果集（dbSQLExecRecord.Res尚未赋值，需传入本次执行结果）
			d.saveSQLExecLog(ctx, dbSQLExecRecord, execRes.Res)
		}
		allExecRes = append(allExecRes, execRes)
	}

	return allExecRes, nil
}

func (d *dbSQLExecAppImpl) ExecReader(ctx context.Context, execReader *dto.SQLReaderExec) error {
	dbConn := execReader.DbConn

	clientId := execReader.ClientId
	filename := stringx.Truncate(execReader.Filename, 20, 10, "...")
	la := contextx.GetLoginAccount(ctx)
	needSendMsg := la != nil && clientId != "" && execReader.UploadId != ""

	startTime := time.Now()
	executedStatements := 0

	dbInfo := dbConn.Info
	msgEvent := &msgdto.MsgTmplSendEvent{
		TmplChannel: msgdto.MsgTmplSQLScriptRunSuccess,
		Params:      collx.M{"filename": filename, "dbId": dbInfo.Id, "dbName": dbInfo.Name},
	}

	progressMsgEvent := &msgdto.MsgTmplSendEvent{
		TmplChannel: msgdto.MsgTmplSQLScriptRunProgress,
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
				msgEvent.TmplChannel = msgdto.MsgTmplSQLScriptRunFail
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
	// 文件内的 BEGIN/COMMIT/ROLLBACK 等事务控制语句按脚本原样执行（与 mysql CLI、psql 一致的脚本自治语义）。
	// 真实提交点由各数据库自身决定（mysql 执行 BEGIN 会隐式提交在途写入，pg 仅告警，sqlite 直接报错），
	// 无法从语句文本可靠推断（DDL 同样会隐式提交），故这里不做猜测与改写，失败一律原样返回底层数据库错误
	err = splitter.SplitSQL(execReader.Reader, func(sql string) error {
		// 检查context是否已取消
		if ctx.Err() != nil {
			if needSendMsg {
				progressMsgEvent.Params["executedStatements"] = executedStatements
				progressMsgEvent.Params["status"] = "cancelled"
				global.EventBus.Publish(ctx, event.EventTopicMsgTmplSend, progressMsgEvent)
			}
			return errorx.NewBizI(ctx, imsg.ErrSQLExecCancelled)
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
		err = imsg.SplitError(ctx, err)
		_ = tx.Rollback()
		if needSendMsg {
			msgEvent.TmplChannel = msgdto.MsgTmplSQLScriptRunFail
			msgEvent.Params["error"] = err.Error()
			global.EventBus.Publish(ctx, event.EventTopicMsgTmplSend, msgEvent)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		logx.Errorf("commit sql file exec failed: %s", err.Error())
		return err
	}

	if needSendMsg {
		msgEvent.Params["cost"] = fmt.Sprintf("%dms", time.Since(startTime).Milliseconds())
		global.EventBus.Publish(ctx, event.EventTopicMsgTmplSend, msgEvent)
	}
	return nil
}

type FlowDbExecSQLBizForm struct {
	DbId   uint64 `json:"dbId"`   //  库id
	DbName string `json:"dbName"` // 库名
	SQL    string `json:"sql"`    // sql
}

func (d *dbSQLExecAppImpl) FlowBizHandle(ctx context.Context, bizHandleParam *flowapp.BizHandleParam) (any, error) {
	procinst := bizHandleParam.Procinst
	bizKey := procinst.BizKey
	procinstStatus := procinst.Status

	logx.Debugf("DbSqlExec FlowBizHandle -> bizKey: %s, procinstStatus: %s", bizKey, flowentity.ProcinstStatusEnum.GetDesc(procinstStatus))
	// 流程非完成状态不处理
	if procinstStatus != flowentity.ProcinstStatusCompleted {
		return nil, nil
	}

	execSQLBizForm, err := jsonx.ToByStr[FlowDbExecSQLBizForm](procinst.BizForm)
	if err != nil {
		return nil, errorx.NewBizf("failed to parse the business form information: %s", err.Error())
	}

	dbConn, err := d.dbApp.GetDbConn(ctx, execSQLBizForm.DbId, execSQLBizForm.DbName)
	if err != nil {
		return nil, err
	}

	execRes, err := d.Exec(contextx.NewLoginAccount(&model.LoginAccount{Id: procinst.CreatorId, Username: procinst.Creator}), &dto.DbSQLExecReq{
		DbId:      execSQLBizForm.DbId,
		Db:        execSQLBizForm.DbName,
		SQL:       execSQLBizForm.SQL,
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
			return execRes, errorx.NewBizI(ctx, imsg.ErrExistRunFailSQL)
		}
	}

	return execRes, nil
}

func (d *dbSQLExecAppImpl) DeleteBy(ctx context.Context, condition *entity.DbSQLExec) error {
	return d.dbSQLExecRepo.DeleteByCond(ctx, condition)
}

func (d *dbSQLExecAppImpl) GetPageList(condition *entity.DbSQLExecQuery, orderBy ...string) (*model.PageResult[*entity.DbSQLExec], error) {
	return d.dbSQLExecRepo.GetPageList(condition, orderBy...)
}

// 保存sql执行记录，如果是查询类则根据系统配置判断是否保存
func (d *dbSQLExecAppImpl) saveSQLExecLog(ctx context.Context, dbSQLExecRecord *entity.DbSQLExec, res any) {
	if dbSQLExecRecord.Type != entity.DbSQLExecTypeQuery {
		dbSQLExecRecord.Res = jsonx.ToStr(res)
		if err := d.dbSQLExecRepo.Insert(ctx, dbSQLExecRecord); err != nil {
			logx.Errorf("save sql exec record failed: %s", err.Error())
		}
		return
	}

	if config.GetDbms().QuerySQLSave {
		dbSQLExecRecord.Table = "-"
		dbSQLExecRecord.OldValue = "-"
		dbSQLExecRecord.Type = entity.DbSQLExecTypeQuery
		if err := d.dbSQLExecRepo.Insert(ctx, dbSQLExecRecord); err != nil {
			logx.Errorf("save sql exec query record failed: %s", err.Error())
		}
	}
}

func (d *dbSQLExecAppImpl) doSelect(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSQLExecRes, error) {
	maxCount := config.GetDbms().MaxResultSet
	sqlExecParam.SQLExecRecord.Type = entity.DbSQLExecTypeQuery

	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSQLExecFlowBizType, collx.Kvs("stmtType", "select")); needStartProc {
			return nil, errorx.NewBizI(ctx, imsg.ErrNeedSubmitWorkTicket)
		}
	}

	return d.doQuery(ctx, sqlExecParam, maxCount)
}

func (d *dbSQLExecAppImpl) doOtherRead(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSQLExecRes, error) {
	sqlExecParam.SQLExecRecord.Type = entity.DbSQLExecTypeQuery

	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSQLExecFlowBizType, collx.Kvs("stmtType", "read")); needStartProc {
			return nil, errorx.NewBizI(ctx, imsg.ErrNeedSubmitWorkTicket)
		}
	}

	return d.doQuery(ctx, sqlExecParam, 0)
}

func (d *dbSQLExecAppImpl) doExecDDL(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSQLExecRes, error) {
	selectSQL := sqlExecParam.SQL
	sqlExecParam.SQLExecRecord.Type = entity.DbSQLExecTypeDDL

	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSQLExecFlowBizType, collx.Kvs("stmtType", "ddl")); needStartProc {
			return nil, errorx.NewBizI(ctx, imsg.ErrNeedSubmitWorkTicket)
		}
	}

	return d.doExec(ctx, sqlExecParam.DbConn, selectSQL)
}

func (d *dbSQLExecAppImpl) doUpdate(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSQLExecRes, error) {
	dbConn := sqlExecParam.DbConn

	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSQLExecFlowBizType, collx.Kvs("stmtType", "update")); needStartProc {
			return nil, errorx.NewBizI(ctx, imsg.ErrNeedSubmitWorkTicket)
		}
	}

	execRecord := sqlExecParam.SQLExecRecord
	execRecord.Type = entity.DbSQLExecTypeUpdate

	stmt := sqlExecParam.Stmt
	if stmt == nil {
		return d.doExec(ctx, dbConn, sqlExecParam.SQL)
	}

	updatestmt, ok := stmt.(*sqlstmt.UpdateStmt)
	if !ok {
		return d.doExec(ctx, dbConn, sqlExecParam.SQL)
	}

	// 不支持多表更新记录旧值
	if len(updatestmt.Tables) != 1 {
		logx.ErrorContext(ctx, "update SQL - logging old values only supports single-table updates")
		return d.doExec(ctx, dbConn, sqlExecParam.SQL)
	}

	tableName := updatestmt.Tables[0].Name
	tableAlias := updatestmt.Tables[0].Alias

	if tableName == "" {
		logx.ErrorContext(ctx, "update SQL - failed to get table name")
		return d.doExec(ctx, dbConn, sqlExecParam.SQL)
	}
	execRecord.Table = tableName

	if updatestmt.Where == nil {
		logx.ErrorContext(ctx, "update SQL - there is no where condition")
		return d.doExec(ctx, dbConn, sqlExecParam.SQL)
	}
	whereStr := updatestmt.Where.Text

	// 获取表全部主键列（联合主键多列），排除使用别名
	primaryKeys, err := dbConn.Metadata().GetPrimaryKeys(tableName)
	if err != nil {
		logx.ErrorfContext(ctx, "update SQL - failed to get primary key columns: %s", err.Error())
		return d.doExec(ctx, dbConn, sqlExecParam.SQL)
	}

	updateColumns := collx.ArrayMap[sqlstmt.Assignment, string](updatestmt.Set, func(a sqlstmt.Assignment) string {
		return a.Column
	})

	// SELECT 需带上全部主键列，否则记录的旧值无法定位行（联合主键少列即歧义）
	pkColumns := make([]string, 0, len(primaryKeys))
	for _, pk := range primaryKeys {
		if tableAlias != "" {
			pk = tableAlias + "." + pk
		}
		pkColumns = append(pkColumns, pk)
	}
	updateColumnsAndPrimaryKey := strings.Join(append(updateColumns, pkColumns...), ",")
	// 查询要更新字段数据的旧值，以及主键值
	selectSQL := fmt.Sprintf("SELECT %s FROM %s where %s", updateColumnsAndPrimaryKey, tableName+" "+tableAlias, whereStr)

	// WalkQuery查出最多200条数据
	maxRec := 200
	nowRec := 0
	res := make([]map[string]any, 0)
	_, err = dbConn.WalkQueryRows(ctx, selectSQL, func(row map[string]any, columns []*dbi.QueryColumn) error {
		nowRec++
		res = append(res, row)
		if nowRec == maxRec {
			return errorx.NewBizf("update SQL - the maximum number of updated queries is exceeded: %d", maxRec)
		}
		return nil
	})
	if err != nil {
		logx.ErrorfContext(ctx, "update SQL - failed to get the updated old value: %s", err.Error())
		return d.doExec(ctx, dbConn, sqlExecParam.SQL)
	}
	execRecord.OldValue = jsonx.ToStr(res)

	return d.doExec(ctx, dbConn, sqlExecParam.SQL)
}

func (d *dbSQLExecAppImpl) doDelete(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSQLExecRes, error) {
	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSQLExecFlowBizType, collx.Kvs("stmtType", "delete")); needStartProc {
			return nil, errorx.NewBizI(ctx, imsg.ErrNeedSubmitWorkTicket)
		}
	}

	dbConn := sqlExecParam.DbConn
	execRecord := sqlExecParam.SQLExecRecord
	execRecord.Type = entity.DbSQLExecTypeDelete

	stmt := sqlExecParam.Stmt
	if stmt == nil {
		return d.doExec(ctx, dbConn, sqlExecParam.SQL)
	}

	deletestmt, ok := stmt.(*sqlstmt.DeleteStmt)
	if !ok {
		return d.doExec(ctx, dbConn, sqlExecParam.SQL)
	}

	// 不支持多表删除记录旧值
	if len(deletestmt.Tables) != 1 {
		logx.ErrorContext(ctx, "delete SQL - logging old values only supports single-table deletion")
		return d.doExec(ctx, dbConn, sqlExecParam.SQL)
	}

	tableName := deletestmt.Tables[0].Name
	tableAlias := deletestmt.Tables[0].Alias

	if tableName == "" {
		logx.ErrorContext(ctx, "delete SQL - failed to get table name")
		return d.doExec(ctx, dbConn, sqlExecParam.SQL)
	}
	execRecord.Table = tableName

	if deletestmt.Where == nil {
		logx.ErrorContext(ctx, "delete SQL - there is no where condition")
		return d.doExec(ctx, dbConn, sqlExecParam.SQL)
	}

	whereStr := deletestmt.Where.Text
	// 查询删除数据（仅用于记录旧值审计，失败不影响删除执行）
	// 用 WalkQueryRows 遍历并计数截断，避免写死 MySQL/PG 的 LIMIT 语法（Oracle/DM/MSSQL 不识别会整条报错），
	// 与 doUpdate 的旧值采集保持一致的跨方言做法；达到上限以 StopWalkQueryError 提前终止扫描
	const maxAuditRows = 200
	selectSQL := fmt.Sprintf("SELECT * FROM %s where %s", tableName+" "+tableAlias, whereStr)
	auditRes := make([]map[string]any, 0, maxAuditRows)
	if _, err := dbConn.WalkQueryRows(ctx, selectSQL, func(row map[string]any, _ []*dbi.QueryColumn) error {
		auditRes = append(auditRes, row)
		if len(auditRes) >= maxAuditRows {
			return dbi.NewStopWalkQueryError("reached the maximum number of audit rows")
		}
		return nil
	}); err != nil {
		logx.ErrorfContext(ctx, "delete SQL - failed to query old values for audit: %s", err.Error())
	} else {
		execRecord.OldValue = jsonx.ToStr(auditRes)
	}

	return d.doExec(ctx, dbConn, sqlExecParam.SQL)
}

func (d *dbSQLExecAppImpl) doInsert(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSQLExecRes, error) {
	if procdef := sqlExecParam.Procdef; procdef != nil {
		if needStartProc := procdef.MatchCondition(DbSQLExecFlowBizType, collx.Kvs("stmtType", "insert")); needStartProc {
			return nil, errorx.NewBizI(ctx, imsg.ErrNeedSubmitWorkTicket)
		}
	}

	dbConn := sqlExecParam.DbConn
	execRecord := sqlExecParam.SQLExecRecord
	execRecord.Type = entity.DbSQLExecTypeInsert

	stmt := sqlExecParam.Stmt
	if stmt == nil {
		return d.doExec(ctx, dbConn, sqlExecParam.SQL)
	}

	insertstmt, ok := stmt.(*sqlstmt.InsertStmt)
	if !ok {
		return d.doExec(ctx, dbConn, sqlExecParam.SQL)
	}

	execRecord.Table = insertstmt.Table.Name

	return d.doExec(ctx, sqlExecParam.DbConn, sqlExecParam.SQL)
}

func (d *dbSQLExecAppImpl) doQuery(ctx context.Context, sqlExecParam *sqlExecParam, maxRows int) (*dto.DbSQLExecRes, error) {
	dbConn := sqlExecParam.DbConn
	sql := sqlExecParam.SQL

	// 查询结果脱敏开关（服务端强制执行，fail-close模式下构建失败阻断查询）
	maskEnabled := d.maskEngine != nil && config.GetDbms().MaskEnabled

	res := make([]map[string]any, 0, 16)
	nowRows := 0
	var rowMasker *masksvc.RowMasker
	cols, err := dbConn.WalkQueryRows(ctx, sql, func(row map[string]any, columns []*dbi.QueryColumn) error {
		nowRows++
		// 超过指定的最大查询记录数，则停止查询
		if maxRows != 0 && nowRows > maxRows {
			return dbi.NewStopWalkQueryError(fmt.Sprintf("exceed the maximum number of query records %d", maxRows))
		}
		// 行级脱敏，脱敏器在首次行回调时构建（此时才拿到结果列信息）
		if maskEnabled {
			if rowMasker == nil {
				var maskErr error
				rowMasker, maskErr = d.maskEngine.BuildStmtRowMasker(ctx, dbConn, sqlExecParam.Stmt, columns)
				if maskErr != nil {
					// fail-close：脱敏计划不可用时阻断本次查询，避免敏感数据明文透出
					return maskErr
				}
			}
			if rowMasker != nil {
				rowMasker.MaskRow(row)
			}
		}
		res = append(res, row)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &dto.DbSQLExecRes{
		SQL:     sql,
		Columns: cols,
		Res:     res,
	}, nil
}

func (d *dbSQLExecAppImpl) doExec(ctx context.Context, dbConn *dbi.DbConn, sql string) (*dto.DbSQLExecRes, error) {
	rowsAffected, err := dbConn.ExecContext(ctx, sql)
	if err != nil {
		return nil, err
	}

	res := make([]map[string]any, 0)
	res = append(res, collx.Kvs("rowsAffected", rowsAffected))

	return &dto.DbSQLExecRes{
		Columns: []*dbi.QueryColumn{
			{Name: "rowsAffected", Key: "rowsAffected", Type: "number"},
		},
		Res: res,
		SQL: sql,
	}, err
}

// 语句分类逻辑已下沉至 sqlparser.Classify（单一入口），此处不再保留字符串兜底判定。
