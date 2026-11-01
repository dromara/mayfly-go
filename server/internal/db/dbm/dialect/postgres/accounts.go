package postgres

// postgres 账号盘点：读 pg_roles。
//
// 与 MySQL 的差别写在这里而不是藏起来：
//   - PG 没有 user@host，账号就是 role；
//   - 没有 account_locked，禁用登录的做法是 NOLOGIN；
//   - 没有 password_expired，只有 rolvaliduntil（为空=永不过期）。

import (
	"context"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/spf13/cast"
)

func pgAccountStr(value any) string {
	return cast.ToString(value)
}

func pgAccountBool(value any) bool {
	switch cast.ToString(value) {
	case "true", "t", "1":
		return true
	}
	return false
}

var _ dbi.AccountInspector = (*PgsqlMetadata)(nil)

// Accounts 列出角色
func (md *PgsqlMetadata) Accounts(ctx context.Context) ([]*dbi.AccountInfo, error) {
	_, rows, err := md.di.QueryContext(ctx,
		"SELECT rolname, rolsuper, rolcanlogin, rolcreaterole,"+
			" COALESCE(rolvaliduntil::text, '') valid_until FROM pg_roles ORDER BY rolname")
	if err != nil {
		return nil, dbi.NewPerfUnavailable("accounts", strings.TrimSpace(err.Error()))
	}
	accounts := make([]*dbi.AccountInfo, 0, len(rows))
	for _, row := range rows {
		validUntil := strings.TrimSpace(pgAccountStr(row["valid_until"]))
		accounts = append(accounts, &dbi.AccountInfo{
			Name:        pgAccountStr(row["rolname"]),
			CanLogin:    pgAccountBool(row["rolcanlogin"]),
			Superuser:   pgAccountBool(row["rolsuper"]),
			CanGrant:    pgAccountBool(row["rolcreaterole"]),
			NeverExpire: validUntil == "",
			LastChanged: validUntil,
		})
	}
	return accounts, nil
}
