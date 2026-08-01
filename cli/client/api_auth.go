package client

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"mayfly-go/cli/i18n"
)

// GetPublicKey 获取服务器 RSA 公钥（用于密码加密）
func (c *ApiClient) GetPublicKey() (string, error) {
	var result string
	if err := c.Get("/common/public-key", &result); err != nil {
		return "", err
	}
	return result, nil
}

// Login 执行登录，返回 accessToken 和 refreshToken
func (c *ApiClient) Login(username, encryptedPassword string) (accessToken, refreshToken string, err error) {
	loginData := map[string]string{
		"username": username,
		"password": encryptedPassword,
	}

	var result struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Post("/auth/accounts/login", loginData, &result); err != nil {
		return "", "", err
	}
	if result.Token == "" {
		return "", "", fmt.Errorf("%s", i18n.T(i18n.MsgClientNoToken))
	}
	return result.Token, result.RefreshToken, nil
}

// EncryptPassword 使用 RSA 公钥加密密码
func EncryptPassword(password, publicKeyPEM string) (string, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return "", fmt.Errorf("%s", i18n.T(i18n.MsgClientParsePEMFailed))
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T(i18n.MsgClientParseKeyFailed), err)
	}

	pubKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("%s", i18n.T(i18n.MsgClientNotRSAKey))
	}

	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, []byte(password))
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T(i18n.MsgClientEncryptFailed), err)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}
