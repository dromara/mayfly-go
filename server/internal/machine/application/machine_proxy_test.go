package application

import (
	"testing"

	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/pkg/config"
	"mayfly-go/internal/pkg/utils"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/netx"
)

func newSocks5Machine(name string) *entity.Machine {
	me := &entity.Machine{Name: name}
	me.SetExtraValue(entity.ExtraKeyProxyType, entity.MachineProxyTypeSocks5)
	me.SetExtraValue(entity.ExtraKeyProxyAddr, " 127.0.0.1:1080 ")
	me.SetExtraValue(entity.ExtraKeyProxyUsername, "audit")
	return me
}

// TestNormalizeAndResolveMachineProxy 机器级代理保存/装配：密码加密、留空沿用、显式清空、类型白名单
func TestNormalizeAndResolveMachineProxy(t *testing.T) {
	oldConf := config.Conf
	defer func() { config.Conf = oldConf }()
	config.Conf = &config.Config{Aes: config.Aes{Key: "1111111111111111"}}

	// 1. 默认（空）与显式 none 都直连
	for _, proxyType := range []string{"", entity.MachineProxyTypeNone} {
		me := &entity.Machine{Name: "direct"}
		me.SetExtraValue(entity.ExtraKeyProxyType, proxyType)
		dialer, err := resolveMachineProxy(me)
		if err != nil || dialer != nil {
			t.Fatalf("proxyType=%q 应直连，实际 dialer=%v err=%v", proxyType, dialer, err)
		}
	}

	// 2. 未知类型必须报错，不能静默直连
	unknown := &entity.Machine{Name: "typo"}
	unknown.SetExtraValue(entity.ExtraKeyProxyType, "Socks5")
	if _, err := resolveMachineProxy(unknown); err == nil {
		t.Fatal("未知 proxyType 应报错")
	}
	if err := normalizeMachineProxy(unknown, nil); err == nil {
		t.Fatal("保存未知 proxyType 应报错")
	}

	// 3. 启用 socks5 但缺地址：保存与装配都要报错
	missingAddr := &entity.Machine{Name: "missing-addr"}
	missingAddr.SetExtraValue(entity.ExtraKeyProxyType, entity.MachineProxyTypeSocks5)
	if _, err := resolveMachineProxy(missingAddr); err == nil {
		t.Fatal("装配时缺地址应报错")
	}
	if err := normalizeMachineProxy(missingAddr, nil); err == nil {
		t.Fatal("保存时缺地址应报错")
	}

	// 4. 正常保存：地址去空白、密码加密落库；装配时还原明文
	me := newSocks5Machine("proxy-machine")
	me.SetExtraValue(entity.ExtraKeyProxyPassword, "secret")
	if err := normalizeMachineProxy(me, nil); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	stored := me.GetExtraString(entity.ExtraKeyProxyPassword)
	if stored == "" || stored == "secret" {
		t.Fatalf("密码应加密落库，实际: %q", stored)
	}
	if me.GetExtraString(entity.ExtraKeyProxyAddr) != "127.0.0.1:1080" {
		t.Fatalf("地址应去除首尾空白，实际: %q", me.GetExtraString(entity.ExtraKeyProxyAddr))
	}
	dialer, err := resolveMachineProxy(me)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	cfg, ok := dialer.(netx.Socks5Config)
	if !ok || cfg.Addr != "127.0.0.1:1080" || cfg.Username != "audit" || cfg.Password != "secret" {
		t.Fatalf("装配结果不对: %#v", dialer)
	}

	// 5. 编辑时密码留空：沿用库中密文
	edit := newSocks5Machine("proxy-machine")
	edit.SetExtraValue(entity.ExtraKeyProxyPassword, "")
	if err := normalizeMachineProxy(edit, me); err != nil {
		t.Fatalf("normalize edit: %v", err)
	}
	if got := edit.GetExtraString(entity.ExtraKeyProxyPassword); got != stored {
		t.Fatalf("密码留空应沿用旧密文，old=%q new=%q", stored, got)
	}

	// 6. 显式清空密码：密文被删除且一次性旗标不落库
	cleared := newSocks5Machine("proxy-machine")
	cleared.SetExtraValue(entity.ExtraKeyProxyPassword, "")
	cleared.SetExtraValue(entity.ExtraKeyProxyPasswordClear, "true")
	if err := normalizeMachineProxy(cleared, me); err != nil {
		t.Fatalf("normalize cleared: %v", err)
	}
	if got := cleared.GetExtraString(entity.ExtraKeyProxyPassword); got != "" {
		t.Fatalf("显式清空后应无密码密文，实际: %q", got)
	}
	if got := cleared.GetExtraString(entity.ExtraKeyProxyPasswordClear); got != "" {
		t.Fatalf("一次性旗标不应落库，实际: %q", got)
	}

	// 7. 关闭代理：清掉密码密文，避免残留凭据
	disabled := &entity.Machine{Name: "proxy-machine"}
	disabled.SetExtraValue(entity.ExtraKeyProxyType, entity.MachineProxyTypeNone)
	disabled.SetExtraValue(entity.ExtraKeyProxyPassword, stored)
	if err := normalizeMachineProxy(disabled, me); err != nil {
		t.Fatalf("normalize disabled: %v", err)
	}
	if got := disabled.GetExtraString(entity.ExtraKeyProxyPassword); got != "" {
		t.Fatalf("关闭代理后应清除密码密文，实际: %q", got)
	}
}

