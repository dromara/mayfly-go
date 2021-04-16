package config

import (
	"flag"
	"fmt"
	"mayfly-go/base/utils"
	"path/filepath"
)

func init() {
	configFilePath := flag.String("e", "./config.yml", "配置文件路径，默认为可执行文件目录")
	flag.Parse()
	// 获取启动参数中，配置文件的绝对路径
	path, _ := filepath.Abs(*configFilePath)
	startConfigParam = &CmdConfigParam{ConfigFilePath: path}
	// 读取配置文件信息
	yc := &Config{}
	if err := utils.LoadYml(startConfigParam.ConfigFilePath, yc); err != nil {
		panic(fmt.Sprintf("读取配置文件[%s]失败: %s", startConfigParam.ConfigFilePath, err.Error()))
	}
	Conf = yc
}

// 启动配置参数
type CmdConfigParam struct {
	ConfigFilePath string // -e  配置文件路径
}

// 启动可执行文件时的参数
var startConfigParam *CmdConfigParam

// yaml配置文件映射对象
type Config struct {
	App    *App    `yaml:"app"`
	Server *Server `yaml:"server"`
	Redis  *Redis  `yaml:"redis"`
	Mysql  *Mysql  `yaml:"mysql"`
}

// 配置文件映射对象
var Conf *Config

// 获取执行可执行文件时，指定的启动参数
func getStartConfig() *CmdConfigParam {
	configFilePath := flag.String("e", "./config.yml", "配置文件路径，默认为可执行文件目录")
	flag.Parse()
	// 获取配置文件绝对路径
	path, _ := filepath.Abs(*configFilePath)
	sc := &CmdConfigParam{ConfigFilePath: path}
	return sc
}
