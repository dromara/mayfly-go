package api

import (
	"errors"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/application/sqlexec"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/imsg"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/spf13/cast"
)

// DbSession 活动会话与执行取消入口。
//
// 与「SQL 执行」的两条通道分开，是因为它们的授权语义不同：
//   - 取消自己的执行：只需是发起者本人（属主校验在应用层），不需要额外功能权限；
//   - 查看/取消实例上的任意会话：能停掉别人的查询，属运维能力，必须单独授予功能权限码
type DbSession struct {
	dbApp        application.Db        `inject:"T"`
	dbSQLExecApp sqlexec.DbSQLExec     `inject:"T"`
	tagApp       tagapp.TagTreeChecker `inject:"T"`
}

func (d *DbSession) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 活动会话列表：会话属实例级事实，但仍按「所选库」做资源可达性校验，与其余库级入口同源
		req.NewGet("/sessions", d.Sessions).RequiredPermissionCode("db:session:list"),

		// 取消指定会话（停掉该会话当前语句）
		req.NewPost("/sessions/cancel", d.CancelSession).Log(req.NewLogSaveI(imsg.LogDbCancelSession)).RequiredPermissionCode("db:session:cancel"),

		// 取消自己发起的一次执行
		req.NewPost("/cancel-exec", d.CancelExec).Log(req.NewLogSaveI(imsg.LogDbCancelExec)),

		// 慢查询聚合：读 performance_schema / pg_stat_statements 的统计，不执行任何用户语句
		req.NewGet("/:dbId/slow-queries", d.SlowQueries).RequiredPermissionCode("db:session:list"),

		// 账号盘点：只读，列出实例账号与高危标志（不做任何 GRANT / CREATE USER）
		req.NewGet("/:dbId/accounts", d.Accounts).RequiredPermissionCode("db:session:list"),

		// 恢复能力：PITR 前提（binlog / WAL 归档）是否具备，不具备时给原因
		req.NewGet("/:dbId/recovery-readiness", d.RecoveryReadiness).RequiredPermissionCode("db:session:list"),

		// 指标趋势：读平台自己的采样表，不连库（库不可达时历史曲线仍可看）
		req.NewGet("/:dbId/metrics", d.Metrics).RequiredPermissionCode("db:session:list"),

		// 锁等待链：只读系统视图
		req.NewGet("/:dbId/lock-waits", d.LockWaits).RequiredPermissionCode("db:session:list"),

		// 执行计划：只读分析（实现写死不带 ANALYZE，看计划不会改动数据），与执行期同一套语句切割口径
		req.NewPost("/sql-explain", d.ExplainSQL).Log(req.NewLogSaveI(imsg.LogDbExplainPlan)).RequiredPermissionCode("db:sqlscript:run"),
	}

	return req.NewConfs("/dbs", reqs[:]...)
}

// ExplainSQL 取一批语句的执行计划。
//
// 权限取「能执行SQL脚本」这一档：看计划是执行前的一步，没执行权限的人拿它做调优探测无意义；
// 且后端只用不带 ANALYZE 的 EXPLAIN，本入口不会因为「看一下」而改数据
func (d *DbSession) ExplainSQL(rc *req.Ctx) {
	dbId, dbName, sqls := rcBindSqlAnalysis(rc)

	d.accessibleConn(rc, dbId, dbName)
	plans, err := d.dbApp.Explain(rc.MetaCtx, dbId, dbName, sqls)
	biz.ErrIsNil(err)
	rc.ResData = plans
}

// rcBindSqlAnalysis 解析只读分析入参：把 sql 原文与已切割的 sqls 合并成一批语句
func rcBindSqlAnalysis(rc *req.Ctx) (uint64, string, []string) {
	reqForm := rc.BindJson[form.DbSqlAnalysisForm]()
	dbId := reqForm.Id
	dbName := reqForm.Db
	// 允许两种给法：整段脚本（sql）或已分好的语句数组（sqls）；切分统一由后端按目标库方言完成
	sqls := reqForm.SQLs
	if strings.TrimSpace(reqForm.SQL) != "" {
		sqls = append(sqls, reqForm.SQL)
	}
	biz.IsTrue(len(sqls) > 0, "sqls cannot be empty")
	rc.ReqParam = collx.Kvs("dbId", dbId, "db", dbName, "statementCount", len(sqls))
	return dbId, dbName, sqls
}

// accessibleConn 取连接并校验资源可达性：会话与分析类入口都不能仅凭实例/库 id 就能看到别人的库
func (d *DbSession) accessibleConn(rc *req.Ctx, dbId uint64, dbName string) *dbi.DbConn {
	dbConn, err := d.dbApp.GetDbConn(rc.MetaCtx, dbId, dbName)
	biz.ErrIsNil(err)
	biz.ErrIsNilAppendErr(d.tagApp.CanAccess(rc.GetLoginAccount().Id, dbConn.Info.CodePath...), "%s")
	return dbConn
}

