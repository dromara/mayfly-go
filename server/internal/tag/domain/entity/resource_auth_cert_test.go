package entity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// CiphertextClear 是列表接口返回前的脱敏动作：私钥口令（以密文形态存在 Extra 里）必须和
// Ciphertext 一起清掉。
//
// 它原本调 `SetExtraValue(ExtraKeyPassphrase, "")`，而基座当时对空值是「跳过」，结果是口令
// 原样留在 `extra` 字段里随响应回给前端（无报错的敏感信息外泄）。基座把零值语义改成删键后，
// 这行才真正生效，本用例钉住它不再退回去
func TestCiphertextClearStripsPassphrase(t *testing.T) {
	ac := &ResourceAuthCert{
		Name:           "prod-key",
		CiphertextType: AuthCertCiphertextTypePrivateKey,
	}
	ac.Ciphertext = "cipher-of-password"
	ac.SetExtraValue(ExtraKeyPassphrase, "cipher-of-passphrase")

	ac.CiphertextClear()

	require.Empty(t, ac.Ciphertext, "密文须清空")
	require.NotContains(t, ac.Extra, ExtraKeyPassphrase, "私钥口令必须从 extra 中删除，否则随列表接口外泄")
	require.Empty(t, ac.GetExtraString(ExtraKeyPassphrase))
}

// 公共授权凭证的密文刻意保留（靠它做继承），只清口令
func TestCiphertextClearKeepsPublicCiphertext(t *testing.T) {
	ac := &ResourceAuthCert{Name: "public", CiphertextType: AuthCertCiphertextTypePublic}
	ac.Ciphertext = "shared-cipher"
	ac.SetExtraValue(ExtraKeyPassphrase, "cipher-of-passphrase")

	ac.CiphertextClear()

	require.Equal(t, "shared-cipher", ac.Ciphertext, "公共凭证密文不清")
	require.NotContains(t, ac.Extra, ExtraKeyPassphrase)
}
