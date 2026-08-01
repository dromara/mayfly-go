package i18n

import (
	"bytes"
	"os"
	"strings"
	"text/template"
)

// 语言常量
const (
	ZhCN = "zh-cn"
	En   = "en"
)

// 当前语言（默认中文）
var currentLang = ZhCN

// 语言消息映射：lang -> msgId -> message
var langMsgs = map[string]map[string]string{
	ZhCN: zhCN,
	En:   en,
}

// i18n 包在初始化时即确定语言，保证依赖方（如 cobra 命令描述）
// 在包变量初始化阶段调用 T() 就能得到正确语言的文本
func init() {
	InitLangFromArgs()
}

// SetLang 设置当前语言
func SetLang(lang string) {
	if lang == "" {
		return
	}
	currentLang = strings.ToLower(strings.ReplaceAll(lang, "_", "-"))
}

// GetLang 获取当前语言
func GetLang() string {
	return currentLang
}

// InitLang 根据环境变量初始化语言
func InitLang() {
	// 优先使用环境变量
	if lang := os.Getenv("MAYFLY_LANG"); lang != "" {
		SetLang(lang)
		return
	}
	// 尝试从 LANG 环境变量推断
	if lang := os.Getenv("LANG"); lang != "" {
		if strings.Contains(strings.ToLower(lang), "zh") {
			currentLang = ZhCN
		} else {
			currentLang = En
		}
	}
}

// InitLangFromArgs 早期语言初始化（扫描 os.Args 中的 --lang 标志）
// 应在 init() 中调用，以确保命令描述使用正确的语言
func InitLangFromArgs() {
	// 先检查环境变量
	InitLang()
	// 然后扫描命令行参数（优先于环境变量）
	args := os.Args
	for i := 1; i < len(args); i++ {
		if args[i] == "--lang" && i+1 < len(args) {
			SetLang(args[i+1])
			return
		}
		if strings.HasPrefix(args[i], "--lang=") {
			SetLang(strings.TrimPrefix(args[i], "--lang="))
			return
		}
	}
}

// T 翻译消息（支持模板变量）
// 用法: T(MsgDbListFailed, "err", err.Error())
func T(msgId string, attrs ...any) string {
	msgs := langMsgs[currentLang]
	if msgs == nil {
		msgs = langMsgs[ZhCN]
	}

	msg, ok := msgs[msgId]
	if !ok {
		// 回退到中文
		if msgs = langMsgs[ZhCN]; msgs != nil {
			msg = msgs[msgId]
		}
	}
	if msg == "" {
		return msgId
	}

	// 无模板变量时直接返回，避免不必要的模板解析
	if len(attrs) == 0 || !strings.Contains(msg, "{{") {
		return msg
	}

	// 模板变量替换
	if len(attrs)%2 != 0 {
		return msg
	}
	vars := make(map[string]any, len(attrs)/2)
	for i := 0; i < len(attrs); i += 2 {
		key, ok := attrs[i].(string)
		if !ok {
			continue
		}
		vars[key] = attrs[i+1]
	}

	tmpl, err := template.New("i18n").Parse(msg)
	if err != nil {
		return msg
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return msg
	}
	return buf.String()
}
