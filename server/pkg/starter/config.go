package starter

import (
	"fmt"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/stringx"
)

type Conf struct {
	Server ServerConf `yaml:"server"`
	DB     DBConf     `yaml:"db"`
	Jwt    JwtConf    `yaml:"jwt"`
	Log    LogConf    `yaml:"log"`
	Redis  RedisConf  `yaml:"redis"`

	isApplyDefaults bool
}

func (c *Conf) ApplyDefaults() error {
	if c.isApplyDefaults {
		return nil
	}

	if err := c.Log.ApplyDefaults(); err != nil {
		return err
	}
	// 优先初始化log，因为后续的一些配置ApplyDefaults方法中可能会用到
	logx.Init(logx.Config{
		Level:     c.Log.Level,
		Type:      c.Log.Type,
		AddSource: c.Log.AddSource,
		Filename:  c.Log.File.Name,
		Filepath:  c.Log.File.Path,
		MaxSize:   c.Log.File.MaxSize,
		MaxAge:    c.Log.File.MaxAge,
		Compress:  c.Log.File.Compress,
	})

	if err := ApplyConfigDefaults(c); err != nil {
		return err
	}

	c.isApplyDefaults = true
	return nil
}

/************************ server ************************/

// ServerConf 配置
type ServerConf struct {
	Lang        string `yaml:"lang" default:"zh-cn" options:"zh-cn,en"`
	Port        int    `yaml:"port" default:"18888"`
	Model       string `yaml:"model" default:"release" options:"release,debug"`
	ContextPath string `yaml:"context-path"` // 请求路径上下文
	Cors        bool   `yaml:"cors"`

	// TLS 配置
	TLS struct {
		Enable   bool   `yaml:"enable"`    // 是否启用tls
		KeyFile  string `yaml:"key-file"`  // 私钥文件路径
		CertFile string `yaml:"cert-file"` // 证书文件路径
	} `yaml:"tls"`

	Statics []struct {
		RelativePath string `yaml:"relative-path"`
		Root         string `yaml:"root"`
	} `yaml:"statics"`

	StaticFiles []struct {
		RelativePath string `yaml:"relative-path"`
		Filepath     string `yaml:"filepath"`
	} `yaml:"static-files"`
}

func (s *ServerConf) GetPort() string {
	return fmt.Sprintf(":%d", s.Port)
}

/************************ db ************************/

type DbDialect string

const (
	DialectMySQL  DbDialect = "mysql"
	DialectSQLite DbDialect = "sqlite"
)

// DBConf 配置
type DBConf struct {
	Dialect      DbDialect `yaml:"dialect" default:"sqlite" options:"mysql,sqlite"` // 数据库类型
	Address      string    `yaml:"address" default:"mayfly-go.db"`                  // 地址
	Name         string    `yaml:"name"`                                            // 数据库名
	Username     string    `yaml:"username"`
	Password     string    `yaml:"password"`
	Config       string    `yaml:"config"` // 额外配置，如 charset=utf8&loc=Local&parseTime=true
	MaxIdleConns int       `yaml:"max-idle-conns" default:"5"`
	MaxOpenConns int       `yaml:"max-open-conns"`
}

/************************ jwt ************************/

// JwtConf 配置
type JwtConf struct {
	Key                    string `yaml:"key"`
	ExpireTime             uint64 `yaml:"expire-time" default:"1440"`               // 过期时间，单位分钟
	RefreshTokenExpireTime uint64 `yaml:"refresh-token-expire-time" default:"7200"` // 刷新token的过期时间，单位分钟
}

var _ ConfigItem = (*JwtConf)(nil)

func (j *JwtConf) ApplyDefaults() error {
	if j.Key == "" {
		// 如果配置文件中的jwt key为空，则随机生成字符串
		j.Key = stringx.Rand(32)
		LogDefaultValue("jwt.key", j.Key)
	}

	return ApplyConfigDefaults(j, "jwt")
}

/************************ log ************************/

// LogConf 配置
type LogConf struct {
	Level     string `yaml:"level" default:"info" options:"debug,info,warn,error"`
	Type      string `yaml:"type" default:"text" options:"text,json"`
	AddSource bool   `yaml:"add-source"`

	File struct {
		Name     string `yaml:"name" default:"mayfly-go.log"`
		Path     string `yaml:"path"`
		MaxSize  int    `yaml:"max-size"`
		MaxAge   int    `yaml:"max-age"`
		Compress bool   `yaml:"compress"`
	} `yaml:"file"`
}

var _ ConfigItem = (*LogConf)(nil)

func (l *LogConf) ApplyDefaults() error {
	return ApplyConfigDefaults(l, "log")
}

/************************ redis ************************/

// RedisConf 配置
type RedisConf struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	Db       int    `yaml:"db"`
}
