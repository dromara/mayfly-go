// Package tlsx 目标端 TLS 连接配置的统一内核。
//
// 定位：只回答「一份 TLS 配置如何变成可用的 crypto/tls.Config 或证书文件」，
// 不关心配置存在哪张表、哪个目标类型支持哪些档位、以及如何塞进具体客户端。
// 因此本包不含任何 db/redis/mongo/es 语义，可被各业务模块平等复用。
//
// 消费方接入清单（新增一个目标端时照此三步，本包零改动）：
//  1. 存储：把 Mode/ServerName/UseSystemCA 与证书材料分开存（前者是可读配置，
//     后者是密文；Material.Seal/Open 负责后者，存储位置由各模块自己决定）；
//  2. 能力：在自身能力声明里列出支持的 Mode（不支持 verify-ca 的目标别声明，
//     Config.Validate 会按声明拒绝，而不是静默降级）；
//  3. 装配：把 Config.Build() 的 *tls.Config 交给客户端；只能接收文件路径的
//     驱动（如 libpq）改用 Config.WriteFiles() 并在连接关闭时 Close()。
//
// 为什么不复用 internal/pkg/utils.PwdAesEncrypt 直接落库：该函数在未配置 aes.key
// 时会原样返回明文，对「口令」是既定行为，对「证书私钥」则等于明文落盘。
// 本包 Seal 在缺少密钥时直接报错，宁可不可用也不静默降级。
package tlsx

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mayfly-go/internal/pkg/config"
	"mayfly-go/internal/pkg/utils"
	"mayfly-go/pkg/errorx"
)

// Mode 连接加密档位。
//
// 刻意不提供 prefer/allow（「能加密就加密、失败回落明文」）：在合规语境下那等于
// 没有 TLS，且界面无法回答「这条连接到底加密了没有」。要么加密，要么显式选明文。
type Mode string

const (
	// ModeDisable 明文
	ModeDisable Mode = "disable"
	// ModeRequire 强制加密，但不校验证书（等价 libpq 的 require）
	ModeRequire Mode = "require"
	// ModeVerifyCA 校验证书链与有效期，不校验主机名
	ModeVerifyCA Mode = "verify-ca"
	// ModeVerifyFull 校验证书链、有效期与主机名
	ModeVerifyFull Mode = "verify-full"
)

// AllModes 全部档位（供能力声明默认值与前端渲染参考，顺序即界面推荐顺序）
var AllModes = []Mode{ModeDisable, ModeRequire, ModeVerifyCA, ModeVerifyFull}

// Valid 档位是否为本包可识别的取值
func (m Mode) Valid() bool {
	switch m {
	case ModeDisable, ModeRequire, ModeVerifyCA, ModeVerifyFull:
		return true
	}
	return false
}

// Encrypted 是否要求加密（disable 与空值均为明文）
func (m Mode) Encrypted() bool { return m != "" && m != ModeDisable }

// VerifiesChain 是否要求校验证书链
func (m Mode) VerifiesChain() bool { return m == ModeVerifyCA || m == ModeVerifyFull }

// Material 证书材料。三者都是 PEM 原文，整体加密后存储。
//
// 私钥只接受**未加密**的 PEM：Go 标准库没有 PKCS#8 私钥解密能力，
// 半吊子支持（仅旧版 RFC1423 PEM）会让用户以为加密私钥可用。
// 需要口令的场景由使用方在导入前自行解密，本包给出明确的错误提示。
type Material struct {
	RootCA     string `json:"rootCa,omitempty"`
	ClientCert string `json:"clientCert,omitempty"`
	ClientKey  string `json:"clientKey,omitempty"`
}

// Empty 是否完全没有材料
func (m Material) Empty() bool {
	return m.RootCA == "" && m.ClientCert == "" && m.ClientKey == ""
}

