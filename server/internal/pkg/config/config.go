package config

import (
	"flag"
	"fmt"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/starter"
	"mayfly-go/pkg/utils/ymlx"
	"os"
	"path/filepath"
	"strconv"
)

// 配置文件映射对象
var Conf *Config

func Init() (*Config, error) {
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

	if err := yc.ApplyDefaults(); err != nil {
		return nil, err
	}

	Conf = yc
	return yc, nil
}

// 启动配置参数
type CmdConfigParam struct {
	ConfigFilePath string // -e  配置文件路径
}

// yaml配置文件映射对象
type Config struct {
	starter.Conf `yaml:",inline"`

	Aes Aes `yaml:"aes"`
}

var _ starter.ConfigItem = (*Config)(nil)

func (c *Config) ApplyDefaults() error {
	if err := c.Conf.ApplyDefaults(); err != nil {
		return err
	}

	if err := c.Aes.ApplyDefaults(); err != nil {
		return err
	}

	return nil
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
		c.DB.Address = dbHost
		c.DB.Dialect = starter.DialectMySQL
	}

	dbName := os.Getenv("MAYFLY_DB_NAME")
	if dbName != "" {
		c.DB.Name = dbName
	}

	dbUser := os.Getenv("MAYFLY_DB_USER")
	if dbUser != "" {
		c.DB.Username = dbUser
	}

	dbPwd := os.Getenv("MAYFLY_DB_PASS")
	if dbPwd != "" {
		c.DB.Password = dbPwd
	}

	sqlitePath := os.Getenv("MAYFLY_SQLITE_PATH")
	if sqlitePath != "" {
		c.DB.Address = sqlitePath
		c.DB.Dialect = starter.DialectSQLite
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
