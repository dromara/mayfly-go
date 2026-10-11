package pool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 连接池的回归用例。
//
// 这里的每一条都对应一个「不会报错、只会静默漏资源或静默卡住」的问题：连接泄漏要等到
// 目标实例的连接数打满才暴露，锁内做 IO 要等到某条连接变慢才暴露，两者在功能测试里都看不见

// slowCloseConn 的 Close 会一直阻塞到 release 被关闭，用来模拟「对端不响应」的慢连接
type slowCloseConn struct {
	*mockConn
	release chan struct{}
	once    sync.Once
}

func (c *slowCloseConn) Close() error {
	c.once.Do(func() { <-c.release })
	return c.mockConn.Close()
}

// TestChanPoolSlowCloseDoesNotBlockPool 关连接是网络 IO，不能持锁做。
//
// 旧实现里 Put 全程持有写锁、并在锁内 Close：一条卡住的连接会把整池的 Get/Put 全部挡在锁上，
// 连「池已关闭」这么个纯内存判断都答不出来。池的可用性于是被最慢的那条连接决定
func TestChanPoolSlowCloseDoesNotBlockPool(t *testing.T) {
	release := make(chan struct{})

	var idGen int32
	p := NewChannelPool(func() (*slowCloseConn, error) {
		return &slowCloseConn{mockConn: newMockConn(int(atomic.AddInt32(&idGen, 1))), release: release}, nil
	}, WithMaxConns[*slowCloseConn](1))
	// 一个 defer 里按序做两件事：先放开慢连接、再关池。
	// 拆成两个 defer 就会在实现退化成「锁内 Close」时死锁（关池要拿写锁，而写锁正被卡在
	// Close 上的 Put 持有），用例本该快速变红却会把整个测试二进制挂死
	defer func() {
		close(release)
		p.Close()
	}()

	ctx := context.Background()
	conn, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	// 池关闭后再归还，必然走到「关掉这条连接」这条路
	p.Close()

	putDone := make(chan error, 1)
	go func() { putDone <- p.Put(conn) }()

	// 先确认 Put 确实卡在 Close 上，否则这条用例什么也没验到
	select {
	case err := <-putDone:
		t.Fatalf("Put 在 Close 返回前就结束了，用例没能复现「锁内做 IO」: %v", err)
	case <-time.After(200 * time.Millisecond):
	}

	got := make(chan error, 1)
	go func() {
		_, err := p.Get(ctx)
		got <- err
	}()
	select {
	case err := <-got:
		if !errors.Is(err, ErrPoolClosed) {
			t.Errorf("Get 的错误 = %v，期望 ErrPoolClosed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("慢连接的 Close 把整池锁住了：Get 连「池已关闭」都答不出来")
	}
}

// TestChanPoolResizeDoesNotOrphanIdleConns Resize 会重建空闲通道，而通道里躺着的都是活连接。
//
// 「channel 本身无状态」这个前提是错的：换掉通道等于把里面的连接原地丢弃，它们既回不了池、
// 也再没有任何人会去 Close，只能等对端的空闲超时。缩容路径只关掉 oldMax-newMax 条，
// 剩下的同样被换通道丢掉；不缩容时更是一条都关不掉
func TestChanPoolResizeDoesNotOrphanIdleConns(t *testing.T) {
	for _, tc := range []struct {
		name   string
		newMax int
	}{
		{"容量不变", 3},
		{"缩容", 1},
		{"扩容", 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var (
				regMu sync.Mutex
				conns []*mockConn
			)
			p := NewChannelPool(func() (*mockConn, error) {
				c := newMockConn(0)
				regMu.Lock()
				c.id = len(conns) + 1
				conns = append(conns, c)
				regMu.Unlock()
				return c, nil
			}, WithMaxConns[*mockConn](3), WithHealthCheckInterval[*mockConn](time.Hour))

			ctx := context.Background()
			obtained := make([]*mockConn, 0, 3)
			for i := 0; i < 3; i++ {
				c, err := p.Get(ctx)
				if err != nil {
					t.Fatalf("第 %d 次 Get 失败: %v", i, err)
				}
				obtained = append(obtained, c)
			}
			for _, c := range obtained {
				if err := p.Put(c); err != nil {
					t.Fatalf("Put 失败: %v", err)
				}
			}

			p.Resize(tc.newMax)
			p.Close()

			regMu.Lock()
			defer regMu.Unlock()
			if len(conns) != 3 {
				t.Fatalf("一共创建了 %d 条连接，期望 3 条", len(conns))
			}
			for _, c := range conns {
				if c.Ping() == nil {
					t.Errorf("连接 %d 在 Resize+Close 之后仍然是活的：它被换通道时丢掉了，再也没人会去关它", c.id)
				}
			}
		})
	}
}

// TestChanPoolHealthCheckKeepsHealthyConns 健康检查改成「锁内排空 → 锁外体检 → 锁内回填」之后，
// 仍可用的连接必须原样回到池里。这条是防止把 IO 挪出锁的时候顺手把连接弄丢
func TestChanPoolHealthCheckKeepsHealthyConns(t *testing.T) {
	p := NewChannelPool(func() (*mockConn, error) {
		return newMockConn(1), nil
	}, WithMaxConns[*mockConn](1), WithIdleTimeout[*mockConn](time.Hour), WithHealthCheckInterval[*mockConn](time.Hour))
	defer p.Close()

	ctx := context.Background()
	conn, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if err := p.Put(conn); err != nil {
		t.Fatalf("put failed: %v", err)
	}

	p.checkIdleConns()

	if got := p.Stats().IdleConns; got != 1 {
		t.Fatalf("健康检查之后空闲连接数 = %d，期望 1：未超时的健康连接不该被清掉", got)
	}
	back, err := p.Get(ctx)
	if err != nil {
		t.Fatalf("健康检查之后取不到连接: %v", err)
	}
	if back != conn {
		t.Error("健康检查之后拿到的不是原来那条连接")
	}
}
