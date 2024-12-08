package i18n

import "context"

type CtxKey string

const (
	LangKey CtxKey = "lang"
)

// NewCtxWithLang 将lang放置context中
func NewCtxWithLang(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, LangKey, lang)
}

// GetLangFromCtx 从context中获取lang
func GetLangFromCtx(ctx context.Context) string {
	if val, ok := ctx.Value(LangKey).(string); ok {
		return val
	}
	return ""
}
