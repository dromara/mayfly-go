package api

import (
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ctx"
	"mayfly-go/pkg/utils"
)

type Common struct {
}

func (i *Common) RasPublicKey(rc *ctx.ReqCtx) {
	publicKeyStr, err := utils.GetRsaPublicKey()
	biz.ErrIsNilAppendErr(err, "rsa生成公私钥失败")
	rc.ResData = publicKeyStr
}
