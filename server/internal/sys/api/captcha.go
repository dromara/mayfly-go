package api

import (
	"mayfly-go/pkg/captcha"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

func GenerateCaptcha(rc *req.Ctx) {
	id, image := captcha.Generate()
	rc.ResData = collx.M{"base64Captcha": image, "cid": id}
}
