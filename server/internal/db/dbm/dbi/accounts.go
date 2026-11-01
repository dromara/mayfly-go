package dbi

import "context"

// 实例账号盘点：只读，回答「这台库上有哪些账号、哪些是高危」。
//
// 刻意不做写操作（CREATE USER / GRANT / ALTER）：
//   - 账号是数据库里权限最大的对象，平台代改一旦出错就是事故；
//   - 各方言账号模型差异极大（PG 的 role 同时是用户与组、MySQL 是 user@host 二元组），
//     抽象成统一写接口会造出一堆「看起来一样、语义不同」的坑。
//
// 读不到（权限不足/视图不可见）一律返回 ErrPerfUnavailable + 原因，
// **不返回空列表**：空列表在界面上等于「这台库没有账号」，与事实相反。

// AccountInfo 一个数据库账号的事实
type AccountInfo struct {
	// Name 账号名（MySQL 为 User，PG 为 rolname）
	Name string `json:"name"`
	// Host MySQL 的登录来源；PG 无此概念，留空
	Host string `json:"host,omitempty"`
	// CanLogin 能否用于登录（PG 的 rolcanlogin；MySQL 恒为 true）
	CanLogin bool `json:"canLogin"`
	// Locked 账号被锁定（MySQL account_locked；PG 无锁定概念，恒 false）
	Locked bool `json:"locked"`
	// PasswordExpired 密码已过期（MySQL password_expired；PG 无此标志，恒 false）
	PasswordExpired bool `json:"passwordExpired"`
	// Superuser 具备超级权限（MySQL Super_priv；PG rolsuper）
	Superuser bool `json:"superuser"`
	// CanGrant 具备授权能力（MySQL Grant_priv；PG rolcreaterole）
	CanGrant bool `json:"canGrant"`
	// NeverExpire PG 的 rolvaliduntil 为空（永不失效）；MySQL 留空
	NeverExpire bool `json:"neverExpire"`
	// CreatedAt 账号创建/最后改密时间（能拿到就给，用于「多久没换密码」）
	LastChanged string `json:"lastChanged,omitempty"`
	// Builtin 数据库内置账号（MySQL 的 mysql.sys / mysql.session / mysql.infoschema），
	// 由方言判定：命名约定属于方言知识，不能让应用层按前缀猜
	Builtin bool `json:"builtin,omitempty"`
}

// AccountInspector 实例账号盘点（可选能力，方言声明 FeatAccounts 后实现）
type AccountInspector interface {
	Accounts(ctx context.Context) ([]*AccountInfo, error)
}
