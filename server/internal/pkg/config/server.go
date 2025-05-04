package config

import (
	"fmt"
	"mayfly-go/pkg/i18n"
)

type Server struct {
	Lang        string         `yaml:"lang"`
	Port        int            `yaml:"port"`
	Model       string         `yaml:"model"`
	ContextPath string         `yaml:"context-path"` // 请求路径上下文
	Cors        bool           `yaml:"cors"`
	Tls         *Tls           `yaml:"tls"`
	Static      *[]*Static     `yaml:"static"`
	StaticFile  *[]*StaticFile `yaml:"static-file"`
}

func (s *Server) Default() {
	if s.Lang == "" {
		s.Lang = i18n.Zh_CN
	}
	if s.Model == "" {
		s.Model = "release"
	}
	if s.Port == 0 {
		s.Port = 18888
	}
}

func (s *Server) GetPort() string {
	return fmt.Sprintf(":%d", s.Port)
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
