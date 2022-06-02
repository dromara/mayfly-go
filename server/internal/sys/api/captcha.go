package api

import (
	"mayfly-go/pkg/captcha"
	"mayfly-go/pkg/ctx"
)

func GenerateCaptcha(rc *ctx.ReqCtx) {
	id, image := captcha.Generate()
	rc.ResData = map[string]interface{}{"base64Captcha": image, "cid": id}
}
