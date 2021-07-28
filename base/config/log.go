package config

import "path"

type Log struct {
	Level string   `yaml:"level"`
	File  *LogFile `yaml:"file"`
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
