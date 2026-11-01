package tlsx

import (
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

	"mayfly-go/internal/pkg/config"
)

// newTestCA 生成自签 CA，返回 PEM 与其私钥（供签发叶子证书）
func newTestCA(t *testing.T) (string, *x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate ca key: %v", err)
	}
	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "mayfly-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create ca: %v", err)
	}
	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse ca: %v", err)
	}
	return certPEM, parsed, key
}

// newTestLeaf 签发一张只含内网域名 CN、无 SAN 的服务端证书（用于验证「只校链不校主机名」）
func newTestLeaf(t *testing.T, ca *x509.Certificate, caKey *rsa.PrivateKey) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate leaf key: %v", err)
	}
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "db.internal"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, ca, &key.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func TestValidateRejectsIncompleteConfig(t *testing.T) {
	cases := []struct {
		name string
		cfg  *Config
	}{
		{"verify-ca 缺 CA", &Config{Mode: ModeVerifyCA}},
		{"verify-full 缺 CA", &Config{Mode: ModeVerifyFull}},
		{"未知档位", &Config{Mode: Mode("prefer")}},
		{"只给客户端证书", &Config{Mode: ModeRequire, Material: Material{ClientCert: "x"}}},
		{"非法 CA", &Config{Mode: ModeVerifyFull, Material: Material{RootCA: "not-pem"}}},
	}
	for _, tc := range cases {
		if err := tc.cfg.Validate(); err == nil {
			t.Errorf("%s: expected validation error, got nil", tc.name)
		}
	}

	// 显式信任系统 CA 时 verify-full 合法
	if err := (&Config{Mode: ModeVerifyFull, UseSystemCA: true}).Validate(); err != nil {
		t.Errorf("verify-full + UseSystemCA should be valid, got %v", err)
	}
	// 明文档位永远合法
	if err := (&Config{Mode: ModeDisable}).Validate(); err != nil {
		t.Errorf("disable should be valid, got %v", err)
	}
}

func TestBuildRequireSkipsVerification(t *testing.T) {
	cfg := &Config{Mode: ModeRequire, ServerName: "db.internal"}
	built, err := cfg.Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !built.InsecureSkipVerify || built.VerifyPeerCertificate != nil {
		t.Fatalf("require must encrypt without verifying, got InsecureSkipVerify=%v verifyFn=%v",
			built.InsecureSkipVerify, built.VerifyPeerCertificate != nil)
	}
	if built.ServerName != "db.internal" {
		t.Fatalf("ServerName not propagated: %q", built.ServerName)
	}
	if built.MinVersion == 0 {
		t.Fatal("MinVersion must be pinned to TLS1.2 or above")
	}
}

// TestVerifyCAIgnoresHostname 是本文件的核心断言：
// verify-ca 必须接受「链可信但主机名不匹配」的证书，否则它就退化成 verify-full，
// 而经 SSH 隧道（目标地址被改写）的场景正是靠这个差异才能工作。
func TestVerifyCAIgnoresHostname(t *testing.T) {
	caPEM, ca, caKey := newTestCA(t)
	leafPEM := newTestLeaf(t, ca, caKey)

	cfg := &Config{
		Mode:       ModeVerifyCA,
		ServerName: "not-the-cert-host",
		Material:   Material{RootCA: caPEM},
	}
	built, err := cfg.Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !built.InsecureSkipVerify || built.VerifyPeerCertificate == nil {
		t.Fatal("verify-ca must disable default verification and install a chain-only verifier")
	}

	block, _ := pem.Decode([]byte(leafPEM))
	if block == nil {
		t.Fatal("decode leaf pem failed")
	}
	if err := built.VerifyPeerCertificate([][]byte{block.Bytes}, nil); err != nil {
		t.Fatalf("verify-ca should accept a trusted cert regardless of hostname, got %v", err)
	}
}

