package runner

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var _ Delayable = &delayElement{}

type delayElement struct {
	id       uint64
	value    int
	deadline time.Time
}

func (elm *delayElement) GetDeadline() time.Time {
	return elm.deadline
}

func (elm *delayElement) GetId() uint64 {
	return elm.id
}

func (elm *delayElement) GetKey() string {
	return strconv.FormatUint(elm.id, 16)
}

type testDelayQueue = DelayQueue[*delayElement]

func newTestDelayQueue(cap int) *testDelayQueue {
	return NewDelayQueue[*delayElement](cap)
}

func mustEnqueue(val int, delay int64) func(t *testing.T, queue *testDelayQueue) {
	return func(t *testing.T, queue *testDelayQueue) {
		require.True(t, queue.Enqueue(context.Background(),
			newTestElm(val, delay)))
	}
}

func newTestElm(value int, delay int64) *delayElement {
	return &delayElement{
		id:       elmId.Add(1),
		value:    value,
		deadline: time.Now().Add(time.Millisecond * time.Duration(delay)),
	}
}

var elmId atomic.Uint64

func TestDelayQueue_Enqueue(t *testing.T) {
	type testCase[R int, T Delayable] struct {
		name    string
		queue   *DelayQueue[T]
		before  func(t *testing.T, queue *DelayQueue[T])
		while   func(t *testing.T, queue *DelayQueue[T])
		after   func(t *testing.T, queue *DelayQueue[T])
		value   int
		delay   int64
		timeout int64
		wantOk  bool
	}
	tests := []testCase[int, *delayElement]{
		{
			name:  "enqueue to empty queue",
			queue: newTestDelayQueue(1),
			after: func(t *testing.T, queue *testDelayQueue) {
				val, ok := queue.priorityQueue.Dequeue()
				require.True(t, ok)
				require.Equal(t, 1, val.value)
			},
			timeout: 10,
			value:   1,
			wantOk:  true,
		},
		{
			name:  "enqueue active element to full queue",
			queue: newTestDelayQueue(1),
			before: func(t *testing.T, queue *testDelayQueue) {
				mustEnqueue(1, 60)(t, queue)
			},
			timeout: 40,
			delay:   20,
			wantOk:  false,
		},
		{
			name:    "enqueue inactive element to full queue",
			queue:   newTestDelayQueue(1),
			before:  mustEnqueue(1, 60),
			timeout: 20,
			delay:   40,
			wantOk:  false,
		},
		{
			name:   "enqueue to full queue while dequeue valid element",
			queue:  newTestDelayQueue(1),
			before: mustEnqueue(1, 60),
			while: func(t *testing.T, queue *testDelayQueue) {
				_, ok := queue.Dequeue(context.Background())
				require.True(t, ok)
			},
			timeout: 80,
			wantOk:  true,
		},
		{
			name:   "enqueue active element to full queue while dequeue invalid element",
			queue:  newTestDelayQueue(1),
			before: mustEnqueue(1, 60),
			while: func(t *testing.T, queue *testDelayQueue) {
				elm, ok := queue.Dequeue(context.Background())
				require.True(t, ok)
				require.Equal(t, 2, elm.value)
			},
			timeout: 40,
			value:   2,
			delay:   20,
			wantOk:  true,
		},
		{
			name:   "enqueue inactive element to full queue while dequeue invalid element",
			queue:  newTestDelayQueue(1),
			before: mustEnqueue(1, 60),
			while: func(t *testing.T, queue *testDelayQueue) {
				_, ok := queue.Dequeue(context.Background())
				require.True(t, ok)
			},
			timeout: 20,
			delay:   40,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(),
				time.Millisecond*time.Duration(tt.timeout))
			defer cancel()
			if tt.before != nil {
				tt.before(t, tt.queue)
			}
			if tt.while != nil {
				go tt.while(t, tt.queue)
			}
			ok := tt.queue.Enqueue(ctx, newTestElm(tt.value, tt.delay))
			require.Equal(t, tt.wantOk, ok)
		})
	}
}

func TestDelayQueue_Dequeue(t *testing.T) {
	type testCase[R int, T Delayable] struct {
		name    string
		queue   *DelayQueue[T]
		before  func(t *testing.T, queue *DelayQueue[T])
		while   func(t *testing.T, queue *DelayQueue[T])
		timeout int64
		wantVal int
		wantOk  bool
	}
	tests := []testCase[int, *delayElement]{
		{
			name:    "dequeue from empty queue",
			queue:   newTestDelayQueue(1),
			timeout: 20,
			wantOk:  false,
		},
		{
			name:    "dequeue new active element from empty queue",
			queue:   newTestDelayQueue(1),
			while:   mustEnqueue(1, 20),
			timeout: 4000,
			wantVal: 1,
			wantOk:  true,
		},
		{
			name:    "dequeue new inactive element from empty queue",
			queue:   newTestDelayQueue(1),
			while:   mustEnqueue(1, 60),
			timeout: 20,
			wantOk:  false,
		},
		{
			name:    "dequeue active element from full queue",
			queue:   newTestDelayQueue(1),
			before:  mustEnqueue(1, 60),
			timeout: 80,
			wantVal: 1,
			wantOk:  true,
		},
		{
			name:    "dequeue inactive element from full queue",
			queue:   newTestDelayQueue(1),
			before:  mustEnqueue(1, 60),
			timeout: 20,
			wantOk:  false,
		},
		{
			name:    "dequeue new active element from full queue",
			queue:   newTestDelayQueue(1),
			before:  mustEnqueue(1, 60),
			while:   mustEnqueue(2, 40),
			timeout: 80,
			wantVal: 2,
			wantOk:  true,
		},
		{
			name:    "dequeue new inactive element from full queue",
			queue:   newTestDelayQueue(1),
			before:  mustEnqueue(1, 60),
			while:   mustEnqueue(2, 40),
			timeout: 20,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(),
				time.Millisecond*time.Duration(tt.timeout))
			defer cancel()
			if tt.before != nil {
				tt.before(t, tt.queue)
			}
			if tt.while != nil {
				go tt.while(t, tt.queue)
			}
			got, ok := tt.queue.Dequeue(ctx)
			require.Equal(t, tt.wantOk, ok)
			if !ok {
				return
			}
			require.Equal(t, tt.wantVal, got.value)
		})
	}
}

