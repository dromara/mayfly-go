package pool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

/**
这个单元测试是ai写的：

提示词如下：
我写了一个连接池和缓存池  请帮忙帮我写一下单元测试，以完成对池的功能检查，需要做到以下目标：
1、连接池内的连接不共用，用完连接后需要归还到池中，通过监听ctx事件实现
2、连接池允许连接数满后，需要等待新的连接，等待超时时间为10秒
3、缓存池内的连接可以共用，连接超时或不可用时自动删除连接
4、group也并发测试一下，另外对各个池做一下压力测试
5、你通过阅读代码，尽可能多的覆盖测试

回答如下:
好的，你的需求很明确。下面我会为你的连接池（ChanPool）和缓存池（CachePool）分别设计详细的单元测试，覆盖你提到的所有核心功能点，并尽量覆盖更多边界和异常场景。
测试会用 Go 的 testing 包实现，并会模拟一个简单的 Conn 实现用于测试。
测试点设计说明：
1. 连接池（ChanPool）测试点
不同 goroutine 获取到的连接不是同一个实例。
连接用完后归还池中，池可复用。
连接池满时新请求会等待，超时后报错。
连接池关闭后无法再获取连接。
连接池 Resize 后行为正常。
健康检查能自动剔除失效连接。
2. 缓存池（CachePool）测试点
获取到的连接可以复用。
连接超时后自动删除。
Put 超过最大连接数时，最久未用的被清理。
池关闭后无法再获取连接。
Resize 后行为正常。

PoolGroup 并发测试：
测试多个 goroutine 并发创建和获取连接池
验证连接池的正确创建和管理
基准测试：
使用 testing.B 进行并发基准测试
测试连接池在高并发下的性能表现
压力测试：
模拟真实场景下的高并发使用
统计成功率和失败率
测试连接池在极限情况下的表现
异常情况测试：
测试连接失效时的行为
测试并发关闭时的行为
验证连接池的健壮性

*/

// mockConn 实现 Conn 接口
type mockConn struct {
	id      int
	closed  bool
	pingErr error
	mu      sync.Mutex
}

func (c *mockConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *mockConn) Ping() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("closed")
	}
	return c.pingErr
}

func newMockConn(id int) *mockConn {
	return &mockConn{id: id}
}

// ========== ChanPool 单元测试 ==========

func TestChanPool_Basic(t *testing.T) {
	var idGen int
	pool := NewChannelPool(func() (*mockConn, error) {
		idGen++
		return newMockConn(idGen), nil
	}, WithMaxConns[*mockConn](2), WithIdleTimeout[*mockConn](time.Second))

	ctx := context.Background()
	conn1, _ := pool.Get(ctx)
	conn2, _ := pool.Get(ctx)
	if conn1 == conn2 {
		t.Fatal("连接池应返回不同连接")
	}

	// 归还后可复用
	_ = pool.Put(conn1)
	conn3, _ := pool.Get(ctx)
	if conn3 != conn1 {
		t.Fatal("归还的连接应被复用")
	}
	_ = pool.Put(conn2)
	_ = pool.Put(conn3)
	pool.Close()
}

func TestChanPool_WaitTimeout(t *testing.T) {
	pool := NewChannelPool(func() (*mockConn, error) {
		return newMockConn(1), nil
	}, WithMaxConns[*mockConn](1), WithWaitTimeout[*mockConn](100*time.Millisecond))

	ctx := context.Background()
	conn1, _ := pool.Get(ctx)

	// 第二个请求会阻塞并超时
	ctx2, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := pool.Get(ctx2)
	if err == nil || time.Since(start) < 100*time.Millisecond {
		t.Fatal("应因池满而超时")
	}
	_ = pool.Put(conn1)
	pool.Close()
}

func TestChanPool_ContextCancel(t *testing.T) {
	pool := NewChannelPool(func() (*mockConn, error) {
		return newMockConn(1), nil
	}, WithMaxConns[*mockConn](1))

	ctx, cancel := context.WithCancel(context.Background())
	conn, _ := pool.Get(ctx)
	cancel()
	time.Sleep(10 * time.Millisecond) // 等待归还
	_ = pool.Put(conn)
	pool.Close()
}

