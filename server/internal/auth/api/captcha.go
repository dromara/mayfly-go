package api

import (
	"mayfly-go/internal/auth/pkg/captcha"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

func GenerateCaptcha(rc *req.Ctx) {
	id, image, err := captcha.Generate()
	biz.ErrIsNilAppendErr(err, "failed to generate the CAPTCHA: %s")
	rc.ResData = collx.M{"base64Captcha": image, "cid": id}
}
