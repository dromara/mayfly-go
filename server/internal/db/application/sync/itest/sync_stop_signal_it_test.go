//go:build it

package itest

// StopTask 的跨实例停止信号（真实 Redis）。
//
// 历史缺陷：StopTask 走 runGuard.Release —— Release 的语义是「持有者自释放」，本地没有持有时
// 直接 return、完全不碰 Redis。于是他实例发起的停止对执行侧不可见：任务一路写到底，
// 界面却已显示"已停止"。本用例断言 StopTask 必须写下跨实例可见的停止标记。

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"mayfly-go/pkg/rediscli"
)

func TestITStopTaskWritesCrossInstanceStopSignal(t *testing.T) {
	ctx := context.Background()
	cli := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6332", DialTimeout: 2 * time.Second})
	if err := cli.Ping(ctx).Err(); err != nil {
		t.Skipf("本地 redis 不可达（跨实例停止语义依赖 Redis）: %s", err.Error())
	}
	prev := rediscli.GetCli()
	rediscli.SetCli(cli)
	t.Cleanup(func() { rediscli.SetCli(prev); _ = cli.Close() })

	const taskId = uint64(9931)
	stopKey := fmt.Sprintf("mayfly:run-stop:%d", taskId)
	_ = cli.Del(ctx, stopKey).Err()
	t.Cleanup(func() { _ = cli.Del(ctx, stopKey).Err() })

	app, fake := newApp()
	require.NoError(t, app.StopTask(ctx, taskId))
	require.NotEmpty(t, fake.updated, "StopTask 仍应把任务状态置为停止")

	v, err := cli.Get(ctx, stopKey).Result()
	require.NoError(t, err, "StopTask 必须写跨实例停止标记（否则他实例的执行侧感知不到）")
	require.Equal(t, "1", v)

	// 对照：清理后标记消失，证明上一条断言读到的就是本次写入
	require.NoError(t, cli.Del(ctx, stopKey).Err())
	_, err = cli.Get(ctx, stopKey).Result()
	require.ErrorIs(t, err, redis.Nil)
}
