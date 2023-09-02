package config

import (
	"mayfly-go/pkg/logx"
	"path"
)

type Log struct {
	Level     string  `yaml:"level"`
	Type      string  `yaml:"type"`
	AddSource bool    `yaml:"add-source"`
	File      LogFile `yaml:"file"`
}

func (l *Log) Default() {
	if l.Level == "" {
		l.Level = "info"
		logx.Warnf("未配置log.level, 默认值: %s", l.Level)
	}
	if l.Type == "" {
		l.Type = "text"
	}
	if l.File.Name == "" {
		l.File.Name = "mayfly-go.log"
	}
}

type LogFile struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

// 获取完整路径文件名
func (l *LogFile) GetFilename() string {
	var filepath, filename string
	if fp := l.Path; fp == "" {
		filepath = "./"
	} else {
		filepath = fp
	}
	if fn := l.Name; fn == "" {
		filename = "default.log"
	} else {
		filename = fn
	}

	return path.Join(filepath, filename)
}
