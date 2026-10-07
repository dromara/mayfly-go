package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"mayfly-go/internal/mongo/application/dto"
	"mayfly-go/internal/mongo/application/mongodoc"
	"mayfly-go/internal/mongo/application/mongoexport"
	"mayfly-go/internal/mongo/config"
	"mayfly-go/internal/mongo/mgm"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/spf13/cast"
)

// 数据面哨兵错误。API 层据此选择国际化文案，application 层不拼装面向用户的提示。
var (
	// ErrNothingToWrite 调用方没有提交任何文档。
	ErrNothingToWrite = errors.New("mongo: no document provided")
	// ErrDocNotFound 按主键定位不到文档。
	ErrDocNotFound = errors.New("mongo document not found")
	// ErrDocConflict 文档在读取之后已被他人改动，本次写入被乐观锁拦下。
	ErrDocConflict = errors.New("mongo document was modified by others")
	// ErrStatsUnavailable 集合统计取不到且没有具体原因（空结果），按统计不可用处理。
	// 视图、无权限等场景走的是聚合报错，会把驱动原因带回来供读数区说明
	ErrStatsUnavailable = errors.New("mongo collection stats unavailable")
)

// ExecTimeoutError 执行超出时间上限，携带生效上限以便提示「超过多少秒被取消」。
//
// 与一般执行失败分开：前者要提示「收窄条件/加索引」，后者要提示「检查命令本身」。
type ExecTimeoutError struct {
	Seconds int
	Err     error
}

func (e *ExecTimeoutError) Error() string {
	return fmt.Sprintf("mongo operation timed out after %d seconds: %s", e.Seconds, e.Err.Error())
}

func (e *ExecTimeoutError) Unwrap() error { return e.Err }

// MongoData Mongo 数据面：拓扑浏览、文档读写、集合结构与命令执行。
//
// 与 Mongo（实例元数据管理）拆成两个接口：前者操作远端 Mongo，后者管理平台上登记的连接，
// 生命周期与权限语义都不同。
//
// 接口不感知 HTTP：入参是已解码的 BSON 文档，出参是已编码的保真文档；
// 失败一律返回 error，禁止在本层使用断言（见 AGENTS.md 的分层约束）。
type MongoData interface {
	// ListDatabases 列出实例下的库
	ListDatabases(ctx context.Context, conn *mgm.MongoConn) ([]*dto.Database, error)

	// ListCollections 列出库下的集合（含视图与只读标记）
	ListCollections(ctx context.Context, conn *mgm.MongoConn, database string) ([]*dto.Collection, error)

	// CreateCollection 创建集合
	CreateCollection(ctx context.Context, conn *mgm.MongoConn, database, collection string) error

	// DropCollection 删除集合（调用方须已完成名称确认与权限校验）
	DropCollection(ctx context.Context, conn *mgm.MongoConn, database, collection string) error

	// DropDatabase 删除数据库
	DropDatabase(ctx context.Context, conn *mgm.MongoConn, database string) error

	// QueryDocuments 按条件分页查询文档，集合统计信息作为附带信息一并返回
	QueryDocuments(ctx context.Context, conn *mgm.MongoConn, query *dto.DocQuery) (*dto.DocPage, error)

	// InsertDocuments 插入一个或多个文档
	InsertDocuments(ctx context.Context, conn *mgm.MongoConn, insert *dto.DocInsert) (*dto.WriteResult, error)

	// UpdateDocument 以主键令牌定位并按差异更新单个文档
	UpdateDocument(ctx context.Context, conn *mgm.MongoConn, update *dto.DocUpdate) (*dto.WriteResult, error)

	// DeleteDocuments 按主键令牌批量删除
	DeleteDocuments(ctx context.Context, conn *mgm.MongoConn, del *dto.DocDelete) (*dto.WriteResult, error)

	// RunCommand 执行管理命令并返回保真 JSON 结果
	RunCommand(ctx context.Context, conn *mgm.MongoConn, database string, command bson.D) (json.RawMessage, error)

	// Aggregate 执行聚合管道（含写出 stage 时由 API 层按写级别鉴权）
	Aggregate(ctx context.Context, conn *mgm.MongoConn, query *dto.AggQuery) (*dto.DocPage, error)

	// CollectionMeta 集合元信息（统计与索引列表），供元信息面板一次拉齐
	CollectionMeta(ctx context.Context, conn *mgm.MongoConn, database, collection string) (*dto.CollectionMeta, error)

	// CreateIndexes 按索引定义数组创建索引
	CreateIndexes(ctx context.Context, conn *mgm.MongoConn, database, collection string, specs json.RawMessage) error

	// DropIndex 按名删除索引
	DropIndex(ctx context.Context, conn *mgm.MongoConn, database, collection, index string) error

	// UpdateByFilter 批量更新，命中数与预期不符则中止
	UpdateByFilter(ctx context.Context, conn *mgm.MongoConn, batch *dto.BatchUpdate) (*dto.WriteResult, error)

	// DeleteByFilter 批量删除，命中数与预期不符则中止
	DeleteByFilter(ctx context.Context, conn *mgm.MongoConn, batch *dto.BatchDelete) (*dto.WriteResult, error)

	// ExportDocuments 按条件取数并生成导出内容
	ExportDocuments(ctx context.Context, conn *mgm.MongoConn, export *dto.ExportRequest) (*dto.ExportFile, error)
}

