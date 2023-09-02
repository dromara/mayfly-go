package logx

import (
	"context"
	"fmt"
	"log"
	"log/slog"
)

type CustomeTextHandlerOptions struct {
	SlogOpts slog.HandlerOptions
}

type CustomeTextHandler struct {
	slog.Handler
	l *log.Logger
}

func (h *CustomeTextHandler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String()
	timeStr := r.Time.Format("2006-01-02 15:04:05.000")

	attrsStr := ""
	r.Attrs(func(a slog.Attr) bool {
		// 如果是source，则忽略key，简洁些
		if a.Key == slog.SourceKey {
			attrsStr += fmt.Sprintf("[%s]", a.Value.Any())
			return true
		}
		attrsStr += fmt.Sprintf("[%s=%v]", a.Key, a.Value.Any())
		return true
	})
	if attrsStr != "" {
		attrsStr = " " + attrsStr
	}

	// 格式为：time [level] [key=value][key2=value2] : message
	h.l.Printf("%s [%s]%s : %s", timeStr, level, attrsStr, r.Message)
	return nil
}

func NewTextHandler(config *Config) *CustomeTextHandler {
	opts := CustomeTextHandlerOptions{
		SlogOpts: slog.HandlerOptions{
			Level:     config.GetLevel(),
			AddSource: false, // 统一由添加公共commonAttrs时判断添加
		}}

	out := config.GetLogOut()
	return &CustomeTextHandler{
		Handler: slog.NewTextHandler(out, &opts.SlogOpts),
		l:       log.New(out, "", 0),
	}
}
