package api

import (
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"strconv"
	"strings"
)

type AuthCert struct {
	AuthCertApp application.AuthCert `inject:""`
}

func (ac *AuthCert) BaseAuthCerts(rc *req.Ctx) {
	queryCond, page := ginx.BindQueryAndPage(rc.GinCtx, new(entity.AuthCertQuery))
	res, err := ac.AuthCertApp.GetPageList(queryCond, page, new([]vo.AuthCertBaseVO))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (ac *AuthCert) AuthCerts(rc *req.Ctx) {
	queryCond, page := ginx.BindQueryAndPage(rc.GinCtx, new(entity.AuthCertQuery))

	res := new([]*entity.AuthCert)
	pageRes, err := ac.AuthCertApp.GetPageList(queryCond, page, res)
	biz.ErrIsNil(err)
	for _, r := range *res {
		r.PwdDecrypt()
	}
	rc.ResData = pageRes
}

func (c *AuthCert) SaveAuthCert(rc *req.Ctx) {
	acForm := &form.AuthCertForm{}
	ac := ginx.BindJsonAndCopyTo(rc.GinCtx, acForm, new(entity.AuthCert))

	// 脱敏记录日志
	acForm.Passphrase = "***"
	acForm.Password = "***"
	rc.ReqParam = acForm

	biz.ErrIsNil(c.AuthCertApp.Save(rc.MetaCtx, ac))
}

func (c *AuthCert) Delete(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		c.AuthCertApp.DeleteById(rc.MetaCtx, uint64(value))
	}
}
