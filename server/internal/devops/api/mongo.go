package api

import (
	"context"
	"mayfly-go/internal/devops/api/form"
	"mayfly-go/internal/devops/application"
	"mayfly-go/internal/devops/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ctx"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mongo struct {
	MongoApp application.Mongo
}

func (m *Mongo) Mongos(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	mc := &entity.Mongo{EnvId: uint64(ginx.QueryInt(g, "envId", 0)),
		ProjectId: uint64(ginx.QueryInt(g, "projectId", 0)),
	}
	mc.CreatorId = rc.LoginAccount.Id
	rc.ResData = m.MongoApp.GetPageList(mc, ginx.GetPageParam(rc.GinCtx), new([]entity.Mongo))
}

func (m *Mongo) Save(rc *ctx.ReqCtx) {
	form := &form.Mongo{}
	ginx.BindJsonAndValid(rc.GinCtx, form)

	rc.ReqParam = form

	mongo := new(entity.Mongo)
	utils.Copy(mongo, form)
	mongo.SetBaseInfo(rc.LoginAccount)
	m.MongoApp.Save(mongo)
}

func (m *Mongo) DeleteMongo(rc *ctx.ReqCtx) {
	m.MongoApp.Delete(m.GetMongoId(rc.GinCtx))
}

func (m *Mongo) Databases(rc *ctx.ReqCtx) {
	cli := m.MongoApp.GetMongoCli(m.GetMongoId(rc.GinCtx))
	res, err := cli.ListDatabases(context.TODO(), bson.D{})
	biz.ErrIsNilAppendErr(err, "获取mongo所有库信息失败: %s")
	rc.ResData = res
}

func (m *Mongo) Collections(rc *ctx.ReqCtx) {
	cli := m.MongoApp.GetMongoCli(m.GetMongoId(rc.GinCtx))
	db := rc.GinCtx.Query("database")
	biz.NotEmpty(db, "database不能为空")
	ctx := context.TODO()
	res, err := cli.Database(db).ListCollectionNames(ctx, bson.D{})
	biz.ErrIsNilAppendErr(err, "获取库集合信息失败: %s")
	rc.ResData = res
}

func (m *Mongo) RunCommand(rc *ctx.ReqCtx) {
	commandForm := new(form.MongoRunCommand)
	ginx.BindJsonAndValid(rc.GinCtx, commandForm)
	cli := m.MongoApp.GetMongoCli(m.GetMongoId(rc.GinCtx))

	ctx := context.TODO()
	var bm bson.M
	err := cli.Database(commandForm.Database).RunCommand(
		ctx,
		commandForm.Command,
	).Decode(&bm)

	biz.ErrIsNilAppendErr(err, "执行命令失败: %s")
	rc.ResData = bm
}

func (m *Mongo) FindCommand(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	cli := m.MongoApp.GetMongoCli(m.GetMongoId(g))
	commandForm := new(form.MongoFindCommand)
	ginx.BindJsonAndValid(g, commandForm)

	limit := commandForm.Limit
	if limit != 0 {
		biz.IsTrue(limit <= 100, "limit不能超过100")
	}
	opts := options.Find().SetSort(commandForm.Sort).
		SetSkip(commandForm.Skip).
		SetLimit(limit)
	ctx := context.TODO()
	cur, err := cli.Database(commandForm.Database).Collection(commandForm.Collection).Find(ctx, commandForm.Filter, opts)
	biz.ErrIsNilAppendErr(err, "命令执行失败: %s")

	var res []bson.M
	cur.All(ctx, &res)
	rc.ResData = res
}

func (m *Mongo) UpdateByIdCommand(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	cli := m.MongoApp.GetMongoCli(m.GetMongoId(g))
	commandForm := new(form.MongoUpdateByIdCommand)
	ginx.BindJsonAndValid(g, commandForm)

	// 解析docId文档id，如果为string类型则使用ObjectId解析，解析失败则为普通字符串
	docId := commandForm.DocId
	docIdVal, ok := docId.(string)
	if ok {
		objId, err := primitive.ObjectIDFromHex(docIdVal)
		if err == nil {
			docId = objId
		}
	}

	res, err := cli.Database(commandForm.Database).Collection(commandForm.Collection).UpdateByID(context.TODO(), docId, commandForm.Update)
	biz.ErrIsNilAppendErr(err, "命令执行失败: %s")

	rc.ReqParam = commandForm
	rc.ResData = res
}

func (m *Mongo) DeleteByIdCommand(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	cli := m.MongoApp.GetMongoCli(m.GetMongoId(g))
	commandForm := new(form.MongoUpdateByIdCommand)
	ginx.BindJsonAndValid(g, commandForm)

	// 解析docId文档id，如果为string类型则使用ObjectId解析，解析失败则为普通字符串
	docId := commandForm.DocId
	docIdVal, ok := docId.(string)
	if ok {
		objId, err := primitive.ObjectIDFromHex(docIdVal)
		if err == nil {
			docId = objId
		}
	}

	res, err := cli.Database(commandForm.Database).Collection(commandForm.Collection).DeleteOne(context.TODO(), bson.D{{"_id", docId}})
	biz.ErrIsNilAppendErr(err, "命令执行失败: %s")

	rc.ReqParam = commandForm
	rc.ResData = res
}

func (m *Mongo) InsertOneCommand(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	cli := m.MongoApp.GetMongoCli(m.GetMongoId(g))
	commandForm := new(form.MongoInsertCommand)
	ginx.BindJsonAndValid(g, commandForm)

	res, err := cli.Database(commandForm.Database).Collection(commandForm.Collection).InsertOne(context.TODO(), commandForm.Doc)
	biz.ErrIsNilAppendErr(err, "命令执行失败: %s")

	rc.ReqParam = commandForm
	rc.ResData = res
}

// 获取请求路径上的mongo id
func (m *Mongo) GetMongoId(g *gin.Context) uint64 {
	dbId, _ := strconv.Atoi(g.Param("id"))
	biz.IsTrue(dbId > 0, "mongoId错误")
	return uint64(dbId)
}
