package api

import (
	"mayfly-go/internal/mongo/api/form"
	"mayfly-go/internal/mongo/application/dto"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

// QueryDocs 按条件分页查询文档。
//
// 只读操作：受资源可见性约束（getConn 内统一校验），不额外要求写权限。
// filter/sort/projection 以原始 JSON 文本进入、由 decodeJSONField 解成有序 BSON 文档，
// 因此复合排序键的顺序不会被 map 迭代序打乱。
func (m *Mongo) QueryDocs(rc *req.Ctx) {
	queryForm := rc.BindJson[form.QueryDocsForm]()
	conn := m.getConn(rc)

	query := &dto.DocQuery{
		Database:   queryForm.Database,
		Collection: queryForm.Collection,
		Filter:     decodeJSONField(rc, queryForm.Filter, "filter"),
		Sort:       decodeJSONField(rc, queryForm.Sort, "sort"),
		Projection: decodeJSONField(rc, queryForm.Projection, "projection"),
		Skip:       max(queryForm.Skip, 0),
		Limit:      queryForm.Limit,
		WithCount:  queryForm.WithCount,
	}

	page, err := m.mongoData.QueryDocuments(rc.MetaCtx, conn, query)
	biz.ErrIsNil(mongoError(rc.MetaCtx, err))
	rc.ResData = page
}

// InsertDocs 插入一个或多个文档。
func (m *Mongo) InsertDocs(rc *req.Ctx) {
	insertForm := rc.BindJson[form.InsertDocsForm]()
	conn := m.getConn(rc)

	docs := decodeJSONDocs(rc, insertForm.Docs, "docs")
	// 空数组是合法 JSON 写法但等于「什么都没做」，由应用层报 ErrNothingToWrite，
	// 这里不再重复一次判据

	rc.ReqParam = collx.Kvs(
		"mongo", conn.Info,
		"database", insertForm.Database,
		"collection", insertForm.Collection,
		"count", len(docs),
		"docs", string(insertForm.Docs),
	)
	m.publishResourceOp(rc, conn)

	res, err := m.mongoData.InsertDocuments(rc.MetaCtx, conn, &dto.DocInsert{
		Database:   insertForm.Database,
		Collection: insertForm.Collection,
		Docs:       docs,
	})
	biz.ErrIsNil(mongoError(rc.MetaCtx, err))
	rc.ResData = res
}

// UpdateDoc 更新单个文档。
//
// 调用方提交「编辑后的完整文档」，$set/$unset 由服务端与当前存储值比对得出：
// 从 JSON 里删掉一个 key 才真的等价于删除字段，主键也不可被调用方改写。
// baseHash 让并发编辑显式冲突，而不是后保存的人静默覆盖前一个人。
func (m *Mongo) UpdateDoc(rc *req.Ctx) {
	updateForm := rc.BindJson[form.UpdateDocForm]()
	conn := m.getConn(rc)

	doc := decodeJSONField(rc, updateForm.Doc, "doc")

	rc.ReqParam = collx.Kvs(
		"mongo", conn.Info,
		"database", updateForm.Database,
		"collection", updateForm.Collection,
		"doc", string(updateForm.Doc),
	)
	m.publishResourceOp(rc, conn)

	res, err := m.mongoData.UpdateDocument(rc.MetaCtx, conn, &dto.DocUpdate{
		Database:   updateForm.Database,
		Collection: updateForm.Collection,
		IDToken:    updateForm.IDToken,
		BaseHash:   updateForm.BaseHash,
		Doc:        doc,
	})
	biz.ErrIsNil(mongoError(rc.MetaCtx, err))
	rc.ResData = res
}

// DeleteDocs 按主键令牌批量删除。
//
// 用 POST 承载：主键值可能含逗号与特殊字符，只能按 JSON 数组放在请求体里传。
// 结果里的 deletedCount 是实际删除条数，可能小于请求条数（他人已先删掉部分文档）。
func (m *Mongo) DeleteDocs(rc *req.Ctx) {
	deleteForm := rc.BindJson[form.DeleteDocsForm]()
	conn := m.getConn(rc)

	rc.ReqParam = collx.Kvs(
		"mongo", conn.Info,
		"database", deleteForm.Database,
		"collection", deleteForm.Collection,
		"count", len(deleteForm.IDTokens),
	)
	m.publishResourceOp(rc, conn)

	res, err := m.mongoData.DeleteDocuments(rc.MetaCtx, conn, &dto.DocDelete{
		Database:   deleteForm.Database,
		Collection: deleteForm.Collection,
		IDTokens:   deleteForm.IDTokens,
	})
	biz.ErrIsNil(mongoError(rc.MetaCtx, err))
	rc.ResData = res
}
