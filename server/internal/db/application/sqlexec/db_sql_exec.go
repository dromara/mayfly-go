package sqlexec

import (
	"context"
	"database/sql"
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

	// Notices 本次执行触发的策略提醒（不阻断，随结果回传给操作者）
	Notices []*dto.PolicyNotice

	// RequireWarnAck 命中「仅提醒」时是否需要先让操作者确认，由调用入口声明
	RequireWarnAck bool
	// WarnAcknowledged 操作者已确认过该提醒
	WarnAcknowledged bool

	SQLExecRecord *entity.DbSQLExec // sql执行记录

	// tx 原子批次共享的事务：非 nil 时本批所有写语句都在它内执行，全成功才提交。
	// 挂在参数对象上而不是给各 do* 方法加事务参数，是为了让「是否走事务」只由批次决定，
	// 逐条自动提交（tx==nil）与整批原子提交共用同一条执行代码
	tx *sql.Tx
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
	// Exec 执行一次 SQL 请求：必然过触发策略，无需调用方声明「要不要查」
	Exec(ctx context.Context, execSQLReq *dto.DbSQLExecReq) ([]*dto.DbSQLExecRes, error)

	// CheckSqlsWithoutTicket 判定「拿不到工单通道」的入口（AI Agent、数据导入）能否执行这批 SQL，返回与入参等长的提醒数组
	CheckSqlsWithoutTicket(ctx context.Context, conn *dbi.DbConn, sqls []string) ([]string, error)

	// ExecApproved 回放「工单审批通过后」的 SQL，不再过触发策略。
	//
	// 单独一个入口而不是给 Exec 加布尔开关：布尔的零值是「不查」，以后新增调用点忘了置位
	// 就静默绕过治理，而这类失效在界面上完全看不出来。回放是唯一不必查的来源，让它显式点名
	ExecApproved(ctx context.Context, execSQLReq *dto.DbSQLExecReq) ([]*dto.DbSQLExecRes, error)

	// ExecReader 从reader中读取sql并执行
	ExecReader(ctx context.Context, execReader *dto.SQLReaderExec) error

	// CancelExec 中断本实例上仍在执行的语句（execId 由交互式执行登记）。
	//
	// 只能由发起者本人取消：不校验属主就等于给任意持 token 的账号一条中断他人执行的能力
	CancelExec(ctx context.Context, execId string, accountId uint64) error

	// RegisterExec 登记一次交互式执行，返回其可取消上下文与注销函数（execId 为空时透传原 ctx）。
	//
	// 调用方必须在取连接等耗时步骤之前登记：取消按 execId 做有界等待，登记晚于取连接时，
	// 冷连接（拨号/隧道）会让等待耗尽，取消误报「未找到进行中的执行」而语句随后照跑。
	// execId 与他人进行中的执行冲突时报错拒绝：顶替行为不得绕过取消侧的属主校验
	RegisterExec(ctx context.Context, execSQLReq *dto.DbSQLExecReq) (context.Context, func(), error)

	// ExportQuery 全量导出一次查询的结果集（不受结果集行数上限约束），产物写入 writer。
	//
	// 重跑同一条语句而不导出页面上已取回的行：页面行已被截断，拿它导出的是一份缺尾巴的文件。
	// onProgress 可选，异步导出任务用它回报已写出行数
	ExportQuery(ctx context.Context, req *dto.QueryExportReq, openWriter OpenWriter, onProgress ...ExportProgress) error

	// GenRollback 依据执行记录当时采到的旧值生成回滚 SQL（只生成文本，不执行）。
	//
	// 不执行是刻意的：回滚同样是一次变更，必须自己过预检与审批，见 dto.SqlRollback
	GenRollback(ctx context.Context, execRecordId uint64) (*dto.SqlRollback, error)

	// 根据条件删除sql执行记录
	DeleteBy(ctx context.Context, condition *entity.DbSQLExec) error

	// 分页获取
	GetPageList(condition *entity.DbSQLExecQuery, orderBy ...string) (*model.PageResult[*entity.DbSQLExec], error)
}

var _ (DbSQLExec) = (*DbSQLExecAppImpl)(nil)

type DbSQLExecAppImpl struct {
	dbApp         DbConnReader         `inject:"T"`
	dbSQLExecRepo repository.DbSQLExec `inject:"T"`
	maskEngine    mask.MaskEngine      `inject:"T"`

	flowProcdefApp flowapp.Procdef `inject:"T"`
}

