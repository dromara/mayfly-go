package utils

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
)

// md5
func Md5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// 系统统一RSA秘钥对
var RsaPair []string

// 生成RSA私钥和公钥字符串
// bits 证书大小
// @return privateKeyStr publicKeyStr error
func GenerateRSAKey(bits int) (string, string, error) {
	var privateKeyStr, publicKeyStr string

	//GenerateKey函数使用随机数据生成器random生成一对具有指定字位数的RSA密钥
	//Reader是一个全局、共享的密码用强随机数生成器
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return privateKeyStr, publicKeyStr, err
	}
	//保存私钥
	//通过x509标准将得到的ras私钥序列化为ASN.1 的 DER编码字符串
	X509PrivateKey := x509.MarshalPKCS1PrivateKey(privateKey)
	//构建一个pem.Block结构体对象
	privateBlock := pem.Block{Type: "RSA Private Key", Bytes: X509PrivateKey}

	privateBuf := new(bytes.Buffer)
	pem.Encode(privateBuf, &privateBlock)
	privateKeyStr = privateBuf.String()

	//保存公钥
	//获取公钥的数据
	publicKey := privateKey.PublicKey
	//X509对公钥编码
	X509PublicKey, err := x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		return publicKeyStr, privateKeyStr, err
	}
	//创建一个pem.Block结构体对象
	publicBlock := pem.Block{Type: "RSA Public Key", Bytes: X509PublicKey}

	publicBuf := new(bytes.Buffer)
	pem.Encode(publicBuf, &publicBlock)
	publicKeyStr = publicBuf.String()

	return privateKeyStr, publicKeyStr, nil
}

// rsa解密
func RsaDecrypt(privateKeyStr string, data []byte) ([]byte, error) {
	block, _ := pem.Decode([]byte(privateKeyStr))
	if block == nil {
		return nil, errors.New("private key error")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.DecryptPKCS1v15(rand.Reader, priv, data)
}

// 使用系统默认的私钥解密
// @param base64 字符串是否使用base64编码
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
	val, err := RsaDecrypt(priKey, []byte(data))
	if err != nil {
		return "", err
	}
	return string(val), nil
}

// 获取系统的RSA公钥
func GetRsaPublicKey() (string, error) {
	if len(RsaPair) == 2 {
		return RsaPair[1], nil
	}

	privateKey, publicKey, err := GenerateRSAKey(1024)
	if err != nil {
		return "", err
	}
	RsaPair = append(RsaPair, privateKey)
	RsaPair = append(RsaPair, publicKey)
	return publicKey, nil
}

// 获取系统私钥
func GetRsaPrivateKey() (string, error) {
	if len(RsaPair) == 2 {
		return RsaPair[0], nil
	}

	privateKey, publicKey, err := GenerateRSAKey(1024)
	if err != nil {
		return "", err
	}
	RsaPair = append(RsaPair, privateKey)
	RsaPair = append(RsaPair, publicKey)
	return privateKey, nil
}
