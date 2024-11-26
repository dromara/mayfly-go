package api

import (
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/domain/entity"

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
	InstanceApp         application.Instance    `inject:"DbInstanceApp"`
	DbApp               application.Db          `inject:""`
	ResourceAuthCertApp tagapp.ResourceAuthCert `inject:""`
	TagApp              tagapp.TagTree          `inject:"TagTreeApp"`
}

// Instances 获取数据库实例信息
// @router /api/instances [get]
func (d *Instance) Instances(rc *req.Ctx) {
	queryCond, page := req.BindQueryAndPage[*entity.InstanceQuery](rc, new(entity.InstanceQuery))

	tags := d.TagApp.GetAccountTags(rc.GetLoginAccount().Id, &tagentity.TagTreeQuery{
		TypePaths:     collx.AsArray(tagentity.NewTypePaths(tagentity.TagTypeDbInstance, tagentity.TagTypeAuthCert)),
		CodePathLikes: collx.AsArray(queryCond.TagPath),
	})
	// 不存在可操作的数据库，即没有可操作数据
	if len(tags) == 0 {
		rc.ResData = model.EmptyPageResult[any]()
		return
	}

	tagCodePaths := tags.GetCodePaths()
	dbInstCodes := tagentity.GetCodesByCodePaths(tagentity.TagTypeDbInstance, tagCodePaths...)
	queryCond.Codes = dbInstCodes

	var instvos []*vo.InstanceListVO
	res, err := d.InstanceApp.GetPageList(queryCond, page, &instvos)
	biz.ErrIsNil(err)

	// 填充授权凭证信息
	d.ResourceAuthCertApp.FillAuthCertByAcNames(tagentity.GetCodesByCodePaths(tagentity.TagTypeAuthCert, tagCodePaths...), collx.ArrayMap(instvos, func(vos *vo.InstanceListVO) tagentity.IAuthCert {
		return vos
	})...)

	// 填充标签信息
	d.TagApp.FillTagInfo(tagentity.TagType(consts.ResourceTypeDbInstance), collx.ArrayMap(instvos, func(insvo *vo.InstanceListVO) tagentity.ITagResource {
		return insvo
	})...)

	rc.ResData = res
}

func (d *Instance) TestConn(rc *req.Ctx) {
	form := &form.InstanceForm{}
	instance := req.BindJsonAndCopyTo[*entity.DbInstance](rc, form, new(entity.DbInstance))

	biz.ErrIsNil(d.InstanceApp.TestConn(instance, form.AuthCerts[0]))
}

// SaveInstance 保存数据库实例信息
// @router /api/instances [post]
func (d *Instance) SaveInstance(rc *req.Ctx) {
	form := &form.InstanceForm{}
	instance := req.BindJsonAndCopyTo[*entity.DbInstance](rc, form, new(entity.DbInstance))

	rc.ReqParam = form
	id, err := d.InstanceApp.SaveDbInstance(rc.MetaCtx, &dto.SaveDbInstance{
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
	dbEntity, err := d.InstanceApp.GetById(dbId)
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
		biz.ErrIsNilAppendErr(d.InstanceApp.Delete(rc.MetaCtx, cast.ToUint64(v)), "delete db instance failed: %s")
	}
}

// 获取数据库实例的所有数据库名
func (d *Instance) GetDatabaseNames(rc *req.Ctx) {
	form := &form.InstanceDbNamesForm{}
	instance := req.BindJsonAndCopyTo[*entity.DbInstance](rc, form, new(entity.DbInstance))
	res, err := d.InstanceApp.GetDatabases(instance, form.AuthCert)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (d *Instance) GetDatabaseNamesByAc(rc *req.Ctx) {
	res, err := d.InstanceApp.GetDatabasesByAc(rc.PathParam("ac"))
	biz.ErrIsNil(err)
	rc.ResData = res
}

// 获取数据库实例server信息
func (d *Instance) GetDbServer(rc *req.Ctx) {
	instanceId := getInstanceId(rc)
	conn, err := d.DbApp.GetDbConnByInstanceId(instanceId)
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
