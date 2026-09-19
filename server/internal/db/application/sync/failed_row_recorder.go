package sync

import (
	"fmt"
	"sync"
)

// FailedRow 失败行记录：包含行数据和错误信息
type FailedRow struct {
	// Row 源行数据（用于诊断和重试）
	Row map[string]any
	// Err 失败原因
	Err error
	// BatchIndex 批次序号（用于定位）
	BatchIndex int
}

// FailedRowRecorder 失败行记录器（Dead Letter Queue）。
// 线程安全：支持并发写入。
// 容量限制：超过 maxCapacity 后丢弃最早的记录，防止内存溢出。
type FailedRowRecorder struct {
	mu          sync.RWMutex
	failedRows  []FailedRow
	maxCapacity int
	totalCount  int // 总失败数（含被丢弃的）
}

// defaultRecorderCapacity 失败行记录器默认容量（当 maxCapacity <= 0 时使用）
const defaultRecorderCapacity = 1000

// NewFailedRowRecorder 创建失败行记录器。
// maxCapacity 为最大保留数量，0 表示使用默认上限。
func NewFailedRowRecorder(maxCapacity int) *FailedRowRecorder {
	if maxCapacity <= 0 {
		maxCapacity = defaultRecorderCapacity
	}
	return &FailedRowRecorder{
		failedRows:  make([]FailedRow, 0, min(maxCapacity, 100)),
		maxCapacity: maxCapacity,
	}
}

// Record 记录一条失败行
func (r *FailedRowRecorder) Record(row map[string]any, err error, batchIndex int) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.totalCount++

	// 容量限制：超过上限时丢弃最早的记录
	if len(r.failedRows) >= r.maxCapacity {
		// 移除最早的 10% 以腾出空间（批量淘汰减少锁竞争）
		discardCount := r.maxCapacity / 10
		if discardCount < 1 {
			discardCount = 1
		}
		r.failedRows = r.failedRows[discardCount:]
	}

	r.failedRows = append(r.failedRows, FailedRow{
		Row:        row,
		Err:        err,
		BatchIndex: batchIndex,
	})
}

// FailedRows 获取当前保留的失败行列表（只读副本）
func (r *FailedRowRecorder) FailedRows() []FailedRow {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]FailedRow, len(r.failedRows))
	copy(result, r.failedRows)
	return result
}

// TotalCount 返回总失败数（含被丢弃的）
func (r *FailedRowRecorder) TotalCount() int {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.totalCount
}

// Len 返回当前保留的失败行数
func (r *FailedRowRecorder) Len() int {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.failedRows)
}

// Summary 返回失败摘要（用于日志输出）
func (r *FailedRowRecorder) Summary() string {
	if r == nil || r.totalCount == 0 {
		return "无失败行"
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.totalCount > len(r.failedRows) {
		return fmt.Sprintf("共 %d 条失败行（保留最近 %d 条）", r.totalCount, len(r.failedRows))
	}
	return fmt.Sprintf("共 %d 条失败行", r.totalCount)
}

// Clear 清空记录
func (r *FailedRowRecorder) Clear() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failedRows = r.failedRows[:0]
	r.totalCount = 0
}
