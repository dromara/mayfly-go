package config

import "mayfly-go/pkg/logx"

type Sqlite struct {
	Path         string `mapstructure:"path" json:"path" yaml:"path"`
	MaxIdleConns int    `mapstructure:"max-idle-conns" json:"maxIdleConns" yaml:"max-idle-conns"`
	MaxOpenConns int    `mapstructure:"max-open-conns" json:"maxOpenConns" yaml:"max-open-conns"`
}

func (m *Sqlite) Default() {
	if m.Path == "" {
		m.Path = "./mayfly-go.sqlite"
		logx.Warnf("[使用mysql可忽略]未配置sqlite.path, 默认值: %s", m.Path)
	}
	if m.MaxIdleConns == 0 {
		m.MaxIdleConns = 5
	}
	if m.MaxOpenConns == 0 {
		m.MaxOpenConns = m.MaxIdleConns
	}
}
