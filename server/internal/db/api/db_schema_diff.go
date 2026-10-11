package api

// 结构对比（库间表结构漂移检测）的 HTTP 入口。
//
// 单独一个 handler 而不是挂在会话文件里：它比的是两个库，入参与权限语义都与「按库操作」不同。
// 权限上不另设功能码，与元数据类入口（表清单、列信息）同源：能否看到某个库的结构由连接层的
// 资源权限决定，而本接口两侧都要取一次连接，因此两侧各自的资源可达性都会被校验。

import (
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/application/schema"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"

	"github.com/spf13/cast"
)

type DbSchemaDiff struct {
	schemaDiffApp schema.SchemaDiff     `inject:"T"`
	baselineApp   schema.SchemaBaseline `inject:"T"`
}

func (d *DbSchemaDiff) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 比对两个库的表结构，返回有差异或比不了的表清单
		req.NewPost("/schema-diff", d.Diff),

		// 建立/刷新表结构基线（以当前结构为准覆盖）
		req.NewPost("/schema-baseline", d.SaveBaseline).Log(req.NewLogSaveI(imsg.LogDbSchemaBaselineSave)),

		// 查看某个库的基线行
		req.NewGet("/schema-baseline", d.ListBaseline),

		// 立即比对一次（不推通知，结果直接回给界面）
		req.NewPost("/schema-baseline/check", d.CheckBaseline),

		req.NewDelete("/schema-baseline", d.DeleteBaseline).Log(req.NewLogSaveI(imsg.LogDbSchemaBaselineDel)),
	}

	return req.NewConfs("/dbs", reqs[:]...)
}

// Diff 结构对比
func (d *DbSchemaDiff) Diff(rc *req.Ctx) {
	reqForm := rc.BindJson[form.DbSchemaDiffForm]()
	rc.ReqParam = collx.Kvs(
		"srcDbId", reqForm.SrcDbId, "srcDb", reqForm.SrcDb,
		"dstDbId", reqForm.DstDbId, "dstDb", reqForm.DstDb,
		"tableCount", len(reqForm.Tables),
	)

	result, err := d.schemaDiffApp.Diff(rc.MetaCtx, &schema.DiffReq{
		SrcDbId: reqForm.SrcDbId,
		SrcDb:   reqForm.SrcDb,
		DstDbId: reqForm.DstDbId,
		DstDb:   reqForm.DstDb,
		Tables:  reqForm.Tables,
	})
	biz.ErrIsNil(err)
	rc.ResData = result
}

// SaveBaseline 建立或刷新基线
func (d *DbSchemaDiff) SaveBaseline(rc *req.Ctx) {
	reqForm := rc.BindJson[form.DbSchemaBaselineForm]()
	rc.ReqParam = collx.Kvs("dbId", reqForm.Id, "db", reqForm.Db, "tableCount", len(reqForm.Tables))

	result, err := d.baselineApp.Save(rc.MetaCtx, reqForm.Id, reqForm.Db, reqForm.Tables)
	biz.ErrIsNil(err)
	rc.ResData = result
}

func (d *DbSchemaDiff) ListBaseline(rc *req.Ctx) {
	dbId := cast.ToUint64(rc.Query("dbId"))
	biz.IsTrue(dbId > 0, "dbId error")

	rows, err := d.baselineApp.List(rc.MetaCtx, dbId, rc.Query("db"))
	biz.ErrIsNil(err)
	rc.ResData = rows
}

// CheckBaseline 立即比对一次：notify 为假，用户在看结果，再推一条通知就是叠床架屋
func (d *DbSchemaDiff) CheckBaseline(rc *req.Ctx) {
	reqForm := rc.BindJson[form.DbSchemaBaselineForm]()
	rc.ReqParam = collx.Kvs("dbId", reqForm.Id, "db", reqForm.Db)

	result, err := d.baselineApp.Check(rc.MetaCtx, reqForm.Id, reqForm.Db, false)
	biz.ErrIsNil(err)
	rc.ResData = result
}

func (d *DbSchemaDiff) DeleteBaseline(rc *req.Ctx) {
	dbId := cast.ToUint64(rc.Query("dbId"))
	biz.IsTrue(dbId > 0, "dbId error")
	rc.ReqParam = collx.Kvs("dbId", dbId, "db", rc.Query("db"))

	biz.ErrIsNil(d.baselineApp.Delete(rc.MetaCtx, dbId, rc.Query("db")))
}
