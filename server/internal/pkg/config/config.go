package config

import (
	"flag"
	"fmt"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/ymlx"
	"os"
	"path/filepath"
	"strconv"
)

type ConfigItem interface {
	// 验证配置
	Valid()

	// 如果不存在配置值，则设置默认值
	Default()
}

// 配置文件映射对象
var Conf *Config

func Init() {
	configFilePath := flag.String("e", "./config.yml", "配置文件路径，默认为可执行文件目录")
	flag.Parse()
	// 获取启动参数中，配置文件的绝对路径
	path, _ := filepath.Abs(*configFilePath)
	startConfigParam := &CmdConfigParam{ConfigFilePath: path}
	// 读取配置文件信息
	yc := &Config{}
	if err := ymlx.LoadYml(startConfigParam.ConfigFilePath, yc); err != nil {
		logx.Warn(fmt.Sprintf("读取配置文件[%s]失败: %s, 使用系统默认配置或环境变量配置", startConfigParam.ConfigFilePath, err.Error()))
	}
	// 尝试使用系统环境变量替换配置信息
	yc.ReplaceOsEnv()

	yc.IfBlankDefaultValue()
	// 校验配置文件内容信息
	yc.Valid()
	Conf = yc
}

// 启动配置参数
type CmdConfigParam struct {
	ConfigFilePath string // -e  配置文件路径
}

// yaml配置文件映射对象
type Config struct {
	Server Server `yaml:"server"`
	Jwt    Jwt    `yaml:"jwt"`
	Aes    Aes    `yaml:"aes"`
	Mysql  Mysql  `yaml:"mysql"`
	Sqlite Sqlite `yaml:"sqlite"`
	Redis  Redis  `yaml:"redis"`
	Log    Log    `yaml:"log"`
}

func (c *Config) IfBlankDefaultValue() {
	c.Log.Default()
	// 优先初始化log，因为后续的一些default方法中会需要用到。统一日志输出
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

	c.Server.Default()
	c.Jwt.Default()
	c.Mysql.Default()
	c.Sqlite.Default()
}

// 配置文件内容校验
func (c *Config) Valid() {
	c.Jwt.Valid()
	c.Aes.Valid()
}

// 替换系统环境变量，如果环境变量中存在该值，则优先使用环境变量设定的值
func (c *Config) ReplaceOsEnv() {
	serverPort := os.Getenv("MAYFLY_SERVER_PORT")
	if serverPort != "" {
		if num, err := strconv.Atoi(serverPort); err != nil {
			panic("环境变量-[MAYFLY_SERVER_PORT]-服务端口号需为数字")
		} else {
			c.Server.Port = num
		}
	}

	dbHost := os.Getenv("MAYFLY_DB_HOST")
	if dbHost != "" {
		c.Mysql.Host = dbHost
	}

	dbName := os.Getenv("MAYFLY_DB_NAME")
	if dbName != "" {
		c.Mysql.Dbname = dbName
	}

	dbUser := os.Getenv("MAYFLY_DB_USER")
	if dbUser != "" {
		c.Mysql.Username = dbUser
	}

	dbPwd := os.Getenv("MAYFLY_DB_PASS")
	if dbPwd != "" {
		c.Mysql.Password = dbPwd
	}

	sqlitePath := os.Getenv("MAYFLY_SQLITE_PATH")
	if sqlitePath != "" {
		c.Sqlite.Path = sqlitePath
	}

	aesKey := os.Getenv("MAYFLY_AES_KEY")
	if aesKey != "" {
		c.Aes.Key = aesKey
	}

	jwtKey := os.Getenv("MAYFLY_JWT_KEY")
	if jwtKey != "" {
		c.Jwt.Key = jwtKey
	}
}
