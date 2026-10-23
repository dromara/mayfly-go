package api

import (
	"testing"

	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/domain/entity"
)

// TestStripMachineCredential 列表下发前必须剥离代理密码密文与一次性旗标，其它代理字段保留
func TestStripMachineCredential(t *testing.T) {
	mv := &vo.MachineVO{}
	mv.SetExtraValue(entity.ExtraKeyProxyType, entity.MachineProxyTypeSocks5)
	mv.SetExtraValue(entity.ExtraKeyProxyAddr, "127.0.0.1:1080")
	mv.SetExtraValue(entity.ExtraKeyProxyPassword, "ciphertext")
	mv.SetExtraValue(entity.ExtraKeyProxyPasswordClear, "true")

	stripMachineCredential(mv)

	if got := mv.GetExtraString(entity.ExtraKeyProxyPassword); got != "" {
		t.Fatalf("代理密码不应下发，实际: %q", got)
	}
	if got := mv.GetExtraString(entity.ExtraKeyProxyPasswordClear); got != "" {
		t.Fatalf("清空旗标不应下发，实际: %q", got)
	}
	if got := mv.GetExtraString(entity.ExtraKeyProxyAddr); got != "127.0.0.1:1080" {
		t.Fatalf("代理地址应保留，实际: %q", got)
	}
	if got := mv.GetExtraString(entity.ExtraKeyProxyType); got != entity.MachineProxyTypeSocks5 {
		t.Fatalf("代理类型应保留，实际: %q", got)
	}

	// nil 安全
	stripMachineCredential(nil)
}
