package api

import (
	"context"
	"mayfly-go/internal/event"
	"mayfly-go/internal/mongo/api/form"
	"mayfly-go/internal/mongo/api/vo"
	"mayfly-go/internal/mongo/application"
	"mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/internal/mongo/imsg"
	"mayfly-go/internal/pkg/consts"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"regexp"
	"strings"

	"github.com/may-fly/cast"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mongo struct {
	mongoApp   application.Mongo `inject:"T"`
	tagTreeApp tagapp.TagTree    `inject:"T"`
}

func (ma *Mongo) ReqConfs() *req.Confs {
	saveDataPerm := req.NewPermission("mongo:data:save")

	reqs := [...]*req.Conf{
		// 获取所有mongo列表
		req.NewGet("", ma.Mongos),

		req.NewPost("/test-conn", ma.TestConn),

		req.NewPost("", ma.Save).Log(req.NewLogSaveI(imsg.LogMongoSave)),

		req.NewDelete(":id", ma.DeleteMongo).Log(req.NewLogSaveI(imsg.LogMongoDelete)),

		// 获取mongo下的所有数据库
		req.NewGet(":id/databases", ma.Databases),

		// 获取mongo指定库的所有集合
		req.NewGet(":id/collections", ma.Collections),

		// mongo runCommand
		req.NewPost(":id/run-command", ma.RunCommand).Log(req.NewLogSaveI(imsg.LogMongoRunCmd)),

		// 执行mongo find命令
		req.NewPost(":id/command/find", ma.FindCommand),

		req.NewPost(":id/command/update-by-id", ma.UpdateByIdCommand).RequiredPermission(saveDataPerm).Log(req.NewLogSaveI(imsg.LogUpdateDocs)),

		// 执行mongo delete by id命令
		req.NewPost(":id/command/delete-by-id", ma.DeleteByIdCommand).RequiredPermission(req.NewPermission("mongo:data:del")).Log(req.NewLogSaveI(imsg.LogDelDocs)),

		// 执行mongo insert 命令
		req.NewPost(":id/command/insert", ma.InsertOneCommand).RequiredPermission(saveDataPerm).Log(req.NewLogSaveI(imsg.LogInsertDocs)),
	}

	return req.NewConfs("/mongos", reqs[:]...)
}

func (m *Mongo) Mongos(rc *req.Ctx) {
	queryCond := req.BindQuery[*entity.MongoQuery](rc)

	// 不存在可访问标签id，即没有可操作数据
	tags := m.tagTreeApp.GetAccountTags(rc.GetLoginAccount().Id, &tagentity.TagTreeQuery{
		TypePaths:     collx.AsArray(tagentity.NewTypePaths(tagentity.TagTypeMongo)),
		CodePathLikes: []string{queryCond.TagPath},
	})
	if len(tags) == 0 {
		rc.ResData = model.NewEmptyPageResult[any]()
		return
	}
	queryCond.Codes = tags.GetCodes()

	res, err := m.mongoApp.GetPageList(queryCond)
	biz.ErrIsNil(err)
	resVo := model.PageResultConv[*entity.Mongo, *vo.Mongo](res)
	mongovos := resVo.List

	// 填充标签信息
	m.tagTreeApp.FillTagInfo(tagentity.TagType(consts.ResourceTypeMongo), collx.ArrayMap(mongovos, func(mvo *vo.Mongo) tagentity.ITagResource {
		return mvo
	})...)

	rc.ResData = resVo
}

func (m *Mongo) TestConn(rc *req.Ctx) {
	_, mongo := req.BindJsonAndCopyTo[*form.Mongo, *entity.Mongo](rc)
	biz.ErrIsNilAppendErr(m.mongoApp.TestConn(mongo), "connection error: %s")
}

func (m *Mongo) Save(rc *req.Ctx) {
	form, mongo := req.BindJsonAndCopyTo[*form.Mongo, *entity.Mongo](rc)

	// 密码脱敏记录日志
	form.Uri = func(str string) string {
		reg := regexp.MustCompile(`(^mongodb://.+?:)(.+)(@.+$)`)
		return reg.ReplaceAllString(str, `${1}****${3}`)
	}(form.Uri)
	rc.ReqParam = form

	biz.ErrIsNil(m.mongoApp.SaveMongo(rc.MetaCtx, mongo, form.TagCodePaths...))
}

func (m *Mongo) DeleteMongo(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		m.mongoApp.Delete(rc.MetaCtx, cast.ToUint64(v))
	}
}

