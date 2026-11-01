package itest

// 真库 TLS 握手验证（mysql 8.0 自带自签证书，默认 have_ssl=YES）。
//
// 运行方式（本机已有 it-mysql 容器时）：
//
//	docker exec mysql cat /var/lib/mysql/ca.pem > /tmp/mysql-ca.pem
//	MAYFLY_IT_MYSQL_TLS_CA_FILE=/tmp/mysql-ca.pem go test ./internal/db/dbm/itest -run TestMySQLTLS -count=1
//
// 本用例刻意不测 verify-full：MySQL 自动生成的服务端证书只有 CN、没有 SAN，
// 而 Go 1.15+ 不再用 CN 做主机名校验，verify-full 对这类证书必然失败。
// 这恰好说明三档必须分开——verify-ca 才是这种证书唯一能落地的档位，
// 也是本用例真正要证明的东西（只校链的实现能在真握手里成立）。

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/pkg/tlsx"
)

func TestMySQLTLSHandshake(t *testing.T) {
	caFile := os.Getenv("MAYFLY_IT_MYSQL_TLS_CA_FILE")
	if caFile == "" {
		t.Skip("set MAYFLY_IT_MYSQL_TLS_CA_FILE to run the real TLS handshake test")
	}
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		t.Fatalf("read CA file: %v", err)
	}
	if !strings.Contains(string(caPEM), "BEGIN CERTIFICATE") {
		t.Fatalf("%s does not look like a PEM CA file", caFile)
	}

	password := os.Getenv("MAYFLY_IT_MYSQL_PASSWORD")
	if password == "" {
		password = "111049"
	}
	connInfo := func(mode tlsx.Mode, rootCA string) *dbi.DbInfo {
		return &dbi.DbInfo{
			Type:     "mysql",
			Host:     "127.0.0.1",
			Port:     3306,
			Username: "root",
			Password: password,
			TLS: &tlsx.Config{
				Mode:     mode,
				Material: tlsx.Material{RootCA: rootCA},
			},
		}
	}
	connect := func(info *dbi.DbInfo) (*dbi.DbConn, context.CancelFunc, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		conn, connErr := dbm.Conn(ctx, info)
		if connErr != nil {
			cancel()
			return nil, nil, connErr
		}
		return conn, cancel, nil
	}

	t.Run("require negotiates encryption", func(t *testing.T) {
		conn, cancel, connErr := connect(connInfo(tlsx.ModeRequire, ""))
		if connErr != nil {
			t.Fatalf("require mode should connect to a TLS-enabled server: %v", connErr)
		}
		defer cancel()
		defer conn.Close()

		status, statusErr := conn.Metadata().TLSStatus(context.Background())
		if statusErr != nil {
			t.Fatalf("read tls status: %v", statusErr)
		}
		if !status.Available || !status.Encrypted {
			t.Fatalf("connection should be encrypted, got %+v", status)
		}
		if status.Cipher == "" {
			t.Fatalf("negotiated cipher must not be empty, got %+v", status)
		}
	})

	t.Run("verify-ca trusts the configured CA", func(t *testing.T) {
		conn, cancel, connErr := connect(connInfo(tlsx.ModeVerifyCA, string(caPEM)))
		if connErr != nil {
			t.Fatalf("verify-ca with the correct CA should connect: %v", connErr)
		}
		defer cancel()
		defer conn.Close()
	})

	t.Run("verify-ca rejects an untrusted CA", func(t *testing.T) {
		conn, cancel, connErr := connect(connInfo(tlsx.ModeVerifyCA, selfSignedPEM(t)))
		if connErr == nil {
			cancel()
			conn.Close()
			t.Fatal("verify-ca must reject a certificate that is not signed by the configured CA")
		}
	})

	t.Run("disable stays plaintext and reports it", func(t *testing.T) {
		conn, cancel, connErr := connect(connInfo(tlsx.ModeDisable, ""))
		if connErr != nil {
			t.Fatalf("disable mode should connect to a plaintext-permitting server: %v", connErr)
		}
		defer cancel()
		defer conn.Close()

		status, statusErr := conn.Metadata().TLSStatus(context.Background())
		if statusErr != nil {
			t.Fatalf("read tls status: %v", statusErr)
		}
		if !status.Available {
			t.Fatalf("mysql should support tls status, got %+v", status)
		}
		if status.Encrypted {
			t.Fatalf("disable mode must not negotiate encryption, got %+v", status)
		}
	})
}

// selfSignedPEM 生成一张与本机服务端无关的 CA，用于「不可信 CA 必须被拒」。
func selfSignedPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(9),
		Subject:               pkix.Name{CommonName: "untrusted-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}
