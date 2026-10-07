package api

import (
	"mayfly-go/internal/mongo/api/form"
	"mayfly-go/internal/mongo/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

// CollectionMeta 集合元信息（统计 + 索引）。只读，随查询共用资源可见性约束。
func (m *Mongo) CollectionMeta(rc *req.Ctx) {
	conn := m.getConn(rc)
	database, collection := databaseParam(rc), collectionParam(rc)

	meta, err := m.mongoData.CollectionMeta(rc.MetaCtx, conn, database, collection)
	biz.ErrIsNil(mongoError(rc.MetaCtx, err))
	rc.ResData = meta
}

// CreateIndexes 创建索引。权限用结构变更码：索引定义改的是集合结构而非文档内容。
func (m *Mongo) CreateIndexes(rc *req.Ctx) {
	indexForm := rc.BindJson[form.CreateIndexesForm]()
	conn := m.getConn(rc)
	database, collection := databaseParam(rc), collectionParam(rc)

	rc.ReqParam = collx.Kvs(
		"mongo", conn.Info,
		"database", database,
		"collection", collection,
		"indexes", string(indexForm.Specs),
	)
	m.publishResourceOp(rc, conn)

	biz.ErrIsNil(mongoError(rc.MetaCtx, m.mongoData.CreateIndexes(rc.MetaCtx, conn, database, collection, indexForm.Specs)))
}

// DropIndex 删除索引。
//
// 要求把索引名原样再输一遍：删索引会让依赖它的查询退化全集合扫描并影响线上性能，
// 一次气泡确认不足以拦住点错的行。
func (m *Mongo) DropIndex(rc *req.Ctx) {
	conn := m.getConn(rc)
	database, collection := databaseParam(rc), collectionParam(rc)
	index := rc.PathParam("index")
	biz.IsTrueI(rc.MetaCtx, index != "", imsg.ErrMongoIndexSpecsInvalid)
	m.requireConfirmName(rc, index)

	rc.ReqParam = collx.Kvs(
		"mongo", conn.Info,
		"database", database,
		"collection", collection,
		"index", index,
	)
	m.publishResourceOp(rc, conn)

	biz.ErrIsNil(mongoError(rc.MetaCtx, m.mongoData.DropIndex(rc.MetaCtx, conn, database, collection, index)))
}