// Validate 校验 PEM 可解析、证书与私钥配对。
//
// 这里只校验「材料本身是否自洽」，不判断当前档位需不需要它——那是 Config.Validate 的事。
// MergeMissing 返回「本次缺少的字段沿用旧材料」的合并结果。
//
// 为什么要单独成方法：TLS 材料是三个独立输入框提交的，用户最常见的动作是只轮换 CA。
// 若按「整体为空才回退」处理，非空的新材料会整体替换旧值，把已保存的客户端证书与私钥
// 静默清掉——保存成功、下次建连才失败，而旧密文已被覆盖，无法恢复。
//
// 证书与私钥按一对处理：只提供其中一个视为「这一对没改」，避免造出配不上的组合。
func (m Material) MergeMissing(old Material) Material {
	merged := m
	if merged.RootCA == "" {
		merged.RootCA = old.RootCA
	}
	// 证书与私钥必须成对整体替换：只给一半时保留旧的一对，
	// 否则会写出「新证书 + 空私钥」这种必然建连失败的组合。
	// 需要彻底清空客户端证书的场景请关掉 TLS 或改用系统 CA，不做「只清一半」。
	if merged.ClientCert == "" || merged.ClientKey == "" {
		merged.ClientCert, merged.ClientKey = old.ClientCert, old.ClientKey
	}
	return merged
}

func (m Material) Validate() error {
	if m.RootCA != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(m.RootCA)) {
			return errorx.NewBiz("the CA certificate is not a valid PEM file")
		}
	}
	if (m.ClientCert == "") != (m.ClientKey == "") {
		return errorx.NewBiz("the client certificate and private key must be provided together")
	}
	if m.ClientCert != "" {
		if _, err := tls.X509KeyPair([]byte(m.ClientCert), []byte(m.ClientKey)); err != nil {
			// 口令加密的私钥会走到这里（x509: failed to parse private key ...）
			return errorx.NewBizf("the client certificate and private key do not match, or the private key is passphrase-protected (please provide a decrypted PEM): %s", err.Error())
		}
	}
	return nil
}

// EarliestExpiry 返回材料中最早到期的证书时间；没有可解析证书时 ok=false。
//
// 只取证书（CERTIFICATE 块），不取私钥：到期告警关心的是对端会拒绝的那张证书。
// 解析失败视为「不可判定」而不是「不过期」——把坏证书当永不过期会让到期提醒彻底失效。
func (m Material) EarliestExpiry() (time.Time, bool) {
	var earliest time.Time
	found := false
	for _, pemText := range []string{m.RootCA, m.ClientCert} {
		rest := []byte(pemText)
		for {
			var block *pem.Block
			block, rest = pem.Decode(rest)
			if block == nil {
				break
			}
			if block.Type != "CERTIFICATE" {
				continue
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				continue
			}
			if !found || cert.NotAfter.Before(earliest) {
				earliest = cert.NotAfter
				found = true
			}
		}
	}
	return earliest, found
}

// Fingerprint 证书材料指纹：用于展示「已配置哪一套证书」与判断轮换。
//
// 取全部材料而非仅叶子证书：换私钥但沿用旧证书同样是配置变更，连接池必须失效。
func (m Material) Fingerprint() string {
	if m.Empty() {
		return ""
	}
	sum := sha256.Sum256([]byte(m.RootCA + "\x00" + m.ClientCert + "\x00" + m.ClientKey))
	return hex.EncodeToString(sum[:])[:16]
}

// Seal 把材料封成可落库的密文字符串。
//
// 未配置 aes.key 时直接报错：此时 PwdAesEncrypt 会原样返回明文，
// 把私钥以明文写进数据库是比「功能不可用」严重得多的后果。
func (m Material) Seal() (string, error) {
	if m.Empty() {
		return "", nil
	}
	if !aesKeyConfigured() {
		return "", errorx.NewBiz("tls certificate storage requires aes.key to be configured (refusing to store the private key in plaintext)")
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return "", errorx.NewBizf("serialize tls material failed: %s", err.Error())
	}
	return utils.PwdAesEncrypt(string(raw))
}