type mongoDataAppImpl struct{}

var _ MongoData = (*mongoDataAppImpl)(nil)

func (m *mongoDataAppImpl) ListDatabases(ctx context.Context, conn *mgm.MongoConn) ([]*dto.Database, error) {
	ctx, cancel, tl := execContext(ctx)
	defer cancel()

	res, err := conn.Cli.ListDatabases(ctx, bson.D{})
	if err != nil {
		return nil, wrapExecError(err, tl)
	}

	databases := make([]*dto.Database, 0, len(res.Databases))
	for _, spec := range res.Databases {
		databases = append(databases, &dto.Database{
			Name:       spec.Name,
			SizeOnDisk: spec.SizeOnDisk,
			Empty:      spec.Empty,
		})
	}
	return databases, nil
}

func (m *mongoDataAppImpl) ListCollections(ctx context.Context, conn *mgm.MongoConn, database string) ([]*dto.Collection, error) {
	ctx, cancel, tl := execContext(ctx)
	defer cancel()

	specs, err := conn.Cli.Database(database).ListCollectionSpecifications(ctx, bson.D{})
	if err != nil {
		return nil, wrapExecError(err, tl)
	}

	collections := make([]*dto.Collection, 0, len(specs))
	for _, spec := range specs {
		collections = append(collections, &dto.Collection{
			Name:     spec.Name,
			Type:     spec.Type,
			ReadOnly: spec.ReadOnly,
		})
	}
	return collections, nil
}

func (m *mongoDataAppImpl) CreateCollection(ctx context.Context, conn *mgm.MongoConn, database, collection string) error {
	ctx, cancel, tl := execContext(ctx)
	defer cancel()

	if err := conn.Cli.Database(database).CreateCollection(ctx, collection); err != nil {
		return wrapExecError(err, tl)
	}
	return nil
}

func (m *mongoDataAppImpl) DropCollection(ctx context.Context, conn *mgm.MongoConn, database, collection string) error {
	ctx, cancel, tl := execContext(ctx)
	defer cancel()

	if err := conn.Cli.Database(database).Collection(collection).Drop(ctx); err != nil {
		return wrapExecError(err, tl)
	}
	return nil
}

func (m *mongoDataAppImpl) DropDatabase(ctx context.Context, conn *mgm.MongoConn, database string) error {
	ctx, cancel, tl := execContext(ctx)
	defer cancel()

	if err := conn.Cli.Database(database).Drop(ctx); err != nil {
		return wrapExecError(err, tl)
	}
	return nil
}

