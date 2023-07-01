package api

import (
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils"
	"strconv"
	"strings"
)

type AuthCert struct {
	AuthCertApp application.AuthCert
}

func (ac *AuthCert) BaseAuthCerts(rc *req.Ctx) {
	g := rc.GinCtx
	condition := &entity.AuthCert{
		Name:       g.Query("name"),
		AuthMethod: int8(ginx.QueryInt(g, "authMethod", 0)),
	}
	condition.Id = uint64(ginx.QueryInt(g, "id", 0))
	rc.ResData = ac.AuthCertApp.GetPageList(condition, ginx.GetPageParam(g), new([]vo.AuthCertBaseVO))
}

func (ac *AuthCert) AuthCerts(rc *req.Ctx) {
	g := rc.GinCtx
	condition := &entity.AuthCert{
		Name:       g.Query("name"),
		AuthMethod: int8(ginx.QueryInt(g, "authMethod", 0)),
	}
	condition.Id = uint64(ginx.QueryInt(g, "id", 0))

	res := new([]*entity.AuthCert)
	pageRes := ac.AuthCertApp.GetPageList(condition, ginx.GetPageParam(g), res)
	for _, r := range *res {
		r.PwdDecrypt()
	}
	rc.ResData = pageRes
}

func (c *AuthCert) SaveAuthCert(rc *req.Ctx) {
	g := rc.GinCtx
	acForm := &form.AuthCertForm{}
	ginx.BindJsonAndValid(g, acForm)

	ac := new(entity.AuthCert)
	utils.Copy(ac, acForm)

	// 脱敏记录日志
	acForm.Passphrase = "***"
	acForm.Password = "***"
	rc.ReqParam = acForm

	ac.SetBaseInfo(rc.LoginAccount)
	c.AuthCertApp.Save(ac)
}

func (c *AuthCert) Delete(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		c.AuthCertApp.DeleteById(uint64(value))
	}
}