func (m *Mongo) Databases(rc *req.Ctx) {
	conn, err := m.mongoApp.GetMongoConn(rc.MetaCtx, m.GetMongoId(rc))
	biz.ErrIsNil(err)

	res, err := conn.Cli.ListDatabases(context.TODO(), bson.D{})
	biz.ErrIsNilAppendErr(err, "get mongo dbs error: %s")
	rc.ResData = res
}

func (m *Mongo) Collections(rc *req.Ctx) {
	conn, err := m.mongoApp.GetMongoConn(rc.MetaCtx, m.GetMongoId(rc))
	biz.ErrIsNil(err)

	global.EventBus.Publish(rc.MetaCtx, event.EventTopicResourceOp, conn.Info.CodePath[0])

	db := rc.Query("database")
	biz.NotEmpty(db, "database cannot be empty")
	ctx := context.TODO()
	res, err := conn.Cli.Database(db).ListCollectionNames(ctx, bson.D{})
	biz.ErrIsNilAppendErr(err, "get db collections error: %s")
	rc.ResData = res
}

func (m *Mongo) RunCommand(rc *req.Ctx) {
	commandForm := req.BindJsonAndValid[*form.MongoRunCommand](rc)

	conn, err := m.mongoApp.GetMongoConn(rc.MetaCtx, m.GetMongoId(rc))
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

	biz.ErrIsNilAppendErr(err, "command execution failed: %s")
	rc.ResData = bm
}

func (m *Mongo) FindCommand(rc *req.Ctx) {
	commandForm := req.BindJsonAndValid[*form.MongoFindCommand](rc)

	conn, err := m.mongoApp.GetMongoConn(rc.MetaCtx, m.GetMongoId(rc))
	biz.ErrIsNil(err)

	cli := conn.Cli

	limit := commandForm.Limit
	if limit != 0 {
		biz.IsTrue(limit <= 100, "the limit cannot exceed 100")
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
	biz.ErrIsNilAppendErr(err, "command execution failed: %s")

	var res []bson.M
	cur.All(ctx, &res)
	rc.ResData = res
}

func (m *Mongo) UpdateByIdCommand(rc *req.Ctx) {
	commandForm := req.BindJsonAndValid[*form.MongoUpdateByIdCommand](rc)

	conn, err := m.mongoApp.GetMongoConn(rc.MetaCtx, m.GetMongoId(rc))
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
	biz.ErrIsNilAppendErr(err, "command execution failed: %s")

	rc.ResData = res
}

func (m *Mongo) DeleteByIdCommand(rc *req.Ctx) {
	commandForm := req.BindJsonAndValid[*form.MongoUpdateByIdCommand](rc)

	conn, err := m.mongoApp.GetMongoConn(rc.MetaCtx, m.GetMongoId(rc))
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
	biz.ErrIsNilAppendErr(err, "command execution failed: %s")
	rc.ResData = res
}

func (m *Mongo) InsertOneCommand(rc *req.Ctx) {
	commandForm := req.BindJsonAndValid[*form.MongoInsertCommand](rc)

	conn, err := m.mongoApp.GetMongoConn(rc.MetaCtx, m.GetMongoId(rc))
	biz.ErrIsNil(err)

	rc.ReqParam = collx.Kvs("mongo", conn.Info, "cmd", commandForm)

	res, err := conn.Cli.Database(commandForm.Database).Collection(commandForm.Collection).InsertOne(context.TODO(), commandForm.Doc)
	biz.ErrIsNilAppendErr(err, "command execution failed: %s")
	rc.ResData = res
}

// 获取请求路径上的mongo id
func (m *Mongo) GetMongoId(rc *req.Ctx) uint64 {
	dbId := rc.PathParamInt("id")
	biz.IsTrue(dbId > 0, "mongoId error")
	return uint64(dbId)
}