var _ DbSQLExec = (*DbSQLExecAppImpl)(nil)

func createSQLExecRecord(ctx context.Context, execSQLReq *dto.DbSQLExecReq, sql string) *entity.DbSQLExec {
	dbSQLExecRecord := new(entity.DbSQLExec)
	dbSQLExecRecord.DbId = execSQLReq.DbId
	dbSQLExecRecord.Db = execSQLReq.Db
	dbSQLExecRecord.SQL = sql
	dbSQLExecRecord.Remark = execSQLReq.Remark
	dbSQLExecRecord.Status = entity.DbSQLExecStatusSuccess
	return dbSQLExecRecord
}

func (d *DbSQLExecAppImpl) Exec(ctx context.Context, execSQLReq *dto.DbSQLExecReq) ([]*dto.DbSQLExecRes, error) {
	// 流程定义按请求解析一次，本次的多条语句共用同一份
	return d.exec(ctx, execSQLReq, d.flowProcdefApp.GetProcdefByCodePath(ctx, execSQLReq.DbConn.Info.CodePath...))
}

func (d *DbSQLExecAppImpl) ExecApproved(ctx context.Context, execSQLReq *dto.DbSQLExecReq) ([]*dto.DbSQLExecRes, error) {
	// procdef 传 nil 即整条链路不判策略（见接口注释），而不是再引入一个「已审批」标记位
	return d.exec(ctx, execSQLReq, nil)
}

