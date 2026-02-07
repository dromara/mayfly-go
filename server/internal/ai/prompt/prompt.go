package prompt

import (
	"embed"
	"mayfly-go/pkg/utils/stringx"
)

//go:embed prompts/*.md
var prompts embed.FS

// GetPrompt 获取本地prompts文件内容，并进行模板解析
func GetPrompt(filename string, values any) (string, error) {
	// 自动添加 prompts/ 前缀
	fullPath := "prompts/" + filename

	bytes, err := prompts.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return stringx.TemplateParse(string(bytes), values)
}
