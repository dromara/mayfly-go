package utils

import (
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/config"
	"regexp"
)

// 检查用户密码安全等级
func CheckAccountPasswordLever(ps string) bool {
	if len(ps) < 8 {
		return false
	}
	num := `[0-9]{1}`
	a_z := `[a-zA-Z]{1}`
	symbol := `[!@#~$%^&*()+|_.,]{1}`
	if b, err := regexp.MatchString(num, ps); !b || err != nil {
		return false
	}
	if b, err := regexp.MatchString(a_z, ps); !b || err != nil {
		return false
	}
	if b, err := regexp.MatchString(symbol, ps); !b || err != nil {
		return false
	}
	return true
}

// 使用config.yml的aes.key进行密码加密
func PwdAesEncrypt(password string) string {
	if password == "" {
		return ""
	}
	aes := config.Conf.Aes
	if aes == nil {
		return password
	}
	encryptPwd, err := aes.EncryptBase64([]byte(password))
	biz.ErrIsNilAppendErr(err, "密码加密失败: %s")
	return encryptPwd
}

// 使用config.yml的aes.key进行密码解密
func PwdAesDecrypt(encryptPwd string) string {
	if encryptPwd == "" {
		return ""
	}
	aes := config.Conf.Aes
	if aes == nil {
		return encryptPwd
	}
	decryptPwd, err := aes.DecryptBase64(encryptPwd)
	biz.ErrIsNilAppendErr(err, "密码解密失败: %s")
	// 解密后的密码
	return string(decryptPwd)
}
