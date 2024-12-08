package i18n

import (
	"context"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
)

type MsgId int

const (
	Zh_CN = "zh-cn"
	En    = "en"
)

// langMsgs key=lang, value=msgs
var langMsgs = make(map[string]map[MsgId]string)

var defaultLang = Zh_CN

// AppendLangMsg append lang msg
func AppendLangMsg(lang string, msgs map[MsgId]string) {
	langMsgs[lang] = collx.MapMerge(langMsgs[lang], msgs)
}

// SetLang set default lang
func SetLang(lang string) {
	defaultLang = lang
}

// T load msg by key, and use default lang
//
//	// NameErr =  {{.name}} is invalid  =>  xxx is invalid
//	T(imsg.NameErr, "name", "xxxx")
func T(msgId MsgId, attrs ...any) string {
	return TL(defaultLang, msgId, attrs...)
}

// TC load msg by key, and use context lang
func TC(ctx context.Context, msgId MsgId, attrs ...any) string {
	return TL(GetLangFromCtx(ctx), msgId, attrs...)
}

// TL load msg by lang
//
//	// NameErr =  {{.name}} is invalid  =>  xxx is invalid
//	T(imsg.NameErr, "name", "xxxx")
func TL(lang string, msgId MsgId, attrs ...any) string {
	if lang == "" {
		lang = defaultLang
	}

	msgs := langMsgs[lang]
	if msgs == nil {
		msgs = langMsgs[defaultLang]
	}

	msg := msgs[msgId]
	if len(attrs) == 0 {
		return msg
	}

	if parseMsg, err := stringx.TemplateParse(msg, collx.Kvs(attrs...)); err != nil {
		return msg
	} else {
		return parseMsg
	}
}
