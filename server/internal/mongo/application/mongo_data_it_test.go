package application

// Mongo 数据面集成测试：对本机 Docker 起的真实 MongoDB 跑完整链路
// （连接 → 保真读写 → 主键令牌 → 差异更新 → 乐观锁 → 排序/分页 → 索引键序 → 统计降级 → 结构销毁）。
//
// 环境准备（任一种）：
//   docker run -d --name mayfly-mongo-it -p 27017:27017 mongo:7.0
//   或带鉴权：docker run -d --name mayfly-mongo-it -p 27017:27017 \
//       -e MONGO_INITDB_ROOT_USERNAME=mayfly -e MONGO_INITDB_ROOT_PASSWORD='Mayfly#123' mongo:7.0
//       并用 MAYFLY_MONGO_URI 覆盖连接串
//
// 运行：cd server && go test -count=1 -run TestITMongo ./internal/mongo/application/
//
// 连不上时整组跳过（普通 go test ./... 不应因为本机没起 mongo 而变红）。
// 测试自建库 mayfly_mongo_it，用例开始前清、结束后删，不给环境留残留数据。

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"mayfly-go/internal/mongo/application/dto"
	"mayfly-go/internal/mongo/application/mongodoc"
	"mayfly-go/internal/mongo/application/mongoexport"
	"mayfly-go/internal/mongo/mgm"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/ioc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	itDatabase = "mayfly_mongo_it"
	// 默认按「无鉴权mongo跑在27017」的最省事形态；带鉴权的环境用 MAYFLY_MONGO_URI 覆盖
	itMongoUriDefault = "mongodb://127.0.0.1:27017"
)

// itConfigFake 只提供 GetConfig：数据面的上限/超时都来自系统配置，测试里给出确定值，
// 避免用例断言随部署环境默认值漂移（其余接口方法不会被调用）
type itConfigFake struct {
	sysapp.Config
}

func (itConfigFake) GetConfig(string) *sysentity.Config {
	return &sysentity.Config{
		Id:    1,
		Value: `{"execTl":"30","defaultLimit":"20","maxResultSet":"500","poolSize":"4"}`,
	}
}

// itEnv 每个用例的环境：注册配置假实现、拿到连接、清库并在用例结束时删库
func itEnv(t *testing.T) (*mongoDataAppImpl, *mgm.MongoConn) {
	t.Helper()

	ioc.Register(itConfigFake{})

	uri := os.Getenv("MAYFLY_MONGO_URI")
	if uri == "" {
		uri = itMongoUriDefault
	}

	conn, err := (&mgm.MongoInfo{Id: 990001, Name: "it-mongo", Uri: uri}).Conn()
	if err != nil {
		t.Skipf("跳过 Mongo 集成测试（本机 %s 不可连接）: %v", uri, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 不用 ping 探可达性：未鉴权的客户端也能 ping 成功，失败会跑到后面的 dropDatabase 上，
	// 报成一个看不出与 URI 有关的权限错误。ListDatabases 需要凭证，能把它换成明确的跳过提示
	if _, err = conn.Cli.ListDatabases(ctx, bson.D{}); err != nil {
		conn.Close()
		t.Skipf("跳过 Mongo 集成测试（%s 不可用或凭证不匹配，可用 MAYFLY_MONGO_URI 覆盖）: %v", uri, err)
	}

	// 前置清一次，避免上次异常中断留下的残留影响断言
	require.NoError(t, conn.Cli.Database(itDatabase).Drop(ctx))
	t.Cleanup(func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dropCancel()
		_ = conn.Cli.Database(itDatabase).Drop(dropCtx)
		_ = conn.Close()
	})

	return &mongoDataAppImpl{}, conn
}

func itCtx() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	_ = cancel // 用例级上下文统一由 30s 上限约束，收尾由 t.Cleanup 负责
	return ctx
}

// itRawDoc 用驱动直连读取原始 BSON，验证库里真实存储的类型。
//
// 必须独立于 mongodoc：只比较「我的编码器解出来对不对」会自洽地错过「写进库的其实已经是错的」。
func itRawDoc(t *testing.T, conn *mgm.MongoConn, collection string, filter bson.D) bson.D {
	t.Helper()

	var doc bson.D
	err := conn.Cli.Database(itDatabase).Collection(collection).FindOne(itCtx(), filter).Decode(&doc)
	require.NoError(t, err, "read raw document %s", collection)
	return doc
}