func TestVerifyCARejectsUntrustedChain(t *testing.T) {
	_, ca, caKey := newTestCA(t)
	leafPEM := newTestLeaf(t, ca, caKey)
	otherCAPEM, _, _ := newTestCA(t)

	cfg := &Config{Mode: ModeVerifyCA, Material: Material{RootCA: otherCAPEM}}
	built, err := cfg.Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	block, _ := pem.Decode([]byte(leafPEM))
	if err := built.VerifyPeerCertificate([][]byte{block.Bytes}, nil); err == nil {
		t.Fatal("verify-ca must reject a cert that is not signed by the configured CA")
	}
}

func TestWriteFilesUsesRestrictivePermissions(t *testing.T) {
	certPEM, _, _ := newTestCA(t)
	keyPEM := "-----BEGIN RSA PRIVATE KEY-----\nMIIBog==\n-----END RSA PRIVATE KEY-----\n"
	cfg := &Config{Mode: ModeVerifyFull, Material: Material{RootCA: certPEM, ClientCert: certPEM, ClientKey: keyPEM}}
	fs, err := cfg.WriteFiles()
	if err != nil {
		t.Fatalf("write files: %v", err)
	}
	defer func() { _ = fs.Close() }()

	for _, path := range []string{fs.CAPath, fs.ClientCertPath, fs.ClientKeyPath} {
		info, statErr := os.Stat(path)
		if statErr != nil {
			t.Fatalf("stat %s: %v", path, statErr)
		}
		// libpq 会拒绝权限过宽的私钥；0600 是唯一安全档
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Fatalf("%s permission = %o, want 600", path, perm)
		}
	}
	dir := fs.Dir
	if err := fs.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Fatalf("temp dir %s should be removed after Close", dir)
	}
}

func TestWriteFilesReturnsNilWithoutMaterial(t *testing.T) {
	fs, err := (&Config{Mode: ModeVerifyFull, UseSystemCA: true}).WriteFiles()
	if err != nil {
		t.Fatalf("write files: %v", err)
	}
	if fs != nil {
		t.Fatalf("no material should not create files, got %+v", fs)
	}
}

func TestStripKV(t *testing.T) {
	raw := "tls=true&charset=utf8mb4&sslrootcert=/tmp/ca.pem&TLS=skip-verify"
	got := StripKV(raw, "tls", "sslrootcert")
	if got != "charset=utf8mb4" {
		t.Fatalf("StripKV = %q, want %q", got, "charset=utf8mb4")
	}
	if StripKV("", "tls") != "" {
		t.Fatal("empty input must stay empty")
	}
	if got := StripKV("a=1&tls=true", "TLS"); got != "a=1" {
		t.Fatalf("key match must be case-insensitive, got %q", got)
	}
}

// withAesKey 在测试期内替换全局配置（config.Conf 由进程引导创建，单测里为 nil）
func withAesKey(t *testing.T, key string) {
	t.Helper()
	original := config.Conf
	conf := &config.Config{}
	conf.Aes.Key = key
	config.Conf = conf
	t.Cleanup(func() { config.Conf = original })
}

func TestSealRefusesPlaintextWithoutAesKey(t *testing.T) {
	withAesKey(t, "")

	if _, err := (Material{ClientKey: "secret"}).Seal(); err == nil {
		t.Fatal("Seal must refuse to store the key when aes.key is missing")
	}
	if _, err := OpenMaterial("cipher"); err == nil {
		t.Fatal("OpenMaterial must fail when aes.key is missing")
	}
}

func TestSealOpenRoundTrip(t *testing.T) {
	withAesKey(t, "0123456789abcdef") // 16 字节，满足 AES 要求

	m := Material{RootCA: "ca-pem", ClientCert: "cert-pem", ClientKey: "key-pem"}
	sealed, err := m.Seal()
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if strings.Contains(sealed, "key-pem") {
		t.Fatal("sealed material must not contain plaintext")
	}
	opened, err := OpenMaterial(sealed)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if opened != m {
		t.Fatalf("round trip mismatch: %+v != %+v", opened, m)
	}

	empty, err := OpenMaterial("")
	if err != nil || !empty.Empty() {
		t.Fatalf("empty ciphertext must open to empty material, got %+v err=%v", empty, err)
	}
}

