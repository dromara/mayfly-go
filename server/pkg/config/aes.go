package config

import (
	"fmt"
	"mayfly-go/pkg/utils/assert"
	"mayfly-go/pkg/utils/cryptox"
)

type Aes struct {
	Key string `yaml:"key"`
}

// 编码并base64
func (a *Aes) EncryptBase64(data []byte) (string, error) {
	return cryptox.AesEncryptBase64(data, []byte(a.Key))
}

// base64解码后再aes解码
func (a *Aes) DecryptBase64(data string) ([]byte, error) {
	return cryptox.AesDecryptBase64(data, []byte(a.Key))
}

func (a *Aes) Valid() {
	if a.Key == "" {
		return
	}
	aesKeyLen := len(a.Key)
	assert.IsTrue(aesKeyLen == 16 || aesKeyLen == 24 || aesKeyLen == 32,
		fmt.Sprintf("config.yml之 [aes.key] 长度需为16、24、32位长度, 当前为%d位", aesKeyLen))
}
