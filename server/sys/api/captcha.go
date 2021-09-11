package api

import (
	"mayfly-go/base/captcha"
	"mayfly-go/base/ctx"
)

func GenerateCaptcha(rc *ctx.ReqCtx) {
	id, image := captcha.Generate()
	rc.ResData = map[string]interface{}{"base64Captcha": image, "cid": id}
}
