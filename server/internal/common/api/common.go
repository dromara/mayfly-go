package api

import (
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/cryptox"
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
	publicKeyStr, err := cryptox.GetRsaPublicKey()
	biz.ErrIsNilAppendErr(err, "rsa - failed to genenrate public key")
	rc.ResData = publicKeyStr
}