// TestResolveTestProxyUsesPlaintextPassword 测试连接走表单明文密码，不走「密文解密」语义
func TestResolveTestProxyUsesPlaintextPassword(t *testing.T) {
	oldConf := config.Conf
	defer func() { config.Conf = oldConf }()
	config.Conf = &config.Config{Aes: config.Aes{Key: "1111111111111111"}}

	me := newSocks5Machine("form")
	me.SetExtraValue(entity.ExtraKeyProxyPassword, "plain-input")
	dialer, err := (&machineAppImpl{}).resolveTestProxy(me, 0)
	if err != nil {
		t.Fatalf("明文密码不应走解密，实际报错: %v", err)
	}
	cfg := dialer.(netx.Socks5Config)
	if cfg.Password != "plain-input" {
		t.Fatalf("密码 = %q, want plain-input", cfg.Password)
	}

	// 表单未填地址：报错
	noAddr := &entity.Machine{Name: "no-addr"}
	noAddr.SetExtraValue(entity.ExtraKeyProxyType, entity.MachineProxyTypeSocks5)
	if _, err := (&machineAppImpl{}).resolveTestProxy(noAddr, 0); err == nil {
		t.Fatal("测试连接缺地址应报错")
	}
}

// TestSanitizeMachineExtraCredential 进 MachineInfo 的 extra 不得携带代理密码与一次性旗标
func TestSanitizeMachineExtraCredential(t *testing.T) {
	me := newSocks5Machine("log-safe")
	me.SetExtraValue(entity.ExtraKeyProxyPassword, "ciphertext")
	me.SetExtraValue(entity.ExtraKeyProxyPasswordClear, "true")

	clean := sanitizeMachineExtra(me.ExtraData)
	if got := clean.GetExtraString(entity.ExtraKeyProxyPassword); got != "" {
		t.Fatalf("MachineInfo extra 不应带代理密码，实际: %q", got)
	}
	if got := clean.GetExtraString(entity.ExtraKeyProxyPasswordClear); got != "" {
		t.Fatalf("MachineInfo extra 不应带清空旗标，实际: %q", got)
	}
	// 非凭据键必须保留（建链依赖 ciphers/keyExchanges 等）
	if got := clean.GetExtraString(entity.ExtraKeyProxyAddr); got != " 127.0.0.1:1080 " {
		t.Fatalf("代理地址应保留，实际: %q", got)
	}
	// 不能改动原 map（原 extra 还用于落库）
	if got := me.GetExtraString(entity.ExtraKeyProxyPassword); got != "ciphertext" {
		t.Fatalf("原 extra 不应被修改，实际: %q", got)
	}
}

