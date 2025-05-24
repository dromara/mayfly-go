package api

import (
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/internal/pkg/consts"

	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/may-fly/cast"
)

type Instance struct {
	instanceApp         application.Instance    `inject:"T"`
	dbApp               application.Db          `inject:"T"`
	resourceAuthCertApp tagapp.ResourceAuthCert `inject:"T"`
	tagApp              tagapp.TagTree          `inject:"T"`
}

func (d *Instance) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 获取数据库列表
		req.NewGet("", d.Instances),

		req.NewPost("/test-conn", d.TestConn),

		req.NewPost("", d.SaveInstance).Log(req.NewLogSaveI(imsg.LogDbInstSave)),

		req.NewGet(":instanceId", d.GetInstance),

		// 获取数据库实例的所有数据库名
		req.NewPost("/databases", d.GetDatabaseNames),

		// 根据授权凭证名获取其所有库名
		req.NewGet("/databases/:ac", d.GetDatabaseNamesByAc),

		req.NewGet(":instanceId/server-info", d.GetDbServer),

		req.NewDelete(":instanceId", d.DeleteInstance).Log(req.NewLogSaveI(imsg.LogDbInstDelete)),
	}

	return req.NewConfs("/instances", reqs[:]...)
}

// Instances 获取数据库实例信息
// @router /api/instances [get]
func (d *Instance) Instances(rc *req.Ctx) {
	queryCond := req.BindQuery[*entity.InstanceQuery](rc)

	tags := d.tagApp.GetAccountTags(rc.GetLoginAccount().Id, &tagentity.TagTreeQuery{
		TypePaths:     collx.AsArray(tagentity.NewTypePaths(tagentity.TagTypeDbInstance, tagentity.TagTypeAuthCert)),
		CodePathLikes: collx.AsArray(queryCond.TagPath),
	})
	// 不存在可操作的数据库，即没有可操作数据
	if len(tags) == 0 {
		rc.ResData = model.NewEmptyPageResult[any]()
		return
	}

	tagCodePaths := tags.GetCodePaths()
	dbInstCodes := tagentity.GetCodesByCodePaths(tagentity.TagTypeDbInstance, tagCodePaths...)
	queryCond.Codes = dbInstCodes

	res, err := d.instanceApp.GetPageList(queryCond)
	biz.ErrIsNil(err)
	resVo := model.PageResultConv[*entity.DbInstance, *vo.InstanceListVO](res)
	instvos := resVo.List

	// 填充授权凭证信息
	d.resourceAuthCertApp.FillAuthCertByAcNames(tagentity.GetCodesByCodePaths(tagentity.TagTypeAuthCert, tagCodePaths...), collx.ArrayMap(instvos, func(vos *vo.InstanceListVO) tagentity.IAuthCert {
		return vos
	})...)

	// 填充标签信息
	d.tagApp.FillTagInfo(tagentity.TagType(consts.ResourceTypeDbInstance), collx.ArrayMap(instvos, func(insvo *vo.InstanceListVO) tagentity.ITagResource {
		return insvo
	})...)

	rc.ResData = resVo
}

func (d *Instance) TestConn(rc *req.Ctx) {
	form, instance := req.BindJsonAndCopyTo[*form.InstanceForm, *entity.DbInstance](rc)
	biz.ErrIsNil(d.instanceApp.TestConn(rc.MetaCtx, instance, form.AuthCerts[0]))
}

// SaveInstance 保存数据库实例信息
// @router /api/instances [post]
func (d *Instance) SaveInstance(rc *req.Ctx) {
	form, instance := req.BindJsonAndCopyTo[*form.InstanceForm, *entity.DbInstance](rc)

	rc.ReqParam = form
	id, err := d.instanceApp.SaveDbInstance(rc.MetaCtx, &dto.SaveDbInstance{
		DbInstance:   instance,
		AuthCerts:    form.AuthCerts,
		TagCodePaths: form.TagCodePaths,
	})
	biz.ErrIsNil(err)
	rc.ResData = id
}

// GetInstance 获取数据库实例密码，由于数据库是加密存储，故提供该接口展示原文密码
// @router /api/instances/:instance [GET]
func (d *Instance) GetInstance(rc *req.Ctx) {
	dbId := getInstanceId(rc)
	dbEntity, err := d.instanceApp.GetById(dbId)
	biz.ErrIsNilAppendErr(err, "get db instance failed: %s")
	rc.ResData = dbEntity
}

// DeleteInstance 删除数据库实例信息
// @router /api/instances/:instance [DELETE]
func (d *Instance) DeleteInstance(rc *req.Ctx) {
	idsStr := rc.PathParam("instanceId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		biz.ErrIsNilAppendErr(d.instanceApp.Delete(rc.MetaCtx, cast.ToUint64(v)), "delete db instance failed: %s")
	}
}

// 获取数据库实例的所有数据库名
func (d *Instance) GetDatabaseNames(rc *req.Ctx) {
	form, instance := req.BindJsonAndCopyTo[*form.InstanceDbNamesForm, *entity.DbInstance](rc)
	res, err := d.instanceApp.GetDatabases(rc.MetaCtx, instance, form.AuthCert)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (d *Instance) GetDatabaseNamesByAc(rc *req.Ctx) {
	res, err := d.instanceApp.GetDatabasesByAc(rc.MetaCtx, rc.PathParam("ac"))
	biz.ErrIsNil(err)
	rc.ResData = res
}

// 获取数据库实例server信息
func (d *Instance) GetDbServer(rc *req.Ctx) {
	instanceId := getInstanceId(rc)
	conn, err := d.dbApp.GetDbConnByInstanceId(rc.MetaCtx, instanceId)
	biz.ErrIsNil(err)
	res, err := conn.GetMetadata().GetDbServer()
	biz.ErrIsNil(err)
	rc.ResData = res
}

func getInstanceId(rc *req.Ctx) uint64 {
	instanceId := rc.PathParamInt("instanceId")
	biz.IsTrue(instanceId > 0, "instanceId error")
	return uint64(instanceId)
}
