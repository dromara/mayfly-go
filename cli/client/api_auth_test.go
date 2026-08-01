package client

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestEncryptPassword(t *testing.T) {
	// 生成测试用 RSA 密钥对
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// 导出公钥为 PEM 格式
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}
	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	tests := []struct {
		name        string
		password    string
		publicKey   string
		wantErr     bool
		errContains string
	}{
		{"valid password", "test123", string(pubKeyPEM), false, ""},
		{"empty password", "", string(pubKeyPEM), false, ""},
		{"invalid pem", "test123", "invalid", true, ""},
		{"empty pem", "test123", "", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EncryptPassword(tt.password, tt.publicKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncryptPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == "" && tt.password != "" {
				t.Error("EncryptPassword() should return non-empty result for non-empty password")
			}
		})
	}
}

func TestApiClient_WithTimeout(t *testing.T) {
	client := NewApiClient("http://localhost:8888", "test-token")

	// 默认超时应该是 30 秒
	if client.client.Timeout.Seconds() != 30 {
		t.Errorf("Default timeout should be 30s, got %v", client.client.Timeout)
	}

	// 设置新超时
	client.WithTimeout(60)
	if client.client.Timeout.Seconds() != 60 {
		t.Errorf("Timeout should be 60s after WithTimeout(60), got %v", client.client.Timeout)
	}
	if client.rawClient.Timeout.Seconds() != 60 {
		t.Errorf("rawClient timeout should be 60s after WithTimeout(60), got %v", client.rawClient.Timeout)
	}

	// 0 或负数不应该改变超时
	client.WithTimeout(0)
	if client.client.Timeout.Seconds() != 60 {
		t.Errorf("Timeout should remain 60s after WithTimeout(0), got %v", client.client.Timeout)
	}

	client.WithTimeout(-1)
	if client.client.Timeout.Seconds() != 60 {
		t.Errorf("Timeout should remain 60s after WithTimeout(-1), got %v", client.client.Timeout)
	}
}
