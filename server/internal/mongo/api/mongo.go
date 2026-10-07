package api

import (
	"strings"

	"mayfly-go/internal/mongo/api/form"
	"mayfly-go/internal/mongo/api/vo"
	"mayfly-go/internal/mongo/application"
	"mayfly-go/internal/mongo/application/mongodoc"
	"mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/internal/mongo/imsg"
	"mayfly-go/internal/mongo/mgm"
	"mayfly-go/internal/pkg/event"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"

	"github.com/spf13/cast"
)

// PermDataExport 导出权限码。只在 API 层使用（不属于命令分级），因此不放进 mongodoc，
// 避免让人误以为 run-command 也能按这个码放行。
const PermDataExport = "mongo:data:export"

// Mongo 模块的 HTTP 入口。
//
// 本文件只做四件事：绑定参数、调用应用层、把错误断言成国际化业务错误、返回结果；
// 驱动调用与文档编解码一律在 application 层（见 application/mongo_data.go）。
// 方法按职责分散在同包的 mongo.go（实例与拓扑）、mongo_meta.go（集合元信息与索引）、
// mongo_data.go（文档读写）、mongo_ops.go（聚合/批量/导出）、mongo_cmd.go（命令）。
type Mongo struct {
	mongoApp   application.Mongo     `inject:"T"`
	mongoData  application.MongoData `inject:"T"`
	tagTreeApp tagapp.TagTreeReader  `inject:"T"`
}

func (m *Mongo) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// ---- 实例元数据 ----
		req.NewGet("", m.Mongos),
		req.NewPost("/test-conn", m.TestConn),
		req.NewPost("", m.Save).RequiredPermissionCode("mongo:save").Log(req.NewLogSaveI(imsg.LogMongoSave)),
		req.NewDelete(":id", m.DeleteMongo).RequiredPermissionCode("mongo:del").Log(req.NewLogSaveI(imsg.LogMongoDelete)),

		// ---- 拓扑浏览 ----
		req.NewGet(":id/databases", m.Databases),
		req.NewGet(":id/databases/:db/collections", m.Collections),
		req.NewGet(":id/databases/:db/collections/:coll/meta", m.CollectionMeta),
		req.NewPost(":id/databases/:db/collections/:coll/indexes", m.CreateIndexes).RequiredPermissionCode(mongodoc.PermDDLSave).Log(req.NewLogSaveI(imsg.LogMongoCreateIndex)),
		req.NewDelete(":id/databases/:db/collections/:coll/indexes/:index", m.DropIndex).RequiredPermissionCode(mongodoc.PermDDLDel).Log(req.NewLogSaveI(imsg.LogMongoDropIndex)),
		req.NewPost(":id/databases/:db/collections", m.CreateCollection).RequiredPermissionCode(mongodoc.PermDDLSave).Log(req.NewLogSaveI(imsg.LogMongoCreateCollection)),
		req.NewDelete(":id/databases/:db/collections/:coll", m.DropCollection).RequiredPermissionCode(mongodoc.PermDDLDel).Log(req.NewLogSaveI(imsg.LogMongoDropCollection)),
		req.NewDelete(":id/databases/:db", m.DropDatabase).RequiredPermissionCode(mongodoc.PermDDLDel).Log(req.NewLogSaveI(imsg.LogMongoDropDatabase)),

		// ---- 文档数据面 ----
		req.NewPost(":id/query", m.QueryDocs),
		req.NewPost(":id/aggregate", m.Aggregate).Log(req.NewLogSaveI(imsg.LogMongoAggregate)),
		req.NewPost(":id/docs", m.InsertDocs).RequiredPermissionCode(mongodoc.PermDataSave).Log(req.NewLogSaveI(imsg.LogInsertDocs)),
		req.NewPut(":id/doc", m.UpdateDoc).RequiredPermissionCode(mongodoc.PermDataSave).Log(req.NewLogSaveI(imsg.LogUpdateDocs)),
		req.NewPost(":id/docs/delete", m.DeleteDocs).RequiredPermissionCode(mongodoc.PermDataDel).Log(req.NewLogSaveI(imsg.LogDelDocs)),
		// 批量改与按主键删同样是删数据，用同一个权限码；两者都必须先确认命中数
		req.NewPost(":id/docs/update", m.UpdateByFilter).RequiredPermissionCode(mongodoc.PermDataSave).Log(req.NewLogSaveI(imsg.LogMongoBatchUpdate)),
		req.NewPost(":id/docs/delete-by-filter", m.DeleteByFilter).RequiredPermissionCode(mongodoc.PermDataDel).Log(req.NewLogSaveI(imsg.LogMongoBatchDelete)),
		// 导出是把数据带走的路径，单独权限码且默认不授予公共角色（脱敏能力接入前只能由管理员使用）
		req.NewPost(":id/export", m.ExportDocs).RequiredPermissionCode(PermDataExport).Log(req.NewLogSaveI(imsg.LogMongoExport)),

		// ---- 命令控制台 ----
		req.NewGet(":id/commands", m.Commands),
		// run-command 的权限取决于命令语义（读/写/删/结构/管理），由 handler 按 Classify 结果动态校验，
		// 因此这里不声明静态权限码
		req.NewPost(":id/run-command", m.RunCommand).Log(req.NewLogSaveI(imsg.LogMongoRunCmd)),
	}

	return req.NewConfs("/mongos", reqs[:]...)
}