// TestITMongoTypesRoundTripKeepBSONTypes 本次重构的核心承诺：
// 「读出来 → 原样写回」不改变任何字段的 BSON 类型，且库里真实存的还是原类型。
func TestITMongoTypesRoundTripKeepBSONTypes(t *testing.T) {
	app, conn := itEnv(t)
	ctx := itCtx()

	id := bson.NewObjectID()
	createdAt := bson.NewDateTimeFromTime(time.Unix(1700000000, 0).UTC())
	nestedID := bson.NewObjectID()

	original := bson.D{
		{Key: "_id", Value: id},
		{Key: "name", Value: "mayfly"},
		{Key: "flag", Value: true},
		{Key: "i32", Value: int32(-7)},
		{Key: "big", Value: int64(9007199254740993)}, // 超出 JS 安全整数
		{Key: "ratio", Value: 1.25},
		{Key: "nan", Value: math.NaN()},
		{Key: "createdAt", Value: createdAt},
		{Key: "amount", Value: itDecimal(t, "129.005")},
		{Key: "uuid", Value: bson.Binary{Subtype: 4, Data: []byte("0011223344556677")}},
		{Key: "opTs", Value: bson.Timestamp{T: 1700000000, I: 3}},
		{Key: "pattern", Value: bson.Regex{Pattern: "^a[0-9]+$", Options: "i"}},
		{Key: "userId", Value: nestedID}, // 非主键位置的 ObjectID
		{Key: "items", Value: bson.A{"x", int32(2), bson.D{{Key: "price", Value: itDecimal(t, "0.5")}}}},
		{Key: "meta", Value: bson.D{{Key: "at", Value: createdAt}, {Key: "ref", Value: nestedID}}},
	}

	_, err := app.InsertDocuments(ctx, conn, &dto.DocInsert{
		Database: itDatabase, Collection: "it_fidelity", Docs: []bson.D{original},
	})
	require.NoError(t, err)

	// 1) 库里真实存的就是这些类型
	stored := itRawDoc(t, conn, "it_fidelity", bson.D{{Key: "_id", Value: id}})
	require.Equal(t, int64(9007199254740993), stored[4].Value, "int64 must stay int64 in storage")
	assert.IsType(t, bson.DateTime(0), stored[7].Value)
	assert.IsType(t, bson.Decimal128{}, stored[8].Value)
	assert.IsType(t, bson.Binary{}, stored[9].Value)
	assert.IsType(t, bson.Timestamp{}, stored[10].Value)
	assert.IsType(t, bson.Regex{}, stored[11].Value)
	assert.IsType(t, bson.ObjectID{}, stored[12].Value)

	// 2) 查询返回的是保真形态：类型以 $ 包装显式表达
	page, err := app.QueryDocuments(ctx, conn, &dto.DocQuery{
		Database: itDatabase, Collection: "it_fidelity", Limit: 10,
	})
	require.NoError(t, err)
	require.Len(t, page.Docs, 1)
	assert.Equal(t, mongodoc.ModeExtJSON, page.Docs[0].Mode)

	text := string(page.Docs[0].Doc)
	for _, wrapper := range []string{"$oid", "$date", "$numberDecimal", "$binary", "$timestamp", "$regularExpression", "$numberLong"} {
		assert.Contains(t, text, wrapper, "response must carry type wrapper %s", wrapper)
	}

	// 3) 把返回内容原样写回，再读一次：库里类型不得发生变化（旧实现在这一步把 Date 变成 String）
	decoded, err := mongodoc.Decode(page.Docs[0].Doc)
	require.NoError(t, err)
	_, err = app.UpdateDocument(ctx, conn, &dto.DocUpdate{
		Database: itDatabase, Collection: "it_fidelity",
		IDToken: page.Docs[0].IDToken, BaseHash: page.Docs[0].Hash, Doc: decoded,
	})
	require.NoError(t, err)

	after := itRawDoc(t, conn, "it_fidelity", bson.D{{Key: "_id", Value: id}})
	afterBytes, err := bson.Marshal(after)
	require.NoError(t, err)
	beforeBytes, err := bson.Marshal(stored)
	require.NoError(t, err)
	assert.Equal(t, string(beforeBytes), string(afterBytes), "read -> write-back must be a no-op on stored BSON")
}

// TestITMongoDateOnlyDocumentKeepsDateType 只含 Date 的普通文档（现实里最常见的形态）必须单独覆盖。
//
// 为什么不能只靠上面那个多类型文档：它含 NaN，而 NaN 会独立把整份文档判成 Extended JSON，
// 就算类型判定对其他 BSON 类型失效它也绿。这个用例里没有别的特殊类型，Date 一丢就会立刻变红。
func TestITMongoDateOnlyDocumentKeepsDateType(t *testing.T) {
	app, conn := itEnv(t)
	ctx := itCtx()

	id := bson.NewObjectID()
	at := bson.NewDateTimeFromTime(time.Unix(1760000000, 0).UTC())
	_, err := app.InsertDocuments(ctx, conn, &dto.DocInsert{
		Database: itDatabase, Collection: "it_dateonly",
		Docs: []bson.D{{{Key: "_id", Value: id}, {Key: "name", Value: "n"}, {Key: "createdAt", Value: at}}},
	})
	require.NoError(t, err)

	page, err := app.QueryDocuments(ctx, conn, &dto.DocQuery{Database: itDatabase, Collection: "it_dateonly", Limit: 10})
	require.NoError(t, err)
	require.Len(t, page.Docs, 1)

	// 只因为有 Date 就必须走保真形态；同时不能把不需要的字段也包装起来
	// （canonical 下 Date 本身就是 {$date:{$numberLong:毫秒}}，所以这里只排除真正无关的包装）
	assert.Equal(t, mongodoc.ModeExtJSON, page.Docs[0].Mode)
	text := string(page.Docs[0].Doc)
	assert.Contains(t, text, "$date")
	assert.Contains(t, text, `"name":"n"`, "普通字符串字段必须保持普通 JSON 形态")
	assert.NotContains(t, text, "$numberDecimal")
	assert.NotContains(t, text, "$binary")

	// 原样写回后库里存的仍是 Date（旧实现在这一步把它写成了字符串）
	decoded, err := mongodoc.Decode(page.Docs[0].Doc)
	require.NoError(t, err)
	_, err = app.UpdateDocument(ctx, conn, &dto.DocUpdate{
		Database: itDatabase, Collection: "it_dateonly", IDToken: page.Docs[0].IDToken, BaseHash: page.Docs[0].Hash, Doc: decoded,
	})
	require.NoError(t, err)

	stored := itRawDoc(t, conn, "it_dateonly", bson.D{{Key: "_id", Value: id}})
	createdAt, ok := mongodoc.Lookup(stored, "createdAt")
	require.True(t, ok)
	assert.IsType(t, bson.DateTime(0), createdAt, "Date 字段写回后不得变成字符串")
	assert.Equal(t, at, createdAt)
}

