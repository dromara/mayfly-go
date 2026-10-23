package mvm

import (
	"fmt"
	"sync"
	"testing"
)

// resetConnIds 隔离全局登记表，避免用例之间互相污染
func resetConnIds() {
	connIdsLock.Lock()
	defer connIdsLock.Unlock()
	connIds = make(map[uint64][]string)
}

// snapshot 在锁内拷贝某个实例的连接登记
func snapshot(milvusId uint64) []string {
	connIdsLock.Lock()
	defer connIdsLock.Unlock()
	return append([]string(nil), connIds[milvusId]...)
}

// TestAddConnIdKeepsAllEntries 同一实例的多个库/账号连接必须全部登记：
// 旧实现把 append 结果赋给局部变量而未写回 map，第二条起被静默丢弃，CloseAll 只能收到第一条
func TestAddConnIdKeepsAllEntries(t *testing.T) {
	resetConnIds()
	addConnId(1, "a")
	addConnId(1, "b")
	addConnId(1, "b") // 重复登记需去重

	got := snapshot(1)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("connIds = %v, want [a b]", got)
	}
}

// TestRemoveConnIdDropsEntry 关闭单条连接后必须摘链，CloseAll 不应再关闭已释放的池
func TestRemoveConnIdDropsEntry(t *testing.T) {
	resetConnIds()
	addConnId(2, "x")
	addConnId(2, "y")

	removeConnId(2, "x")
	if got := snapshot(2); len(got) != 1 || got[0] != "y" {
		t.Fatalf("connIds = %v, want [y]", got)
	}

	removeConnId(2, "y")
	connIdsLock.Lock()
	_, exists := connIds[2]
	connIdsLock.Unlock()
	if exists {
		t.Fatal("最后一个连接移除后应删除实例键")
	}
}

// TestAddConnIdConcurrent 连接工厂在并发请求中执行：并发登记不得触发 concurrent map writes
func TestAddConnIdConcurrent(t *testing.T) {
	resetConnIds()

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			addConnId(uint64(i%8), fmt.Sprintf("conn-%d", i))
		}(i)
	}
	wg.Wait()

	total := 0
	for id := uint64(0); id < 8; id++ {
		total += len(snapshot(id))
	}
	if total != 64 {
		t.Fatalf("登记总数 = %d, want 64", total)
	}
}