// collectionStats 取集合头部读数，作为查询的附带能力而不是独立入口。
//
// 走 $collStats 聚合而非 collStats 命令：后者在 MongoDB 7 起已废弃，且在分片集群上需要额外权限。
// scaleBytes=1 固定按字节返回，避免不同服务端版本的默认缩放差异让前端读数抖动一个量级。
//
// 共用调用方的 ctx：头部读数不应比查询本身享有更长的超时预算。
func (m *mongoDataAppImpl) collectionStats(ctx context.Context, conn *mgm.MongoConn, database, collection string) (*dto.CollectionStats, error) {
	// 管道必须是数组：driver 拒接 bson.D（报 "bson.D is not an allowed pipeline type"），
	// 单个聚合 stage 也不例外
	pipeline := bson.A{
		bson.D{{Key: "$collStats", Value: bson.D{
			{Key: "storageStats", Value: bson.D{{Key: "scaleBytes", Value: int32(1)}}},
		}}},
	}

	cur, err := conn.Cli.Database(database).Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var stats []bson.D
	if err = cur.All(ctx, &stats); err != nil {
		return nil, err
	}
	if len(stats) == 0 {
		return nil, ErrStatsUnavailable
	}

	raw := stats[0]
	nested, _ := mongodoc.Lookup(raw, "storageStats")
	nestedDoc, _ := nested.(bson.D)

	// 大小与索引类字段在不同服务版本里分别位于顶层与 storageStats 子文档（实测 7.0 全部在嵌套里），
	// 两处都找一次：版本差异吸收在服务端，前端只面对一份稳定契约
	lookup := func(key string) (any, bool) {
		if value, ok := mongodoc.Lookup(raw, key); ok && value != nil {
			return value, true
		}
		if nestedDoc != nil {
			if value, ok := mongodoc.Lookup(nestedDoc, key); ok && value != nil {
				return value, true
			}
		}
		return nil, false
	}

	pick := func(key string) int64 {
		value, ok := lookup(key)
		if !ok {
			return 0
		}
		return cast.ToInt64(value)
	}

	// 索引数：新版 $collStats 不再直接给 nindexes，改从 indexSizes 的键个数推出
	pickIndexCount := func() int64 {
		if value, ok := lookup("nindexes"); ok {
			return cast.ToInt64(value)
		}
		if sizes, ok := lookup("indexSizes"); ok {
			if doc, ok := sizes.(bson.D); ok {
				return int64(len(doc))
			}
		}
		return 0
	}

	// indexSizes 是个文档（索引名 → 字节数），位置与标量字段同样有两种
	pickIndexSizes := func() map[string]int64 {
		sizes := make(map[string]int64)
		value, ok := lookup("indexSizes")
		if !ok {
			return sizes
		}
		doc, ok := value.(bson.D)
		if !ok {
			return sizes
		}
		for _, e := range doc {
			sizes[e.Key] = cast.ToInt64(e.Value)
		}
		return sizes
	}

	return &dto.CollectionStats{
		Ns:              pickString(raw, "ns"),
		Count:           pick("count"),
		AvgObjSize:      pick("avgObjSize"),
		StorageSize:     pick("storageSize"),
		FreeStorageSize: pick("freeStorageSize"),
		TotalIndexSize:  pick("totalIndexSize"),
		TotalSize:       pick("totalSize"),
		NIndexes:        pickIndexCount(),
		IndexSizes:      pickIndexSizes(),
	}, nil
}

func pickString(doc bson.D, key string) string {
	value, ok := mongodoc.Lookup(doc, key)
	if !ok {
		return ""
	}
	return cast.ToString(value)
}

