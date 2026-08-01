package client

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"mayfly-go/cli/i18n"
)

// AesEncryptBase64 使用 AES-192-CBC 加密数据，返回 base64 编码字符串
// key 必须是 24 字节（从 JWT token 前 24 字符获取）
func AesEncryptBase64(data []byte, key []byte) (string, error) {
	if len(key) < 24 {
		return "", fmt.Errorf("%s", i18n.T(i18n.MsgClientAesKeyTooShort, "len", len(key)))
	}
	key = key[:24]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T(i18n.MsgClientAesCipherFailed), err)
	}

	blockSize := block.BlockSize()
	// PKCS7 填充
	padding := blockSize - len(data)%blockSize
	padText := make([]byte, padding)
	for i := range padText {
		padText[i] = byte(padding)
	}
	paddedData := append(data, padText...)

	// CBC 加密，IV = key[:blockSize]
	crypted := make([]byte, len(paddedData))
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	blockMode.CryptBlocks(crypted, paddedData)

	return base64.StdEncoding.EncodeToString(crypted), nil
}
