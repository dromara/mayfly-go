package api

import (
	"context"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/mongo/api/form"
	"mayfly-go/internal/mongo/application"
	"mayfly-go/internal/mongo/domain/entity"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mongo struct {
	MongoApp application.Mongo `inject:""`
	TagApp   tagapp.TagTree    `inject:"TagTreeApp"`
}

func (m *Mongo) Mongos(rc *req.Ctx) {
	queryCond, page := ginx.BindQueryAndPage[*entity.MongoQuery](rc.GinCtx, new(entity.MongoQuery))

	// 不存在可访问标签id，即没有可操作数据
	codes := m.TagApp.GetAccountResourceCodes(rc.GetLoginAccount().Id, consts.TagResourceTypeMongo, queryCond.TagPath)
	if len(codes) == 0 {
		rc.ResData = model.EmptyPageResult[any]()
		return
	}
	queryCond.Codes = codes

	res, err := m.MongoApp.GetPageList(queryCond, page, new([]entity.Mongo))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (m *Mongo) TestConn(rc *req.Ctx) {
	form := &form.Mongo{}
	mongo := ginx.BindJsonAndCopyTo[*entity.Mongo](rc.GinCtx, form, new(entity.Mongo))
	biz.ErrIsNilAppendErr(m.MongoApp.TestConn(mongo), "连接失败: %s")
}

func (m *Mongo) Save(rc *req.Ctx) {
	form := &form.Mongo{}
	mongo := ginx.BindJsonAndCopyTo[*entity.Mongo](rc.GinCtx, form, new(entity.Mongo))

	// 密码脱敏记录日志
	form.Uri = func(str string) string {
		reg := regexp.MustCompile(`(^mongodb://.+?:)(.+)(@.+$)`)
		return reg.ReplaceAllString(str, `${1}****${3}`)
	}(form.Uri)
	rc.ReqParam = form

	biz.ErrIsNil(m.MongoApp.SaveMongo(rc.MetaCtx, mongo, form.TagId...))
}

func (m *Mongo) DeleteMongo(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		m.MongoApp.Delete(rc.MetaCtx, uint64(value))
	}
}

func (m *Mongo) Databases(rc *req.Ctx) {
	conn, err := m.MongoApp.GetMongoConn(m.GetMongoId(rc.GinCtx))
	biz.ErrIsNil(err)
	res, err := conn.Cli.ListDatabases(context.TODO(), bson.D{})
	biz.ErrIsNilAppendErr(err, "获取mongo所有库信息失败: %s")
	rc.ResData = res
}

func (m *Mongo) Collections(rc *req.Ctx) {
	conn, err := m.MongoApp.GetMongoConn(m.GetMongoId(rc.GinCtx))
	biz.ErrIsNil(err)
	db := rc.GinCtx.Query("database")
	biz.NotEmpty(db, "database不能为空")
	ctx := context.TODO()
	res, err := conn.Cli.Database(db).ListCollectionNames(ctx, bson.D{})
	biz.ErrIsNilAppendErr(err, "获取库集合信息失败: %s")
	rc.ResData = res
}

func (m *Mongo) RunCommand(rc *req.Ctx) {
	commandForm := new(form.MongoRunCommand)
	ginx.BindJsonAndValid(rc.GinCtx, commandForm)

	conn, err := m.MongoApp.GetMongoConn(m.GetMongoId(rc.GinCtx))
	biz.ErrIsNil(err)
	rc.ReqParam = collx.Kvs("mongo", conn.Info, "cmd", commandForm)

	// 顺序执行
	commands := bson.D{}
	for _, cmd := range commandForm.Command {
		e := bson.E{}
		for k, v := range cmd {
			e.Key = k
			e.Value = v
		}
		commands = append(commands, e)
	}

	ctx := context.TODO()
	var bm bson.M
	err = conn.Cli.Database(commandForm.Database).RunCommand(
		ctx,
		commands,
	).Decode(&bm)

	biz.ErrIsNilAppendErr(err, "执行命令失败: %s")
	rc.ResData = bm
}

func (m *Mongo) FindCommand(rc *req.Ctx) {
	g := rc.GinCtx
	commandForm := new(form.MongoFindCommand)
	ginx.BindJsonAndValid(g, commandForm)

	conn, err := m.MongoApp.GetMongoConn(m.GetMongoId(g))
	biz.ErrIsNil(err)
	cli := conn.Cli

	limit := commandForm.Limit
	if limit != 0 {
		biz.IsTrue(limit <= 100, "limit不能超过100")
	}
	opts := options.Find().SetSort(commandForm.Sort).
		SetSkip(commandForm.Skip).
		SetLimit(limit)
	ctx := context.TODO()

	filter := commandForm.Filter
	// 处理_id查询字段,使用ObjectId函数包装
	id, ok := filter["_id"].(string)
	if ok && id != "" {
		objId, err := primitive.ObjectIDFromHex(id)
		if err == nil {
			filter["_id"] = objId
		}
	}

	cur, err := cli.Database(commandForm.Database).Collection(commandForm.Collection).Find(ctx, commandForm.Filter, opts)
	biz.ErrIsNilAppendErr(err, "命令执行失败: %s")

	var res []bson.M
	cur.All(ctx, &res)
	rc.ResData = res
}

func (m *Mongo) UpdateByIdCommand(rc *req.Ctx) {
	g := rc.GinCtx
	commandForm := new(form.MongoUpdateByIdCommand)
	ginx.BindJsonAndValid(g, commandForm)

	conn, err := m.MongoApp.GetMongoConn(m.GetMongoId(g))
	biz.ErrIsNil(err)
	rc.ReqParam = collx.Kvs("mongo", conn.Info, "cmd", commandForm)

	// 解析docId文档id，如果为string类型则使用ObjectId解析，解析失败则为普通字符串
	docId := commandForm.DocId
	docIdVal, ok := docId.(string)
	if ok {
		objId, err := primitive.ObjectIDFromHex(docIdVal)
		if err == nil {
			docId = objId
		}
	}

	res, err := conn.Cli.Database(commandForm.Database).Collection(commandForm.Collection).UpdateByID(context.TODO(), docId, commandForm.Update)
	biz.ErrIsNilAppendErr(err, "命令执行失败: %s")

	rc.ResData = res
}

func (m *Mongo) DeleteByIdCommand(rc *req.Ctx) {
	g := rc.GinCtx
	commandForm := new(form.MongoUpdateByIdCommand)
	ginx.BindJsonAndValid(g, commandForm)

	conn, err := m.MongoApp.GetMongoConn(m.GetMongoId(g))
	biz.ErrIsNil(err)
	rc.ReqParam = collx.Kvs("mongo", conn.Info, "cmd", commandForm)

	// 解析docId文档id，如果为string类型则使用ObjectId解析，解析失败则为普通字符串
	docId := commandForm.DocId
	docIdVal, ok := docId.(string)
	if ok {
		objId, err := primitive.ObjectIDFromHex(docIdVal)
		if err == nil {
			docId = objId
		}
	}

	res, err := conn.Cli.Database(commandForm.Database).Collection(commandForm.Collection).DeleteOne(context.TODO(), bson.D{{Key: "_id", Value: docId}})
	biz.ErrIsNilAppendErr(err, "命令执行失败: %s")
	rc.ResData = res
}

func (m *Mongo) InsertOneCommand(rc *req.Ctx) {
	g := rc.GinCtx
	commandForm := new(form.MongoInsertCommand)
	ginx.BindJsonAndValid(g, commandForm)

	conn, err := m.MongoApp.GetMongoConn(m.GetMongoId(g))
	biz.ErrIsNil(err)
	rc.ReqParam = collx.Kvs("mongo", conn.Info, "cmd", commandForm)

	res, err := conn.Cli.Database(commandForm.Database).Collection(commandForm.Collection).InsertOne(context.TODO(), commandForm.Doc)
	biz.ErrIsNilAppendErr(err, "命令执行失败: %s")
	rc.ResData = res
}

// 获取请求路径上的mongo id
func (m *Mongo) GetMongoId(g *gin.Context) uint64 {
	dbId, _ := strconv.Atoi(g.Param("id"))
	biz.IsTrue(dbId > 0, "mongoId错误")
	return uint64(dbId)
}
