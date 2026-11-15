package api

// SQL 草稿（自然语言转 SQL）的 HTTP 入口。
//
// 单独一个 handler 而不是塞进 db.go：它的产物是文本草稿而不是库操作结果，权限与
// 审计口径也不同（生成不触达数据、不执行语句）。
//
// 权限码复用 db:sqlscript:run：能在这张库上跑脚本的人才需要生成脚本。刻意不新开
// 功能码——新码若没在权限表里建行，上线即除 admin 外全体 403（仓库已踩过）。

import (
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/application/sqldraft"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"

	"github.com/spf13/cast"
)

type DbSqlDraft struct {
	draftApp sqldraft.Service `inject:"T"`
}

func (d *DbSqlDraft) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 按描述生成 SQL 草稿：只读结构 + 一次模型调用，不执行任何语句
		req.NewPost(":dbId/sql-draft", d.Draft).
			Log(req.NewLogSaveI(imsg.LogDbSqlDraft)).
			RequiredPermissionCode("db:sqlscript:run"),
	}
	return req.NewConfs("/dbs", reqs[:]...)
}

// Draft 生成 SQL 草稿
func (d *DbSqlDraft) Draft(rc *req.Ctx) {
	reqForm := rc.BindJson[form.SqlDraft]()
	dbId := cast.ToUint64(rc.PathParam("dbId"))
	// 审计只记「谁在哪张库上要了哪种意图 + 描述摘要」：prompt 原文可能含业务数据，
	// 整段落库等于把用户输入旁路成了日志正文
	rc.ReqParam = collx.Kvs(
		"dbId", dbId, "db", reqForm.Db,
		"intent", reqForm.Intent,
		"promptLength", len([]rune(reqForm.Prompt)),
		"hasSelection", reqForm.Selection != "",
		"isRefine", reqForm.RefineOf != "",
		"includePlans", reqForm.IncludePlans,
	)

	draft, err := d.draftApp.Draft(rc.MetaCtx, &sqldraft.Req{
		DbId:            dbId,
		Db:              reqForm.Db,
		Intent:          reqForm.Intent,
		Prompt:          reqForm.Prompt,
		Selection:       reqForm.Selection,
		AnchorStatement: reqForm.AnchorStatement,
		RefineOf:        reqForm.RefineOf,
		IncludePlans:    reqForm.IncludePlans,
	})
	biz.ErrIsNil(err)
	rc.ResData = draft
}