func (d *DbSQLExecAppImpl) exec(ctx context.Context, execSQLReq *dto.DbSQLExecReq, flowProcdef *flowentity.Procdef) ([]*dto.DbSQLExecRes, error) {
	// 交互式执行的登记已前置到入口 handler（RegisterExec，见 api/db.go ExecSQL）：
	// 登记必须早于取连接，否则冷连接拨号期间到达的「取消」会落空；
	// 回放等非交互入口 execId 为空，本就不登记
	dbConn := execSQLReq.DbConn
	execSQL := execSQLReq.SQL

	allExecRes := make([]*dto.DbSQLExecRes, 0)

	// 先使用方言切割器切割 SQL
	splitter := dbConn.GetDialect().GetSQLSplitter()
	var sqlList []string
	var err error
	err = splitter.SplitSQL(strings.NewReader(execSQL), func(oneSQL string) error {
		sqlList = append(sqlList, oneSQL)
		return nil
	})
	if err != nil {
		return nil, imsg.SplitError(ctx, err)
	}

	// 获取解析器
	sp := dbConn.GetDialect().GetSQLParser()

	// 先逐条解析并分类：原子批次要在执行前判定「这一批能不能原子」，
	// 而分类结果执行时还要用一遍——放在这里算一次并复用，避免同一语句解析两遍
	plans := make([]*stmtPlan, 0, len(sqlList))
	for _, sql := range sqlList {
		stmt, parseErr := sp.Parse(sql)
		// 语句分类单点收敛至 sqlparser.Classify：AST 优先，其次方言分类能力，最后整词关键字兜底
		class := sqlparser.Classify(sp, splitter, sql, stmt, parseErr)
		if execSQLReq.Atomic && !atomicClasses[class] {
			return nil, errorx.NewBizI(ctx, imsg.ErrAtomicUnsupportedStmt, "sql", stringx.Truncate(sql, 120, 50, "..."))
		}
		plans = append(plans, &stmtPlan{sql: sql, stmt: stmt, parseErr: parseErr, class: class})
	}

	// 原子批次：整批共用一个事务，全成功才提交，任一条失败（含被策略拦下）即整批回滚
	var atomicTx *sql.Tx
	if execSQLReq.Atomic {
		if atomicTx, err = dbConn.Begin(); err != nil {
			return nil, err
		}
		// 回滚兜底：提交成功后 atomicTx 置 nil，未提交路径（含 panic）由这里回滚；
		// RollbackTx 对 nil 与「事务已完成」都安全，不会掩盖真实回滚失败
		defer func() {
			dbi.RollbackTx(atomicTx)
		}()
	}

	// 原子批次的执行记录先攒着，等目标库事务真正提交后再落：执行记录写在平台自己的元数据库里，
	// 不随目标库事务回滚。若在提交前逐条落库，整批回滚后列表里会留下一堆「成功」记录——语句其实
	// 一条都没生效，而「生成回滚 SQL」入口还能据当时采到的旧值把陈旧数据写回现网。
	// 与既有语义对齐：失败语句本来就不落执行记录，作废的整批也就应当一条不留
	type pendingExecLog struct {
		record *entity.DbSQLExec
		res    any
	}
	var pendingExecLogs []pendingExecLog

	// 逐条执行
	for _, plan := range plans {
		sql := plan.sql
		var execRes *dto.DbSQLExecRes
		var err error

		dbSQLExecRecord := createSQLExecRecord(ctx, execSQLReq, sql)
		dbSQLExecRecord.Type = entity.DbSQLExecTypeOther
		sqlExec := &sqlExecParam{
			DbConn:           dbConn,
			SQL:              sql,
			Stmt:             plan.stmt,
			Procdef:          flowProcdef,
			SQLExecRecord:    dbSQLExecRecord,
			RequireWarnAck:   execSQLReq.RequireWarnAck,
			WarnAcknowledged: execSQLReq.WarnAcknowledged,
			tx:               atomicTx,
		}

		switch plan.class {
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
			execRes, err = d.doExec(ctx, sqlExec)
		}

		// 执行错误
		if err != nil {
			if execRes == nil {
				execRes = &dto.DbSQLExecRes{SQL: sql}
			}
			execRes.ErrorMsg = err.Error()
			// 被策略要求审批的语句带上提单标记，前端据此在失败结果处直接给出提交入口；
			// 等待确认提醒的语句带另一个标记：两者后续动作不同（一个提单、一个确认后重试）
			execRes.NeedApproval = flowapp.IsNeedApprovalError(err)
			execRes.WarnAck = flowapp.IsWarnAckError(err)

			// 原子批次里一条失败即整批作废：不再执行后续语句。
			if execSQLReq.Atomic {
				rolledBackErr := errorx.NewBizI(ctx, imsg.ErrAtomicBatchRolledBack,
					"index", len(allExecRes)+1, "total", len(plans), "error", err.Error())
				// 需审批/待确认是「可继续动作」的失败：整批已回滚，只回这一条并带上结构化标记，
				// 让前端据此给出（并自动打开）提单入口；不回前面已回滚的语句，否则界面把作废的显示成成功
				if execRes.NeedApproval || execRes.WarnAck {
					return []*dto.DbSQLExecRes{{SQL: sql, ErrorMsg: rolledBackErr.Error(), NeedApproval: execRes.NeedApproval, WarnAck: execRes.WarnAck}}, nil
				}
				// 其余失败（语法/权限/约束等）保持整批回滚错误上抛，界面按执行失败呈现
				return nil, rolledBackErr
			}
		} else if execSQLReq.Atomic {
			pendingExecLogs = append(pendingExecLogs, pendingExecLog{record: dbSQLExecRecord, res: execRes.Res})
		} else {
			// 保存执行结果集（dbSQLExecRecord.Res尚未赋值，需传入本次执行结果）
			d.saveSQLExecLog(ctx, dbSQLExecRecord, execRes.Res)
		}
		// 策略提醒随该条语句的结果回传：不阻断执行，但必须让操作者看到
		if execRes != nil && len(sqlExec.Notices) > 0 {
			execRes.Notices = sqlExec.Notices
		}
		allExecRes = append(allExecRes, execRes)
	}

	if atomicTx != nil {
		if err := dbi.CommitTargetTx(dbConn, atomicTx); err != nil {
			return nil, err
		}
		atomicTx = nil
	}

	// 提交成功才补写执行记录：此后的数据与「库里已生效」一致
	for _, pending := range pendingExecLogs {
		d.saveSQLExecLog(ctx, pending.record, pending.res)
	}

	return allExecRes, nil
}

// stmtPlan 单条语句的解析与分类结果，供执行与原子批次前置判定共用（同一份分类口径）
type stmtPlan struct {
	sql      string
	stmt     sqlstmt.Stmt
	parseErr error
	class    sqlstmt.StmtType
}

// atomicClasses 允许进原子批次的语句类别。
//
// 只收写数据的三类：SELECT 的结果不参与回滚语义，混进来会让「已读到的数据」与「已提交的数据」
// 在同一批次里含义不一致；DDL 在 MySQL 等方言里隐式提交、根本无法回滚，
// 若允许声明为原子批次会给出「可回滚」的假承诺，比拒绝更危险
var atomicClasses = map[sqlstmt.StmtType]bool{
	sqlstmt.StmtTypeInsert: true,
	sqlstmt.StmtTypeUpdate: true,
	sqlstmt.StmtTypeDelete: true,
}

