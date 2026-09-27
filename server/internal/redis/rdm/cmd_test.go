package rdm

import "testing"

// TestCmdCaseInsensitive Redis 命令名大小写不敏感，命令控制台又是原样透传用户输入，
// 因此两张判定表必须对任意大小写给出同一结论：否则小写写命令会绕过写权限与工单审批
func TestCmdCaseInsensitive(t *testing.T) {
	for _, cmd := range []string{"SET", "set", "SeT", "HSET", "hset", "DEL", "del", "HEXPIRE", "hexpire"} {
		if !IsWriteCmd(cmd) {
			t.Fatalf("IsWriteCmd(%q) should be true", cmd)
		}
	}

	for _, cmd := range []string{"FLUSHDB", "flushdb", "Keys", "CONFIG", "config"} {
		if !IsDangerousCmd(cmd) {
			t.Fatalf("IsDangerousCmd(%q) should be true", cmd)
		}
	}

	for _, cmd := range []string{"GET", "get", "HGETALL", "hgetall", "TTL", "ttl"} {
		if IsWriteCmd(cmd) || IsDangerousCmd(cmd) {
			t.Fatalf("read command %q must not be classified as write/dangerous", cmd)
		}
	}
}
