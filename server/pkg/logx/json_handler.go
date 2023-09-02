package logx

import (
	"log/slog"
	"time"
)

func NewJsonHandler(config *Config) *slog.JSONHandler {
	replace := func(groups []string, a slog.Attr) slog.Attr {
		// 格式化时间.
		if a.Key == slog.TimeKey && len(groups) == 0 {
			return slog.Attr{Key: "time", Value: slog.StringValue(time.Now().Local().Format("2006-01-02 15:04:05.000"))}
		}
		return a
	}

	return slog.NewJSONHandler(config.GetLogOut(), &slog.HandlerOptions{
		Level:       config.GetLevel(),
		AddSource:   false, // 统一由添加公共commonAttrs时判断添加
		ReplaceAttr: replace,
	})
}
