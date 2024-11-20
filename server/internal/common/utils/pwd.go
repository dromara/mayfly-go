package utils

import (
	"mayfly-go/pkg/config"
	"regexp"
)

// 检查用户密码安全等级
func CheckAccountPasswordLever(ps string) bool {
	if len(ps) < 8 {
		return false
	}
	// 包含大写字母
	charRegex := regexp.MustCompile(`[a-zA-Z]`)
	// 包含数字
	digitRegex := regexp.MustCompile(`[0-9]`)
	// 包含特殊符号
	specialCharRegex := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`)

	return charRegex.MatchString(ps) &&
		digitRegex.MatchString(ps) &&
		specialCharRegex.MatchString(ps)
}

// 使用config.yml的aes.key进行密码加密
func PwdAesEncrypt(password string) (string, error) {
	if password == "" {
		return "", nil
	}
	aes := config.Conf.Aes
	if aes.Key == "" {
		return password, nil
	}
	encryptPwd, err := aes.EncryptBase64([]byte(password))
	if err != nil {
		return "", err
	}
	return encryptPwd, nil
}

// 使用config.yml的aes.key进行密码解密
func PwdAesDecrypt(encryptPwd string) (string, error) {
	if encryptPwd == "" {
		return "", nil
	}
	aes := config.Conf.Aes
	if aes.Key == "" {
		return encryptPwd, nil
	}
	decryptPwd, err := aes.DecryptBase64(encryptPwd)
	if err != nil {
		return "", err
	}
	// 解密后的密码
	return string(decryptPwd), nil
}