// ============ 实例元数据 ============

func (m *Mongo) Mongos(rc *req.Ctx) {
	queryCond := rc.BindQuery[entity.MongoQuery]()

	// 不存在可访问标签，即没有可操作数据
	codes := m.tagTreeApp.GetAccountResourceCodes(rc.GetLoginAccount().Id, queryCond.TagPath, tagentity.TagTypeMongo)
	if len(codes) == 0 {
		rc.ResData = model.NewEmptyPageResult[*vo.Mongo]()
		return
	}
	queryCond.Codes = codes

	res, err := m.mongoApp.GetPageList(queryCond)
	biz.ErrIsNil(err)
	rc.ResData = vo.PageOfMongos(res)
}

// TestConn 测试连接。
//
// uri 留空且带了实例 id 时按「测试已保存的连接」处理：列表接口不再回传明文连接串，
// 编辑场景下用户手上没有 uri 原文，否则「测试连接」按钮在改过密码前无法使用。
func (m *Mongo) TestConn(rc *req.Ctx) {
	_, mongo := rc.BindJsonAndCopyTo[form.Mongo, entity.Mongo]()

	if mongo.Uri == "" {
		biz.IsTrueI(rc.MetaCtx, mongo.Id > 0, imsg.ErrMongoUriRequired)
		oldMongo, err := m.mongoApp.GetById(mongo.Id)
		biz.ErrIsNil(err)
		mongo.Uri = oldMongo.Uri
	}

	// 必须先判空再取 message：TestConn 成功返回 nil，直接对 nil 调 .Error() 会 panic，
	// 而 panic 被请求兜底成「server error 500」，表现就是「连得上反而报服务器错误」
	if err := m.mongoApp.TestConn(rc.MetaCtx, mongo); err != nil {
		biz.ErrIsNil(errorx.NewBizI(rc.MetaCtx, imsg.ErrMongoConnFailed, "detail", err.Error()))
	}
}

func (m *Mongo) Save(rc *req.Ctx) {
	mongoForm := rc.BindJson[form.Mongo]()

	// 日志与请求参数里只留脱敏连接串：明文 uri 含密码，落日志等于泄露凭证
	rc.ReqParam = collx.Kvs(
		"id", mongoForm.Id,
		"name", mongoForm.Name,
		"uri", mgm.MaskUri(mongoForm.Uri),
		"sshTunnelMachineId", mongoForm.SshTunnelMachineId,
		"tagCodePaths", mongoForm.TagCodePaths,
	)

	mongo := &entity.Mongo{
		Name:               mongoForm.Name,
		Uri:                mongoForm.Uri,
		SshTunnelMachineId: mongoForm.SshTunnelMachineId,
	}
	mongo.Id = mongoForm.Id

	biz.ErrIsNil(m.mongoApp.SaveMongo(rc.MetaCtx, mongo, mongoForm.TagCodePaths...))
}

// DeleteMongo 支持逗号分隔的批量删除（与全站资源删除入口一致）
func (m *Mongo) DeleteMongo(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr

	for _, idStr := range strings.Split(idsStr, ",") {
		mongoId := cast.ToUint64(idStr)
		biz.IsTrueI(rc.MetaCtx, mongoId > 0, imsg.ErrMongoInvalidId)
		biz.ErrIsNil(m.mongoApp.Delete(rc.MetaCtx, mongoId))
	}
}