func TestChanPool_Resize(t *testing.T) {
	pool := NewChannelPool(func() (*mockConn, error) {
		return newMockConn(1), nil
	}, WithMaxConns[*mockConn](2))
	ctx := context.Background()
	conn1, _ := pool.Get(ctx)
	conn2, _ := pool.Get(ctx)
	pool.Resize(1)
	_ = pool.Put(conn1)
	_ = pool.Put(conn2)
	pool.Close()
}

func TestChanPool_HealthCheck(t *testing.T) {
	pool := NewChannelPool(func() (*mockConn, error) {
		return newMockConn(1), nil
	}, WithMaxConns[*mockConn](1), WithIdleTimeout[*mockConn](10*time.Millisecond), WithHealthCheckInterval[*mockConn](10*time.Millisecond))
	ctx := context.Background()
	conn, _ := pool.Get(ctx)
	_ = pool.Put(conn)
	time.Sleep(30 * time.Millisecond)
	stats := pool.Stats()
	if stats.IdleConns != 0 {
		t.Fatal("健康检查应清理超时连接")
	}
	pool.Close()
}

// ========== CachePool 单元测试 ==========

func TestCachePool_Basic(t *testing.T) {
	var idGen int
	pool := NewCachePool(func() (*mockConn, error) {
		idGen++
		return newMockConn(idGen), nil
	}, WithMaxConns[*mockConn](2), WithIdleTimeout[*mockConn](time.Second))

	ctx := context.Background()
	conn1, _ := pool.Get(ctx)
	_ = pool.Put(conn1)
	conn2, _ := pool.Get(ctx)
	if conn1 != conn2 {
		t.Fatal("缓存池应复用同一连接")
	}
	_ = pool.Put(conn2)
	pool.Close()
}

func TestCachePool_TimeoutCleanup(t *testing.T) {
	pool := NewCachePool(func() (*mockConn, error) {
		return newMockConn(1), nil
	}, WithMaxConns[*mockConn](1), WithIdleTimeout[*mockConn](10*time.Millisecond), WithHealthCheckInterval[*mockConn](10*time.Millisecond))
	ctx := context.Background()
	conn, _ := pool.Get(ctx)
	_ = pool.Put(conn)
	time.Sleep(30 * time.Millisecond)
	stats := pool.Stats()
	if stats.TotalConns != 0 {
		t.Fatal("超时连接应被清理")
	}
	pool.Close()
}

func TestCachePool_OverMaxConns(t *testing.T) {
	var idGen int
	pool := NewCachePool(func() (*mockConn, error) {
		idGen++
		return newMockConn(idGen), nil
	}, WithMaxConns[*mockConn](1))
	ctx := context.Background()
	conn1, _ := pool.Get(ctx)
	_ = pool.Put(conn1)
	conn2, _ := pool.Get(ctx)
	_ = pool.Put(conn2)
	if conn1 != conn2 {
		t.Fatal("缓存池应复用同一连接")
	}
	// 放入第二个不同连接，最老的会被清理
	conn3 := newMockConn(999)
	_ = pool.Put(conn3)
	if pool.Stats().TotalConns != 1 {
		t.Fatal("超出最大连接数应只保留一个")
	}
	pool.Close()
}

func TestCachePool_Resize(t *testing.T) {
	pool := NewCachePool(func() (*mockConn, error) {
		return newMockConn(1), nil
	}, WithMaxConns[*mockConn](2))
	ctx := context.Background()
	conn1, _ := pool.Get(ctx)
	_ = pool.Put(conn1)
	pool.Resize(1)
	if pool.Stats().TotalConns != 1 {
		t.Fatal("Resize 后应只保留一个连接")
	}
	pool.Close()
}

// ========== PoolGroup 并发测试 ==========

