package captcha

import (
	"mayfly-go/pkg/rediscli"
	"time"

	"github.com/mojocn/base64Captcha"
)

var store base64Captcha.Store
var driver base64Captcha.Driver = base64Captcha.DefaultDriverDigit

// 生成验证码
func Generate() (string, string, error) {
	if store == nil {
		if rediscli.GetCli() != nil {
			store = new(RedisStore)
		} else {
			store = base64Captcha.DefaultMemStore
		}
	}

	c := base64Captcha.NewCaptcha(driver, store)
	// 获取
	id, b64s, _, err := c.Generate()
	return id, b64s, err
}

// 验证验证码
func Verify(id string, val string) bool {
	if store == nil || id == "" || val == "" {
		return false
	}
	// 同时清理掉这个图片
	return store.Verify(id, val, true)
}

type RedisStore struct {
}

const CAPTCHA = "mayfly:captcha:"

// 实现设置captcha的方法
func (r RedisStore) Set(id string, value string) error {
	//time.Minute*2：有效时间2分钟
	rediscli.Set(CAPTCHA+id, value, time.Minute*2)
	return nil
}

// 实现获取captcha的方法
func (r RedisStore) Get(id string, clear bool) string {
	key := CAPTCHA + id
	val, err := rediscli.Get(key)
	if err != nil {
		return ""
	}
	if clear {
		//clear为true，验证通过，删除这个验证码
		rediscli.Del(key)
	}
	return val
}

// 实现验证captcha的方法
func (r RedisStore) Verify(id, answer string, clear bool) bool {
	return r.Get(id, clear) == answer
}
