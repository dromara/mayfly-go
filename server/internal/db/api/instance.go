package api

import (
	"github.com/gin-gonic/gin"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	msgapp "mayfly-go/internal/msg/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/cryptox"
	"strconv"
	"strings"
)

type Instance struct {
	InstanceApp application.Instance
	MsgApp      msgapp.Msg
}

// @router /api/instances [get]
func (d *Instance) Instances(rc *req.Ctx) {
	queryCond, page := ginx.BindQueryAndPage[*entity.InstanceQuery](rc.GinCtx, new(entity.InstanceQuery))
	rc.ResData = d.InstanceApp.GetPageList(queryCond, page, new([]vo.SelectDataInstanceVO))
}

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

// 获取数据库实例密码，由于数据库是加密存储，故提供该接口展示原文密码
func (d *Instance) GetInstancePwd(rc *req.Ctx) {
	dbId := GetInstanceId(rc.GinCtx)
	dbEntity := d.InstanceApp.GetById(dbId, "Password")
	dbEntity.PwdDecrypt()
	rc.ResData = dbEntity.Password
}

func (d *Instance) DeleteInstance(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "dbId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		dbId := uint64(value)
		d.InstanceApp.Delete(dbId)
	}
}

func GetInstanceId(g *gin.Context) uint64 {
	dbId, _ := strconv.Atoi(g.Param("dbId"))
	biz.IsTrue(dbId > 0, "dbId错误")
	return uint64(dbId)
}