func TestPoolGroup_Concurrent(t *testing.T) {
	group := NewPoolGroup[Conn]()
	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 100

	// 并发创建和获取连接池
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("pool_%d", id)
				pool, err := group.GetChanPool(key, func() (Conn, error) {
					return newMockConn(id), nil
				})
				if err != nil {
					t.Errorf("获取连接池失败: %v", err)
					return
				}
				if pool == nil {
					t.Error("连接池不应为nil")
					return
				}
			}
		}(i)
	}
	wg.Wait()

	// 验证所有池都被正确创建
	pools := group.AllPool()
	if len(pools) != goroutines {
		t.Errorf("期望 %d 个连接池，实际有 %d 个", goroutines, len(pools))
	}

	// 清理所有池
	group.CloseAll()
}

// ========== 压力测试 ==========

func BenchmarkChanPool_Concurrent(b *testing.B) {
	pool := NewChannelPool(func() (*mockConn, error) {
		return newMockConn(1), nil
	}, WithMaxConns[*mockConn](100))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			conn, err := pool.Get(ctx)
			if err != nil {
				b.Fatal(err)
			}
			_ = pool.Put(conn)
		}
	})
	pool.Close()
}

func BenchmarkCachePool_Concurrent(b *testing.B) {
	pool := NewCachePool(func() (*mockConn, error) {
		return newMockConn(1), nil
	}, WithMaxConns[*mockConn](100))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			conn, err := pool.Get(ctx)
			if err != nil {
				b.Fatal(err)
			}
			_ = pool.Put(conn)
		}
	})
	pool.Close()
}

// 模拟真实场景的压力测试
func TestChanPool_Stress(t *testing.T) {
	const (
		goroutines = 50
		iterations = 1000
	)

	pool := NewChannelPool(func() (*mockConn, error) {
		return newMockConn(1), nil
	}, WithMaxConns[*mockConn](20), WithWaitTimeout[*mockConn](time.Second))

	var wg sync.WaitGroup
	var errCount int32
	var successCount int32

	// 添加一个 done channel 用于通知所有 goroutine 停止
	done := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				select {
				case <-done:
					return
				default:
					ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
					conn, err := pool.Get(ctx)
					if err != nil {
						atomic.AddInt32(&errCount, 1)
						cancel()
						continue
					}

					// 模拟使用连接
					time.Sleep(time.Millisecond)
					_ = pool.Put(conn)
					atomic.AddInt32(&successCount, 1)
					cancel()
				}
			}
		}()
	}

	// 等待所有操作完成
	wg.Wait()
	close(done) // 通知所有 goroutine 停止

	// 确保所有连接都被正确关闭
	pool.Close()

	t.Logf("总请求数: %d", goroutines*iterations)
	t.Logf("成功请求数: %d", successCount)
	t.Logf("失败请求数: %d", errCount)
	t.Logf("成功率: %.2f%%", float64(successCount)/float64(goroutines*iterations)*100)
}

func TestCachePool_Stress(t *testing.T) {
	const (
		goroutines = 50
		iterations = 1000
	)

	pool := NewCachePool(func() (*mockConn, error) {
		return newMockConn(1), nil
	}, WithMaxConns[*mockConn](20), WithIdleTimeout[*mockConn](time.Minute))

	var wg sync.WaitGroup
	var errCount int32
	var successCount int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				conn, err := pool.Get(ctx)
				if err != nil {
					atomic.AddInt32(&errCount, 1)
					cancel()
					continue
				}

				// 模拟使用连接
				time.Sleep(time.Millisecond)
				_ = pool.Put(conn)
				atomic.AddInt32(&successCount, 1)
				cancel()
			}
		}()
	}

	wg.Wait()
	pool.Close()

	t.Logf("总请求数: %d", goroutines*iterations)
	t.Logf("成功请求数: %d", successCount)
	t.Logf("失败请求数: %d", errCount)
	t.Logf("成功率: %.2f%%", float64(successCount)/float64(goroutines*iterations)*100)
}