func TestDelayQueue(t *testing.T) {
	const delay = 1000
	const timeout = 1000
	const capacity = 100
	const count = 100
	var wg sync.WaitGroup
	var (
		enqueueSeq atomic.Int32
		dequeueSeq atomic.Int32
		checksum   atomic.Int64
	)
	queue := newTestDelayQueue(capacity)
	procs := runtime.GOMAXPROCS(0)
	wg.Add(procs)
	for i := 0; i < procs; i++ {
		go func(i int) {
			defer wg.Done()
			for {
				ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*timeout)
				if i%2 == 0 {
					if seq := int(enqueueSeq.Add(1)); seq <= count {
						for ctx.Err() == nil {
							if ok := queue.Enqueue(ctx, newTestElm(seq, int64(rand.Intn(delay)))); ok {
								break
							}
						}
					} else {
						cancel()
						return
					}
				} else {
					if seq := int(dequeueSeq.Add(1)); seq > count {
						cancel()
						return
					}
					for ctx.Err() == nil {
						if elm, ok := queue.Dequeue(ctx); ok {
							require.Less(t, elm.GetDeadline().Sub(time.Now()), minTimerDelay)
							checksum.Add(int64(elm.value))
							break
						}
					}
				}
				cancel()
			}
		}(i)
	}
	wg.Wait()
	assert.Zero(t, queue.priorityQueue.Len())
	assert.Equal(t, int64((1+count)*count/2), checksum.Load())
}

func BenchmarkDelayQueueV3(b *testing.B) {
	const delay = 0
	const capacity = 100

	b.Run("enqueue", func(b *testing.B) {
		queue := newTestDelayQueue(b.N)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = queue.Enqueue(context.Background(), newTestElm(1, delay))
		}
	})

	b.Run("parallel to enqueue", func(b *testing.B) {
		queue := newTestDelayQueue(b.N)
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = queue.Enqueue(context.Background(), newTestElm(1, delay))
			}
		})
	})

	b.Run("dequeue", func(b *testing.B) {
		queue := newTestDelayQueue(b.N)
		for i := 0; i < b.N; i++ {
			require.True(b, queue.Enqueue(context.Background(), newTestElm(1, delay)))
		}
		time.Sleep(time.Millisecond * delay)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = queue.Dequeue(context.Background())
		}
	})

	b.Run("parallel to dequeue", func(b *testing.B) {
		queue := newTestDelayQueue(b.N)
		for i := 0; i < b.N; i++ {
			require.True(b, queue.Enqueue(context.Background(), newTestElm(1, delay)))
		}
		time.Sleep(time.Millisecond * delay)
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = queue.Dequeue(context.Background())
			}
		})
	})

	b.Run("parallel to dequeue while enqueue", func(b *testing.B) {
		queue := newTestDelayQueue(capacity)
		go func() {
			for i := 0; i < b.N; i++ {
				_ = queue.Enqueue(context.Background(), newTestElm(i, delay))
			}
		}()
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = queue.Dequeue(context.Background())
			}
		})
	})

	b.Run("parallel to enqueue while dequeue", func(b *testing.B) {
		queue := newTestDelayQueue(capacity)
		go func() {
			for i := 0; i < b.N; i++ {
				_, _ = queue.Dequeue(context.Background())
			}
		}()
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = queue.Enqueue(context.Background(), newTestElm(1, delay))
			}
		})
	})

	b.Run("parallel to enqueue and dequeue", func(b *testing.B) {
		var wg sync.WaitGroup
		var (
			enqueueSeq atomic.Int32
			dequeueSeq atomic.Int32
		)
		queue := newTestDelayQueue(capacity)
		b.ReportAllocs()
		b.ResetTimer()
		procs := runtime.GOMAXPROCS(0)
		wg.Add(procs)
		for i := 0; i < procs; i++ {
			go func(i int) {
				defer wg.Done()
				for {
					if i%2 == 0 {
						if seq := int(enqueueSeq.Add(1)); seq <= b.N {
							for {
								if ok := queue.Enqueue(context.Background(), newTestElm(seq, delay)); ok {
									break
								}
							}
						} else {
							return
						}
					} else {
						if seq := int(dequeueSeq.Add(1)); seq > b.N {
							return
						}
						for {
							if _, ok := queue.Dequeue(context.Background()); ok {
								break
							}
						}
					}
				}
			}(i)
		}
		wg.Wait()
	})
}
