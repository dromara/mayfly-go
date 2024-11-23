package api

import (
	"context"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/event"
	"mayfly-go/internal/mongo/api/form"
	"mayfly-go/internal/mongo/api/vo"
	"mayfly-go/internal/mongo/application"
	"mayfly-go/internal/mongo/domain/entity"
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
	MongoApp application.Mongo `inject:""`
	TagApp   tagapp.TagTree    `inject:"TagTreeApp"`
}

func (m *Mongo) Mongos(rc *req.Ctx) {
	queryCond, page := req.BindQueryAndPage[*entity.MongoQuery](rc, new(entity.MongoQuery))

	// 不存在可访问标签id，即没有可操作数据
	tags := m.TagApp.GetAccountTags(rc.GetLoginAccount().Id, &tagentity.TagTreeQuery{
		Types:         []tagentity.TagType{tagentity.TagTypeMongo},
		CodePathLikes: []string{queryCond.TagPath},
	})
	if len(tags) == 0 {
		rc.ResData = model.EmptyPageResult[any]()
		return
	}
	queryCond.Codes = tags.GetCodes()

	var mongovos []*vo.Mongo
	res, err := m.MongoApp.GetPageList(queryCond, page, &mongovos)
	biz.ErrIsNil(err)

	// 填充标签信息
	m.TagApp.FillTagInfo(tagentity.TagType(consts.ResourceTypeMongo), collx.ArrayMap(mongovos, func(mvo *vo.Mongo) tagentity.ITagResource {
		return mvo
	})...)

	rc.ResData = res
}

func (m *Mongo) TestConn(rc *req.Ctx) {
	form := &form.Mongo{}
	mongo := req.BindJsonAndCopyTo[*entity.Mongo](rc, form, new(entity.Mongo))
	biz.ErrIsNilAppendErr(m.MongoApp.TestConn(mongo), "connection error: %s")
}

func (m *Mongo) Save(rc *req.Ctx) {
	form := &form.Mongo{}
	mongo := req.BindJsonAndCopyTo[*entity.Mongo](rc, form, new(entity.Mongo))

	// 密码脱敏记录日志
	form.Uri = func(str string) string {
		reg := regexp.MustCompile(`(^mongodb://.+?:)(.+)(@.+$)`)
		return reg.ReplaceAllString(str, `${1}****${3}`)
	}(form.Uri)
	rc.ReqParam = form

	biz.ErrIsNil(m.MongoApp.SaveMongo(rc.MetaCtx, mongo, form.TagCodePaths...))
}

func (m *Mongo) DeleteMongo(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		m.MongoApp.Delete(rc.MetaCtx, cast.ToUint64(v))
	}
}

func (m *Mongo) Databases(rc *req.Ctx) {
	conn, err := m.MongoApp.GetMongoConn(m.GetMongoId(rc))
	biz.ErrIsNil(err)
	res, err := conn.Cli.ListDatabases(context.TODO(), bson.D{})
	biz.ErrIsNilAppendErr(err, "get mongo dbs error: %s")
	rc.ResData = res
}

func (m *Mongo) Collections(rc *req.Ctx) {
	conn, err := m.MongoApp.GetMongoConn(m.GetMongoId(rc))
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
	commandForm := new(form.MongoRunCommand)
	req.BindJsonAndValid(rc, commandForm)

	conn, err := m.MongoApp.GetMongoConn(m.GetMongoId(rc))
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
	commandForm := req.BindJsonAndValid(rc, new(form.MongoFindCommand))

	conn, err := m.MongoApp.GetMongoConn(m.GetMongoId(rc))
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
	commandForm := req.BindJsonAndValid(rc, new(form.MongoUpdateByIdCommand))

	conn, err := m.MongoApp.GetMongoConn(m.GetMongoId(rc))
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
	commandForm := req.BindJsonAndValid(rc, new(form.MongoUpdateByIdCommand))

	conn, err := m.MongoApp.GetMongoConn(m.GetMongoId(rc))
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
	commandForm := req.BindJsonAndValid(rc, new(form.MongoInsertCommand))

	conn, err := m.MongoApp.GetMongoConn(m.GetMongoId(rc))
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