func TestKeyDistinguishesModeAndMaterial(t *testing.T) {
	base := &Config{Mode: ModeVerifyFull, ServerName: "db.internal", Material: Material{RootCA: "a"}}
	if base.Key() == "" {
		t.Fatal("key must not be empty for an enabled config")
	}
	sameMaterialOtherMode := &Config{Mode: ModeRequire, ServerName: "db.internal", Material: Material{RootCA: "a"}}
	if base.Key() == sameMaterialOtherMode.Key() {
		t.Fatal("same material with different modes must not share a registry key")
	}
	if (&Config{Mode: ModeDisable}).Key() != "" {
		t.Fatal("disabled config must not produce a registry key")
	}
}

// TestEarliestExpiry 证书到期取「最早的那张」：只认证书块、忽略私钥，坏证书视为不可判定。
func TestEarliestExpiry(t *testing.T) {
	if _, ok := (Material{}).EarliestExpiry(); ok {
		t.Fatal("empty material must not report an expiry")
	}

	caPEM, ca, caKey := newTestCA(t)
	// 一张 24 小时到期（newTestLeaf 的固定值）、一张 60 天到期：
	// 取最早的才说明实现没有「返回最后解析到的那张」
	shortPEM := newTestLeaf(t, ca, caKey)
	long := newTestLeafWithDays(t, ca, caKey, 60)

	m := Material{RootCA: caPEM, ClientCert: shortPEM + long}
	expiry, ok := m.EarliestExpiry()
	if !ok {
		t.Fatal("expected an expiry for parsed certificates")
	}
	// 最近的那张应在 24h 左右（newTestLeaf）而不是 60 天
	if hours := time.Until(expiry).Hours(); hours > 48 {
		t.Fatalf("expiry should come from the soonest certificate, got %.1fh away", hours)
	}
}

// newTestLeafWithDays 签发指定有效期的叶子证书（仅用于比较「最早到期」）
func newTestLeafWithDays(t *testing.T, ca *x509.Certificate, caKey *rsa.PrivateKey, days int) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate leaf key: %v", err)
	}
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(int64(100 + days)),
		Subject:      pkix.Name{CommonName: "db.internal"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Duration(days) * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, ca, &key.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

// TestMaterialMergeMissing TLS 材料是三个独立输入框提交的：只轮换 CA 时，
// 已存的客户端证书与私钥必须保留——整体替换会让「保存成功」变成下次建连才暴露的数据丢失。
func TestMaterialMergeMissing(t *testing.T) {
	old := Material{RootCA: "old-ca", ClientCert: "old-cert", ClientKey: "old-key"}

	merged := Material{RootCA: "new-ca"}.MergeMissing(old)
	if merged.RootCA != "new-ca" || merged.ClientCert != "old-cert" || merged.ClientKey != "old-key" {
		t.Fatalf("rotating CA must keep client pair, got %+v", merged)
	}

	// 证书与私钥成对：只给其中一个视为「这一对没改」
	merged = Material{ClientCert: "new-cert"}.MergeMissing(old)
	if merged.ClientCert != "old-cert" || merged.ClientKey != "old-key" {
		t.Fatalf("half pair must not replace the stored pair, got %+v", merged)
	}

	// 成对提供才替换
	merged = Material{ClientCert: "new-cert", ClientKey: "new-key"}.MergeMissing(old)
	if merged.ClientCert != "new-cert" || merged.ClientKey != "new-key" {
		t.Fatalf("complete pair must replace, got %+v", merged)
	}

	// 旧材料为空时不能凭空造值
	merged = Material{}.MergeMissing(Material{})
	if !merged.Empty() {
		t.Fatalf("empty merge must stay empty, got %+v", merged)
	}
}