// TestITMongoIDTokenSeparatesStringAndObjectID 锁死旧实现的类型猜测缺陷：
// 库里同时存在「24 位十六进制的字符串主键」与「同形状 ObjectID 主键」时，
// 按形状猜会让更新/删除打到错误的文档上；主键令牌必须精确区分。
func TestITMongoIDTokenSeparatesStringAndObjectID(t *testing.T) {
	app, conn := itEnv(t)
	ctx := itCtx()

	const hexLike = "507f1f77bcf86cd799439011"
	objectID, err := bson.ObjectIDFromHex(hexLike)
	require.NoError(t, err)

	_, err = app.InsertDocuments(ctx, conn, &dto.DocInsert{
		Database: itDatabase, Collection: "it_idkinds",
		Docs: []bson.D{
			{{Key: "_id", Value: hexLike}, {Key: "kind", Value: "string"}},
			{{Key: "_id", Value: objectID}, {Key: "kind", Value: "objectid"}},
		},
	})
	require.NoError(t, err)

	page, err := app.QueryDocuments(ctx, conn, &dto.DocQuery{
		Database: itDatabase, Collection: "it_idkinds", Limit: 10, Sort: bson.D{{Key: "kind", Value: 1}},
	})
	require.NoError(t, err)
	require.Len(t, page.Docs, 2)

	// 只更新字符串主键那条。按下标与按 _id 文本形状都不可靠（ObjectID 与同形字符串主键
	// 在展示上长得一样），靠后端下发的 idKind 认类型
	var stringDoc, objectIDDoc *mongodoc.QueryDoc
	for _, item := range page.Docs {
		switch item.IDKind {
		case "string":
			stringDoc = item
		case "objectId":
			objectIDDoc = item
		}
	}
	require.NotNil(t, stringDoc, "字符串主键必须原样回传且 idKind 为 string")
	require.NotNil(t, objectIDDoc, "ObjectID 主键的 idKind 必须为 objectId")
	assert.Equal(t, hexLike, itField(t, stringDoc, "_id"), "字符串主键值不得被改写")

	decoded, err := mongodoc.Decode(stringDoc.Doc)
	require.NoError(t, err)
	_, err = app.UpdateDocument(ctx, conn, &dto.DocUpdate{
		Database: itDatabase, Collection: "it_idkinds",
		IDToken: stringDoc.IDToken, BaseHash: stringDoc.Hash, Doc: decoded,
	})
	require.NoError(t, err)

	// 库里两条都还在，且被改的只是字符串那条
	rawString := itRawDoc(t, conn, "it_idkinds", bson.D{{Key: "_id", Value: hexLike}})
	assert.Equal(t, "string", rawString[1].Value)
	_, err = conn.Cli.Database(itDatabase).Collection("it_idkinds").CountDocuments(ctx, bson.D{{Key: "_id", Value: objectID}})
	require.NoError(t, err, "objectid document must not be touched by a string-id operation")

	// 令牌的类型与存储类型一致：一个解出 string，一个解出 ObjectID
	oidValue, err := mongodoc.ParseID(objectIDDoc.IDToken)
	require.NoError(t, err)
	assert.IsType(t, bson.ObjectID{}, oidValue)

	decodedValue, err := mongodoc.ParseID(stringDoc.IDToken)
	require.NoError(t, err)
	assert.IsType(t, "", decodedValue)
}

