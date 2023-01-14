package api

import (
	"mayfly-go/pkg/captcha"
	"mayfly-go/pkg/req"
)

func GenerateCaptcha(rc *req.Ctx) {
	id, image := captcha.Generate()
	rc.ResData = map[string]interface{}{"base64Captcha": image, "cid": id}
}
