package mysql

// mysql 账号盘点：读 mysql.user。
//
// 权限要求：普通账号读不到 mysql.user，此时返回「不可用 + 原因」，
// 由管理员决定是否授予只读权限——平台不为了看账号去要 SUPER。

import (
	"context"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/spf13/cast"
)

var _ dbi.AccountInspector = (*MysqlMetadata)(nil)

// Accounts 列出实例上的账号（user@host 维度）
func (md *MysqlMetadata) Accounts(ctx context.Context) ([]*dbi.AccountInfo, error) {
	_, rows, err := md.di.QueryContext(ctx,
		"SELECT User, Host, Super_priv, Grant_priv, account_locked, password_expired,"+
			" IFNULL(password_last_changed, '') last_changed FROM mysql.user ORDER BY User, Host")
	if err != nil {
		return nil, dbi.NewPerfUnavailable("accounts", strings.TrimSpace(err.Error()))
	}
	accounts := make([]*dbi.AccountInfo, 0, len(rows))
	for _, row := range rows {
		name := cast.ToString(row["User"])
		accounts = append(accounts, &dbi.AccountInfo{
			Name:            name,
			Builtin:         isBuiltinMysqlAccount(name),
			Host:            cast.ToString(row["Host"]),
			CanLogin:        true,
			Superuser:       strings.EqualFold(cast.ToString(row["Super_priv"]), "Y"),
			CanGrant:        strings.EqualFold(cast.ToString(row["Grant_priv"]), "Y"),
			Locked:          strings.EqualFold(cast.ToString(row["account_locked"]), "Y"),
			PasswordExpired: strings.EqualFold(cast.ToString(row["password_expired"]), "Y"),
			LastChanged:     cast.ToString(row["last_changed"]),
		})
	}
	return accounts, nil
}

// isBuiltinMysqlAccount MySQL 内置账号：由数据库自己创建、默认锁定，不能删也不该改。
// 判定放在方言里，应用层只消费 AccountInfo.Builtin——命名约定属于方言知识。
func isBuiltinMysqlAccount(name string) bool {
	return strings.HasPrefix(name, "mysql.")
}
