package config

import "fmt"

type Server struct {
	Port       int            `yaml:"port"`
	Model      string         `yaml:"model"`
	Cors       bool           `yaml:"cors"`
	Static     *[]*Static     `yaml:"static"`
	StaticFile *[]*StaticFile `yaml:"static-file"`
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
