package api

// 文生 SQL 业务词典（语义层）的 HTTP 入口。
//
// 权限码沿用 db:save：与表访问控制同理——这是「编辑某个库的配置」，不是一项独立业务
// 能力，另造代码会让同一件事在角色配置页里分成两处勾选。
//
// 只按 dbId 列出全部条目（不按库过滤）：管理界面要能看见别的库登记了什么，否则
// 「同一术语在两个库口径不同」这种问题在界面上根本发现不了。生成侧的按库取数走
// sqldraft.Terms.ListForSchema，两个视图不共用一个查询语义。

import (
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/application/sqldraft"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"

	"github.com/spf13/cast"
)

type DbAiTerm struct {
	termsApp sqldraft.Terms        `inject:"T"`
	dbApp    application.Db        `inject:"T"`
	tagApp   tagapp.TagTreeService `inject:"T"`
}

// assertDbAccess 资源可达校验：权限码只说明「能改配置」，不说明「能改哪个库的配置」。
func (d *DbAiTerm) assertDbAccess(rc *req.Ctx, dbId uint64) {
	biz.IsTrue(dbId > 0, "dbId error")
	db, err := d.dbApp.GetById(dbId)
	biz.ErrIsNil(err)
	biz.IsTrue(db != nil && db.Code != "", "db not found")
	biz.ErrIsNilAppendErr(d.tagApp.CanAccessByCode(rc.MetaCtx, int8(tagentity.TagTypeDb), db.Code), "%s")
}

func (d *DbAiTerm) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("/ai-terms", d.List).RequiredPermissionCode("db:save"),
		req.NewPost("/ai-terms", d.Save).Log(req.NewLogSaveI(imsg.LogDbAiTermSave)).RequiredPermissionCode("db:save"),
		req.NewPut("/ai-terms", d.Save).Log(req.NewLogSaveI(imsg.LogDbAiTermSave)).RequiredPermissionCode("db:save"),
		req.NewDelete("/ai-terms/:id", d.Delete).Log(req.NewLogSaveI(imsg.LogDbAiTermDelete)).RequiredPermissionCode("db:save"),
	}

	return req.NewConfs("/dbs", reqs[:]...)
}

// List 某库资源的业务词典
// @router /api/dbs/ai-terms [get]
func (d *DbAiTerm) List(rc *req.Ctx) {
	dbId := cast.ToUint64(rc.Query("dbId"))
	d.assertDbAccess(rc, dbId)
	res, err := d.termsApp.ListByDb(rc.MetaCtx, dbId)
	biz.ErrIsNil(err)
	rc.ResData = res
}

// Save 保存词典条目（同一库同名术语按更新处理）
// @router /api/dbs/ai-terms [post]
func (d *DbAiTerm) Save(rc *req.Ctx) {
	f := rc.BindJson[form.DbAiTermForm]()
	rc.ReqParam = f

	d.assertDbAccess(rc, f.DbId)
	term := &entity.DbAiTerm{
		DbId:    f.DbId,
		Schema:  f.Schema,
		Term:    f.Term,
		Tables:  f.Tables,
		Details: f.Details,
	}
	term.Id = f.Id
	biz.ErrIsNil(d.termsApp.SaveTerm(rc.MetaCtx, term))
}

// Delete 删除词典条目
// @router /api/dbs/ai-terms/:id [delete]
func (d *DbAiTerm) Delete(rc *req.Ctx) {
	id := cast.ToUint64(rc.PathParam("id"))
	// 按条目自己的库判可达，而不是信请求里带的 dbId：后者能让调用方拿一个「我有权的库」
	// 去删他库条目（表访问控制同一手法）
	entry, err := d.termsApp.GetTerm(rc.MetaCtx, id)
	biz.ErrIsNil(err)
	d.assertDbAccess(rc, entry.DbId)
	rc.ReqParam = collx.Kvs("id", id, "dbId", entry.DbId)
	biz.ErrIsNil(d.termsApp.DeleteTerm(rc.MetaCtx, entry.DbId, id))
}
