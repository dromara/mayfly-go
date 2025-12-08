package prompt

import (
	"embed"
	"fmt"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/stringx"
	"strings"
)

const (
	FLOW_BIZ_AUDIT = "FLOW_BIZ_AUDIT"
)

//go:embed prompts.txt
var prompts embed.FS

// prompt缓存 key: XXX_YYY  value: 内容
var promptCache = make(map[string]string, 20)

// 获取本地文件的prompt内容，并进行解析，获取对应key的prompt内容
func GetPrompt(key string, formatValues ...any) string {
	prompt := promptCache[key]
	if prompt != "" {
		return fmt.Sprintf(prompt, formatValues...)
	}

	bytes, err := prompts.ReadFile("prompts.txt")
	if err != nil {
		logx.Error("failed to read prompt file: prompts.txt, err: %v", err)
		return ""
	}
	allPrompts := string(bytes)

	propmts := strings.Split(allPrompts, "---------------------------------------")
	var res string
	for _, keyAndPrompt := range propmts {
		keyAndPrompt = stringx.TrimSpaceAndBr(keyAndPrompt)
		// 获取第一行的Key信息如：--XXX_YYY
		info := strings.SplitN(keyAndPrompt, "\n", 2)
		// prompt，即去除第一行的key与备注信息
		prompt := info[1]
		// 获取key；如：XXX_YYY
		promptKey := strings.Split(strings.Split(info[0], " ")[0], "--")[1]
		if key == promptKey {
			res = prompt
		}
		promptCache[promptKey] = prompt
	}
	return fmt.Sprintf(res, formatValues...)
}
