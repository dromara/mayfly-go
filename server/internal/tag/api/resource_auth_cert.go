package api

import (
	"mayfly-go/internal/tag/api/form"
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"strings"

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

	var racs []*entity.ResourceAuthCert
	res, err := r.ResourceAuthCertApp.PageQuery(cond, rc.GetPageParam(), &racs)
	biz.ErrIsNil(err)
	for _, rac := range racs {
		rac.CiphertextClear()
	}
	rc.ResData = res
}

func (r *ResourceAuthCert) GetCompleteAuthCert(rc *req.Ctx) {
	acName := rc.Query("name")
	biz.NotEmpty(acName, "授权凭证名不能为空")
	res := &entity.ResourceAuthCert{Name: acName}
	err := r.ResourceAuthCertApp.GetBy(res)
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

	biz.ErrIsNil(c.ResourceAuthCertApp.SavePulbicAuthCert(rc.MetaCtx, ac))
}

func (c *ResourceAuthCert) Delete(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	ids := strings.Split(idsStr, ",")

	acIds := make([]uint64, 0)
	acNames := make([]string, 0)
	for _, v := range ids {
		id := cast.ToUint64(v)
		rac, err := c.ResourceAuthCertApp.GetById(new(entity.ResourceAuthCert), id)
		biz.ErrIsNil(err, "存在错误授权凭证id")
		biz.IsTrue(rac.Type == entity.AuthCertTypePublic, "只允许删除公共授权凭证")
		biz.IsTrue(c.ResourceAuthCertApp.CountByCond(&entity.ResourceAuthCert{Ciphertext: rac.Name}) == 0, "[%s]该授权凭证已被关联", rac.Name)
		acIds = append(acIds, id)
		acNames = append(acNames, rac.Name)
	}

	rc.ReqParam = acNames
	biz.ErrIsNil(c.ResourceAuthCertApp.DeleteByWheres(rc.MetaCtx, collx.M{"id in ?": acIds}))
}