func (m *mongoDataAppImpl) QueryDocuments(ctx context.Context, conn *mgm.MongoConn, query *dto.DocQuery) (*dto.DocPage, error) {
	cfg := config.GetMongo()
	limit := cfg.Limit(query.Limit)

	ctx, cancel, tl := execContext(ctx)
	defer cancel()

	coll := conn.Cli.Database(query.Database).Collection(query.Collection)

	// 「没写条件」在调用侧就是 nil，而 driver 不能把 nil 文档编成 BSON，进驱动前统一归一（见 mongodoc.Doc）
	filter := mongodoc.Doc(query.Filter)

	// 多取一条用于判定「是否还有后续结果」，比再发一次 count 便宜得多，
	// 也让前端能明确区分「取完了」与「被上限截断了」
	opts := options.Find().SetSkip(query.Skip).SetLimit(limit + 1)
	if sort := mongodoc.Doc(query.Sort); len(sort) > 0 {
		opts.SetSort(sort)
	}
	if projection := mongodoc.Doc(query.Projection); len(projection) > 0 {
		opts.SetProjection(projection)
	}

	cur, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, wrapExecError(err, tl)
	}

	// 解码到 []bson.D 而不是 []bson.M：前者保留字段顺序，后者的 map 迭代序随机会让展示与 diff 不稳定
	var rawDocs []bson.D
	if err = cur.All(ctx, &rawDocs); err != nil {
		return nil, wrapExecError(err, tl)
	}

	truncated := int64(len(rawDocs)) > limit
	if truncated {
		rawDocs = rawDocs[:limit]
	}

	docs, err := mongodoc.EncodeQueryDocs(rawDocs)
	if err != nil {
		return nil, err
	}

	page := &dto.DocPage{Docs: docs, Total: -1, Truncated: truncated, Limit: limit}
	if query.WithCount {
		// 统计的是「匹配总数」，不含 skip/limit；未开启统计时为 -1，
		// 前端不得把已加载条数或集合总条数当匹配数展示
		total, err := coll.CountDocuments(ctx, filter)
		if err != nil {
			return nil, wrapExecError(err, tl)
		}
		page.Total = total
	}

	// 集合统计随查询一并取回：视图、无 collStats 权限等场景会失败，但那些失败不应让
	// 已经取到数据的整页看起来出错，只把原因交给读数区说明
	stats, statsErr := m.collectionStats(ctx, conn, query.Database, query.Collection)
	if statsErr != nil {
		page.StatsError = statsErr.Error()
	} else {
		page.Stats = stats
	}
	return page, nil
}

func (m *mongoDataAppImpl) InsertDocuments(ctx context.Context, conn *mgm.MongoConn, insert *dto.DocInsert) (*dto.WriteResult, error) {
	if len(insert.Docs) == 0 {
		return nil, ErrNothingToWrite
	}

	ctx, cancel, tl := execContext(ctx)
	defer cancel()

	coll := conn.Cli.Database(insert.Database).Collection(insert.Collection)

	// 插入保留调用方给定的 _id（显式指定主键是合法操作），与更新不同——更新必须不能改主键
	if len(insert.Docs) == 1 {
		res, err := coll.InsertOne(ctx, insert.Docs[0])
		if err != nil {
			return nil, wrapExecError(err, tl)
		}
		token, err := mongodoc.SignIDValue(res.InsertedID)
		if err != nil {
			return nil, err
		}
		return &dto.WriteResult{InsertedCount: 1, InsertedIDs: []string{token}}, nil
	}

	docs := make([]any, 0, len(insert.Docs))
	for _, doc := range insert.Docs {
		docs = append(docs, doc)
	}

	res, err := coll.InsertMany(ctx, docs)
	if err != nil {
		// 有序插入可能在中间失败，已插入条数仍如实返回，便于前端提示「部分成功」
		written := 0
		if res != nil {
			written = len(res.InsertedIDs)
		}
		return &dto.WriteResult{InsertedCount: int64(written)}, wrapExecError(err, tl)
	}

	tokens := make([]string, 0, len(res.InsertedIDs))
	for _, id := range res.InsertedIDs {
		token, err := mongodoc.SignIDValue(id)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return &dto.WriteResult{InsertedCount: int64(len(tokens)), InsertedIDs: tokens}, nil
}

// UpdateDocument 读取当前存储文档 → 校验内容指纹 → 按差异写入。
//
// 差异在服务端计算而不是由调用方提交 $set/$unset：调用方只表达「期望的最终文档」，
// 于是「删掉一个 key」自然等价于删除字段，而操作符注入面也一并关闭（见 mongodoc.BuildUpdateDiff）。
func (m *mongoDataAppImpl) UpdateDocument(ctx context.Context, conn *mgm.MongoConn, update *dto.DocUpdate) (*dto.WriteResult, error) {
	idValue, err := mongodoc.ParseID(update.IDToken)
	if err != nil {
		return nil, err
	}

	ctx, cancel, tl := execContext(ctx)
	defer cancel()

	coll := conn.Cli.Database(update.Database).Collection(update.Collection)

	var current bson.D
	if err = coll.FindOne(ctx, mongodoc.IDFilter(idValue)).Decode(&current); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrDocNotFound
		}
		return nil, wrapExecError(err, tl)
	}

	if update.BaseHash != "" {
		currentHash, err := mongodoc.DocHash(current)
		if err != nil {
			return nil, err
		}
		if currentHash != update.BaseHash {
			return nil, ErrDocConflict
		}
	}

	diff, changed, err := mongodoc.BuildUpdateDiff(current, update.Doc)
	if err != nil {
		return nil, err
	}
	if !changed {
		return &dto.WriteResult{MatchedCount: 1, NoChange: true}, nil
	}

	res, err := coll.UpdateByID(ctx, idValue, diff)
	if err != nil {
		return nil, wrapExecError(err, tl)
	}
	return &dto.WriteResult{MatchedCount: res.MatchedCount, ModifiedCount: res.ModifiedCount}, nil
}