// aesKeyConfigured 是否配置了落库加密密钥。
//
// 同时防住 config.Conf 为 nil 的场景（单元测试/未完成引导的进程），
// 否则会以空指针崩溃代替一条可读的配置错误。
func aesKeyConfigured() bool {
	return config.Conf != nil && config.Conf.Aes.Key != ""
}

// OpenMaterial 解密并解析落库的材料密文；空串表示未配置。
//
// 解密失败必须上抛：把它当成「没有证书」会让一条本该 verify-full 的连接
// 以其它方式继续，属于静默降级。
func OpenMaterial(cipher string) (Material, error) {
	var m Material
	if strings.TrimSpace(cipher) == "" {
		return m, nil
	}
	if !aesKeyConfigured() {
		return m, errorx.NewBiz("tls material is encrypted but aes.key is not configured, cannot decrypt it")
	}
	plain, err := utils.PwdAesDecrypt(cipher)
	if err != nil {
		return m, errorx.NewBizf("decrypt tls material failed: %s", err.Error())
	}
	if err := json.Unmarshal([]byte(plain), &m); err != nil {
		return m, errorx.NewBizf("parse tls material failed: %s", err.Error())
	}
	return m, nil
}

// Config 一次连接要用的完整 TLS 配置。
//
// Mode/ServerName/UseSystemCA 是可直接展示与编辑的标量，Material 是密文材料，
// 二者存储位置不同（见包注释），这里只做运行期聚合。
type Config struct {
	Mode        Mode
	ServerName  string
	UseSystemCA bool

	Material
}

// Enabled 是否需要为该连接启用 TLS
func (c *Config) Enabled() bool { return c != nil && c.Mode.Encrypted() }

// Validate 保存期校验：档位合法、该档位所需的材料齐全、材料自洽。
//
// 调用方还应在此之前校验「该目标类型是否声明了这个档位」（能力位），
// 本包不知道哪个目标支持什么。
func (c *Config) Validate() error {
	if c == nil {
		return nil
	}
	if c.Mode == "" {
		return nil
	}
	if !c.Mode.Valid() {
		return errorx.NewBizf("unsupported tls mode: %q", string(c.Mode))
	}
	if !c.Mode.Encrypted() {
		return nil
	}
	if c.Mode.VerifiesChain() && c.RootCA == "" && !c.UseSystemCA {
		return errorx.NewBiz("this tls mode requires a CA certificate (or explicitly trust the system CA)")
	}
	if c.Mode == ModeRequire && (c.ClientCert != "") == (c.ClientKey == "") {
		// require 档允许无客户端证书，但不允许只给一半
		return errorx.NewBiz("the client certificate and private key must be provided together")
	}
	return c.Material.Validate()
}

// Build 构造驱动可直接使用的 tls.Config。
//
// 三个档位的语义差异全部收敛在这里，消费方不再各写一遍：
//   - require：加密但不验证对端（InsecureSkipVerify），等价 libpq 的 require；
//   - verify-ca：Go 的 tls.Config 在关闭 InsecureSkipVerify 后**必然**校主机名，
//     无法单独表达「只校链」，故关闭默认校验并用 VerifyPeerCertificate 手工校链；
//   - verify-full：交给标准库，RootCAs 为空时使用系统根证书。
func (c *Config) Build() (*tls.Config, error) {
	if !c.Enabled() {
		return nil, nil
	}
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: c.ServerName,
	}

	if c.ClientCert != "" {
		cert, err := tls.X509KeyPair([]byte(c.ClientCert), []byte(c.ClientKey))
		if err != nil {
			return nil, errorx.NewBizf("load client certificate failed: %s", err.Error())
		}
		cfg.Certificates = []tls.Certificate{cert}
	}

	switch c.Mode {
	case ModeRequire:
		cfg.InsecureSkipVerify = true
	case ModeVerifyFull:
		roots, err := c.rootCAs()
		if err != nil {
			return nil, err
		}
		cfg.RootCAs = roots
	case ModeVerifyCA:
		roots, err := c.rootCAs()
		if err != nil {
			return nil, err
		}
		cfg.InsecureSkipVerify = true
		cfg.VerifyPeerCertificate = chainOnlyVerifier(roots)
	}
	return cfg, nil
}

