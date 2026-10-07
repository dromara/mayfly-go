package application

import (
	"context"
	"mayfly-go/pkg/gox"
	"sync"
	"time"
)

// runBatchConcurrently 批量操作通用并发调度：信号量限并发 + 整体超时 + 逐台 panic 兜底，
// 单台失败/超时不影响他台，按入参顺序聚合结果。
//
// 泛型使「批量命令执行」与「批量文件分发」共用同一套调度而互不依赖（接口隔离）：
//   - per:     单台执行逻辑，返回该台结果（内部自行处理整体超时后的记账）
//   - onPanic: 单台 panic 时的兜底结果工厂
func runBatchConcurrently[T any](ctx context.Context, machineIds []uint64, totalTimeout time.Duration, per func(ctx context.Context, machineId uint64) T, onPanic func(machineId uint64, err error) T) []T {
	// 各 goroutine 只写自己的索引位，无数据竞争
	results := make([]T, len(machineIds))

	// 整体限时：多台不可达时最坏可达「台数×单台超时÷并发」的超长请求会挂死调用方
	// （前端 loading、反向代理超时）。到期后未执行的机器由 per 自行按超时记账
	runCtx, cancel := context.WithTimeout(ctx, totalTimeout)
	defer cancel()

	sem := make(chan struct{}, batchConcurrency)
	wg := &sync.WaitGroup{}
	for i, id := range machineIds {
		wg.Add(1)
		// 信号量在主循环获取：超限时不再创建新协程，天然限制总并发
		sem <- struct{}{}
		go func(i int, id uint64) {
			defer wg.Done()
			defer gox.Recover(func(err error) { results[i] = onPanic(id, err) })
			defer func() { <-sem }()
			results[i] = per(runCtx, id)
		}(i, id)
	}
	wg.Wait()
	return results
}
