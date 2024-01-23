package logx

import (
	"io"
	"log/slog"
	"os"
	"path"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
	Level     string
	Type      string // 日志类型；text、json
	AddSource bool   // 是否记录调用方法

	Filename string // 日志文件名
	Filepath string // 日志路径
	MaxSize  int    // 日志文件的最大大小（以兆字节为单位）。当日志文件大小达到该值时，将触发切割操作，默认为 100 兆字节
	MaxAge   int    // 根据文件名中的时间戳，设置保留旧日志文件的最大天数。一天被定义为 24 小时
	Compress bool   // 是否使用 gzip 压缩方式压缩轮转后的日志文件

	writer io.Writer
}

// 获取日志输出源
func (c *Config) GetLogOut() io.Writer {
	if c.writer != nil {
		return c.writer
	}
	var writer io.Writer
	writer = os.Stdout
	// 根据配置文件设置日志级别
	if c.Filepath != "" && c.Filename != "" {
		// 写入文件
		writer = &lumberjack.Logger{
			Filename:  path.Join(c.Filepath, c.Filename),
			MaxSize:   c.MaxSize,
			MaxAge:    c.MaxAge,
			Compress:  c.Compress,
			LocalTime: true,
		}
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
