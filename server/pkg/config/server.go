package config

import "fmt"

type Server struct {
	Port           int            `yaml:"port"`
	Model          string         `yaml:"model"`
	Cors           bool           `yaml:"cors"`
	Tls            *Tls           `yaml:"tls"`
	Static         *[]*Static     `yaml:"static"`
	StaticFile     *[]*StaticFile `yaml:"static-file"`
	MachineRecPath string         `yaml:"machine-rec-path"` // 机器终端操作回放文件存储路径
}

func (s *Server) GetPort() string {
	return fmt.Sprintf(":%d", s.Port)
}

// 获取终端回访记录存放基础路径, 如果配置文件未配置，则默认为./rec
func (s *Server) GetMachineRecPath() string {
	path := s.MachineRecPath
	if path == "" {
		return "./rec"
	}
	return path
}

type Static struct {
	RelativePath string `yaml:"relative-path"`
	Root         string `yaml:"root"`
}

type StaticFile struct {
	RelativePath string `yaml:"relative-path"`
	Filepath     string `yaml:"filepath"`
}

type Tls struct {
	Enable   bool   `yaml:"enable"`    // 是否启用tls
	KeyFile  string `yaml:"key-file"`  // 私钥文件路径
	CertFile string `yaml:"cert-file"` // 证书文件路径
}