// 测试连接池在连接失效时的行为
func TestChanPool_InvalidConn(t *testing.T) {
	pool := NewChannelPool(func() (*mockConn, error) {
		conn := newMockConn(1)
		conn.pingErr = errors.New("connection invalid")
		return conn, nil
	}, WithMaxConns[*mockConn](1), WithHealthCheckInterval[*mockConn](10*time.Millisecond))

	ctx := context.Background()
	conn, _ := pool.Get(ctx)
	_ = pool.Put(conn)

	// 等待健康检查
	time.Sleep(20 * time.Millisecond)

	// 获取新连接
	newConn, err := pool.Get(ctx)
	if err != nil {
		t.Fatal("应该能获取到新连接")
	}
	if newConn == conn {
		t.Fatal("应该获取到新的连接实例")
	}

	_ = pool.Put(newConn)
	pool.Close()
}

// 测试连接池在并发关闭时的行为
func TestChanPool_ConcurrentClose(t *testing.T) {
	pool := NewChannelPool(func() (*mockConn, error) {
		return newMockConn(1), nil
	}, WithMaxConns[*mockConn](10))

	var wg sync.WaitGroup
	const goroutines = 10
	done := make(chan struct{}) // 用于通知所有 goroutine 停止

	// 并发获取连接
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
					conn, err := pool.Get(ctx)
					if err != nil {
						cancel()
						continue
					}
					_ = pool.Put(conn)
					cancel()
				}
			}
		}()
	}

	// 等待一段时间让 goroutine 运行
	time.Sleep(100 * time.Millisecond)

	// 关闭连接池
	pool.Close()

	// 通知所有 goroutine 停止
	close(done)

	// 等待所有 goroutine 完成
	wg.Wait()

	// 验证连接池已关闭
	ctx := context.Background()
	_, err := pool.Get(ctx)
	if err != ErrPoolClosed {
		t.Errorf("期望错误 %v，实际错误 %v", ErrPoolClosed, err)
	}
}

func TestPoolGroup_ConcurrentAccess(t *testing.T) {
	group := NewPoolGroup[Conn]()
	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 100

	// 并发创建和获取连接池
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("pool_%d", id)
				pool, err := group.GetChanPool(key, func() (Conn, error) {
					return newMockConn(id), nil
				})
				if err != nil {
					t.Errorf("获取连接池失败: %v", err)
					return
				}
				if pool == nil {
					t.Error("连接池不应为nil")
					return
				}

				// 模拟使用连接池
				ctx := context.Background()
				conn, err := pool.Get(ctx)
				if err != nil {
					t.Errorf("获取连接失败: %v", err)
					continue
				}
				_ = pool.Put(conn)
			}
		}(i)
	}
	wg.Wait()

	// 验证所有池都被正确创建
	pools := group.AllPool()
	if len(pools) != goroutines {
		t.Errorf("期望 %d 个连接池，实际有 %d 个", goroutines, len(pools))
	}

	// 并发关闭所有池
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("pool_%d", id)
			_ = group.Close(key)
		}(i)
	}
	wg.Wait()

	// 等待所有池关闭完成
	err := group.WaitForClose(10 * time.Second)
	if err != nil {
		t.Errorf("等待池关闭超时: %v", err)
	}

	// 验证所有池都已关闭
	pools = group.AllPool()
	if len(pools) != 0 {
		t.Errorf("所有池应已关闭，但还有 %d 个池", len(pools))
	}
}

func TestPoolGroup_ConcurrentClose(t *testing.T) {
	group := NewPoolGroup[Conn]()
	const goroutines = 10

	// 先创建一些池
	for i := 0; i < goroutines; i++ {
		key := fmt.Sprintf("pool_%d", i)
		_, err := group.GetChanPool(key, func() (Conn, error) {
			return newMockConn(i), nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	// 并发关闭所有池
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			group.CloseAll()
		}()
	}
	wg.Wait()

	// 等待所有池关闭完成
	err := group.WaitForClose(10 * time.Second)
	if err != nil {
		t.Errorf("等待池关闭超时: %v", err)
	}

	// 验证所有池都已关闭
	pools := group.AllPool()
	if len(pools) != 0 {
		t.Errorf("所有池应已关闭，但还有 %d 个池", len(pools))
	}
}
