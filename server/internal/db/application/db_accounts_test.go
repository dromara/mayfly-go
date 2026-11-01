package application

// 账号风险规则：只报能处置的项，且等级要能区分「必须马上处理」与「知会一下」。

import (
	"testing"

	"mayfly-go/internal/db/dbm/dbi"
)

func riskCodes(risks []AccountRisk) map[string]string {
	codes := make(map[string]string, len(risks))
	for _, risk := range risks {
		codes[risk.Code] = risk.Level
	}
	return codes
}

func TestAccountRisks(t *testing.T) {
	accounts := []*dbi.AccountInfo{
		// MySQL 超管 + 任意来源：两条都要报
		{Name: "root", Host: "%", Superuser: true, CanLogin: true},
		// 密码已过期：必须马上处理
		{Name: "app", Host: "10.0.0.%", PasswordExpired: true},
		// 已锁定：知会即可（锁着本身不是风险）
		{Name: "old", Host: "10.0.0.1", Locked: true},
		// PG 的「能登录 + 密码永不过期」
		{Name: "pguser", CanLogin: true, NeverExpire: true},
		// 普通账号、来源受限：不该有任何风险
		{Name: "report", Host: "10.0.0.2"},
	}
	risks := accountRisks(accounts)
	levels := riskCodes(risks)

	if levels["superuser"] != AccountRiskHigh {
		t.Fatalf("superuser must be high, got %q", levels["superuser"])
	}
	if levels["any_host"] != AccountRiskWarn {
		t.Fatalf("host=%% must be warn, got %q", levels["any_host"])
	}
	if levels["password_expired"] != AccountRiskHigh {
		t.Fatalf("expired password must be high, got %q", levels["password_expired"])
	}
	if levels["never_expire"] != AccountRiskWarn {
		t.Fatalf("pg never-expire password must be warn, got %q", levels["never_expire"])
	}
	if levels["locked"] != AccountRiskInfo {
		t.Fatalf("locked account must be info, got %q", levels["locked"])
	}

	// 普通账号不得产生任何风险（噪声会让真风险被忽略）
	for _, risk := range risks {
		if risk.Account == "report" {
			t.Fatalf("plain account must not raise any risk, got %+v", risk)
		}
	}
	// 风险条目必须带上是哪个账号（否则界面无法处置）
	for _, risk := range risks {
		if risk.Account == "" {
			t.Fatalf("risk must carry account name: %+v", risk)
		}
	}
}

func TestAccountRisksSkipsBuiltinAccounts(t *testing.T) {
	// MySQL 内置账号默认锁定、不可改，报出来只会淹掉真风险
	// Builtin 由方言判定（MySQL 把 mysql.* 标成内置），应用层只消费这个事实
	risks := accountRisks([]*dbi.AccountInfo{
		{Name: "mysql.sys", Host: "localhost", Locked: true, Superuser: true, Builtin: true},
		{Name: "mysql.infoschema", Host: "localhost", Locked: true, Builtin: true},
		// 名字像内置但方言没标记的账号仍然要报（不靠前缀猜）
		{Name: "mysql.rogue", Host: "%", Superuser: true},
	})
	if len(risks) != 2 {
		t.Fatalf("only dialect-marked builtin accounts are skipped, got %+v", risks)
	}
	for _, risk := range risks {
		if risk.Account != "mysql.rogue" {
			t.Fatalf("unexpected risk from builtin account: %+v", risk)
		}
	}
}

func TestAccountRisksIgnoresNilAndNonLoginRoles(t *testing.T) {
	// PG 的纯组角色（NOLOGIN + 永不过期）不该报「密码永不过期」——它根本不登录
	risks := accountRisks([]*dbi.AccountInfo{nil, {Name: "grp", CanLogin: false, NeverExpire: true}})
	if len(risks) != 0 {
		t.Fatalf("nologin role must not raise never_expire, got %+v", risks)
	}
}
