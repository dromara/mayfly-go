package api

import (
	"mayfly-go/internal/pkg/utils"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

type Common struct {
}

func (c *Common) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 获取公钥
		req.NewGet("/public-key", c.RasPublicKey).DontNeedToken(),
	}

	return req.NewConfs("/common", reqs[:]...)
}

func (i *Common) RasPublicKey(rc *req.Ctx) {
	publicKeyStr, err := utils.GetRsaPublicKey()
	biz.ErrIsNilAppendErr(err, "rsa - failed to genenrate public key")
	rc.ResData = publicKeyStr
}
