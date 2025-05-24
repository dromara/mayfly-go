package api

import (
	"mayfly-go/internal/tag/api/form"
	"mayfly-go/internal/tag/api/vo"
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"strings"

	"github.com/may-fly/cast"
)

type ResourceAuthCert struct {
	resourceAuthCertApp application.ResourceAuthCert `inject:"T"`
}

func (r *ResourceAuthCert) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", r.ListByQuery),

		req.NewGet("/simple", r.SimpleAc),

		req.NewGet("/detail", r.GetCompleteAuthCert).Log(req.NewLogSaveI(imsg.LogAcShowPwd)).RequiredPermissionCode("authcert:showciphertext"),

		req.NewPost("", r.SaveAuthCert).Log(req.NewLogSaveI(imsg.LogAcSave)).RequiredPermissionCode("authcert:save"),

		req.NewDelete(":id", r.Delete).Log(req.NewLogSaveI(imsg.LogAcDelete)).RequiredPermissionCode("authcert:del"),
	}

	return req.NewConfs("/auth-certs", reqs[:]...)
}

func (r *ResourceAuthCert) ListByQuery(rc *req.Ctx) {
	cond := new(entity.ResourceAuthCert)
	cond.ResourceCode = rc.Query("resourceCode")
	cond.ResourceType = int8(rc.QueryInt("resourceType"))
	cond.Type = entity.AuthCertType(rc.QueryInt("type"))
	cond.CiphertextType = entity.AuthCertCiphertextType(rc.QueryInt("ciphertextType"))
	cond.Name = rc.Query("name")

	res, err := r.resourceAuthCertApp.PageByCond(cond, rc.GetPageParam())
	biz.ErrIsNil(err)
	for _, rac := range res.List {
		rac.CiphertextClear()
	}
	rc.ResData = res
}

func (m *ResourceAuthCert) SimpleAc(rc *req.Ctx) {
	acCodesStr := rc.Query("codes")
	biz.NotEmpty(acCodesStr, "codes不能为空")

	var vos []vo.SimpleResourceAuthCert
	m.resourceAuthCertApp.ListByCondToAny(model.NewCond().In("name", strings.Split(acCodesStr, ",")), &vos)
	rc.ResData = vos
}

func (r *ResourceAuthCert) GetCompleteAuthCert(rc *req.Ctx) {
	acName := rc.Query("name")
	biz.NotEmpty(acName, "授权凭证名不能为空")
	rc.ReqParam = acName

	res := &entity.ResourceAuthCert{Name: acName}
	err := r.resourceAuthCertApp.GetByCond(res)
	biz.ErrIsNil(err)
	res.CiphertextDecrypt()
	rc.ResData = res
}

func (c *ResourceAuthCert) SaveAuthCert(rc *req.Ctx) {
	acForm, ac := req.BindJsonAndCopyTo[*form.AuthCertForm, *entity.ResourceAuthCert](rc)

	// 脱敏记录日志
	acForm.Ciphertext = "***"
	rc.ReqParam = acForm

	biz.ErrIsNil(c.resourceAuthCertApp.SaveAuthCert(rc.MetaCtx, ac))
}

func (c *ResourceAuthCert) Delete(rc *req.Ctx) {
	id := rc.PathParamInt("id")
	rc.ReqParam = id
	biz.ErrIsNil(c.resourceAuthCertApp.DeleteAuthCert(rc.MetaCtx, cast.ToUint64(id)))
}
