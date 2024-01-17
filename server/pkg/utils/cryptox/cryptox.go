package cryptox

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
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/logx"
	"os"

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
	publicBlock := pem.Block{Type: "PUBLIC KEY", Bytes: X509PublicKey}

	publicBuf := new(bytes.Buffer)
	pem.Encode(publicBuf, &publicBlock)
	publicKeyStr = publicBuf.String()

	return privateKeyStr, publicKeyStr, nil
}

// rsa加密
func RsaEncrypt(publicKeyStr string, data []byte) ([]byte, error) {
	block, _ := pem.Decode([]byte(publicKeyStr))
	if block == nil {
		return nil, errors.New("private key error")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.EncryptPKCS1v15(rand.Reader, pub.(*rsa.PublicKey), data)
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

const (
	// 公钥文件路径
	publicKeyFile = "./mayfly_rsa.pub"
	// 私钥文件路径
	privateKeyFile = "./mayfly_rsa"

	publicKeyK  = "mayfly:public-key"
	privateKeyK = "mayfly:private-key"
)

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
	privateKey, publicKey, err := GenerateRSAKey(1024)
	if err != nil {
		return "", "", err
	}

	// 如果使用了redis缓存，则优先存入redis
	if cache.UseRedisCache() {
		logx.Debug("系统配置了redis, rsa存入redis")
		cache.SetStr(privateKeyK, privateKey, -1)
		cache.SetStr(publicKeyK, publicKey, -1)
		return privateKey, publicKey, nil
	}

	err = os.WriteFile(privateKeyFile, []byte(privateKey), 0644)
	if err != nil {
		logx.ErrorTrace("RSA私钥写入磁盘文件失败, 使用缓存存储该私钥", err)
		cache.SetStr(privateKeyK, privateKey, -1)
	}

	err = os.WriteFile(publicKeyFile, []byte(publicKey), 0644)
	if err != nil {
		logx.ErrorTrace("RSA公钥写入磁盘文件失败, 使用缓存存储该公钥", err)
		cache.SetStr(publicKeyK, publicKey, -1)
	}

	return privateKey, publicKey, nil
}

// AesEncrypt 加密
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

// AesDecrypt 解密
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

// pkcs7Padding 填充
func pkcs7Padding(data []byte, blockSize int) []byte {
	//判断缺少几位长度。最少1，最多 blockSize
	padding := blockSize - len(data)%blockSize
	//补足位数。把切片[]byte{byte(padding)}复制padding个
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// pkcs7UnPadding 填充的反向操作
func pkcs7UnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("加密字符串错误！")
	}
	//获取填充的个数
	unPadding := int(data[length-1])
	if unPadding > length {
		return nil, errors.New("解密字符串时去除填充个数超出字符串长度")
	}
	return data[:(length - unPadding)], nil
}
