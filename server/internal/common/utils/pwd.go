package utils

import (
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/config"
)

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
