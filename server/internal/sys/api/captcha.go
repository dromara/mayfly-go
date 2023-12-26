package api

import (
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/captcha"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

func GenerateCaptcha(rc *req.Ctx) {
	id, image, err := captcha.Generate()
	biz.ErrIsNilAppendErr(err, "获取验证码错误: %s")
	rc.ResData = collx.M{"base64Captcha": image, "cid": id}
}
