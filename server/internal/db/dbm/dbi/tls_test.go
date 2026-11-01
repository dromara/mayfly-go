package dbi

import (
	"os"
	"testing"

	"mayfly-go/internal/pkg/tlsx"
)

// TestEffectiveTLSUsesOriginalHost 守护「隧道改写地址后仍按原始主机名校验证书」。
//
// dialTunnel 会把 Host 改成通道本地地址（127.0.0.1），而 verify-full 用的是证书里的
// 主机名——若 ServerName 跟着 di.Host 走，所有经 SSH 隧道的 verify-full 连接都会失败。
func TestEffectiveTLSUsesOriginalHost(t *testing.T) {
	tunneled := &DbInfo{
		Host:       "127.0.0.1",
		Port:       40001,
		RemoteAddr: "db.internal:5432",
		TLS:        &tlsx.Config{Mode: tlsx.ModeVerifyFull},
	}
	if got := tunneled.EffectiveTLS().ServerName; got != "db.internal" {
		t.Fatalf("tunneled ServerName = %q, want db.internal", got)
	}

	direct := &DbInfo{Host: "db.example.com", Port: 5432, TLS: &tlsx.Config{Mode: tlsx.ModeVerifyFull}}
	if got := direct.EffectiveTLS().ServerName; got != "db.example.com" {
		t.Fatalf("direct ServerName = %q, want db.example.com", got)
	}

	// 显式配置的 ServerName 优先于推导值
	explicit := &DbInfo{
		Host:       "127.0.0.1",
		Port:       40001,
		RemoteAddr: "db.internal:5432",
		TLS:        &tlsx.Config{Mode: tlsx.ModeVerifyFull, ServerName: "db-override.internal"},
	}
	if got := explicit.EffectiveTLS().ServerName; got != "db-override.internal" {
		t.Fatalf("explicit ServerName = %q, want db-override.internal", got)
	}
}

func TestEffectiveTLSDisabledReturnsNil(t *testing.T) {
	if (&DbInfo{}).EffectiveTLS() != nil {
		t.Fatal("nil TLS config must yield nil")
	}
	disabled := &DbInfo{TLS: &tlsx.Config{Mode: tlsx.ModeDisable}}
	if disabled.EffectiveTLS() != nil {
		t.Fatal("disable mode must yield nil: 明文连接不参与任何 TLS 参数拼装")
	}
}

// TestPrepareTLSFilesIsIdempotentAndCleaned 证书临时目录必须可重复取用、且清理后不留文件。
func TestPrepareTLSFilesIsIdempotentAndCleaned(t *testing.T) {
	material := tlsx.Material{
		RootCA:     "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n",
		ClientCert: "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n",
		ClientKey:  "-----BEGIN RSA PRIVATE KEY-----\nMIIB\n-----END RSA PRIVATE KEY-----\n",
	}
	di := &DbInfo{Host: "db.internal", Port: 5432, TLS: &tlsx.Config{Mode: tlsx.ModeVerifyFull, Material: material}}

	first, err := di.PrepareTLSFiles()
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if first == nil || first.CAPath == "" || first.ClientKeyPath == "" {
		t.Fatalf("expected materialized files, got %+v", first)
	}
	second, err := di.PrepareTLSFiles()
	if err != nil {
		t.Fatalf("prepare again: %v", err)
	}
	if second.Dir != first.Dir {
		t.Fatalf("PrepareTLSFiles must be idempotent, got dirs %q and %q", first.Dir, second.Dir)
	}
	if info, statErr := os.Stat(first.ClientKeyPath); statErr != nil {
		t.Fatalf("stat key: %v", statErr)
	} else if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("key permission = %o, want 600", perm)
	}

	dir := first.Dir
	di.CleanupTLSFiles()
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Fatalf("temp dir %s must be removed", dir)
	}
	if di.tlsFiles != nil {
		t.Fatal("tlsFiles must be reset after cleanup")
	}
}

// TestPrepareTLSFilesWithoutMaterial 无材料（系统 CA 场景）不创建目录。
func TestPrepareTLSFilesWithoutMaterial(t *testing.T) {
	di := &DbInfo{Host: "db.internal", TLS: &tlsx.Config{Mode: tlsx.ModeVerifyFull, UseSystemCA: true}}
	fs, err := di.PrepareTLSFiles()
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if fs != nil {
		t.Fatalf("system CA should not materialize files, got %+v", fs)
	}
}