// checkImportedSQL 判定导入文件里的单条语句能否执行：与交互式执行共用同一个触发策略入口，
// 提醒不阻断（导入过程里没有能逐条应答确认的人），需审批与禁止执行一律拦下。
//
// procdef 为空表示该库没有绑定流程定义，此时不付任何解析开销
func (d *DbSQLExecAppImpl) checkImportedSQL(ctx context.Context, procdef *flowentity.Procdef, dbConn *dbi.DbConn, sql string) error {
	if procdef == nil {
		return nil
	}
	sp := dbConn.GetDialect().GetSQLParser()
	stmt, parseErr := sp.Parse(sql)
	// 分类口径与执行、审计同源（见 Exec），否则同一条语句在导入与交互执行下会得到不同结论
	sqlExec := &sqlExecParam{DbConn: dbConn, SQL: sql, Stmt: stmt, Procdef: procdef}
	stmtType := string(sqlparser.Classify(sp, dbConn.GetDialect().GetSQLSplitter(), sql, stmt, parseErr))
	if err := d.checkSQLTrigger(ctx, sqlExec, stmtType); err != nil {
		return err
	}
	// 提醒拦不住执行，但必须留痕：导入结束后界面只报「成功 N 条」，不记日志就等于无人知晓
	for _, notice := range sqlExec.Notices {
		logx.WarnContext(ctx, fmt.Sprintf("sql file import hit policy notice: %s", notice.Title))
	}
	return nil
}