func (m *mongoDataAppImpl) DeleteDocuments(ctx context.Context, conn *mgm.MongoConn, del *dto.DocDelete) (*dto.WriteResult, error) {
	// 令牌逐个解析后再合并成一次 $in 删除：既不按形状猜主键类型，也不产生 N 次往返
	values := make([]any, 0, len(del.IDTokens))
	for _, token := range del.IDTokens {
		value, err := mongodoc.ParseID(token)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}

	ctx, cancel, tl := execContext(ctx)
	defer cancel()

	cond := bson.D{{Key: mongodoc.FieldID, Value: bson.D{{Key: "$in", Value: values}}}}
	res, err := conn.Cli.Database(del.Database).Collection(del.Collection).DeleteMany(ctx, cond)
	if err != nil {
		return nil, wrapExecError(err, tl)
	}

	// DeletedCount 可能小于请求条数（文档已被他人删除），如实回传由调用方决定提示口径
	return &dto.WriteResult{DeletedCount: res.DeletedCount}, nil
}

func (m *mongoDataAppImpl) RunCommand(ctx context.Context, conn *mgm.MongoConn, database string, command bson.D) (json.RawMessage, error) {
	ctx, cancel, tl := execContext(ctx)
	defer cancel()

	var result bson.D
	if err := conn.Cli.Database(database).RunCommand(ctx, command).Decode(&result); err != nil {
		return nil, wrapExecError(err, tl)
	}

	encoded, err := mongodoc.Encode(result)
	if err != nil {
		return nil, err
	}
	return encoded.Doc, nil
}

// ErrEmptyPipeline 聚合管道为空：Mongo 会报「pipeline 不能为空」但文案含糊，在这里拦下
var ErrEmptyPipeline = errors.New("mongo: aggregation pipeline is empty")

// BatchCountMismatchError 批量操作的命中数与调用方确认的不一致。
//
// 典型成因是「预览后又改了条件」或「并发修改」：此时不能照旧执行，
// 否则用户是对 N 条做的决策却作用到了 M 条上。
type BatchCountMismatchError struct {
	Expect int64
	Actual int64
}

func (e *BatchCountMismatchError) Error() string {
	return fmt.Sprintf("mongo: matched %d documents but expected %d", e.Actual, e.Expect)
}

