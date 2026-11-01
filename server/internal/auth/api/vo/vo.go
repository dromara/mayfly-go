package vo

type Oauth2Status struct {
	Enable bool `json:"enable"`
	Bind   bool `json:"bind"`
}

// LdapEnabled LDAP 登录开关
//
// 必须是具名结构体而不能退化成裸 bool：登录页按 { enabled } 取值，若响应体是裸标量，
// 前端只能读到 undefined（falsy）→ 勾选框永不渲染且登录永远走本地账号。带字段名后
// 契约可被 scripts/api-contract-audit 静态比对，裸标量比不出来
type LdapEnabled struct {
	Enabled bool `json:"enabled"`
}
