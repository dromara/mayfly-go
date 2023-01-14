package api

import (
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils"
)

type Common struct {
}

func (i *Common) RasPublicKey(rc *req.Ctx) {
	publicKeyStr, err := utils.GetRsaPublicKey()
	biz.ErrIsNilAppendErr(err, "rsa生成公私钥失败")
	rc.ResData = publicKeyStr
}
