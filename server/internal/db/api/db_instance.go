package api

import (
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/cryptox"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Instance struct {
	InstanceApp application.Instance `inject:"DbInstanceApp"`
	DbApp       application.Db       `inject:""`
}

// Instances 获取数据库实例信息
// @router /api/instances [get]
func (d *Instance) Instances(rc *req.Ctx) {
	queryCond, page := ginx.BindQueryAndPage[*entity.InstanceQuery](rc.GinCtx, new(entity.InstanceQuery))
	res, err := d.InstanceApp.GetPageList(queryCond, page, new([]vo.InstanceListVO))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (d *Instance) TestConn(rc *req.Ctx) {
	form := &form.InstanceForm{}
	instance := ginx.BindJsonAndCopyTo[*entity.DbInstance](rc.GinCtx, form, new(entity.DbInstance))

	// 密码解密，并使用解密后的赋值
	originPwd, err := cryptox.DefaultRsaDecrypt(form.Password, true)
	biz.ErrIsNilAppendErr(err, "解密密码错误: %s")
	instance.Password = originPwd

	biz.ErrIsNil(d.InstanceApp.TestConn(instance))
}

// SaveInstance 保存数据库实例信息
// @router /api/instances [post]
func (d *Instance) SaveInstance(rc *req.Ctx) {
	form := &form.InstanceForm{}
	instance := ginx.BindJsonAndCopyTo[*entity.DbInstance](rc.GinCtx, form, new(entity.DbInstance))

	// 密码解密，并使用解密后的赋值
	originPwd, err := cryptox.DefaultRsaDecrypt(form.Password, true)
	biz.ErrIsNilAppendErr(err, "解密密码错误: %s")
	instance.Password = originPwd

	// 密码脱敏记录日志
	form.Password = "****"
	rc.ReqParam = form
	biz.ErrIsNil(d.InstanceApp.Save(rc.MetaCtx, instance))
}

// GetInstance 获取数据库实例密码，由于数据库是加密存储，故提供该接口展示原文密码
// @router /api/instances/:instance [GET]
func (d *Instance) GetInstance(rc *req.Ctx) {
	dbId := getInstanceId(rc.GinCtx)
	dbEntity, err := d.InstanceApp.GetById(new(entity.DbInstance), dbId)
	biz.ErrIsNil(err, "获取数据库实例错误")
	dbEntity.Password = ""
	rc.ResData = dbEntity
}

// GetInstancePwd 获取数据库实例密码，由于数据库是加密存储，故提供该接口展示原文密码
// @router /api/instances/:instance/pwd [GET]
func (d *Instance) GetInstancePwd(rc *req.Ctx) {
	instanceId := getInstanceId(rc.GinCtx)
	instanceEntity, err := d.InstanceApp.GetById(new(entity.DbInstance), instanceId, "Password")
	biz.ErrIsNil(err, "获取数据库实例错误")
	biz.ErrIsNil(instanceEntity.PwdDecrypt())
	rc.ResData = instanceEntity.Password
}

// DeleteInstance 删除数据库实例信息
// @router /api/instances/:instance [DELETE]
func (d *Instance) DeleteInstance(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "instanceId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "删除数据库实例失败: %s")
		instanceId := uint64(value)
		err = d.InstanceApp.Delete(rc.MetaCtx, instanceId)
		biz.ErrIsNilAppendErr(err, "删除数据库实例失败: %s")
	}
}

// 获取数据库实例的所有数据库名
func (d *Instance) GetDatabaseNames(rc *req.Ctx) {
	instanceId := getInstanceId(rc.GinCtx)
	instance, err := d.InstanceApp.GetById(new(entity.DbInstance), instanceId, "Password")
	biz.ErrIsNil(err, "获取数据库实例错误")
	biz.ErrIsNil(instance.PwdDecrypt())
	res, err := d.InstanceApp.GetDatabases(instance)
	biz.ErrIsNil(err)
	rc.ResData = res
}

// 获取数据库实例server信息
func (d *Instance) GetDbServer(rc *req.Ctx) {
	instanceId := getInstanceId(rc.GinCtx)
	conn, err := d.DbApp.GetDbConnByInstanceId(instanceId)
	biz.ErrIsNil(err)
	res, err := conn.GetDialect().GetDbServer()
	biz.ErrIsNil(err)
	rc.ResData = res
}

func getInstanceId(g *gin.Context) uint64 {
	instanceId, _ := strconv.Atoi(g.Param("instanceId"))
	biz.IsTrue(instanceId > 0, "instanceId 错误")
	return uint64(instanceId)
}
