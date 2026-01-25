package static

import (
	"embed"
	"io/fs"
	"mayfly-go/pkg/starter"
)

// 使用1.16特性编译阶段将静态资源文件打包进编译好的程序
var (
	//go:embed static/**
	Static embed.FS
)

func Router() *starter.StaticRouter {
	sys, _ := fs.Sub(Static, "static")
	return &starter.StaticRouter{
		Fs: sys,
		Paths: []string{"/",
			"/favicon.ico",
			"/config.js",
			"/assets/*file",
		},
	}
}
