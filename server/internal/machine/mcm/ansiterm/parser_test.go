package ansiterm

import "testing"

// ==================== MyParser 核心测试 ====================

func TestMyParser_SendAndNext(t *testing.T) {
	p := &MyParser{
		CharChan: make(chan string, 10),
		IsPlain:  make(chan bool, 10),
	}
	ok := p.Send("a")
	if !ok {
		t.Fatal("Send should return true when not closed")
	}
	got := p.Next()
	if got != "a" {
		t.Errorf("Next() = %q, want 'a'", got)
	}
}

// Send 在 Close 后应返回 false（防止向已关闭 channel 写入 panic）
func TestMyParser_SendAfterClose(t *testing.T) {
	p := &MyParser{
		CharChan: make(chan string, 10),
		IsPlain:  make(chan bool, 10),
	}
	p.Close()
	ok := p.Send("a")
	if ok {
		t.Error("Send should return false after Close")
	}
}

func TestMyParser_RunningAndStart(t *testing.T) {
	p := &MyParser{
		CharChan: make(chan string, 10),
		IsPlain:  make(chan bool, 10),
	}
	if p.Running() {
		t.Error("should not be running initially")
	}
	p.Start()
	if !p.Running() {
		t.Error("should be running after Start")
	}
}