// TestNormalizeMachineProxyConflict 同时填新密码与勾选清空属于矛盾输入，必须报错
func TestNormalizeMachineProxyConflict(t *testing.T) {
	me := newSocks5Machine("conflict")
	me.SetExtraValue(entity.ExtraKeyProxyPassword, "new-secret")
	me.SetExtraValue(entity.ExtraKeyProxyPasswordClear, "true")
	if err := normalizeMachineProxy(me, nil); err == nil {
		t.Fatal("同时设置与清空密码应报错")
	}
}

// TestDecryptStoredProxyPassword 库中密文解密：正常还原、空密文为空、坏密文报错
func TestDecryptStoredProxyPassword(t *testing.T) {
	oldConf := config.Conf
	defer func() { config.Conf = oldConf }()
	config.Conf = &config.Config{Aes: config.Aes{Key: "1111111111111111"}}

	cipher, err := utils.PwdAesEncrypt("secret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	extra := model.ExtraData{}
	extra.SetExtraValue(entity.ExtraKeyProxyPassword, cipher)
	got, err := decryptStoredProxyPassword(extra)
	if err != nil || got != "secret" {
		t.Fatalf("解密结果 = %q err=%v, want secret", got, err)
	}

	if got, err := decryptStoredProxyPassword(model.ExtraData{}); err != nil || got != "" {
		t.Fatalf("空密文应为空串，实际 %q err=%v", got, err)
	}

	broken := model.ExtraData{}
	broken.SetExtraValue(entity.ExtraKeyProxyPassword, "not-a-ciphertext")
	if _, err := decryptStoredProxyPassword(broken); err == nil {
		t.Fatal("坏密文应报错")
	}
}

// TestLogSanitizeDoesNotEatPassword 回归：日志脱敏必须基于副本。
// 表单与实体在 BindJsonAndCopyTo 后共享同一个 extra map，若脱敏直接删键，
// 会把待加密落库的明文密码/清空旗标一起删掉，表现为“保存成功但密码没存”。
func TestLogSanitizeDoesNotEatPassword(t *testing.T) {
	oldConf := config.Conf
	defer func() { config.Conf = oldConf }()
	config.Conf = &config.Config{Aes: config.Aes{Key: "1111111111111111"}}

	me := newSocks5Machine("shared-map")
	me.SetExtraValue(entity.ExtraKeyProxyPassword, "secret")
	me.SetExtraValue(entity.ExtraKeyProxyPasswordClear, "true")

	// 模拟 SaveMachine 的日志脱敏步骤（基于副本）
	logExtra := entity.CloneExtraWithoutProxyCredential(me.ExtraData)
	if logExtra.GetExtraString(entity.ExtraKeyProxyPassword) != "" {
		t.Fatal("日志副本不应包含密码")
	}

	// 脱敏后原 map 必须保留密码与旗标：同时设置+清空应触发冲突报错
	if err := normalizeMachineProxy(me, nil); err == nil {
		t.Fatal("脱敏副本不应删掉原 map 中的清空旗标（应触发冲突报错）")
	}

	// 去掉旗标后保存，密码必须加密落库
	me.SetExtraValue(entity.ExtraKeyProxyPasswordClear, "")
	if err := normalizeMachineProxy(me, nil); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	stored := me.GetExtraString(entity.ExtraKeyProxyPassword)
	if stored == "" || stored == "secret" {
		t.Fatalf("密码应加密落库，实际: %q", stored)
	}
	plain, err := decryptStoredProxyPassword(me.ExtraData)
	if err != nil || plain != "secret" {
		t.Fatalf("回读密码 = %q err=%v, want secret", plain, err)
	}
}
