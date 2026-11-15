package dbi

// 连接池为空时的行为回归锁。
//
// 真机崩溃现场：DbConn.Close() 会把与 DbInfo 共享的 db 置 nil，而 DbInfo 仍被实例缓存
// 与健康巡检持有。巡检协程随后调 QueryContext → nil 解引用 panic → **整个进程被带走**，
// 表现是「后端莫名退出」而不是某次查询失败。后台协程里的 panic 不会被任何业务错误处理接住，
// 所以这条守卫必须以「返回错误」的形式存在，并且由用例钉住不再退回裸解引用。

import (
	"context"
	"testing"
)

func TestDbInfoWithoutPoolReturnsErrorInsteadOfPanicking(t *testing.T) {
	// db 字段刻意留空，等价于「连接被关闭 / 从未建立」的实例
	info := &DbInfo{Id: 7, Name: "closed-instance", Type: "mysql"}

	t.Run("QueryContext", func(t *testing.T) {
		cols, rows, err := info.QueryContext(context.Background(), "SHOW MASTER STATUS")
		if err == nil {
			t.Fatal("a missing pool must be an error, not a nil dereference")
		}
		if cols != nil || rows != nil {
			t.Fatalf("no result may be returned together with an error, got %v / %v", cols, rows)
		}
	})

	t.Run("ExecContext", func(t *testing.T) {
		affected, err := info.ExecContext(context.Background(), "FLUSH LOGS")
		if err == nil {
			t.Fatal("a missing pool must be an error, not a nil dereference")
		}
		if affected != 0 {
			t.Fatalf("affected rows must stay zero, got %d", affected)
		}
	})

	t.Run("Ping", func(t *testing.T) {
		conn := &DbConn{Info: info}
		if err := conn.Ping(); err == nil {
			t.Fatal("pinging a closed instance must report an error")
		}
	})

	// 错误文案要能指到是哪个实例：巡检会遍历所有实例，没有标识的报错无法定位
	if _, _, err := info.QueryContext(context.Background(), "SELECT 1"); err == nil || len(err.Error()) < 10 {
		t.Fatalf("the error must carry the instance description, got %v", err)
	}
}
