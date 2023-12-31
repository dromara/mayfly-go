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
