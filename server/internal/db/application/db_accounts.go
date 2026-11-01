package application

// 实例账号盘点：只读。
//
// 平台不做 CREATE USER / GRANT：账号是权限最大的对象，代改出错即事故，
// 且 MySQL(user@host) 与 PG(role) 的模型差异无法用一套写接口表达。
// 这里只把「有哪些账号、哪些是高危」摊开给人看。

import (
	"context"
	"errors"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/ioc"
)

// 账号风险等级
const (
	AccountRiskHigh = "high"
	AccountRiskWarn = "warn"
	AccountRiskInfo = "info"
)

// AccountRisk 一条账号风险
type AccountRisk struct {
	Level   string `json:"level"`
	Code    string `json:"code"`
	Account string `json:"account"`
	Host    string `json:"host,omitempty"`
}

// AccountOverview 账号盘点结果
type AccountOverview struct {
	Accounts []*dbi.AccountInfo `json:"accounts"`
	Risks    []AccountRisk      `json:"risks"`
}

// AccountsApp 实例账号盘点
type AccountsApp interface {
	Overview(ctx context.Context, dbId uint64, dbName string) (*AccountOverview, error)
}

func GetAccountsApp() AccountsApp {
	return ioc.Get[AccountsApp]()
}

type accountsAppImpl struct {
	dbApp Db `inject:"T"`
}

var _ AccountsApp = (*accountsAppImpl)(nil)

// accountRisks 纯函数：账号事实 → 风险清单
//
// 只报「能处置」的项。像「账号很多」这种不构成风险，不列，
// 否则真风险会被噪声淹掉。
func accountRisks(accounts []*dbi.AccountInfo) []AccountRisk {
	risks := make([]AccountRisk, 0, len(accounts))
	for _, account := range accounts {
		if account == nil {
			continue
		}
		// 内置系统账号（方言自己判定）跳过：它们默认锁定、由数据库维护，
		// 既不能删也不该改，报出来只会淹掉真风险
		if account.Builtin {
			continue
		}
		appendRisk := func(level, code string) {
			risks = append(risks, AccountRisk{Level: level, Code: code, Account: account.Name, Host: account.Host})
		}

		if account.Superuser {
			appendRisk(AccountRiskHigh, "superuser")
		}
		if account.PasswordExpired {
			appendRisk(AccountRiskHigh, "password_expired")
		}
		if account.Host == "%" {
			appendRisk(AccountRiskWarn, "any_host")
		}
		// PG 特有的「密码永不到期」：能登录且 rolvaliduntil 为空
		if account.NeverExpire && account.CanLogin {
			appendRisk(AccountRiskWarn, "never_expire")
		}
		if account.Locked {
			appendRisk(AccountRiskInfo, "locked")
		}
	}
	return risks
}

// Overview 盘点实例账号
func (a *accountsAppImpl) Overview(ctx context.Context, dbId uint64, dbName string) (*AccountOverview, error) {
	conn, err := a.dbApp.GetDbConn(ctx, dbId, dbName)
	if err != nil {
		return nil, err
	}
	accounts, err := conn.Metadata().Accounts(ctx)
	if err != nil {
		// 「方言不提供」与「读不到（权限不足/视图不可见）」处置方向完全不同：
		// 前者换方言也没用、只能接受；后者给只读权限就能看。合成一条会让人朝错方向排查
		if errors.Is(err, dbi.ErrUnsupportedKind) {
			return nil, errorx.NewBizI(ctx, imsg.ErrAccountsUnsupported, "reason", err.Error())
		}
		return nil, errorx.NewBizI(ctx, imsg.ErrAccountsUnreadable, "reason", err.Error())
	}
	return &AccountOverview{Accounts: accounts, Risks: accountRisks(accounts)}, nil
}