func (d *DbSession) Sessions(rc *req.Ctx) {
	// 该路由挂在 /dbs/sessions 下（无 :dbId 路径参数），库信息一律按查询参数取
	dbId := cast.ToUint64(rc.Query("dbId"))
	biz.IsTrue(dbId > 0, "dbId error")
	dbName := rc.Query("db")
	biz.NotEmpty(dbName, "db cannot be empty")

	d.accessibleConn(rc, dbId, dbName)

	sessions, err := d.dbApp.ListSessions(rc.MetaCtx, dbId, dbName)
	biz.ErrIsNilAppendErr(err, "list database sessions failed: %s")
	rc.ResData = sessions
}

func (d *DbSession) CancelSession(rc *req.Ctx) {
	reqForm := rc.BindJson[form.DbSessionCancelForm]()
	dbId := reqForm.Id
	dbName := reqForm.Db
	sessionId := reqForm.SessionId
	biz.NotEmpty(sessionId, "sessionId cannot be empty")
	rc.ReqParam = collx.Kvs("dbId", dbId, "db", dbName, "sessionId", sessionId)

	d.accessibleConn(rc, dbId, dbName)
	biz.ErrIsNil(d.dbApp.CancelSession(rc.MetaCtx, dbId, dbName, sessionId))
}

// CancelExec 中断一次仍在执行的交互式执行。
//
// 不加功能权限码：能不能停，取决于「这条执行是不是你发起的」，由应用层属主校验把关
func (d *DbSession) CancelExec(rc *req.Ctx) {
	reqForm := rc.BindJson[form.DbCancelExecForm]()
	execId := reqForm.ExecId
	biz.NotEmpty(execId, "execId cannot be empty")
	rc.ReqParam = execId

	if err := d.dbSQLExecApp.CancelExec(rc.MetaCtx, execId, rc.GetLoginAccount().Id); err != nil {
		// 登记表的错误全是业务错误（属主校验/未找到/刚刚结束），原样 panic 保住分流码：
		// biz.ErrIsNil 会重包成 code=400，4003 一丢前端就压不掉「刚刚结束，无需取消」的
		// 红色 toast，操作者会被一个不是错误的事实喊话（与 flowapp.PreserveDecisionCode 同一接缝）。
		// 用 errors.As 而非直接类型断言：日后应用层若给错误加一层上下文包装，仍能穿透保住分流码
		var bizErr *errorx.BizError
		if errors.As(err, &bizErr) {
			panic(bizErr)
		}
		biz.ErrIsNil(err)
	}
}

// SlowQueries 实例级慢查询聚合（按累计耗时降序）
// @router /api/dbs/:dbId/slow-queries [get]
func (d *DbSession) SlowQueries(rc *req.Ctx) {
	dbId, dbName := getDbId(rc), rc.Query("db")
	d.accessibleConn(rc, dbId, dbName)
	queries, err := d.dbApp.SlowQueries(rc.MetaCtx, dbId, dbName, cast.ToInt(rc.Query("limit")))
	biz.ErrIsNil(err)
	rc.ResData = queries
}

// LockWaits 实例级锁等待链（等待方 → 阻塞方）
// @router /api/dbs/:dbId/lock-waits [get]
func (d *DbSession) LockWaits(rc *req.Ctx) {
	dbId, dbName := getDbId(rc), rc.Query("db")
	d.accessibleConn(rc, dbId, dbName)
	waits, err := d.dbApp.LockWaits(rc.MetaCtx, dbId, dbName)
	biz.ErrIsNil(err)
	rc.ResData = waits
}

// Metrics 实例指标趋势（来自平台采样表，不连接目标库）
// @router /api/dbs/:dbId/metrics [get]
func (d *DbSession) Metrics(rc *req.Ctx) {
	dbId := getDbId(rc)
	dbEntity, err := d.dbApp.GetById(dbId)
	biz.ErrIsNil(err)
	biz.IsTrue(dbEntity != nil && dbEntity.InstanceId > 0, "db instance not found")
	series, err := application.GetDbMetricApp().Series(rc.MetaCtx, dbEntity.InstanceId, rc.Query("metric"), cast.ToInt(rc.Query("minutes")))
	biz.ErrIsNil(err)
	rc.ResData = series
}

// RecoveryReadiness 时间点恢复前提探测（只读，不做任何恢复动作）
// @router /api/dbs/:dbId/recovery-readiness [get]
func (d *DbSession) RecoveryReadiness(rc *req.Ctx) {
	readiness, err := application.GetRecoveryApp().Readiness(rc.MetaCtx, getDbId(rc), rc.Query("db"))
	biz.ErrIsNil(err)
	rc.ResData = readiness
}

// Accounts 实例账号盘点（只读）
// @router /api/dbs/:dbId/accounts [get]
func (d *DbSession) Accounts(rc *req.Ctx) {
	overview, err := application.GetAccountsApp().Overview(rc.MetaCtx, getDbId(rc), rc.Query("db"))
	biz.ErrIsNil(err)
	rc.ResData = overview
}
