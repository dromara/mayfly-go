package config

import (
	"flag"
	"fmt"
	"mayfly-go/pkg/utils"
	"mayfly-go/pkg/utils/assert"
	"os"
	"path/filepath"
)

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
	if err := utils.LoadYml(startConfigParam.ConfigFilePath, yc); err != nil {
		panic(fmt.Sprintf("读取配置文件[%s]失败: %s", startConfigParam.ConfigFilePath, err.Error()))
	}
	// 校验配置文件内容信息
	yc.Valid()
	// 尝试使用系统环境变量替换配置信息
	yc.ReplaceOsEnv()
	Conf = yc
}

// 启动配置参数
type CmdConfigParam struct {
	ConfigFilePath string // -e  配置文件路径
}

// yaml配置文件映射对象
type Config struct {
	Server *Server `yaml:"server"`
	Jwt    *Jwt    `yaml:"jwt"`
	Aes    *Aes    `yaml:"aes"`
	Redis  *Redis  `yaml:"redis"`
	Mysql  *Mysql  `yaml:"mysql"`
	Log    *Log    `yaml:"log"`
}

// 配置文件内容校验
func (c *Config) Valid() {
	assert.IsTrue(c.Jwt != nil, "配置文件的[jwt]信息不能为空")
	c.Jwt.Valid()
	if c.Aes != nil {
		c.Aes.Valid()
	}
}

// 替换系统环境变量，如果环境变量中存在该值，则优秀使用环境变量设定的值
func (c *Config) ReplaceOsEnv() {
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

	aesKey := os.Getenv("MAYFLY_AES_KEY")
	if aesKey != "" {
		c.Aes.Key = aesKey
	}

	jwtKey := os.Getenv("MAYFLY_JWT_KEY")
	if jwtKey != "" {
		c.Jwt.Key = jwtKey
	}
}
