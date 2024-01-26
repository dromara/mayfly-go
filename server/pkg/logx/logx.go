package logx

import (
	"context"
	"fmt"
	"log/slog"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/utils/runtimex"
	"path/filepath"
	"runtime"
)

var (
	config *Config
)

func GetConfig() *Config {
	if config == nil {
		return &Config{
			Level:     "info",
			Type:      "text",
			AddSource: false,
		}
	}
	return config
}

func Init(logConf Config) {
	config = &logConf
	var handler slog.Handler
	if logConf.IsJsonType() {
		handler = NewJsonHandler(config)
	} else {
		handler = NewTextHandler(config)
	}
	slog.SetDefault(slog.New(handler))
}

func Print(msg string, args ...any) {
	Log(context.Background(), slog.LevelInfo, msg, args...)
}

func Debug(msg string, args ...any) {
	Log(context.Background(), slog.LevelDebug, msg, args...)
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	Log(ctx, slog.LevelDebug, msg, args...)
}

func Debugf(format string, args ...any) {
	Log(context.Background(), slog.LevelDebug, fmt.Sprintf(format, args...))
}

func DebugWithFields(ctx context.Context, msg string, mapFields map[string]any) {
	Log(context.Background(), slog.LevelDebug, msg, map2Attrs(mapFields)...)
}

// debug记录，并将堆栈信息添加至msg里，默认记录10个堆栈信息
func DebugTrace(msg string, err error) {
	Log(context.Background(), slog.LevelDebug, fmt.Sprintf(msg+" %s\n%s", err.Error(), runtimex.StatckStr(2, 10)))
}

func Info(msg string, args ...any) {
	Log(context.Background(), slog.LevelInfo, msg, args...)
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	Log(ctx, slog.LevelInfo, msg, args...)
}

func Infof(format string, args ...any) {
	Log(context.Background(), slog.LevelInfo, fmt.Sprintf(format, args...))
}

func InfoWithFields(ctx context.Context, msg string, mapFields map[string]any) {
	Log(ctx, slog.LevelInfo, msg, map2Attrs(mapFields)...)
}

func Warn(msg string, args ...any) {
	Log(context.Background(), slog.LevelWarn, msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	Log(ctx, slog.LevelWarn, msg, args...)
}

func Warnf(format string, args ...any) {
	Log(context.Background(), slog.LevelWarn, fmt.Sprintf(format, args...))
}

func WarnWithFields(msg string, mapFields map[string]any) {
	Log(context.Background(), slog.LevelWarn, msg, map2Attrs(mapFields)...)
}

func Error(msg string, args ...any) {
	Log(context.Background(), slog.LevelError, msg, args...)
}

func ErrorContext(ctx context.Context, msg string, args ...any) {
	Log(ctx, slog.LevelError, msg, args...)
}

func Errorf(format string, args ...any) {
	Log(context.Background(), slog.LevelError, fmt.Sprintf(format, args...))
}

// 错误记录，并将堆栈信息添加至msg里，默认记录10个堆栈信息
func ErrorTrace(msg string, err any) {
	errMsg := ""
	switch t := err.(type) {
	case error:
		errMsg = t.Error()
	case string:
		errMsg = t
	default:
		errMsg = fmt.Sprintf("%v", t)
	}
	Log(context.Background(), slog.LevelError, fmt.Sprintf(msg+"\n%s\n%s", errMsg, runtimex.StatckStr(2, 20)))
}

func ErrorWithFields(ctx context.Context, msg string, mapFields map[string]any) {
	Log(ctx, slog.LevelError, msg, map2Attrs(mapFields)...)
}

func Panic(msg string, args ...any) {
	Log(context.Background(), slog.LevelError, msg, args...)
	panic(msg)
}

func Panicf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	Log(context.Background(), slog.LevelError, fmt.Sprintf(format, args...))
	panic(msg)
}

func Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	slog.Log(ctx, level, msg, appendCommonAttr(ctx, level, args)...)
}

// 获取日志公共属性
func getCommonAttr(ctx context.Context, level slog.Level) []any {
	commonAttrs := make([]any, 0)

	// 尝试从上下文获取traceId，若存在则记录
	if traceId := contextx.GetTraceId(ctx); traceId != "" {
		commonAttrs = append(commonAttrs, "tid", traceId)
	}
	// 如果系统配置添加方法信息或者为错误级别时则 记录方法信息及行号
	if GetConfig().AddSource || level == slog.LevelError {
		// skip [runtime.Callers, getCommonAttr, appendCommonAttr, logx.Log, logx.Info|Debug|Warn|Error..]
		var pcs [1]uintptr
		runtime.Callers(5, pcs[:])
		fs := runtime.CallersFrames(pcs[:])
		f, _ := fs.Next()

		source := &Source{
			Function: f.Function,
			Fileline: fmt.Sprintf("%s:%d", filepath.Base(f.File), f.Line),
		}
		commonAttrs = append(commonAttrs, slog.SourceKey, source)
	}

	return commonAttrs
}

func appendCommonAttr(ctx context.Context, level slog.Level, args []any) []any {
	commonAttrs := getCommonAttr(ctx, level)
	if len(commonAttrs) > 0 {
		args = append(commonAttrs, args...)
	}
	return args
}

// map类型转为attr
func map2Attrs(mapArg map[string]any) []any {
	atts := make([]any, 0)
	for k, v := range mapArg {
		atts = append(atts, slog.Any(k, v))
	}
	return atts
}

type Source struct {
	// Function is the package path-qualified function name containing the
	// source line. If non-empty, this string uniquely identifies a single
	// function in the program. This may be the empty string if not known.
	Function string `json:"function"`
	// File and Line are the file name and line number (1-based) of the source
	// line. These may be the empty string and zero, respectively, if not known.
	Fileline string `json:"fileline"`
}

func (s Source) String() string {
	return fmt.Sprintf("%s (%s)", s.Function, s.Fileline)
}

// An Attr is a key-value pair.
type Attr = slog.Attr

// String returns an Attr for a string value.
func String(key, value string) Attr {
	return slog.String(key, value)
}

// Int64 returns an Attr for an int64.
func Int64(key string, value int64) Attr {
	return slog.Int64(key, value)
}

// Bool returns an Attr for an bool.
func Bool(key string, value bool) Attr {
	return slog.Bool(key, value)
}
