package config

import (
	"fmt"
	"mayfly-go/pkg/utils"
	"mayfly-go/pkg/utils/assert"
)

type Aes struct {
	Key string `yaml:"key"`
}

// 编码并base64
func (a *Aes) EncryptBase64(data []byte) (string, error) {
	return utils.AesEncryptBase64(data, []byte(a.Key))
}

// base64解码后再aes解码
func (a *Aes) DecryptBase64(data string) ([]byte, error) {
	return utils.AesDecryptBase64(data, []byte(a.Key))
}

func (j *Aes) Valid() {
	aesKeyLen := len(j.Key)
	assert.IsTrue(aesKeyLen == 16 || aesKeyLen == 24 || aesKeyLen == 32,
		fmt.Sprintf("config.yml之 [aes.key] 长度需为16、24、32位长度, 当前为%d位", aesKeyLen))
}
