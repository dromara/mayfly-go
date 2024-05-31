package api

import (
	"mayfly-go/internal/tag/api/form"
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"

	"github.com/may-fly/cast"
)

type ResourceAuthCert struct {
	ResourceAuthCertApp application.ResourceAuthCert `inject:""`
}

func (r *ResourceAuthCert) ListByQuery(rc *req.Ctx) {
	cond := new(entity.ResourceAuthCert)
	cond.ResourceCode = rc.Query("resourceCode")
	cond.ResourceType = int8(rc.QueryInt("resourceType"))
	cond.Type = entity.AuthCertType(rc.QueryInt("type"))
	cond.CiphertextType = entity.AuthCertCiphertextType(rc.QueryInt("ciphertextType"))
	cond.Name = rc.Query("name")

	res, err := r.ResourceAuthCertApp.PageByCond(cond, rc.GetPageParam())
	biz.ErrIsNil(err)
	for _, rac := range res.List {
		rac.CiphertextClear()
	}
	rc.ResData = res
}

func (r *ResourceAuthCert) GetCompleteAuthCert(rc *req.Ctx) {
	acName := rc.Query("name")
	biz.NotEmpty(acName, "授权凭证名不能为空")
	rc.ReqParam = acName

	res := &entity.ResourceAuthCert{Name: acName}
	err := r.ResourceAuthCertApp.GetByCond(res)
	biz.ErrIsNil(err)
	res.CiphertextDecrypt()
	rc.ResData = res
}

func (c *ResourceAuthCert) SaveAuthCert(rc *req.Ctx) {
	acForm := &form.AuthCertForm{}
	ac := req.BindJsonAndCopyTo(rc, acForm, new(entity.ResourceAuthCert))

	// 脱敏记录日志
	acForm.Ciphertext = "***"
	rc.ReqParam = acForm

	biz.ErrIsNil(c.ResourceAuthCertApp.SaveAuthCert(rc.MetaCtx, ac))
}

func (c *ResourceAuthCert) Delete(rc *req.Ctx) {
	id := rc.PathParamInt("id")
	rc.ReqParam = id
	biz.ErrIsNil(c.ResourceAuthCertApp.DeleteAuthCert(rc.MetaCtx, cast.ToUint64(id)))
}
