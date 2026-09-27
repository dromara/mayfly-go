package application

import (
	"context"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
)

// 与 keyvalue 包测试同一约定：连本地开发实例，连不上则跳过，避免无 redis 的环境变红
const testLocalRedisAddr = "127.0.0.1:6332"

// TestNeedConfirm 执行前确认的命令集合：清空类必须确认，普通读写不该被打断
func TestNeedConfirm(t *testing.T) {
	tests := []struct {
		name string
		info *redis.CommandInfo
		want bool
	}{
		{"平台登记的高危命令", &redis.CommandInfo{Name: "FLUSHALL"}, true},
		{"平台登记的只读但阻塞命令", &redis.CommandInfo{Name: "KEYS", Flags: []string{"readonly"}}, true},
		{"Redis 自标 admin 的命令", &redis.CommandInfo{Name: "DEBUG", Flags: []string{"admin", "noscript"}}, true},
		{"Redis 自标 @dangerous 类别", &redis.CommandInfo{Name: "SORT_RO", ACLFlags: []string{"@read", "@dangerous"}}, true},
		{"普通读命令", &redis.CommandInfo{Name: "HGETALL", Flags: []string{"readonly"}, ACLFlags: []string{"@read", "@hash", "@slow"}}, false},
		{"普通写命令", &redis.CommandInfo{Name: "SET", Flags: []string{"write", "denyoom"}, ACLFlags: []string{"@write", "@string"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needConfirm(strings.ToUpper(tt.info.Name), tt.info); got != tt.want {
				t.Errorf("needConfirm(%s) = %v, want %v", tt.info.Name, got, tt.want)
			}
		})
	}
}

// TestInstanceCommandDirectory 命令目录取自实例自身的 COMMAND 回复：
// 这里验证客户端库能解析当前 Redis 版本的回复结构，且键参数位置、写标志确实可用
// （Commands 方法本体只做字段搬运，其入参 *rdm.RedisConn 需要完整 IOC 装配，故不在单测里构造）
func TestInstanceCommandDirectory(t *testing.T) {
	cli := redis.NewClient(&redis.Options{Addr: testLocalRedisAddr, DB: 15})
	defer func() { _ = cli.Close() }()

	infos, err := cli.Command(context.Background()).Result()
	if err != nil {
		t.Skipf("local redis [%s] command directory unavailable, skip: %s", testLocalRedisAddr, err.Error())
	}
	if len(infos) < 100 {
		t.Fatalf("command directory looks truncated, got %d commands", len(infos))
	}

	byName := make(map[string]*redis.CommandInfo, len(infos))
	for _, info := range infos {
		byName[strings.ToUpper(info.Name)] = info
	}

	hgetall := byName["HGETALL"]
	if hgetall == nil {
		t.Fatal("HGETALL missing from command directory")
	}
	if hgetall.Arity != 2 || hgetall.FirstKeyPos != 1 || hgetall.LastKeyPos != 1 || hgetall.StepCount != 1 {
		t.Errorf("HGETALL metadata mismatch: %+v", hgetall)
	}

	// 清空类命令必须被识别为需要确认，普通读写不需要
	if !needConfirm("FLUSHALL", byName["FLUSHALL"]) {
		t.Error("FLUSHALL should require confirmation")
	}
	if needConfirm("GET", byName["GET"]) {
		t.Error("GET should not require confirmation")
	}
}