// Aggregate 执行聚合管道。
//
// 结果同样走保真编码：聚合输出的 `id`、`total` 常为 int64，按普通 JSON 返回会在前端丢类型。
// explain 形态不执行管道（包括写出 stage），因此只走只读路径。
func (m *mongoDataAppImpl) Aggregate(ctx context.Context, conn *mgm.MongoConn, query *dto.AggQuery) (*dto.DocPage, error) {
	if len(query.Pipeline) == 0 {
		return nil, ErrEmptyPipeline
	}

	cfg := config.GetMongo()
	limit := cfg.Limit(0)

	ctx, cancel, tl := execContext(ctx)
	defer cancel()

	if query.Explain {
		command := bson.D{
			{Key: "explain", Value: aggregateCommand(query, limit)},
			{Key: "verbosity", Value: "executionStats"},
		}
		var result bson.D
		if err := conn.Cli.Database(query.Database).RunCommand(ctx, command).Decode(&result); err != nil {
			return nil, wrapExecError(err, tl)
		}
		docs, err := mongodoc.EncodeQueryDocs([]bson.D{result})
		if err != nil {
			return nil, err
		}
		return &dto.DocPage{Docs: docs, Total: -1, Limit: limit}, nil
	}

	opts := options.Aggregate().SetBatchSize(int32(limit))
	if query.AllowDiskUse {
		opts.SetAllowDiskUse(true)
	}

	cur, err := conn.Cli.Database(query.Database).Collection(query.Collection).Aggregate(ctx, query.Pipeline, opts)
	if err != nil {
		return nil, wrapExecError(err, tl)
	}

	// 多取一条以判断是否被上限截断，与分页查询一致
	var rawDocs []bson.D
	if err = cur.All(ctx, &rawDocs); err != nil {
		return nil, wrapExecError(err, tl)
	}

	truncated := int64(len(rawDocs)) > limit
	if truncated {
		rawDocs = rawDocs[:limit]
	}
	docs, err := mongodoc.EncodeQueryDocs(rawDocs)
	if err != nil {
		return nil, err
	}
	return &dto.DocPage{Docs: docs, Total: -1, Truncated: truncated, Limit: limit}, nil
}

// aggregateCommand 组装聚合命令文档（explain 与直接执行共用同一形状）
func aggregateCommand(query *dto.AggQuery, limit int64) bson.D {
	command := bson.D{
		{Key: "aggregate", Value: query.Collection},
		{Key: "pipeline", Value: query.Pipeline},
		{Key: "cursor", Value: bson.D{{Key: "batchSize", Value: limit}}},
	}
	if query.AllowDiskUse {
		command = append(command, bson.E{Key: "allowDiskUse", Value: true})
	}
	return command
}

// CollectionMeta 集合元信息：统计与索引列表一次取回。
//
// 合并为一个入口而不是两个接口：面板打开时两者同进同退，分开拉会出现一半新一半旧的画面。
// 统计不可用（视图、无 collStats 权限）不令整个元信息失败，只把原因放在 StatsError。
func (m *mongoDataAppImpl) CollectionMeta(ctx context.Context, conn *mgm.MongoConn, database, collection string) (*dto.CollectionMeta, error) {
	ctx, cancel, tl := execContext(ctx)
	defer cancel()

	indexes, err := m.listIndexes(ctx, conn, database, collection)
	if err != nil {
		return nil, wrapExecError(err, tl)
	}

	meta := &dto.CollectionMeta{Indexes: indexes, TotalDocs: -1}
	stats, statsErr := m.collectionStats(ctx, conn, database, collection)
	if statsErr != nil {
		meta.StatsError = statsErr.Error()
	} else {
		meta.Stats = stats
		if stats != nil {
			meta.TotalDocs = stats.Count
		}
	}
	return meta, nil
}

// listIndexes 列索引。
//
// 索引大小来自 $collStats 的 indexSizes；取不到时只为 0，不影响索引列表本身。
func (m *mongoDataAppImpl) listIndexes(ctx context.Context, conn *mgm.MongoConn, database, collection string) ([]*dto.IndexInfo, error) {
	ctx, cancel, tl := execContext(ctx)
	defer cancel()

	cur, err := conn.Cli.Database(database).Collection(collection).Indexes().List(ctx)
	if err != nil {
		return nil, wrapExecError(err, tl)
	}

	var specs []bson.D
	if err = cur.All(ctx, &specs); err != nil {
		return nil, wrapExecError(err, tl)
	}

	indexSizes := m.indexSizes(ctx, conn, database, collection)

	infos := make([]*dto.IndexInfo, 0, len(specs))
	for _, spec := range specs {
		info := &dto.IndexInfo{Name: indexName(spec), Keys: indexKeys(spec)}
		encoded, err := mongodoc.Encode(spec)
		if err != nil {
			return nil, err
		}
		info.Spec = encoded.Doc
		if unique, ok := mongodoc.Lookup(spec, "unique"); ok {
			info.Unique = unique == true
		}
		info.SizeBytes = indexSizes[info.Name]
		infos = append(infos, info)
	}
	return infos, nil
}

