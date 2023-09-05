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
	InstanceApp application.Instance
	DbApp       application.Db
}

// Instances 获取数据库实例信息
// @router /api/instances [get]
func (d *Instance) Instances(rc *req.Ctx) {
	queryCond, page := ginx.BindQueryAndPage[*entity.InstanceQuery](rc.GinCtx, new(entity.InstanceQuery))
	rc.ResData = d.InstanceApp.GetPageList(queryCond, page, new([]vo.SelectDataInstanceVO))
}

// SaveInstance 保存数据库实例信息
// @router /api/instances [post]
func (d *Instance) SaveInstance(rc *req.Ctx) {
	form := &form.InstanceForm{}
	instance := ginx.BindJsonAndCopyTo[*entity.Instance](rc.GinCtx, form, new(entity.Instance))

	// 密码解密，并使用解密后的赋值
	originPwd, err := cryptox.DefaultRsaDecrypt(form.Password, true)
	biz.ErrIsNilAppendErr(err, "解密密码错误: %s")
	instance.Password = originPwd

	// 密码脱敏记录日志
	form.Password = "****"
	rc.ReqParam = form

	instance.SetBaseInfo(rc.LoginAccount)
	d.InstanceApp.Save(instance)
}

// GetInstance 获取数据库实例密码，由于数据库是加密存储，故提供该接口展示原文密码
// @router /api/instances/:instance [GET]
func (d *Instance) GetInstance(rc *req.Ctx) {
	dbId := getInstanceId(rc.GinCtx)
	dbEntity := d.InstanceApp.GetById(dbId)
	biz.IsTrue(dbEntity != nil, "获取数据库实例错误")
	dbEntity.Password = ""
	rc.ResData = dbEntity
}

// GetInstancePwd 获取数据库实例密码，由于数据库是加密存储，故提供该接口展示原文密码
// @router /api/instances/:instance/pwd [GET]
func (d *Instance) GetInstancePwd(rc *req.Ctx) {
	instanceId := getInstanceId(rc.GinCtx)
	instanceEntity := d.InstanceApp.GetById(instanceId, "Password")
	biz.IsTrue(instanceEntity != nil, "获取数据库实例错误")
	instanceEntity.PwdDecrypt()
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
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		instanceId := uint64(value)
		if d.DbApp.Count(&entity.DbQuery{InstanceId: instanceId}) != 0 {
			instance := d.InstanceApp.GetById(instanceId, "name")
			biz.NotNil(instance, "获取数据库实例错误，数据库实例ID为：%d", instance.Id)
			biz.IsTrue(false, "不能删除数据库实例【%s】，请先删除其关联的数据库资源。", instance.Name)
		}
		d.InstanceApp.Delete(instanceId)
	}
}

// 获取数据库实例的所有数据库名
func (d *Instance) GetDatabaseNames(rc *req.Ctx) {
	instanceId := getInstanceId(rc.GinCtx)
	instance := d.InstanceApp.GetById(instanceId, "Password")
	biz.IsTrue(instance != nil, "获取数据库实例错误")
	instance.PwdDecrypt()
	rc.ResData = d.InstanceApp.GetDatabases(instance)
}

func getInstanceId(g *gin.Context) uint64 {
	instanceId, _ := strconv.Atoi(g.Param("instanceId"))
	biz.IsTrue(instanceId > 0, "instanceId 错误")
	return uint64(instanceId)
}
