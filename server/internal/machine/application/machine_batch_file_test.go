package application

import (
	"context"
	"testing"
)

// 批量分发的目标目录校验：不校验会让 SFTP 把含元字符的路径当字面量拼接，
// 既不报错也没人知道文件落在哪，界面却回「分发成功」——虚假成功比报错更难查。
func TestValidateRemotePathAccepts(t *testing.T) {
	ok := []string{
		"/tmp",
		"/tmp/",
		"/srv/app data/",
		"/tmp/中文目录",
		"/var/log-2.0",
		"/home/u_1/.config",
		"/data+backup/2026",
	}
	for _, p := range ok {
		if err := validateRemotePath(context.Background(), p); err != nil {
			t.Fatalf("validateRemotePath(%q) should pass, got %v", p, err)
		}
	}
}

func TestValidateRemotePathRejects(t *testing.T) {
	bad := []string{
		"tmp/relative",           // 非绝对路径
		"/tmp; touch /tmp/pwned", // 命令分隔
		"/tmp|wc",                // 管道
		"/tmp&bg",                // 后台
		"/tmp$(id)",              // 命令替换
		"/tmp`id`",               // 命令替换
		"/etc/../tmp",            // 目录上跳
		"/tmp/..",                // 目录上跳
		"/tmp/*",                 // 通配
		"/tmp/'q'",               // 引号
		`/tmp\"x`,                // 转义/反斜杠
		"/tmp\nnewline",          // 控制字符
	}
	for _, p := range bad {
		if err := validateRemotePath(context.Background(), p); err == nil {
			t.Fatalf("validateRemotePath(%q) must be rejected（放行会产生虚假成功或垃圾路径）", p)
		}
	}
}
