package static

import "embed"

// 使用1.16特性编译阶段将静态资源文件打包进编译好的程序
var (
	//go:embed static/**
	Static embed.FS
)