// indexSizes 索引名 → 字节数。统计不可用时返回空表而不影响主流程
func (m *mongoDataAppImpl) indexSizes(ctx context.Context, conn *mgm.MongoConn, database, collection string) map[string]int64 {
	stats, err := m.collectionStats(ctx, conn, database, collection)
	if err != nil || stats == nil || stats.IndexSizes == nil {
		return map[string]int64{}
	}
	return stats.IndexSizes
}

func indexName(spec bson.D) string {
	if value, ok := mongodoc.Lookup(spec, "name"); ok {
		return cast.ToString(value)
	}
	return ""
}

// indexKeys 把索引键文档转成有序数组，方向按可读形式输出（含 2dsphere/text 这类命名方向）
func indexKeys(spec bson.D) []dto.IndexKey {
	keys, ok := mongodoc.Lookup(spec, "key")
	if !ok {
		return nil
	}
	doc, ok := keys.(bson.D)
	if !ok {
		return nil
	}

	res := make([]dto.IndexKey, 0, len(doc))
	for _, e := range doc {
		res = append(res, dto.IndexKey{Field: e.Key, Direction: indexDirection(e.Value)})
	}
	return res
}

func indexDirection(value any) string {
	switch v := value.(type) {
	case int32:
		return directionOf(int64(v))
	case int64:
		return directionOf(v)
	case float64:
		return directionOf(int64(v))
	case string:
		return v
	default:
		// 地理对象的复合键、通配索引等会带子文档/特殊形状，原样输出而不假装是升/降序
		return fmt.Sprintf("%v", value)
	}
}

func directionOf(v int64) string {
	if v < 0 {
		return "desc"
	}
	return "asc"
}

// CreateIndexes 直接以 createIndexes 命令提交原始定义：
// 索引选项很多（ttl/partial/sparse/collation/wildcard…），手写映射必然漏且会默默丢选项，
// 把用户提交的结构原样交给我服务端最可靠。
func (m *mongoDataAppImpl) CreateIndexes(ctx context.Context, conn *mgm.MongoConn, database, collection string, specs json.RawMessage) error {
	decoded, err := mongodoc.DecodeValue(specs)
	if err != nil {
		return err
	}
	list, ok := decoded.(bson.A)
	if !ok || len(list) == 0 {
		return ErrEmptyIndexSpecs
	}
	for _, item := range list {
		spec, ok := item.(bson.D)
		if !ok {
			return ErrEmptyIndexSpecs
		}
		if _, hasKey := mongodoc.Lookup(spec, "key"); !hasKey {
			return ErrEmptyIndexSpecs
		}
	}

	_, err = m.RunCommand(ctx, conn, database, bson.D{
		{Key: "createIndexes", Value: collection},
		{Key: "indexes", Value: list},
	})
	return err
}

func (m *mongoDataAppImpl) DropIndex(ctx context.Context, conn *mgm.MongoConn, database, collection, index string) error {
	_, err := m.RunCommand(ctx, conn, database, bson.D{
		{Key: "dropIndexes", Value: collection},
		{Key: "index", Value: index},
	})
	return err
}

func (m *mongoDataAppImpl) countMatched(ctx context.Context, conn *mgm.MongoConn, database, collection string, filter bson.D) (int64, error) {
	ctx, cancel, tl := execContext(ctx)
	defer cancel()

	count, err := conn.Cli.Database(database).Collection(collection).CountDocuments(ctx, mongodoc.Doc(filter))
	if err != nil {
		return 0, wrapExecError(err, tl)
	}
	return count, nil
}

