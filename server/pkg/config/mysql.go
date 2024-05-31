package config

import "mayfly-go/pkg/logx"

type Mysql struct {
	AutoMigration bool   `mapstructure:"auto-migration" json:"autoMigration" yaml:"auto-migration"`
	Host          string `mapstructure:"path" json:"host" yaml:"host"`
	Config        string `mapstructure:"config" json:"config" yaml:"config"`
	Dbname        string `mapstructure:"db-name" json:"dbname" yaml:"db-name"`
	Username      string `mapstructure:"username" json:"username" yaml:"username"`
	Password      string `mapstructure:"password" json:"password" yaml:"password"`
	MaxIdleConns  int    `mapstructure:"max-idle-conns" json:"maxIdleConns" yaml:"max-idle-conns"`
	MaxOpenConns  int    `mapstructure:"max-open-conns" json:"maxOpenConns" yaml:"max-open-conns"`
	LogMode       bool   `mapstructure:"log-mode" json:"logMode" yaml:"log-mode"`
	LogZap        string `mapstructure:"log-zap" json:"logZap" yaml:"log-zap"`
}

func (m *Mysql) Default() {
	if m.Host == "" {
		m.Host = "localhost:3306"
		logx.Warnf("[使用sqlite可忽略]未配置mysql.host, 默认值: %s", m.Host)
	}
	if m.Config == "" {
		m.Config = "charset=utf8&loc=Local&parseTime=true"
	}
	if m.MaxIdleConns == 0 {
		m.MaxIdleConns = 5
	}
	if m.MaxOpenConns == 0 {
		m.MaxOpenConns = m.MaxIdleConns
	}
}

func (m *Mysql) Dsn() string {
	return m.Username + ":" + m.Password + "@tcp(" + m.Host + ")/" + m.Dbname + "?" + m.Config
}
