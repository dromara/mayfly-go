package captcha

import (
	"mayfly-go/base/biz"

	"github.com/mojocn/base64Captcha"
)

var store = base64Captcha.DefaultMemStore
var driver base64Captcha.Driver = base64Captcha.DefaultDriverDigit

// 生成验证码
func Generate() (string, string) {
	c := base64Captcha.NewCaptcha(driver, store)
	// 获取
	id, b64s, err := c.Generate()
	biz.ErrIsNilAppendErr(err, "获取验证码错误: %s")
	return id, b64s
}

// 验证验证码
func Verify(id string, val string) bool {
	if id == "" || val == "" {
		return false
	}
	// 同时清理掉这个图片
	return store.Verify(id, val, true)
}
