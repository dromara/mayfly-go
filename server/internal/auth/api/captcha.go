package api

import (
	"mayfly-go/internal/auth/pkg/captcha"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type Captcha struct {
}

func (c *Captcha) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", c.GenerateCaptcha).DontNeedToken(),
	}

	return req.NewConfs("/sys/captcha", reqs[:]...)
}

func (c *Captcha) GenerateCaptcha(rc *req.Ctx) {
	id, image, err := captcha.Generate()
	biz.ErrIsNilAppendErr(err, "failed to generate the CAPTCHA: %s")
	rc.ResData = collx.M{"base64Captcha": image, "cid": id}
}
