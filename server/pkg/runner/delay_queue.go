package runner

import (
	"context"
	"math"
	"sync"
	"time"
)

const minTimerDelay = time.Millisecond * 1
const maxTimerDelay = time.Nanosecond * math.MaxInt64

type DelayQueue[T Delayable] struct {
	enqueuedSignal chan struct{}
	dequeuedSignal chan struct{}
	transferChan   chan T
	singleDequeue  chan struct{}
	mutex          sync.Mutex
	priorityQueue  *PriorityQueue[T]
	zero           T
}

type Delayable interface {
	GetDeadline() time.Time
	GetKey() string
}

var _ Delayable = (*wrapper[Job])(nil)

type wrapper[T Job] struct {
	key      string
	deadline time.Time
	removed  bool
	status   JobStatus
	job      T
}

func newWrapper[T Job](job T) *wrapper[T] {
	return &wrapper[T]{
		key: job.GetKey(),
		job: job,
	}
}

func (d *wrapper[T]) GetDeadline() time.Time {
	return d.deadline
}

func (d *wrapper[T]) GetKey() string {
	return d.key
}

func NewDelayQueue[T Delayable](cap int) *DelayQueue[T] {
	singleDequeue := make(chan struct{}, 1)
	singleDequeue <- struct{}{}
	return &DelayQueue[T]{
		enqueuedSignal: make(chan struct{}),
		dequeuedSignal: make(chan struct{}),
		transferChan:   make(chan T),
		singleDequeue:  singleDequeue,
		priorityQueue: NewPriorityQueue[T](cap, func(src T, dst T) bool {
			return src.GetDeadline().Before(dst.GetDeadline())
		}),
	}
}

func (s *DelayQueue[T]) TryDequeue() (T, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if elm, ok := s.priorityQueue.Peek(0); ok {
		delay := elm.GetDeadline().Sub(time.Now())
		if delay < minTimerDelay {
			// 无需延迟，头部元素出队后直接返回
			_, _ = s.dequeue()
			return elm, true
		}
	}
	return s.zero, false
}

func (s *DelayQueue[T]) Dequeue(ctx context.Context) (T, bool) {
	// 出队锁：避免因重复获取队列头部同一元素降低性能
	select {
	case <-s.singleDequeue:
		defer func() {
			s.singleDequeue <- struct{}{}
		}()
	case <-ctx.Done():
		return s.zero, false
	}

	for {
		// 全局锁：避免入队和出队信号的重置与激活出现并发问题
		s.mutex.Lock()
		if ctx.Err() != nil {
			s.mutex.Unlock()
			return s.zero, false
		}

		// 接收直接转发的不需要延迟的新元素
		select {
		case elm := <-s.transferChan:
			s.mutex.Unlock()
			return elm, true
		default:
		}

		// 延迟时间缺省值为 maxTimerDelay, 表示队列为空
		delay := maxTimerDelay
		if elm, ok := s.priorityQueue.Peek(0); ok {
			now := time.Now()
			delay = elm.GetDeadline().Sub(now)
			if delay < minTimerDelay {
				// 无需延迟，头部元素出队后直接返回
				_, _ = s.dequeue()
				s.mutex.Unlock()
				return elm, ok
			}
		}
		// 重置入队信号，避免历史信号干扰
		select {
		case <-s.enqueuedSignal:
		default:
		}
		s.mutex.Unlock()

		if delay == maxTimerDelay {
			// 队列为空, 等待新元素
			select {
			case elm := <-s.transferChan:
				return elm, true
			case <-s.enqueuedSignal:
				continue
			case <-ctx.Done():
				return s.zero, false
			}
		} else if delay >= minTimerDelay {
			// 等待时间到期或新元素加入
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
				continue
			case elm := <-s.transferChan:
				timer.Stop()
				return elm, true
			case <-s.enqueuedSignal:
				timer.Stop()
				continue
			case <-ctx.Done():
				timer.Stop()
				return s.zero, false
			}
		}
	}
}

func (s *DelayQueue[T]) dequeue() (T, bool) {
	elm, ok := s.priorityQueue.Dequeue()
	if !ok {
		return s.zero, false
	}
	select {
	case s.dequeuedSignal <- struct{}{}:
	default:
	}
	return elm, true
}

func (s *DelayQueue[T]) enqueue(val T) bool {
	if ok := s.priorityQueue.Enqueue(val); !ok {
		return false
	}
	select {
	case s.enqueuedSignal <- struct{}{}:
	default:
	}
	return true
}

func (s *DelayQueue[T]) TryEnqueue(val T) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.priorityQueue.IsFull() {
		return false
	}
	return s.enqueue(val)
}

func (s *DelayQueue[T]) Enqueue(ctx context.Context, val T) bool {
	for {
		// 全局锁：避免入队和出队信号的重置与激活出现并发问题
		s.mutex.Lock()

		if ctx.Err() != nil {
			s.mutex.Unlock()
			return false
		}

		// 如果队列未满，入队后直接返回
		if !s.priorityQueue.IsFull() {
			s.enqueue(val)
			s.mutex.Unlock()
			return true
		}
		// 队列已满，重置出队信号，避免受到历史信号影响
		select {
		case <-s.dequeuedSignal:
		default:
		}
		s.mutex.Unlock()

		if delay := val.GetDeadline().Sub(time.Now()); delay >= minTimerDelay {
			// 新元素需要延迟，等待退出信号、出队信号和到期信号
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
				// 新元素不再需要延迟
			case <-s.dequeuedSignal:
				// 收到出队信号，从头开始尝试入队
				timer.Stop()
				continue
			case <-ctx.Done():
				timer.Stop()
				return false
			}
		} else {
			// 新元素不需要延迟，等待转发成功信号、出队信号和退出信号
			select {
			case s.transferChan <- val:
				// 新元素转发成功，直接返回（避免队列满且元素未到期导致新元素长时间无法入队）
				return true
			case <-s.dequeuedSignal:
				// 收到出队信号，从头开始尝试入队
				continue
			case <-ctx.Done():
				return false
			}
		}
	}
}

func (s *DelayQueue[T]) Remove(_ context.Context, key string) (T, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.priorityQueue.Remove(s.index(key))
}

func (s *DelayQueue[T]) index(key string) int {
	for i := 0; i < s.priorityQueue.Len(); i++ {
		elm, ok := s.priorityQueue.Peek(i)
		if !ok {
			continue
		}
		if key == elm.GetKey() {
			return i
		}
	}
	return -1
}
