package utils

import (
	"encoding/base64"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/cryptox"
	"os"
)

const (
	// 公钥文件路径
	publicKeyFile = "./mayfly_rsa.pub"
	// 私钥文件路径
	privateKeyFile = "./mayfly_rsa"

	publicKeyK  = "mayfly:public-key"
	privateKeyK = "mayfly:private-key"
)

// 使用系统默认的私钥解密
//  -  base64 字符串是否使用base64编码
func DefaultRsaDecrypt(data string, useBase64 bool) (string, error) {
	// 空字符串不解密
	if data == "" {
		return "", nil
	}
	if useBase64 {
		if decodeBase64, err := base64.StdEncoding.DecodeString(data); err != nil {
			return "", err
		} else {
			data = string(decodeBase64)
		}
	}
	priKey, err := GetRsaPrivateKey()
	if err != nil {
		return "", err
	}
	val, err := cryptox.RsaDecrypt(priKey, []byte(data))
	if err != nil {
		return "", err
	}
	return string(val), nil
}

// 获取系统的RSA公钥
func GetRsaPublicKey() (string, error) {
	if cache.UseRedisCache() {
		publicKey := cache.GetStr(publicKeyK)
		if publicKey != "" {
			return publicKey, nil
		}
	} else {
		content, err := os.ReadFile(publicKeyFile)
		if err != nil {
			publicKey := cache.GetStr(publicKeyK)
			if publicKey != "" {
				return publicKey, nil
			}
		} else {
			return string(content), nil
		}
	}

	_, pubKey, err := GenerateAndSaveRSAKey()
	return pubKey, err
}

// 获取系统私钥
func GetRsaPrivateKey() (string, error) {
	if cache.UseRedisCache() {
		priKey := cache.GetStr(privateKeyK)
		if priKey != "" {
			return priKey, nil
		}
	} else {
		content, err := os.ReadFile(privateKeyFile)
		if err != nil {
			priKey := cache.GetStr(privateKeyK)
			if priKey != "" {
				return priKey, nil
			}
		} else {
			return string(content), nil
		}
	}

	priKey, _, err := GenerateAndSaveRSAKey()
	return priKey, err
}

// 生成并保存rsa key，优先保存于磁盘，若磁盘保存失败，则保存至缓存
//
// 依次返回 privateKey, publicKey, error
func GenerateAndSaveRSAKey() (string, string, error) {
	privateKey, publicKey, err := cryptox.GenerateRSAKey(1024)
	if err != nil {
		return "", "", err
	}

	// 如果使用了redis缓存，则优先存入redis
	if cache.UseRedisCache() {
		logx.Debug("系统配置了redis, rsa存入redis")
		cache.Set(privateKeyK, privateKey, -1)
		cache.Set(publicKeyK, publicKey, -1)
		return privateKey, publicKey, nil
	}

	err = os.WriteFile(privateKeyFile, []byte(privateKey), 0644)
	if err != nil {
		logx.ErrorTrace("RSA私钥写入磁盘文件失败, 使用缓存存储该私钥", err)
		cache.Set(privateKeyK, privateKey, -1)
	}

	err = os.WriteFile(publicKeyFile, []byte(publicKey), 0644)
	if err != nil {
		logx.ErrorTrace("RSA公钥写入磁盘文件失败, 使用缓存存储该公钥", err)
		cache.Set(publicKeyK, publicKey, -1)
	}

	return privateKey, publicKey, nil
}

func AesDecryptByLa(data string, la *model.LoginAccount) (string, error) {
	key := []byte(la.GetAesKey())
	res, err := cryptox.AesDecryptBase64(data, key)
	return string(res), err
}
