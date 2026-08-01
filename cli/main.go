package main

import (
	"mayfly-go/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		// 错误渲染（JSON/文本）与退出码统一由 cmd 包处理
		cmd.ExitWithError(err)
	}
}
