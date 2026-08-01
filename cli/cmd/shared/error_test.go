package shared

import (
	"errors"
	"testing"
)

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name   string
		errMsg string
		want   bool
	}{
		{"connection refused", "dial tcp: connection refused", true},
		{"code 501", "error code=501", true},
		{"code 502", "error code=502", true},
		{"normal error", "some other error", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsConnectionError(tt.errMsg)
			if got != tt.want {
				t.Errorf("IsConnectionError(%q) = %v, want %v", tt.errMsg, got, tt.want)
			}
		})
	}
}

func TestFail(t *testing.T) {
	// 测试 ErrSilent 传递
	err := Fail("test.msg", ErrSilent)
	if !errors.Is(err, ErrSilent) {
		t.Errorf("Fail() should return ErrSilent when input is ErrSilent")
	}

	// 测试普通错误
	err = Fail("test.msg", errors.New("test error"))
	if err == nil {
		t.Error("Fail() should return non-nil error")
	}
}

func TestArgsHasJSONFlag(t *testing.T) {
	// 这个函数依赖 os.Args，这里只测试基本逻辑
	// 实际测试需要修改 os.Args，跳过
	t.Skip("ArgsHasJSONFlag depends on os.Args")
}
