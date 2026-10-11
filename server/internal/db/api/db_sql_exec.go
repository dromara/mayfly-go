package api

import (
	"fmt"
	"io"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/application/sqlexec"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/writerx"
	"time"

	"github.com/spf13/cast"

	"strings"
)

type DbSQLExec struct {
	dbSQLExecApp sqlexec.DbSQLExec `inject:"T"`
}

func (d *DbSQLExec) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 获取所有数据库sql执行记录列表
		req.NewGet("/sql-execs", d.DbSQLExecs),

		// 依据执行记录当时采到的旧值生成回滚 SQL（只生成文本，不执行）。
		// 权限取「能执行SQL脚本」这一档：旧值里可能含明文数据，但能对着同一个库跑 SELECT 的人
		// 本就取到过它们；生成动作本身留痕，因为回滚文本会被带进工单或口头交接
		req.NewGet("/sql-execs/:execId/rollback-sql", d.RollbackSQL).Log(req.NewLogSaveI(imsg.LogDbGenRollback)).RequiredPermissionCode("db:sqlscript:run"),

		// 全量导出一次查询的结果集（不受结果集行数上限约束）。
		// 权限取「能执行SQL脚本」这一档：导出会重跑这条语句，不能宽于执行入口；
		// 产物是数据本身，得留痕「这次被拉走了多少行」——语句与行数写入日志字段，不落文件
		req.NewPost(":dbId/export-query", d.ExportQuery).Log(req.NewLogSaveI(imsg.LogDbExportQuery)).RequiredPermissionCode("db:sqlscript:run").NoRes(),
	}

	return req.NewConfs("/dbs", reqs[:]...)
}

// RollbackSQL 生成指定执行记录的回滚 SQL
func (d *DbSQLExec) RollbackSQL(rc *req.Ctx) {
	execId := cast.ToUint64(rc.PathParam("execId"))
	biz.IsTrue(execId > 0, "execId error")

	res, err := d.dbSQLExecApp.GenRollback(rc.MetaCtx, execId)
	biz.ErrIsNil(err)
	rc.ResData = res
}

// ExportQuery 全量导出查询结果集：直接写响应体，不走统一结果封装
func (d *DbSQLExec) ExportQuery(rc *req.Ctx) {
	dbId := cast.ToUint64(rc.PathParam("dbId"))
	biz.IsTrue(dbId > 0, "dbId error")

	reqForm := rc.BindJson[form.QueryExportForm]()
	// 格式留空走 csv：前端不必为了「默认值」再了解一次后端约定
	if strings.TrimSpace(reqForm.Format) == "" {
		reqForm.Format = "csv"
	}
	// 先记下「对哪个库做了什么」：导出中途失败时后面那次赋值跑不到，而审计恰恰需要在这时候
	// 看得到语句本体（操作日志在响应处理完之后才落，拿的就是这里的值）
	rc.ReqParam = collx.Kvs("dbId", dbId, "db", reqForm.Db, "format", reqForm.Format, "sql", reqForm.SQL)

	// 末次进度上报的累计行数，收尾时写进操作日志
	var exportRows int64

	// 产物写入器由应用层在校验全部通过后才创建，因此 close 也必须发生在其后：
	// 用局部变量持有，校验失败时它仍为 nil，响应也就是一次正常的 JSON 错误
	var gzipWriter *writerx.GzipWriter
	// exportDone 表示整批结果已写完并交到了格式层的收尾。只有这一条路径允许补 gzip 尾块：
	// 中途失败时统一响应层会把错误 JSON 追加进已开始的流里，此时若仍 Close 出合法尾块，
	// 前端按魔术号校验就会通过并存盘——用户拿到的是一份看起来正常、实际少了尾巴的数据文件。
	// 故意不 Close 才是这里正确的做法：产物解压即报错，失败出声而不是静默少行
	exportDone := false
	defer func() {
		if gzipWriter == nil || !exportDone {
			return
		}
		// 尾块写入失败会让整份文件解压报错，而界面已经告知过「导出成功」：必须留得下日志行
		if closeErr := gzipWriter.Close(); closeErr != nil {
			logx.ErrorfContext(rc.MetaCtx, "close query export gzip writer failed, dbId=%d, traceId=%s: %v", dbId, contextx.GetTraceId(rc.MetaCtx), closeErr)
		}
	}()

	biz.ErrIsNil(d.dbSQLExecApp.ExportQuery(rc.MetaCtx, &dto.QueryExportReq{
		DbId:   dbId,
		Db:     reqForm.Db,
		SQL:    reqForm.SQL,
		Format: reqForm.Format,
	}, func() (io.Writer, error) {
		headerName := fmt.Sprintf("%s-%s.%s.gz", reqForm.Db, time.Now().Format("20060102150405"), reqForm.Format)
		rc.Header("Content-Type", "application/octet-stream")
		rc.Header("Content-Disposition", contentDisposition(headerName))
		// 与 dump 一致地 gzip：导出不走上限，体积可能远大于页面结果，不压缩会白送一倍流量
		gzipWriter = writerx.NewGzipWriter(rc.GetWriter())
		return gzipWriter, nil
	}, func(rows int64) {
		exportRows = rows
	}))
	exportDone = true

	// 全量导出有意不受结果集行数上限约束，「被拉走了多少行」就必须能在审计里回答出来
	// （与 dump 入口的 exportRows 同一口径）：取末次上报而非累加，进度回调每批报的都是累计值
	rc.ReqParam = collx.Kvs("dbId", dbId, "db", reqForm.Db, "format", reqForm.Format, "sql", reqForm.SQL, "exportRows", exportRows)
}

func (d *DbSQLExec) DbSQLExecs(rc *req.Ctx) {
	queryCond := rc.BindQuery[entity.DbSQLExecQuery]()
	if statusStr := rc.Query("status"); statusStr != "" {
		queryCond.Status = collx.ArrayMap[string, int8](strings.Split(statusStr, ","), func(val string) int8 {
			return cast.ToInt8(val)
		})
	}
	res, err := d.dbSQLExecApp.GetPageList(queryCond)
	biz.ErrIsNil(err)
	rc.ResData = res
}