func (m *mongoDataAppImpl) UpdateByFilter(ctx context.Context, conn *mgm.MongoConn, batch *dto.BatchUpdate) (*dto.WriteResult, error) {
	matched, err := m.guardBatch(ctx, conn, batch.Database, batch.Collection, batch.Filter, batch.ExpectCount)
	if err != nil || matched == 0 {
		return &dto.WriteResult{MatchedCount: matched}, err
	}

	ctx, cancel, tl := execContext(ctx)
	defer cancel()

	opts := options.UpdateMany().SetUpsert(batch.Upsert)
	res, err := conn.Cli.Database(batch.Database).Collection(batch.Collection).
		UpdateMany(ctx, mongodoc.Doc(batch.Filter), batch.Update.Value(), opts)
	if err != nil {
		return nil, wrapExecError(err, tl)
	}
	return &dto.WriteResult{
		MatchedCount:  res.MatchedCount,
		ModifiedCount: res.ModifiedCount,
		UpsertedCount: res.UpsertedCount,
	}, nil
}

func (m *mongoDataAppImpl) DeleteByFilter(ctx context.Context, conn *mgm.MongoConn, batch *dto.BatchDelete) (*dto.WriteResult, error) {
	matched, err := m.guardBatch(ctx, conn, batch.Database, batch.Collection, batch.Filter, batch.ExpectCount)
	if err != nil || matched == 0 {
		return &dto.WriteResult{DeletedCount: 0}, err
	}

	ctx, cancel, tl := execContext(ctx)
	defer cancel()

	res, err := conn.Cli.Database(batch.Database).Collection(batch.Collection).
		DeleteMany(ctx, mongodoc.Doc(batch.Filter))
	if err != nil {
		return nil, wrapExecError(err, tl)
	}
	return &dto.WriteResult{DeletedCount: res.DeletedCount}, nil
}

// guardBatch 先统计命中数并与调用方确认的值比对，不一致则中止。
//
// 这不只能防错条件，也能防「预览后改了条件/别人改了数据」；它是不可逆批量写前的唯一屏障。
func (m *mongoDataAppImpl) guardBatch(ctx context.Context, conn *mgm.MongoConn, database, collection string, filter bson.D, expect int64) (int64, error) {
	matched, err := m.countMatched(ctx, conn, database, collection, filter)
	if err != nil {
		return 0, err
	}
	if matched != expect {
		return matched, &BatchCountMismatchError{Expect: expect, Actual: matched}
	}
	return matched, nil
}

// ExportDocuments 按条件取数并生成文件内容。
//
// 条数硬上限一律用配置值而不是接受调用方传的 limit：导出是把数据带走的路径，
// 让它能被请求参数无限拉大等于开了一道无限的口。
func (m *mongoDataAppImpl) ExportDocuments(ctx context.Context, conn *mgm.MongoConn, export *dto.ExportRequest) (*dto.ExportFile, error) {
	if !mongoexport.Supported(export.Format) {
		return nil, fmt.Errorf("mongoexport: unsupported format %q", export.Format)
	}

	limit := config.GetMongo().Limit(0)
	page, err := m.QueryDocuments(ctx, conn, &dto.DocQuery{
		Database:   export.Database,
		Collection: export.Collection,
		Filter:     export.Filter,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}

	content, contentType, err := mongoexport.Export(export.Format, page.Docs)
	if err != nil {
		return nil, err
	}
	return &dto.ExportFile{Content: content, ContentType: contentType, Count: len(page.Docs)}, nil
}

// ErrEmptyIndexSpecs 索引定义不合法：必须是非空数组且每项含 key。
var ErrEmptyIndexSpecs = errors.New("mongo: index specifications must be a non-empty array of documents with a key field")

// execContext 为单次数据面操作派生带执行时间上限的上下文。
//
// 父上下文就是请求上下文，所以客户端断开同样会取消服务端操作；这里补的是另一类保护：
// 客户端没断、但操作本身很慢（全集合扫描、$where、缺索引的大排序）。旧实现用 context.TODO()，
// 这类操作会永久占住 goroutine 且无法取消。
func execContext(ctx context.Context) (context.Context, context.CancelFunc, int) {
	tl := config.GetMongo().ExecTl
	if tl <= 0 {
		return ctx, func() {}, tl
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(tl)*time.Second)
	return timeoutCtx, cancel, tl
}

// wrapExecError 把超时取消从一般执行失败里区分出来（携带生效上限），其余错误原样上抛由调用方展示细节。
func wrapExecError(err error, seconds int) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &ExecTimeoutError{Seconds: seconds, Err: err}
	}
	return err
}