// TestITMongoUpdateDiffSemantics 「从 JSON 里删掉一个 key」必须真的删除字段，
// 同类型同值必须不产生写入（旧实现整份 $set，既删不掉字段也覆盖面积更大）。
func TestITMongoUpdateDiffSemantics(t *testing.T) {
	app, conn := itEnv(t)
	ctx := itCtx()

	id := bson.NewObjectID()
	_, err := app.InsertDocuments(ctx, conn, &dto.DocInsert{
		Database: itDatabase, Collection: "it_diff",
		Docs: []bson.D{{{Key: "_id", Value: id}, {Key: "keep", Value: "a"}, {Key: "drop", Value: "b"}, {Key: "change", Value: int32(1)}}},
	})
	require.NoError(t, err)

	page, err := app.QueryDocuments(ctx, conn, &dto.DocQuery{Database: itDatabase, Collection: "it_diff", Limit: 10})
	require.NoError(t, err)
	doc := page.Docs[0]

	// 什么都没改：noChange，不产生写入
	untouched, err := mongodoc.Decode(doc.Doc)
	require.NoError(t, err)
	res, err := app.UpdateDocument(ctx, conn, &dto.DocUpdate{
		Database: itDatabase, Collection: "it_diff", IDToken: doc.IDToken, BaseHash: doc.Hash, Doc: untouched,
	})
	require.NoError(t, err)
	assert.True(t, res.NoChange, "identical document must not be written")

	// 删掉 drop、新增 added、改 change
	edited, err := mongodoc.Decode(json.RawMessage(`{"_id":{"$oid":"` + id.Hex() + `"},"keep":"a","change":2,"added":true}`))
	require.NoError(t, err)
	res, err = app.UpdateDocument(ctx, conn, &dto.DocUpdate{
		Database: itDatabase, Collection: "it_diff", IDToken: doc.IDToken, BaseHash: doc.Hash, Doc: edited,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.ModifiedCount)

	stored := itRawDoc(t, conn, "it_diff", bson.D{{Key: "_id", Value: id}})
	_, hasDrop := mongodoc.Lookup(stored, "drop")
	assert.False(t, hasDrop, "removing a key from the json must unset the field")
	added, hasAdded := mongodoc.Lookup(stored, "added")
	assert.True(t, hasAdded)
	assert.Equal(t, true, added)

	// int32(1) -> 提交 2：JSON 整数按 ExtJSON 规则解为 int32；要得到 int64 必须写 $numberLong
	changeValue, hasNext := mongodoc.Lookup(stored, "change")
	require.True(t, hasNext, "改动后的字段必须存在")
	assert.IsType(t, int32(0), changeValue)
	assert.Equal(t, int32(2), changeValue)

	longEdited, err := mongodoc.Decode(json.RawMessage(`{"change":{"$numberLong":"9007199254740993"},"keep":"a","added":true}`))
	require.NoError(t, err)
	second, err := app.QueryDocuments(ctx, conn, &dto.DocQuery{Database: itDatabase, Collection: "it_diff", Limit: 10})
	require.NoError(t, err)
	_, err = app.UpdateDocument(ctx, conn, &dto.DocUpdate{
		Database: itDatabase, Collection: "it_diff", IDToken: second.Docs[0].IDToken, BaseHash: second.Docs[0].Hash, Doc: longEdited,
	})
	require.NoError(t, err)

	reloaded := itRawDoc(t, conn, "it_diff", bson.D{{Key: "_id", Value: id}})
	changeAgain, ok := mongodoc.Lookup(reloaded, "change")
	require.True(t, ok)
	assert.IsType(t, int64(0), changeAgain, "$numberLong must be stored as int64")
	assert.Equal(t, int64(9007199254740993), changeAgain)
}

// TestITMongoOptimisticLockRejectsStaleWrite 后保存的人不能静默覆盖前一个人。
func TestITMongoOptimisticLockRejectsStaleWrite(t *testing.T) {
	app, conn := itEnv(t)
	ctx := itCtx()

	id := bson.NewObjectID()
	_, err := app.InsertDocuments(ctx, conn, &dto.DocInsert{
		Database: itDatabase, Collection: "it_lock",
		Docs: []bson.D{{{Key: "_id", Value: id}, {Key: "v", Value: "first"}}},
	})
	require.NoError(t, err)

	page, err := app.QueryDocuments(ctx, conn, &dto.DocQuery{Database: itDatabase, Collection: "it_lock", Limit: 10})
	require.NoError(t, err)
	doc := page.Docs[0]

	// 另一个会话先改了一次（指纹随之变化）
	other, err := mongodoc.Decode(json.RawMessage(`{"v":"from-other"}`))
	require.NoError(t, err)
	_, err = app.UpdateDocument(ctx, conn, &dto.DocUpdate{
		Database: itDatabase, Collection: "it_lock", IDToken: doc.IDToken, BaseHash: doc.Hash, Doc: other,
	})
	require.NoError(t, err)

	// 用户仍拿着旧指纹提交，必须被拒
	stale, err := mongodoc.Decode(json.RawMessage(`{"v":"from-me"}`))
	require.NoError(t, err)
	_, err = app.UpdateDocument(ctx, conn, &dto.DocUpdate{
		Database: itDatabase, Collection: "it_lock", IDToken: doc.IDToken, BaseHash: doc.Hash, Doc: stale,
	})
	assert.ErrorIs(t, err, ErrDocConflict)

	stored := itRawDoc(t, conn, "it_lock", bson.D{{Key: "_id", Value: id}})
	assert.Equal(t, "from-other", stored[1].Value, "conflicted write must not have reached the database")
}

// TestITMongoCompoundSortIsStableAcrossPages 复合排序键的顺序此前被 map 迭代序打乱，
// 表现为翻页出现重复与漏行。这里按「分页并集 == 一次全量」验证。
func TestITMongoCompoundSortIsStableAcrossPages(t *testing.T) {
	app, conn := itEnv(t)
	ctx := itCtx()

	const docCount = 30
	docs := make([]bson.D, 0, docCount)
	for i := range docCount {
		// score 只有 3 个取值，必须靠第二排序键 name 才能定序
		docs = append(docs, bson.D{
			{Key: "score", Value: int32(i % 3)},
			{Key: "name", Value: fmt.Sprintf("n-%02d", i)},
		})
	}
	_, err := app.InsertDocuments(ctx, conn, &dto.DocInsert{Database: itDatabase, Collection: "it_sort", Docs: docs})
	require.NoError(t, err)

	sortKey := bson.D{{Key: "score", Value: 1}, {Key: "name", Value: -1}}

	var paged []string
	for page := 1; ; page++ {
		res, err := app.QueryDocuments(ctx, conn, &dto.DocQuery{
			Database: itDatabase, Collection: "it_sort", Sort: sortKey, Limit: 7, Skip: int64((page - 1) * 7),
		})
		require.NoError(t, err)
		if len(res.Docs) == 0 {
			break
		}
		for _, item := range res.Docs {
			paged = append(paged, fmt.Sprint(itField(t, item, "name")))
		}
		if len(res.Docs) < 7 {
			break
		}
	}
	require.Len(t, paged, docCount, "分页并集必须覆盖全部文档")
	assert.Equal(t, len(paged), len(uniqueStrings(paged)), "分页之间不得出现重复文档")

	// 与一次全量读取的期望顺序一致：score 升序、同分按 name 降序
	all, err := app.QueryDocuments(ctx, conn, &dto.DocQuery{Database: itDatabase, Collection: "it_sort", Sort: sortKey, Limit: 100})
	require.NoError(t, err)
	expected := make([]string, 0, len(all.Docs))
	for _, item := range all.Docs {
		expected = append(expected, fmt.Sprint(itField(t, item, "name")))
	}
	assert.Equal(t, expected, paged)
}

// TestITMongoQueryLimitClampedAndTruncated 上限归一与截断标记：
// 请求条数超过配置上限时按上限生效并明确告知前端，而不是静默少给。
func TestITMongoQueryLimitClampedAndTruncated(t *testing.T) {
	app, conn := itEnv(t)
	ctx := itCtx()

	docs := make([]bson.D, 0, 25)
	for i := range 25 {
		docs = append(docs, bson.D{{Key: "n", Value: int32(i)}})
	}
	_, err := app.InsertDocuments(ctx, conn, &dto.DocInsert{Database: itDatabase, Collection: "it_limit", Docs: docs})
	require.NoError(t, err)

	// 未指定 limit → 落到默认值；25 条取 20 条，确实被上限截断
	page, err := app.QueryDocuments(ctx, conn, &dto.DocQuery{Database: itDatabase, Collection: "it_limit"})
	require.NoError(t, err)
	assert.Equal(t, int64(20), page.Limit)
	assert.True(t, page.Truncated, "取满且后面还有数据时必须报截断")
	assert.Len(t, page.Docs, 20)

	// 超限 → 归一到配置上限
	page, err = app.QueryDocuments(ctx, conn, &dto.DocQuery{Database: itDatabase, Collection: "it_limit", Limit: 100000})
	require.NoError(t, err)
	assert.Equal(t, int64(500), page.Limit)

	// 触达上限 → truncated 为真
	page, err = app.QueryDocuments(ctx, conn, &dto.DocQuery{Database: itDatabase, Collection: "it_limit", Limit: 10})
	require.NoError(t, err)
	assert.Len(t, page.Docs, 10)
	assert.True(t, page.Truncated)
	assert.Equal(t, int64(-1), page.Total, "未开启统计时总数必须是 -1 而不是猜测值")

	// 开启统计 → 得到匹配总数（与集合总数无关）
	page, err = app.QueryDocuments(ctx, conn, &dto.DocQuery{
		Database: itDatabase, Collection: "it_limit", Limit: 10, WithCount: true,
		Filter: bson.D{{Key: "n", Value: bson.D{{Key: "$gte", Value: 20}}}},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5), page.Total)
	assert.Len(t, page.Docs, 5)
	assert.False(t, page.Truncated)
}

// TestITMongoCompoundIndexKeyOrderPreserved 复合索引键的顺序是语义的一部分：
// 命令以原始 JSON 进入、按序编码为 BSON，才能保证建出的索引就是用户写的顺序。
func TestITMongoCompoundIndexKeyOrderPreserved(t *testing.T) {
	app, conn := itEnv(t)
	ctx := itCtx()

	_, err := app.InsertDocuments(ctx, conn, &dto.DocInsert{
		Database: itDatabase, Collection: "it_index", Docs: []bson.D{{{Key: "a", Value: 1}, {Key: "b", Value: 2}}},
	})
	require.NoError(t, err)

	command := json.RawMessage(`{"createIndexes":"it_index","indexes":[{"key":{"b":1,"a":-1},"name":"idx_b_a"}]}`)
	decoded, err := mongodoc.Decode(command)
	require.NoError(t, err)
	_, level, err := mongodoc.Classify(decoded)
	require.NoError(t, err)
	require.Equal(t, mongodoc.LevelStructSave, level, "建索引必须按结构变更级别鉴权")

	_, err = app.RunCommand(ctx, conn, itDatabase, decoded)
	require.NoError(t, err)

	// 用驱动直连读回索引定义，确认库里真实存的是 b 前 a 后
	cur, err := conn.Cli.Database(itDatabase).Collection("it_index").Indexes().List(ctx)
	require.NoError(t, err)
	var indexDocs []bson.D
	require.NoError(t, cur.All(ctx, &indexDocs))

	for _, idx := range indexDocs {
		name := fmt.Sprint(mustLookup(t, idx, "name"))
		if name != "idx_b_a" {
			continue
		}
		key := mustLookup(t, idx, "key").(bson.D)
		require.Len(t, key, 2)
		assert.Equal(t, "b", key[0].Key, "compound index key order must be preserved")
		assert.Equal(t, "a", key[1].Key)
		return
	}
	t.Fatal("index idx_b_a not found after createIndexes")
}

// TestITMongoStatsUnavailableDoesNotBreakQuery 视图没有 $collStats：
// 统计取不到只应让头部读数缺失，不能把已经取到的数据判成失败。
func TestITMongoStatsUnavailableDoesNotBreakQuery(t *testing.T) {
	app, conn := itEnv(t)
	ctx := itCtx()

	_, err := app.InsertDocuments(ctx, conn, &dto.DocInsert{
		Database: itDatabase, Collection: "it_base", Docs: []bson.D{{{Key: "a", Value: int32(1)}}, {{Key: "a", Value: int32(2)}}},
	})
	require.NoError(t, err)

	// 正常集合：统计可用
	page, err := app.QueryDocuments(ctx, conn, &dto.DocQuery{Database: itDatabase, Collection: "it_base", Limit: 10})
	require.NoError(t, err)
	assert.NotEmpty(t, page.Stats)
	assert.Empty(t, page.StatsError)
	assert.Equal(t, int64(2), page.Stats.Count, "头部读数必须拿到文档数，got %+v", page.Stats)
	assert.Equal(t, itDatabase+".it_base", page.Stats.Ns)
	assert.Greater(t, page.Stats.StorageSize, int64(0))
	assert.Equal(t, int64(1), page.Stats.NIndexes, "新建集合默认只有 _id 索引")

	// 建视图：视图没有 $collStats，取统计会失败
	_, err = app.RunCommand(ctx, conn, itDatabase, itViewCommand())
	require.NoError(t, err)

	page, err = app.QueryDocuments(ctx, conn, &dto.DocQuery{Database: itDatabase, Collection: "it_view", Limit: 10})
	require.NoError(t, err, "querying a view must still succeed even when stats are unavailable")
	assert.Len(t, page.Docs, 2)
	assert.Empty(t, page.Stats)
	assert.NotEmpty(t, page.StatsError, "view stats failure must be reported as a reason, not as a query error")
}

// TestITMongoDropCollectionAndDatabase 结构销毁走专用端点后的真实效果。
func TestITMongoDropCollectionAndDatabase(t *testing.T) {
	app, conn := itEnv(t)
	ctx := itCtx()

	require.NoError(t, app.CreateCollection(ctx, conn, itDatabase, "it_drop"))
	colls, err := app.ListCollections(ctx, conn, itDatabase)
	require.NoError(t, err)
	assert.True(t, containsCollection(colls, "it_drop"))

	require.NoError(t, app.DropCollection(ctx, conn, itDatabase, "it_drop"))
	colls, err = app.ListCollections(ctx, conn, itDatabase)
	require.NoError(t, err)
	assert.False(t, containsCollection(colls, "it_drop"))

	// 空库在 Mongo 里不出现于 listDatabases，因此以「建集合→删库→集合消失」验证
	require.NoError(t, app.CreateCollection(ctx, conn, "mayfly_mongo_it_tmp", "tmp_coll"))
	require.NoError(t, app.DropDatabase(ctx, conn, "mayfly_mongo_it_tmp"))
	dbs, err := app.ListDatabases(ctx, conn)
	require.NoError(t, err)
	for _, db := range dbs {
		assert.NotEqual(t, "mayfly_mongo_it_tmp", db.Name)
	}
}

// TestITMongoBatchDelete 批量删除按实际影响条数如实回传。
func TestITMongoBatchDelete(t *testing.T) {
	app, conn := itEnv(t)
	ctx := itCtx()

	docs := make([]bson.D, 0, 3)
	for i := range 3 {
		docs = append(docs, bson.D{{Key: "n", Value: int32(i)}})
	}
	_, err := app.InsertDocuments(ctx, conn, &dto.DocInsert{Database: itDatabase, Collection: "it_del", Docs: docs})
	require.NoError(t, err)

	page, err := app.QueryDocuments(ctx, conn, &dto.DocQuery{Database: itDatabase, Collection: "it_del", Limit: 10})
	require.NoError(t, err)
	tokens := make([]string, 0, len(page.Docs))
	for _, item := range page.Docs {
		tokens = append(tokens, item.IDToken)
	}
	require.Len(t, tokens, 3)

	res, err := app.DeleteDocuments(ctx, conn, &dto.DocDelete{Database: itDatabase, Collection: "it_del", IDTokens: tokens})
	require.NoError(t, err)
	assert.Equal(t, int64(3), res.DeletedCount)

	// 再删一次：文档已不在，实际删除条数必须是 0 而不是「请求了几条就删了几条」
	res, err = app.DeleteDocuments(ctx, conn, &dto.DocDelete{Database: itDatabase, Collection: "it_del", IDTokens: tokens})
	require.NoError(t, err)
	assert.Equal(t, int64(0), res.DeletedCount)
}

// TestITMongoInsertRejectsEmpty 空提交必须报错，而不是返回一个零计数让人误以为成功。
func TestITMongoInsertRejectsEmpty(t *testing.T) {
	app, conn := itEnv(t)

	_, err := app.InsertDocuments(itCtx(), conn, &dto.DocInsert{Database: itDatabase, Collection: "it_empty"})
	assert.ErrorIs(t, err, ErrNothingToWrite)
}

// TestITMongoAggregate 聚合：只读管道返回分组结果，写出 stage 真的建出集合并保留类型。
func TestITMongoAggregate(t *testing.T) {
	app, conn := itEnv(t)
	ctx := itCtx()

	_, err := app.InsertDocuments(ctx, conn, &dto.DocInsert{
		Database: itDatabase, Collection: "it_agg",
		Docs: []bson.D{
			{{Key: "status", Value: "paid"}, {Key: "amount", Value: itDecimal(t, "10.5")}},
			{{Key: "status", Value: "paid"}, {Key: "amount", Value: itDecimal(t, "20.25")}},
			{{Key: "status", Value: "unpaid"}, {Key: "amount", Value: itDecimal(t, "7")}},
		},
	})
	require.NoError(t, err)

	stages, err := mongodoc.DecodePipeline(json.RawMessage(`[{"$group":{"_id":"$status","total":{"$sum":"$amount"}}}]`))
	require.NoError(t, err)
	page, err := app.Aggregate(ctx, conn, &dto.AggQuery{Database: itDatabase, Collection: "it_agg", Pipeline: stages})
	require.NoError(t, err)
	require.Len(t, page.Docs, 2)

	var paidTotal string
	for _, item := range page.Docs {
		value := itField(t, item, "total")
		if fmt.Sprint(itField(t, item, "_id")) == "paid" {
			encoded, err := bson.MarshalExtJSON(bson.D{{Key: "v", Value: value}}, true, false)
			require.NoError(t, err)
			paidTotal = string(encoded)
		}
	}
	// Decimal128 求和仍是 Decimal128（换成 double 就是静默丢精度）
	assert.Contains(t, paidTotal, "$numberDecimal")
	assert.Contains(t, paidTotal, "30.75")

	// 写出 stage：$out 真的创建目标集合
	outStages, err := mongodoc.DecodePipeline(json.RawMessage(`[{"$match":{"status":"paid"}},{"$out":"it_agg_out"}]`))
	require.NoError(t, err)
	_, err = app.Aggregate(ctx, conn, &dto.AggQuery{Database: itDatabase, Collection: "it_agg", Pipeline: outStages})
	require.NoError(t, err)

	colls, err := app.ListCollections(ctx, conn, itDatabase)
	require.NoError(t, err)
	assert.True(t, containsCollection(colls, "it_agg_out"), "$out must materialize the target collection")

	// explain 不执行管道：带 $out 的解释不得留下新集合
	before, err := app.ListCollections(ctx, conn, itDatabase)
	require.NoError(t, err)
	_, err = app.Aggregate(ctx, conn, &dto.AggQuery{
		Database: itDatabase, Collection: "it_agg", Pipeline: outStages, Explain: true,
	})
	require.NoError(t, err)

	after, err := app.ListCollections(ctx, conn, itDatabase)
	require.NoError(t, err)
	assert.Equal(t, len(before), len(after), "explain must not create collections")
}

// TestITMongoCollectionMetaAndIndexes 元信息面板的数据源：统计 + 索引列表一次取齐，
// 复合索引的键序与大小必须真实可读。
func TestITMongoCollectionMetaAndIndexes(t *testing.T) {
	app, conn := itEnv(t)
	ctx := itCtx()

	_, err := app.InsertDocuments(ctx, conn, &dto.DocInsert{
		Database: itDatabase, Collection: "it_meta",
		Docs: []bson.D{{{Key: "a", Value: int32(1)}}, {{Key: "b", Value: int32(2)}}},
	})
	require.NoError(t, err)

	require.NoError(t, app.CreateIndexes(ctx, conn, itDatabase, "it_meta",
		json.RawMessage(`[{"key":{"b":1,"a":-1},"name":"idx_b_a","unique":true}]`)))

	meta, err := app.CollectionMeta(ctx, conn, itDatabase, "it_meta")
	require.NoError(t, err)
	require.NotNil(t, meta.Stats)
	assert.Equal(t, int64(2), meta.TotalDocs)
	assert.Equal(t, int64(2), meta.Stats.Count)
	assert.Empty(t, meta.StatsError)
	assert.Greater(t, meta.Stats.TotalIndexSize, int64(0))

	var compound *dto.IndexInfo
	for _, index := range meta.Indexes {
		if index.Name == "idx_b_a" {
			compound = index
		}
	}
	require.NotNil(t, compound, "created index must be listed")
	assert.True(t, compound.Unique)
	assert.Greater(t, compound.SizeBytes, int64(0), "index size must come from $collStats.indexSizes")
	require.Len(t, compound.Keys, 2)
	assert.Equal(t, "b", compound.Keys[0].Field)
	assert.Equal(t, "asc", compound.Keys[0].Direction)
	assert.Equal(t, "a", compound.Keys[1].Field)
	assert.Equal(t, "desc", compound.Keys[1].Direction)

	// _id 索引仍在，说明列表不是只返了新建那几个
	names := make([]string, 0, len(meta.Indexes))
	for _, index := range meta.Indexes {
		names = append(names, index.Name)
	}
	assert.Contains(t, names, "_id_")

	// 非法索引定义必须被拦下（空数组、缺 key）
	for _, specs := range []string{`[]`, `[{"name":"x"}]`, `{"key":{"a":1}}`} {
		assert.ErrorIs(t, app.CreateIndexes(ctx, conn, itDatabase, "it_meta", json.RawMessage(specs)), ErrEmptyIndexSpecs,
			"specs %s must be rejected", specs)
	}

	require.NoError(t, app.DropIndex(ctx, conn, itDatabase, "it_meta", "idx_b_a"))
	meta, err = app.CollectionMeta(ctx, conn, itDatabase, "it_meta")
	require.NoError(t, err)
	for _, index := range meta.Indexes {
		assert.NotEqual(t, "idx_b_a", index.Name, "dropped index must disappear")
	}
}

// TestITMongoBatchCountGuard 批量改删的命中数护栏：不一致时不能动任何数据。
func TestITMongoBatchCountGuard(t *testing.T) {
	app, conn := itEnv(t)
	ctx := itCtx()

	docs := []bson.D{}
	for i := range 5 {
		docs = append(docs, bson.D{{Key: "n", Value: int32(i)}, {Key: "status", Value: "new"}})
	}
	_, err := app.InsertDocuments(ctx, conn, &dto.DocInsert{Database: itDatabase, Collection: "it_batch", Docs: docs})
	require.NoError(t, err)

	filter := bson.D{{Key: "status", Value: "new"}}
	spec, err := mongodoc.DecodeUpdateSpec(json.RawMessage(`{"$set":{"status":"done"}}`))
	require.NoError(t, err)

	// 预期与实际不符：中止，且一条也没改
	_, err = app.UpdateByFilter(ctx, conn, &dto.BatchUpdate{
		Database: itDatabase, Collection: "it_batch", Filter: filter, Update: spec, ExpectCount: 3,
	})
	var mismatch *BatchCountMismatchError
	require.ErrorAs(t, err, &mismatch)
	assert.Equal(t, int64(3), mismatch.Expect)
	assert.Equal(t, int64(5), mismatch.Actual)

	page, err := app.QueryDocuments(ctx, conn, &dto.DocQuery{Database: itDatabase, Collection: "it_batch", Limit: 10, WithCount: true})
	require.NoError(t, err)
	assert.Equal(t, int64(5), page.Total)
	for _, item := range page.Docs {
		assert.Equal(t, "new", itField(t, item, "status"), "guarded update must not touch any document")
	}

	// 预期正确：5 条全部更新
	res, err := app.UpdateByFilter(ctx, conn, &dto.BatchUpdate{
		Database: itDatabase, Collection: "it_batch", Filter: filter, Update: spec, ExpectCount: 5,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5), res.MatchedCount)
	assert.Equal(t, int64(5), res.ModifiedCount)

	// 批量删除同样有护栏：上面的更新已把 status 改成 done，旧条件现在匹配 0 条；
	// 拿旧命中数去删必须被中止，而不是「没删到就算成功」
	_, err = app.DeleteByFilter(ctx, conn, &dto.BatchDelete{
		Database: itDatabase, Collection: "it_batch", Filter: filter, ExpectCount: 5,
	})
	require.ErrorAs(t, err, &mismatch)
	assert.Equal(t, int64(5), mismatch.Expect)
	assert.Equal(t, int64(0), mismatch.Actual)

	doneFilter := bson.D{{Key: "status", Value: "done"}}
	_, err = app.DeleteByFilter(ctx, conn, &dto.BatchDelete{
		Database: itDatabase, Collection: "it_batch", Filter: doneFilter, ExpectCount: 2,
	})
	require.ErrorAs(t, err, &mismatch)

	res, err = app.DeleteByFilter(ctx, conn, &dto.BatchDelete{
		Database: itDatabase, Collection: "it_batch", Filter: doneFilter, ExpectCount: 5,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5), res.DeletedCount)

	page, err = app.QueryDocuments(ctx, conn, &dto.DocQuery{Database: itDatabase, Collection: "it_batch", Limit: 10})
	require.NoError(t, err)
	assert.Empty(t, page.Docs)
}

// TestITMongoExport 导出取数：两种格式都能在真实数据上产出可用内容，且不超配置上限。
func TestITMongoExport(t *testing.T) {
	app, conn := itEnv(t)
	ctx := itCtx()

	_, err := app.InsertDocuments(ctx, conn, &dto.DocInsert{
		Database: itDatabase, Collection: "it_export",
		Docs: []bson.D{
			{{Key: "name", Value: "a,b"}, {Key: "at", Value: bson.NewDateTimeFromTime(time.Unix(1700000000, 0).UTC())}},
			{{Key: "name", Value: `say "hi"`}, {Key: "at", Value: bson.NewDateTimeFromTime(time.Unix(1700000001, 0).UTC())}},
		},
	})
	require.NoError(t, err)

	for _, format := range mongoexport.Formats {
		file, err := app.ExportDocuments(ctx, conn, &dto.ExportRequest{
			Database: itDatabase, Collection: "it_export", Format: format,
		})
		require.NoError(t, err, "format %s", format)
		assert.Equal(t, 2, file.Count)
		assert.NotEmpty(t, file.Content)
		assert.NotEmpty(t, file.ContentType)
	}

	csvFile, err := app.ExportDocuments(ctx, conn, &dto.ExportRequest{
		Database: itDatabase, Collection: "it_export", Format: mongoexport.FormatCSV,
	})
	require.NoError(t, err)
	body := string(csvFile.Content)
	assert.Contains(t, body, "name,at\n")
	assert.Contains(t, body, `"a,b"`, "字段里的逗号必须被引号包住")
	assert.Contains(t, body, `"say ""hi"""`, "字段里的引号必须双写转义")

	_, err = app.ExportDocuments(ctx, conn, &dto.ExportRequest{
		Database: itDatabase, Collection: "it_export", Format: "sql",
	})
	assert.Error(t, err, "不支持的格式必须报错而不是静默产出空文件")
}

// ---------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------

func itDecimal(t *testing.T, s string) bson.Decimal128 {
	t.Helper()
	d, err := bson.ParseDecimal128(s)
	require.NoError(t, err)
	return d
}

// itField 从返回文档里取一个顶层字段。Doc 是保真 JSON 文本，断言前统一解码一次。
func itField(t *testing.T, qdoc *mongodoc.QueryDoc, key string) any {
	t.Helper()

	doc, err := mongodoc.Decode(qdoc.Doc)
	require.NoError(t, err)

	value, ok := mongodoc.Lookup(doc, key)
	require.True(t, ok, "field %q missing in %s", key, string(qdoc.Doc))
	return value
}

func itViewCommand() bson.D {
	return bson.D{
		{Key: "create", Value: "it_view"},
		{Key: "viewOn", Value: "it_base"},
		{Key: "pipeline", Value: bson.A{bson.D{{Key: "$match", Value: bson.D{{Key: "a", Value: bson.D{{Key: "$gte", Value: 1}}}}}}}},
	}
}

func mustLookup(t *testing.T, doc bson.D, key string) any {
	t.Helper()
	value, ok := mongodoc.Lookup(doc, key)
	require.True(t, ok, "field %q missing", key)
	return value
}

func containsCollection(colls []*dto.Collection, name string) bool {
	for _, c := range colls {
		if c.Name == name {
			return true
		}
	}
	return false
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]bool, len(items))
	var out []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}