// rootCAs 构造根证书池；nil 表示使用系统根证书。
func (c *Config) rootCAs() (*x509.CertPool, error) {
	if c.RootCA == "" {
		if c.UseSystemCA {
			return nil, nil
		}
		return nil, errorx.NewBiz("a CA certificate is required for the current tls mode")
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(c.RootCA)) {
		return nil, errorx.NewBiz("the CA certificate is not a valid PEM file")
	}
	return pool, nil
}

// chainOnlyVerifier 只校验证书链与有效期，不校验主机名。
// rawCerts[0] 是叶子证书，其余为对端下发的中间证书。
func chainOnlyVerifier(roots *x509.CertPool) func([][]byte, [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return errorx.NewBiz("the server did not present any certificate")
		}
		leaf, err := x509.ParseCertificate(rawCerts[0])
		if err != nil {
			return errorx.NewBizf("parse server certificate failed: %s", err.Error())
		}
		intermediates := x509.NewCertPool()
		for _, raw := range rawCerts[1:] {
			if cert, parseErr := x509.ParseCertificate(raw); parseErr == nil {
				intermediates.AddCert(cert)
			}
		}
		_, err = leaf.Verify(x509.VerifyOptions{
			Roots:         roots, // nil => 系统根证书
			Intermediates: intermediates,
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		})
		if err != nil {
			return errorx.NewBizf("verify server certificate failed: %s", err.Error())
		}
		return nil
	}
}

// Summary 可下发前端的配置摘要：不含任何证书内容，只表达「配了什么」。
type Summary struct {
	Mode          Mode   `json:"mode"`
	ServerName    string `json:"serverName,omitempty"`
	UseSystemCA   bool   `json:"useSystemCA,omitempty"`
	HasCA         bool   `json:"hasCa"`
	HasClientCert bool   `json:"hasClientCert"`
	Fingerprint   string `json:"fingerprint,omitempty"`
}

// Summary 生成下发摘要
func (c *Config) Summary() Summary {
	if c == nil {
		return Summary{Mode: ModeDisable}
	}
	mode := c.Mode
	if mode == "" {
		mode = ModeDisable
	}
	return Summary{
		Mode:          mode,
		ServerName:    c.ServerName,
		UseSystemCA:   c.UseSystemCA,
		HasCA:         c.RootCA != "",
		HasClientCert: c.ClientCert != "",
		Fingerprint:   c.Material.Fingerprint(),
	}
}

// FileSet 落盘后的证书文件集合，仅供只接受文件路径的驱动（libpq、DM 等）使用。
type FileSet struct {
	Dir            string
	CAPath         string
	ClientCertPath string
	ClientKeyPath  string
}

