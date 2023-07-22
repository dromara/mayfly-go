package config

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

func (m *Mysql) Dsn() string {
	return m.Username + ":" + m.Password + "@tcp(" + m.Host + ")/" + m.Dbname + "?" + m.Config
}
