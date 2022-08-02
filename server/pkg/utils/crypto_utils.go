package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// md5
func Md5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// bcrypt加密密码
func PwdHash(password string) string {
	bytes, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes)
}

// 检查密码是否一致
func CheckPwdHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
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

//AesEncrypt 加密
func AesEncrypt(data []byte, key []byte) ([]byte, error) {
	//创建加密实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//判断加密快的大小
	blockSize := block.BlockSize()
	//填充
	encryptBytes := pkcs7Padding(data, blockSize)
	//初始化加密数据接收切片
	crypted := make([]byte, len(encryptBytes))
	//使用cbc加密模式
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	//执行加密
	blockMode.CryptBlocks(crypted, encryptBytes)
	return crypted, nil
}

//AesDecrypt 解密
func AesDecrypt(data []byte, key []byte) ([]byte, error) {
	//创建实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//获取块的大小
	blockSize := block.BlockSize()
	//使用cbc
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	//初始化解密数据接收切片
	crypted := make([]byte, len(data))
	//执行解密
	blockMode.CryptBlocks(crypted, data)
	//去除填充
	crypted, err = pkcs7UnPadding(crypted)
	if err != nil {
		return nil, err
	}
	return crypted, nil
}

// aes加密 后 再base64
func AesEncryptBase64(data []byte, key []byte) (string, error) {
	res, err := AesEncrypt(data, key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(res), nil
}

// base64解码后再 aes解码
func AesDecryptBase64(data string, key []byte) ([]byte, error) {
	dataByte, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	return AesDecrypt(dataByte, key)
}

//pkcs7Padding 填充
func pkcs7Padding(data []byte, blockSize int) []byte {
	//判断缺少几位长度。最少1，最多 blockSize
	padding := blockSize - len(data)%blockSize
	//补足位数。把切片[]byte{byte(padding)}复制padding个
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

//pkcs7UnPadding 填充的反向操作
func pkcs7UnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("加密字符串错误！")
	}
	//获取填充的个数
	unPadding := int(data[length-1])
	return data[:(length - unPadding)], nil
}