// WriteFiles 把材料写到独立临时目录（目录 0700、文件 0600）。
//
// 权限必须收窄：libpq 会拒绝权限过宽的私钥文件（见其 ssl_permissions.go），
// 而同一个进程里其它用户可读私钥本身就是事故。
// 无任何材料时返回 (nil, nil)——调用方据此不拼证书相关参数（如 verify-* 用系统 CA）。
func (c *Config) WriteFiles() (*FileSet, error) {
	if c == nil || c.Material.Empty() {
		return nil, nil
	}
	dir, err := os.MkdirTemp("", "mayfly-tls-*")
	if err != nil {
		return nil, errorx.NewBizf("create tls temp dir failed: %s", err.Error())
	}
	fs := &FileSet{Dir: dir}
	// 文件名全部是编译期常量，只有目录来自 os.MkdirTemp（不受请求输入影响），
	// 因此这里的 Join 不构成路径穿越面。
	writeFile := func(path, content string) error {
		if content == "" {
			return nil
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return errorx.NewBizf("write tls file %s failed: %s", filepath.Base(path), err.Error())
		}
		return nil
	}
	if c.RootCA != "" {
		fs.CAPath = filepath.Join(dir, "ca.pem")
		if err = writeFile(fs.CAPath, c.RootCA); err != nil {
			_ = fs.Close()
			return nil, err
		}
	}
	if c.ClientCert != "" {
		fs.ClientCertPath = filepath.Join(dir, "client-cert.pem")
		if err = writeFile(fs.ClientCertPath, c.ClientCert); err != nil {
			_ = fs.Close()
			return nil, err
		}
	}
	if c.ClientKey != "" {
		fs.ClientKeyPath = filepath.Join(dir, "client-key.pem")
		if err = writeFile(fs.ClientKeyPath, c.ClientKey); err != nil {
			_ = fs.Close()
			return nil, err
		}
	}
	return fs, nil
}

// Close 删除临时目录。nil 接收者安全；删除失败不影响连接关闭（Windows 上
// 句柄未释放时可能失败，只留日志由调用方决定）。
func (f *FileSet) Close() error {
	if f == nil || f.Dir == "" {
		return nil
	}
	err := os.RemoveAll(f.Dir)
	f.Dir = ""
	return err
}

// StripKV 从 `k=v&k=v` 形式的连接参数串里移除指定键（键名大小写不敏感），保留其余原样。
//
// 用途：结构化 TLS 配置生效时，剔除用户在「其他连接参数」里手写的同名项，
// 避免两处真源互相覆盖（谁生效取决于驱动解析顺序，属于不可解释行为）。
// 该串在各模块的既有形态本就以 `&` 分隔（含 pg 的 params：其代码先按 `&` 拆分再转空格）。
func StripKV(raw string, keys ...string) string {
	if strings.TrimSpace(raw) == "" || len(keys) == 0 {
		return raw
	}
	drop := make(map[string]bool, len(keys))
	for _, k := range keys {
		drop[strings.ToLower(strings.TrimSpace(k))] = true
	}
	parts := strings.Split(raw, "&")
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		key := item
		if idx := strings.Index(item, "="); idx >= 0 {
			key = item[:idx]
		}
		if drop[strings.ToLower(strings.TrimSpace(key))] {
			continue
		}
		kept = append(kept, item)
	}
	return strings.Join(kept, "&")
}

// Key 完整配置指纹（档位 + 主机名 + 系统CA + 材料内容）。
//
// 供「把 tls.Config 注册进全局表」的驱动（如 go-sql-driver 的 RegisterTLSConfig）做注册名：
// 只用材料指纹会让「同一套证书、不同档位」的两份配置共用同一个注册项，后注册者覆盖先注册者，
// 表现为「A 库的 verify-full 悄悄变成了 B 库的 require」。
func (c *Config) Key() string {
	if c == nil || !c.Enabled() {
		return ""
	}
	sum := sha256.Sum256([]byte(c.Describe() + "\x00" + c.Material.Fingerprint()))
	return hex.EncodeToString(sum[:])[:16]
}

// Describe 生成用于日志/错误的一句话描述（永不含证书内容）。
func (c *Config) Describe() string {
	if !c.Enabled() {
		return "tls=disable"
	}
	parts := []string{fmt.Sprintf("tls=%s", string(c.Mode))}
	if c.ServerName != "" {
		parts = append(parts, "serverName="+c.ServerName)
	}
	if c.RootCA != "" {
		parts = append(parts, "ca=custom")
	} else if c.UseSystemCA {
		parts = append(parts, "ca=system")
	}
	if c.ClientCert != "" {
		parts = append(parts, "clientCert=yes")
	}
	return strings.Join(parts, " ")
}
