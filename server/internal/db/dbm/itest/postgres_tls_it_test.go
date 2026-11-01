package itest

// 真库 TLS 握手验证（postgres，服务端已配置自签证书，见 docker-compose.it.yaml 的 it-pg-tls）。
//
// 运行方式：
//
//	docker compose -f docker-compose.it.yaml up -d it-pg-tls
//	docker exec mayfly-it-pg-tls cat /tmp/pgtls/ca.crt > /tmp/pg-ca.pem
//	MAYFLY_IT_PG_TLS_CA_FILE=/tmp/pg-ca.pem go test ./internal/db/dbm/itest -run TestPostgresTLSHandshake -count=1
//
// 与 mysql 那套互补：这里的服务端证书**带 SAN**（DNS:localhost, IP:127.0.0.1），
// 因此 verify-full 能真正成立，用来证明「主机名校验」和「ServerName 覆盖」都生效。

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/pkg/tlsx"
)

func TestPostgresTLSHandshake(t *testing.T) {
	caFile := os.Getenv("MAYFLY_IT_PG_TLS_CA_FILE")
	if caFile == "" {
		t.Skip("set MAYFLY_IT_PG_TLS_CA_FILE to run the real TLS handshake test")
	}
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		t.Fatalf("read CA file: %v", err)
	}
	if !strings.Contains(string(caPEM), "BEGIN CERTIFICATE") {
		t.Fatalf("%s does not look like a PEM CA file", caFile)
	}

	const (
		host     = "127.0.0.1"
		port     = 15432
		user     = "postgres"
		password = "postgres"
		database = "postgres"
	)
	connInfo := func(tlsCfg *tlsx.Config) *dbi.DbInfo {
		return &dbi.DbInfo{
			Type:     "postgres",
			Host:     host,
			Port:     port,
			Username: user,
			Password: password,
			Database: database,
			TLS:      tlsCfg,
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
	statusOf := func(t *testing.T, conn *dbi.DbConn) *dbi.TLSStatus {
		t.Helper()
		status, statusErr := conn.Metadata().TLSStatus(context.Background())
		if statusErr != nil {
			t.Fatalf("read tls status: %v", statusErr)
		}
		return status
	}

	t.Run("disable stays plaintext and reports it", func(t *testing.T) {
		conn, cancel, connErr := connect(connInfo(nil))
		if connErr != nil {
			t.Fatalf("plaintext connection should be accepted: %v", connErr)
		}
		defer cancel()
		defer conn.Close()
		if status := statusOf(t, conn); status.Encrypted {
			t.Fatalf("disable mode must not negotiate encryption, got %+v", status)
		}
	})

	t.Run("require negotiates encryption", func(t *testing.T) {
		conn, cancel, connErr := connect(connInfo(&tlsx.Config{Mode: tlsx.ModeRequire}))
		if connErr != nil {
			t.Fatalf("require mode should connect to a TLS-enabled server: %v", connErr)
		}
		defer cancel()
		defer conn.Close()
		status := statusOf(t, conn)
		if !status.Encrypted || status.Version == "" {
			t.Fatalf("connection should be encrypted with a protocol version, got %+v", status)
		}
	})

	t.Run("verify-ca trusts the configured CA", func(t *testing.T) {
		conn, cancel, connErr := connect(connInfo(&tlsx.Config{Mode: tlsx.ModeVerifyCA, Material: tlsx.Material{RootCA: string(caPEM)}}))
		if connErr != nil {
			t.Fatalf("verify-ca with the correct CA should connect: %v", connErr)
		}
		defer cancel()
		defer conn.Close()
	})

	t.Run("verify-full validates the SAN", func(t *testing.T) {
		// 默认 ServerName 取连接主机（127.0.0.1），证书 SAN 里有 IP:127.0.0.1
		conn, cancel, connErr := connect(connInfo(&tlsx.Config{Mode: tlsx.ModeVerifyFull, Material: tlsx.Material{RootCA: string(caPEM)}}))
		if connErr != nil {
			t.Fatalf("verify-full should accept a cert whose SAN matches the host: %v", connErr)
		}
		defer cancel()
		defer conn.Close()
	})

	t.Run("verify-full honors the configured server name", func(t *testing.T) {
		// 连 127.0.0.1，但要求按 DNS 名 localhost 校验：证明 ServerName 覆盖真的生效
		// （libpq 本身没有 servername 参数，靠 host 改写 + 自定义拨号实现）
		cfg := &tlsx.Config{Mode: tlsx.ModeVerifyFull, ServerName: "localhost", Material: tlsx.Material{RootCA: string(caPEM)}}
		conn, cancel, connErr := connect(connInfo(cfg))
		if connErr != nil {
			t.Fatalf("verify-full with an explicit matching server name should connect: %v", connErr)
		}
		defer cancel()
		defer conn.Close()
	})

	t.Run("verify-full rejects a mismatched server name", func(t *testing.T) {
		cfg := &tlsx.Config{Mode: tlsx.ModeVerifyFull, ServerName: "wrong.example.com", Material: tlsx.Material{RootCA: string(caPEM)}}
		conn, cancel, connErr := connect(connInfo(cfg))
		if connErr == nil {
			cancel()
			conn.Close()
			t.Fatal("verify-full must reject a certificate that does not match the server name")
		}
	})

	t.Run("verify-full rejects an untrusted CA", func(t *testing.T) {
		cfg := &tlsx.Config{Mode: tlsx.ModeVerifyFull, Material: tlsx.Material{RootCA: selfSignedPEM(t)}}
		conn, cancel, connErr := connect(connInfo(cfg))
		if connErr == nil {
			cancel()
			conn.Close()
			t.Fatal("verify-full must reject a certificate that is not signed by the configured CA")
		}
	})
}
