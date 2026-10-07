package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"mayfly-go/internal/mongo/api/form"
	"mayfly-go/internal/mongo/api/vo"
	"mayfly-go/internal/mongo/application/dto"
	"mayfly-go/internal/mongo/application/mongodoc"
	"mayfly-go/internal/mongo/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Aggregate 执行聚合管道。
//
// 权限按聚合语义动态判定：含 $out/$merge 即为写文档；explain 不执行任何 stage，因此按只读放行。
// 路由上不声明静态权限码 —— 声明了就无法区分这两种情形，只能一律按最严的来。
func (m *Mongo) Aggregate(rc *req.Ctx) {
	aggForm := rc.BindJson[form.AggregateForm]()
	conn := m.getConn(rc)

	pipeline := decodePipeline(rc, aggForm.Pipeline)
	_, level, err := mongodoc.Classify(mongodoc.PipelineCommand(aggForm.Collection, pipeline))
	biz.ErrIsNil(mongoError(rc.MetaCtx, err))
	if !aggForm.Explain {
		m.requireCmdPerm(rc, "aggregate", level)
	}

	rc.ReqParam = collx.Kvs(
		"mongo", conn.Info,
		"database", aggForm.Database,
		"collection", aggForm.Collection,
		"explain", aggForm.Explain,
		"pipeline", string(aggForm.Pipeline),
	)
	// explain 不改动数据，不上报资源操作；真正的聚合可能写出结果，按用过资源对待
	if !aggForm.Explain {
		m.publishResourceOp(rc, conn)
	}

	page, err := m.mongoData.Aggregate(rc.MetaCtx, conn, &dto.AggQuery{
		Database:     aggForm.Database,
		Collection:   aggForm.Collection,
		Pipeline:     pipeline,
		AllowDiskUse: aggForm.AllowDiskUse,
		Explain:      aggForm.Explain,
	})
	biz.ErrIsNil(mongoError(rc.MetaCtx, err))
	rc.ResData = page
}

// UpdateByFilter 按条件批量更新。
//
// expectCount 来自前一步预览：服务端先统计再比对，不一致即中止。
// 少了这道护栏，一个写错的 filter 会静默改掉整个集合且无法回退。
func (m *Mongo) UpdateByFilter(rc *req.Ctx) {
	batchForm := rc.BindJson[form.BatchUpdateForm]()
	conn := m.getConn(rc)

	spec, err := mongodoc.DecodeUpdateSpec(batchForm.Update)
	biz.ErrIsNil(mongoError(rc.MetaCtx, err))

	rc.ReqParam = collx.Kvs(
		"mongo", conn.Info,
		"database", batchForm.Database,
		"collection", batchForm.Collection,
		"filter", string(batchForm.Filter),
		"update", string(batchForm.Update),
		"expectCount", batchForm.ExpectCount,
		"upsert", batchForm.Upsert,
	)
	m.publishResourceOp(rc, conn)

	res, err := m.mongoData.UpdateByFilter(rc.MetaCtx, conn, &dto.BatchUpdate{
		Database:    batchForm.Database,
		Collection:  batchForm.Collection,
		Filter:      decodeJSONField(rc, batchForm.Filter, "filter"),
		Update:      spec,
		Upsert:      batchForm.Upsert,
		ExpectCount: batchForm.ExpectCount,
	})
	biz.ErrIsNil(mongoError(rc.MetaCtx, err))
	rc.ResData = res
}

// DeleteByFilter 按条件批量删除，同样要求先确认命中数
func (m *Mongo) DeleteByFilter(rc *req.Ctx) {
	batchForm := rc.BindJson[form.BatchDeleteForm]()
	conn := m.getConn(rc)

	rc.ReqParam = collx.Kvs(
		"mongo", conn.Info,
		"database", batchForm.Database,
		"collection", batchForm.Collection,
		"filter", string(batchForm.Filter),
		"expectCount", batchForm.ExpectCount,
	)
	m.publishResourceOp(rc, conn)

	res, err := m.mongoData.DeleteByFilter(rc.MetaCtx, conn, &dto.BatchDelete{
		Database:    batchForm.Database,
		Collection:  batchForm.Collection,
		Filter:      decodeJSONField(rc, batchForm.Filter, "filter"),
		ExpectCount: batchForm.ExpectCount,
	})
	biz.ErrIsNil(mongoError(rc.MetaCtx, err))
	rc.ResData = res
}

// ExportDocs 导出文档。
//
// 用普通 JSON 响应而不是文件流：NoRes 流式响应下框架不会输出错误体，
// 权限不足或超时会让用户拿到一个「成功但空」的响应。条数有服务端硬上限，
// 体量可控，文件名与下载由前端负责。
func (m *Mongo) ExportDocs(rc *req.Ctx) {
	exportForm := rc.BindJson[form.ExportForm]()
	conn := m.getConn(rc)

	rc.ReqParam = collx.Kvs(
		"mongo", conn.Info,
		"database", exportForm.Database,
		"collection", exportForm.Collection,
		"format", exportForm.Format,
		"filter", string(exportForm.Filter),
	)

	file, err := m.mongoData.ExportDocuments(rc.MetaCtx, conn, &dto.ExportRequest{
		Database:   exportForm.Database,
		Collection: exportForm.Collection,
		Filter:     decodeJSONField(rc, exportForm.Filter, "filter"),
		Format:     exportForm.Format,
	})
	biz.ErrIsNil(mongoError(rc.MetaCtx, err))
	rc.ResData = &vo.ExportResult{
		FileName:    exportFileName(exportForm.Database, exportForm.Collection, exportForm.Format),
		ContentType: file.ContentType,
		Count:       file.Count,
		Content:     string(file.Content),
	}
}

// exportFileName 文件名带库.集合与时间戳：同一集合多次导出时，光看文件名就能知道是哪一批、来自哪里。
func exportFileName(database, collection, format string) string {
	ext := format
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return fmt.Sprintf("%s.%s-%s%s", database, collection, time.Now().Format("20060102-150405"), ext)
}

// decodePipeline 解析聚合管道，失败时给出可直接照抄的合法形状示例。
func decodePipeline(rc *req.Ctx, raw json.RawMessage) []bson.D {
	pipeline, err := mongodoc.DecodePipeline(raw)
	biz.ErrIsNil(mongoError(rc.MetaCtx, err))
	if len(pipeline) == 0 {
		biz.ErrIsNil(errorx.NewBizI(rc.MetaCtx, imsg.ErrMongoPipelineInvalid))
	}
	return pipeline
}