// ============ 拓扑浏览 ============

func (m *Mongo) Databases(rc *req.Ctx) {
	conn := m.getConn(rc)

	databases, err := m.mongoData.ListDatabases(rc.MetaCtx, conn)
	biz.ErrIsNil(mongoError(rc.MetaCtx, err))
	rc.ResData = databases
}

func (m *Mongo) Collections(rc *req.Ctx) {
	conn := m.getConn(rc)
	database := databaseParam(rc)

	collections, err := m.mongoData.ListCollections(rc.MetaCtx, conn, database)
	biz.ErrIsNil(mongoError(rc.MetaCtx, err))
	rc.ResData = collections
}

func (m *Mongo) CreateCollection(rc *req.Ctx) {
	collForm := rc.BindJson[form.CreateCollectionForm]()
	conn := m.getConn(rc)
	database := databaseParam(rc)

	rc.ReqParam = collx.Kvs("mongo", conn.Info.Name, "database", database, "collection", collForm.Collection)
	m.publishResourceOp(rc, conn)

	biz.ErrIsNil(mongoError(rc.MetaCtx, m.mongoData.CreateCollection(rc.MetaCtx, conn, database, collForm.Collection)))
}

// DropCollection 删除集合。
//
// 集合名与库名都不可逆地销毁数据与索引结构，因此要求把目标名原样再输一遍：
// 仅靠一次「确定删除?」气泡确认不足以拦住点错行的操作。
func (m *Mongo) DropCollection(rc *req.Ctx) {
	conn := m.getConn(rc)
	database, collection := databaseParam(rc), collectionParam(rc)
	m.requireConfirmName(rc, collection)

	rc.ReqParam = collx.Kvs("mongo", conn.Info.Name, "database", database, "collection", collection)
	m.publishResourceOp(rc, conn)

	biz.ErrIsNil(mongoError(rc.MetaCtx, m.mongoData.DropCollection(rc.MetaCtx, conn, database, collection)))
}

func (m *Mongo) DropDatabase(rc *req.Ctx) {
	conn := m.getConn(rc)
	database := databaseParam(rc)
	m.requireConfirmName(rc, database)

	rc.ReqParam = collx.Kvs("mongo", conn.Info.Name, "database", database)
	m.publishResourceOp(rc, conn)

	biz.ErrIsNil(mongoError(rc.MetaCtx, m.mongoData.DropDatabase(rc.MetaCtx, conn, database)))
}

// ============ 公共辅助 ============

// getConn 取实例 id 并拿到已通过数据权限校验的连接。
//
// 权限校验在 application.GetMongoConn 内统一完成，所有数据面入口都会经过它，
// 因此新增接口不会因漏写鉴权而绕过资源可见性限制。
func (m *Mongo) getConn(rc *req.Ctx) *mgm.MongoConn {
	mongoId := cast.ToUint64(rc.PathParam("id"))
	biz.IsTrueI(rc.MetaCtx, mongoId > 0, imsg.ErrMongoInvalidId)

	conn, err := m.mongoApp.GetMongoConn(rc.MetaCtx, mongoId)
	biz.ErrIsNil(err)
	return conn
}

func databaseParam(rc *req.Ctx) string {
	database := rc.PathParam("db")
	biz.IsTrueI(rc.MetaCtx, database != "", imsg.ErrMongoDatabaseEmpty)
	return database
}

func collectionParam(rc *req.Ctx) string {
	collection := rc.PathParam("coll")
	biz.IsTrueI(rc.MetaCtx, collection != "", imsg.ErrMongoCollectionEmpty)
	return collection
}

// requireConfirmName 校验破坏性操作的确认名称与目标名一致。
func (m *Mongo) requireConfirmName(rc *req.Ctx, expect string) {
	biz.IsTrueI(rc.MetaCtx, rc.Query("confirmName") == expect, imsg.ErrMongoConfirmMismatch, "expect", expect)
}

// publishResourceOp 通知资源正在被操作，供资源树的在线状态与角标刷新。
//
// 只在写操作与结构变更时发：读操作高频且不代表资源被改动。
func (m *Mongo) publishResourceOp(rc *req.Ctx, conn *mgm.MongoConn) {
	if len(conn.Info.CodePath) == 0 {
		return
	}
	event.PublishResourceOp(rc.MetaCtx, conn.Info.CodePath)
}