func (d *DbSQLExecAppImpl) ExecReader(ctx context.Context, execReader *dto.SQLReaderExec) error {
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

	// 流程定义按连接粒度解析一次，别在语句循环里逐条查库
	importProcdef := d.flowProcdefApp.GetProcdefByCodePath(ctx, dbConn.Info.CodePath...)

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
		// 导入与交互式执行必须同一套口径：否则把「禁止执行」的语句写进 .sql 上传即可绕过策略，
		// 这比界面上少弹一次确认框严重得多。导入也无法逐条追问（几百条语句会把导入变成人工批处理），
		// 故提醒级放行、需审批与禁止级直接拦下，拦下就中断本次导入。
		//
		// 这里能给的保证是「被策略拒绝的语句绝不执行」，而不是「整个文件要么全成要么全无」：
		// 前序语句能否回滚取决于脚本所在库的事务语义（见上文，mysql 的 DDL 会隐式提交，
		// 那种情况下前序写入留得住，与用 mysql CLI 跑同一个脚本一致）
		if err := d.checkImportedSQL(ctx, importProcdef, dbConn, sql); err != nil {
			return err
		}
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

func (d *DbSQLExecAppImpl) FlowBizHandle(ctx context.Context, bizHandleParam *flowapp.BizHandleParam) (any, error) {
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

	// 与 Redis 回放同一套身份处理：取连接（含资源鉴权）与执行都用发起人身份
	ctx = flowapp.BizOperatorContext(ctx, bizHandleParam)

	dbConn, err := d.dbApp.GetDbConn(ctx, execSQLBizForm.DbId, execSQLBizForm.DbName)
	if err != nil {
		return nil, err
	}

	// 审批通过后按原样回放，不再查策略：再查一次会命中「需审批」，工单就卡在自己身上
	execRes, err := d.ExecApproved(ctx, &dto.DbSQLExecReq{
		DbId:   execSQLBizForm.DbId,
		Db:     execSQLBizForm.DbName,
		SQL:    execSQLBizForm.SQL,
		DbConn: dbConn,
		Remark: procinst.Remark,
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

func (d *DbSQLExecAppImpl) CancelExec(ctx context.Context, execId string, accountId uint64) error {
	return execRegistry.cancel(ctx, execId, accountId)
}

func (d *DbSQLExecAppImpl) RegisterExec(ctx context.Context, execSQLReq *dto.DbSQLExecReq) (context.Context, func(), error) {
	return execRegistry.register(ctx, execSQLReq.ExecId, execSQLReq)
}

func (d *DbSQLExecAppImpl) DeleteBy(ctx context.Context, condition *entity.DbSQLExec) error {
	return d.dbSQLExecRepo.DeleteByCond(ctx, condition)
}

func (d *DbSQLExecAppImpl) GetPageList(condition *entity.DbSQLExecQuery, orderBy ...string) (*model.PageResult[*entity.DbSQLExec], error) {
	return d.dbSQLExecRepo.GetPageList(condition, orderBy...)
}

// 保存sql执行记录，如果是查询类则根据系统配置判断是否保存
func (d *DbSQLExecAppImpl) saveSQLExecLog(ctx context.Context, dbSQLExecRecord *entity.DbSQLExec, res any) {
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

func (d *DbSQLExecAppImpl) doSelect(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSQLExecRes, error) {
	maxCount := config.GetDbms().MaxResultSet
	sqlExecParam.SQLExecRecord.Type = entity.DbSQLExecTypeQuery

	if err := d.checkSQLTrigger(ctx, sqlExecParam, "select"); err != nil {
		return nil, err
	}

	return d.doQuery(ctx, sqlExecParam, maxCount)
}

func (d *DbSQLExecAppImpl) doOtherRead(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSQLExecRes, error) {
	sqlExecParam.SQLExecRecord.Type = entity.DbSQLExecTypeQuery

	if err := d.checkSQLTrigger(ctx, sqlExecParam, "read"); err != nil {
		return nil, err
	}

	return d.doQuery(ctx, sqlExecParam, 0)
}

func (d *DbSQLExecAppImpl) doExecDDL(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSQLExecRes, error) {
	sqlExecParam.SQLExecRecord.Type = entity.DbSQLExecTypeDDL

	if err := d.checkSQLTrigger(ctx, sqlExecParam, "ddl"); err != nil {
		return nil, err
	}

	return d.doExec(ctx, sqlExecParam)
}

// schemaQualified 判定 DML 目标表是否带了 schema 限定（如 UPDATE s2.t、INSERT INTO s2.t）。
//
// 采集与回滚生成都只按「连接默认 schema」解析标识符：旧值回查 SELECT 的裸表名经 search_path
// 落到默认 schema，元数据查询（GetColumns/GetPrimaryKeys）以 current_schema() 过滤、无法解析
// 限定名。若原语句限定到非默认 schema 而默认 schema 恰有同名表，采集会读错表；更严重的是
// INSERT 的回滚是 DELETE，会删掉默认 schema 同名表的行（数据丢失），而语句界面照样显示成功。
// 故此处 fail-closed：带 schema 限定即跳过采集、不提供回滚（生成侧据空表名/空快照如实报
// 「无旧值/无插入快照」），也不给出指向错误表的回滚。完整跨 schema 回滚需把 schema 贯穿采集、
// record.Table、元数据解析与生成，属更大改动，另行跟踪。
func schemaQualified(schema string) bool {
	return strings.TrimSpace(schema) != ""
}

func (d *DbSQLExecAppImpl) doUpdate(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSQLExecRes, error) {
	dbConn := sqlExecParam.DbConn

	if err := d.checkSQLTrigger(ctx, sqlExecParam, "update"); err != nil {
		return nil, err
	}

	execRecord := sqlExecParam.SQLExecRecord
	execRecord.Type = entity.DbSQLExecTypeUpdate

	stmt := sqlExecParam.Stmt
	if stmt == nil {
		return d.doExec(ctx, sqlExecParam)
	}

	updatestmt, ok := stmt.(*sqlstmt.UpdateStmt)
	if !ok {
		return d.doExec(ctx, sqlExecParam)
	}

	// 不支持多表更新记录旧值
	if len(updatestmt.Tables) != 1 {
		logx.ErrorContext(ctx, "update SQL - logging old values only supports single-table updates")
		return d.doExec(ctx, sqlExecParam)
	}

	// schema 限定表 fail-closed：回查与生成都只认默认 schema，跨 schema 会采错表（见 schemaQualified）
	if schemaQualified(updatestmt.Tables[0].Schema) {
		logx.ErrorContext(ctx, "update SQL - logging old values does not support schema-qualified tables")
		return d.doExec(ctx, sqlExecParam)
	}

	// 带 LIMIT 的行限制 fail-closed：实际只影响 WHERE 匹配集的一个非确定子集，而回查 SELECT
	// 按 WHERE 捞回全部匹配行。UPDATE 会生成覆盖未变更行的多余回滚；无法凭 WHERE 还原真实影响集，故跳过采集
	if updatestmt.Limit != nil {
		logx.ErrorContext(ctx, "update SQL - logging old values does not support LIMIT/row-count restriction")
		return d.doExec(ctx, sqlExecParam)
	}

	tableName := updatestmt.Tables[0].Name
	tableAlias := updatestmt.Tables[0].Alias

	if tableName == "" {
		logx.ErrorContext(ctx, "update SQL - failed to get table name")
		return d.doExec(ctx, sqlExecParam)
	}
	execRecord.Table = tableName

	if updatestmt.Where == nil {
		logx.ErrorContext(ctx, "update SQL - there is no where condition")
		return d.doExec(ctx, sqlExecParam)
	}
	whereStr := updatestmt.Where.Text

	// 获取表全部主键列（联合主键多列），排除使用别名
	primaryKeys, err := dbConn.Metadata().GetPrimaryKeys(tableName)
	if err != nil {
		logx.ErrorfContext(ctx, "update SQL - failed to get primary key columns: %s", err.Error())
		return d.doExec(ctx, sqlExecParam)
	}

	// SET 列取自解析器（已剥引号），可能带表别名前缀（SET x.col）；用 lastSegment 归一为裸列名，
	// 与执行后快照、回滚生成侧（updateSetColumns）以及 OldValue 的键口径保持一致
	updateColumns := collx.ArrayMap[sqlstmt.Assignment, string](updatestmt.Set, func(a sqlstmt.Assignment) string {
		return lastSegment(a.Column)
	})

	// 目录真名索引：回查标识符必须先归一再引用（见 db_sql_ident.go）。
	// 元数据取不到时索引为空，各标识符按裸名输出——与引用改造前同行为，不会引用出一个不存在的名字
	ident := newIdentQuoter(dbConn)
	tableColumns, catalogErr := dbConn.Metadata().GetColumns(tableName)
	if catalogErr != nil {
		logx.ErrorfContext(ctx, "update SQL - failed to load column metadata of %s: %s", tableName, catalogErr.Error())
	}
	columnCatalog, catalogTable := newColumnCatalog(tableColumns)

	// SELECT 需带上全部主键列，否则记录的旧值无法定位行（联合主键少列即歧义）
	selectColumns := make([]string, 0, len(updateColumns)+len(primaryKeys))
	for _, column := range updateColumns {
		selectColumns = append(selectColumns, ident.catalogIdent(columnCatalog, column))
	}
	for _, pk := range primaryKeys {
		// 主键名取自元数据，已是目录真名
		selectColumns = append(selectColumns, ident.quote(pk))
	}
	fromClause := ident.tableIdent(catalogTable, tableName)
	if tableAlias != "" {
		fromClause += " " + ident.aliasIdent(tableAlias)
	}
	// 查询要更新字段数据的旧值，以及主键值（不变量：本回查与随后的 UPDATE 各自独立求值同一段 WHERE，仅当谓词确定时两次匹配集才相等；结构性行限制 LIMIT 已在上方 fail-closed，语义性易变谓词如 RAND() 是本采集模型的固有边界，至多让回滚漏改未采集行而不会写坏数据）
	selectSQL := fmt.Sprintf("SELECT %s FROM %s WHERE %s", strings.Join(selectColumns, ","), fromClause, whereStr)

	// WalkQuery 查出最多 sqlExecAuditRows 条数据；到上限以 StopWalkQueryError 提前终止扫描并保留已采部分
	// （与 doDelete 同策）。曾在此处「超限即报错」使整批旧值归零：一条影响 201 行的 UPDATE
	// 就让回滚能力整体失效。截断导致的缺失改由回滚生成侧的说明如实告知，见 rollbackTruncationNote
	maxRec := sqlExecAuditRows
	res := make([]map[string]any, 0, maxRec)
	if _, err := dbConn.WalkQueryRowsInTx(ctx, sqlExecParam.tx, selectSQL, func(row map[string]any, _ []*dbi.QueryColumn) error {
		res = append(res, row)
		if len(res) >= maxRec {
			return dbi.NewStopWalkQueryError("reached the maximum number of audit rows")
		}
		return nil
	}); err != nil {
		logx.ErrorfContext(ctx, "update SQL - failed to get the updated old value: %s", err.Error())
		res = nil
	}
	if len(res) > 0 {
		execRecord.OldValue = jsonx.ToStr(res)
	}

	execRes, execErr := d.doExec(ctx, sqlExecParam)
	if execErr == nil && len(res) > 0 && len(primaryKeys) > 0 && !updateTouchesPrimaryKey(updateColumns, primaryKeys) {
		// 执行成功后按旧主键回查「变更列+主键」的当前值，作为回滚生成的并发校验基准与 CAS 值来源。
		// SET 触及主键时跳过：主键已被改掉，按旧值回查会落到「恰好拥有这些主键值」的别的行上，
		// 采出的快照与新行无关，宁可没有（生成侧对空快照降级生成并如实说明）
		execRecord.NewValue = captureUpdatePostImage(ctx, sqlExecParam, res, updateColumns, primaryKeys)
	}
	return execRes, execErr
}

func (d *DbSQLExecAppImpl) doDelete(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSQLExecRes, error) {
	if err := d.checkSQLTrigger(ctx, sqlExecParam, "delete"); err != nil {
		return nil, err
	}

	dbConn := sqlExecParam.DbConn
	execRecord := sqlExecParam.SQLExecRecord
	execRecord.Type = entity.DbSQLExecTypeDelete

	stmt := sqlExecParam.Stmt
	if stmt == nil {
		return d.doExec(ctx, sqlExecParam)
	}

	deletestmt, ok := stmt.(*sqlstmt.DeleteStmt)
	if !ok {
		return d.doExec(ctx, sqlExecParam)
	}

	// 不支持多表删除记录旧值
	if len(deletestmt.Tables) != 1 {
		logx.ErrorContext(ctx, "delete SQL - logging old values only supports single-table deletion")
		return d.doExec(ctx, sqlExecParam)
	}

	// schema 限定表 fail-closed：回查与生成都只认默认 schema，跨 schema 会采错表（见 schemaQualified）
	if schemaQualified(deletestmt.Tables[0].Schema) {
		logx.ErrorContext(ctx, "delete SQL - logging old values does not support schema-qualified tables")
		return d.doExec(ctx, sqlExecParam)
	}

	// 带 LIMIT 的行限制 fail-closed：回查 SELECT 按 WHERE 捞回全部匹配行，而实际只删了其中一个子集。
	// 回滚会为未被删除的行生成 INSERT，撞主键报错或插出重复行（数据损坏）；无法凭 WHERE 还原真实影响集，故跳过采集
	if deletestmt.Limit != nil {
		logx.ErrorContext(ctx, "delete SQL - logging old values does not support LIMIT/row-count restriction")
		return d.doExec(ctx, sqlExecParam)
	}

	tableName := deletestmt.Tables[0].Name
	tableAlias := deletestmt.Tables[0].Alias

	if tableName == "" {
		logx.ErrorContext(ctx, "delete SQL - failed to get table name")
		return d.doExec(ctx, sqlExecParam)
	}
	execRecord.Table = tableName

	if deletestmt.Where == nil {
		logx.ErrorContext(ctx, "delete SQL - there is no where condition")
		return d.doExec(ctx, sqlExecParam)
	}

	whereStr := deletestmt.Where.Text
	// 查询删除数据（仅用于记录旧值审计，失败不影响删除执行）
	// 用 WalkQueryRows 遍历并计数截断，避免写死 MySQL/PG 的 LIMIT 语法（Oracle/DM/MSSQL 不识别会整条报错），
	// 与 doUpdate 的旧值采集保持一致的跨方言做法；达到上限以 StopWalkQueryError 提前终止扫描。
	// 表名按目录真名引用：书写形态在折叠大小写的方言上会引用出不存在的对象（见 db_sql_ident.go），
	// 元数据取不到则退回裸名，与逐字回放的 WHERE 同规则折叠
	const maxAuditRows = sqlExecAuditRows // 与回滚生成侧的截断判定同源，见 sqlExecAuditRows
	ident := newIdentQuoter(dbConn)
	catalogTable := ""
	if tables, err := dbConn.Metadata().GetTables(tableName); err == nil && len(tables) > 0 {
		catalogTable = tables[0].TableName
	}
	fromClause := ident.tableIdent(catalogTable, tableName)
	if tableAlias != "" {
		fromClause += " " + ident.aliasIdent(tableAlias)
	}
	selectSQL := fmt.Sprintf("SELECT * FROM %s WHERE %s", fromClause, whereStr)
	auditRes := make([]map[string]any, 0, maxAuditRows)
	if _, err := dbConn.WalkQueryRowsInTx(ctx, sqlExecParam.tx, selectSQL, func(row map[string]any, _ []*dbi.QueryColumn) error {
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

	return d.doExec(ctx, sqlExecParam)
}

func (d *DbSQLExecAppImpl) doInsert(ctx context.Context, sqlExecParam *sqlExecParam) (*dto.DbSQLExecRes, error) {
	if err := d.checkSQLTrigger(ctx, sqlExecParam, "insert"); err != nil {
		return nil, err
	}

	execRecord := sqlExecParam.SQLExecRecord
	execRecord.Type = entity.DbSQLExecTypeInsert

	stmt := sqlExecParam.Stmt
	if stmt == nil {
		return d.doExec(ctx, sqlExecParam)
	}

	insertstmt, ok := stmt.(*sqlstmt.InsertStmt)
	if !ok {
		return d.doExec(ctx, sqlExecParam)
	}

	// schema 限定表 fail-closed：INSERT 回滚是 DELETE，只认默认 schema 会删错同名表的行（见 schemaQualified）。
	// 跳过即不设 Table/NewValue，生成侧据空值如实报「无插入快照」而非删错表
	if schemaQualified(insertstmt.Table.Schema) {
		logx.ErrorContext(ctx, "insert SQL - logging insert values does not support schema-qualified tables")
		return d.doExec(ctx, sqlExecParam)
	}

	// 冲突处理形态 fail-closed：REPLACE / INSERT IGNORE / ON DUPLICATE KEY UPDATE / ON CONFLICT
	// 在目标行已存在时不新增行（忽略/改为更新/先删后插），而 INSERT 回滚是「删掉刚插入的行」：
	// 按 VALUES 字面量生成的 DELETE 会命中并删除那条既有行（数据丢失）。故跳过插入快照采集
	if insertstmt.HasConflictClause {
		logx.ErrorContext(ctx, "insert SQL - logging insert values does not support conflict-handling (REPLACE/IGNORE/ON DUPLICATE/ON CONFLICT)")
		return d.doExec(ctx, sqlExecParam)
	}

	execRecord.Table = insertstmt.Table.Name

	// 插入值快照：INSERT 的回滚是「删掉刚插入的行」，而新行没有执行前可定位的主键
	// （自增主键执行前不存在），字面量是唯一线索。仅接受可确定求值的纯字面量形态，
	// 函数/表达式/子查询整体放弃——猜出的定位值会删错行，宁可不生成（生成侧如实说明）
	execRecord.NewValue = insertSnapshotOf(insertstmt)

	return d.doExec(ctx, sqlExecParam)
}

func (d *DbSQLExecAppImpl) doQuery(ctx context.Context, sqlExecParam *sqlExecParam, maxRows int) (*dto.DbSQLExecRes, error) {
	dbConn := sqlExecParam.DbConn
	sql := sqlExecParam.SQL

	// 查询结果脱敏开关（服务端强制执行，fail-close模式下构建失败阻断查询）
	maskEnabled := d.maskEngine != nil && config.GetDbms().MaskEnabled

	res := make([]map[string]any, 0, 16)
	nowRows := 0
	// truncated 单独记一个事实而不是等调用方拿 len(res) 去猜上限：
	// 游标在「多出一行」时才停止，所以 nowRows > maxRows 就是「库里还有数据没取完」的直接证据
	truncated := false
	var rowMasker *masksvc.RowMasker
	cols, err := dbConn.WalkQueryRows(ctx, sql, func(row map[string]any, columns []*dbi.QueryColumn) error {
		nowRows++
		// 超过指定的最大查询记录数，则停止查询
		if maxRows != 0 && nowRows > maxRows {
			truncated = true
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
		SQL:       sql,
		Columns:   cols,
		Res:       res,
		Truncated: truncated,
		RowLimit:  maxRows,
	}, nil
}

// doExec 执行一条写语句。p.tx 非空时在批次事务内执行，否则走连接池自动提交
func (d *DbSQLExecAppImpl) doExec(ctx context.Context, p *sqlExecParam) (*dto.DbSQLExecRes, error) {
	rowsAffected, err := p.DbConn.TxExecContext(ctx, p.tx, p.SQL)
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
		SQL: p.SQL,
	}, err
}

// 语句分类逻辑已下沉至 sqlparser.Classify（单一入口），此处不再保留字符串兜底判定。
