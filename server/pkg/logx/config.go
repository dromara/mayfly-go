package logx

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"strings"
)

type Config struct {
	Level     string
	Type      string // 日志类型；text、json
	AddSource bool   // 是否记录调用方法
	Filename  string
	Filepath  string

	writer io.Writer
}

// 获取日志输出源
func (c *Config) GetLogOut() io.Writer {
	if c.writer != nil {
		return c.writer
	}
	writer := os.Stdout
	// 根据配置文件设置日志级别
	if c.Filepath != "" && c.Filename != "" {
		// 写入文件
		file, err := os.OpenFile(path.Join(c.Filepath, c.Filename), os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModeAppend|0666)
		if err != nil {
			panic(fmt.Sprintf("创建日志文件失败: %s", err.Error()))
		}
		writer = file
	}

	c.writer = writer
	return writer
}

// 获取日志级别
func (c *Config) GetLevel() slog.Level {
	switch strings.ToLower(c.Level) {
	case "error":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "info":
		return slog.LevelInfo
	case "debug":
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

func (c *Config) IsJsonType() bool {
	return c.Type == "json"
}

func (c *Config) IsDebug() bool {
	return strings.ToLower(c.Level) == "debug"
}
