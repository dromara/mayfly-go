package api

import (
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
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
		rac.CiphertextDecrypt()
	}
	rc.ResData = res
}
