package config

// FieldMapping 表示用户属性和 LDAP 字段名之间的映射关系
type FieldMapping struct {
	// Identifier 表示用户标识
	Identifier string `yaml:"identifier,omitempty"`
	// DisplayName 表示用户姓名
	DisplayName string `yaml:"displayName,omitempty"`
	// Email 表示 Email 地址
	Email string `yaml:"email,omitempty"`
}

// SecurityProtocol 表示连接 LDAP 服务器的安全协议
type SecurityProtocol string

const (
	// SecurityProtocolStartTLS 表示 StartTLS 安全协议
	SecurityProtocolStartTLS SecurityProtocol = "starttls"
	// SecurityProtocolLDAPS 表示 LDAPS 安全协议
	SecurityProtocolLDAPS SecurityProtocol = "ldaps"
)

// Ldap 是 LDAP 服务配置
type Ldap struct {
	// Enabled 表示是否启用 LDAP 登录
	Enabled bool `yaml:"enabled"`
	// Host 是 LDAP 服务地址, 如: "ldap.example.com"
	Host string `yaml:"host"`
	// Port 是 LDAP 服务端口号, 如: 389
	Port int `yaml:"port"`
	// SkipTLSVerify 控制客户端是否跳过 TLS 证书验证
	SkipTLSVerify bool `yaml:"skipTlsVerify"`
	// BindDN 是 LDAP 服务的管理员账号，如: "cn=admin,dc=example,dc=com"
	BindDN string `yaml:"bindDn"`
	// BindPassword 是 LDAP 服务的管理员密码
	BindPassword string `yaml:"bindPassword"`
	// BaseDN 是用户所在的 base DN, 如: "ou=users,dc=example,dc=com".
	BaseDN string `yaml:"baseDn"`
	// UserFilter 是过滤用户的方式, 如: "(uid=%s)".
	UserFilter string `yaml:"userFilter"`
	// SecurityProtocol 是连接使用的 LDAP 安全协议（为空不使用安全协议），如: StartTLS, LDAPS
	SecurityProtocol SecurityProtocol `yaml:"securityProtocol"`
	// FieldMapping 表示用户属性和 LDAP 字段名之间的映射关系
	FieldMapping FieldMapping `yaml:"fieldMapping"`
}
