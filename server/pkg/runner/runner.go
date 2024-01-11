package runner

import (
	"context"
	"fmt"
	"github.com/emirpasic/gods/maps/linkedhashmap"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/timex"
	"sync"
	"time"
)

type JobKey = string
type RunFunc func(ctx context.Context)
type NextFunc func() (Job, bool)
type RunnableFunc func(next NextFunc) bool

type JobStatus int

const (
	JobUnknown JobStatus = iota
	JobDelay
	JobWaiting
	JobRunning
	JobRemoved
)

type Job interface {
	GetKey() JobKey
	GetStatus() JobStatus
	SetStatus(status JobStatus)
	Run(ctx context.Context)
	Runnable(next NextFunc) bool
	GetDeadline() time.Time
	Schedule() bool
	Update(job Job)
}

type iterator[T Job] struct {
	index int
	data  []T
	zero  T
}

func (iter *iterator[T]) Begin() {
	iter.index = -1
}

func (iter *iterator[T]) Next() (T, bool) {
	if iter.index >= len(iter.data)-1 {
		return iter.zero, false
	}
	iter.index++
	return iter.data[iter.index], true
}

type array[T Job] struct {
	size int
	data []T
	zero T
}

func newArray[T Job](size int) *array[T] {
	return &array[T]{
		size: size,
		data: make([]T, 0, size),
	}
}

func (a *array[T]) Iterator() *iterator[T] {
	return &iterator[T]{
		index: -1,
		data:  a.data,
	}
}

func (a *array[T]) Full() bool {
	return len(a.data) >= a.size
}

func (a *array[T]) Append(job T) bool {
	if len(a.data) >= a.size {
		return false
	}
	a.data = append(a.data, job)
	return true
}

func (a *array[T]) Get(key JobKey) (T, bool) {
	for _, job := range a.data {
		if key == job.GetKey() {
			return job, true
		}
	}
	return a.zero, false
}

func (a *array[T]) Remove(key JobKey) {
	length := len(a.data)
	for i, elm := range a.data {
		if key == elm.GetKey() {
			a.data[i], a.data[length-1] = a.data[length-1], a.zero
			a.data = a.data[:length-1]
			return
		}
	}
}

type Runner[T Job] struct {
	maxRunning int
	waiting    *linkedhashmap.Map
	running    *array[T]
	runnable   func(job T, iterateRunning func() (T, bool)) bool
	mutex      sync.Mutex
	wg         sync.WaitGroup
	context    context.Context
	cancel     context.CancelFunc
	zero       T
	signal     chan struct{}
	all        map[string]T
	delayQueue *DelayQueue[T]
}

func NewRunner[T Job](maxRunning int) *Runner[T] {
	ctx, cancel := context.WithCancel(context.Background())
	runner := &Runner[T]{
		maxRunning: maxRunning,
		all:        make(map[string]T, maxRunning),
		waiting:    linkedhashmap.New(),
		running:    newArray[T](maxRunning),
		context:    ctx,
		cancel:     cancel,
		signal:     make(chan struct{}, 1),
		delayQueue: NewDelayQueue[T](0),
	}
	runner.wg.Add(maxRunning + 1)
	for i := 0; i < maxRunning; i++ {
		go runner.run()
	}
	go func() {
		defer runner.wg.Done()
		timex.SleepWithContext(runner.context, time.Second*10)
		for runner.context.Err() == nil {
			job, ok := runner.delayQueue.Dequeue(ctx)
			if !ok {
				continue
			}
			runner.mutex.Lock()
			runner.waiting.Put(job.GetKey(), job)
			job.SetStatus(JobWaiting)
			runner.trigger()
			runner.mutex.Unlock()
		}
	}()
	return runner
}

func (r *Runner[T]) Close() {
	r.cancel()
	r.wg.Wait()
}

func (r *Runner[T]) run() {
	defer r.wg.Done()

	for r.context.Err() == nil {
		select {
		case <-r.signal:
			job, ok := r.pickRunnable()
			if !ok {
				continue
			}
			r.doRun(job)
			r.afterRun(job)
		case <-r.context.Done():
		}
	}
}

func (r *Runner[T]) doRun(job T) {
	defer func() {
		if err := recover(); err != nil {
			logx.Error(fmt.Sprintf("failed to run job: %v", err))
		}
	}()

	job.Run(r.context)
}

func (r *Runner[T]) afterRun(job T) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := job.GetKey()
	r.running.Remove(key)
	r.trigger()
	switch job.GetStatus() {
	case JobRunning:
		r.schedule(r.context, job)
	case JobRemoved:
		delete(r.all, key)
	default:
		panic(fmt.Sprintf("invalid job status %v occurred after run", job.GetStatus()))
	}
}

func (r *Runner[T]) pickRunnable() (T, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	iter := r.running.Iterator()
	var runnable T
	ok := r.waiting.Any(func(key interface{}, value interface{}) bool {
		job := value.(T)
		iter.Begin()
		if job.Runnable(func() (Job, bool) { return iter.Next() }) {
			if r.running.Full() {
				return false
			}
			r.waiting.Remove(key)
			r.running.Append(job)
			job.SetStatus(JobRunning)
			if !r.running.Full() && !r.waiting.Empty() {
				r.trigger()
			}
			runnable = job
			return true
		}
		return false
	})
	if !ok {
		return r.zero, false
	}
	return runnable, true
}

func (r *Runner[T]) schedule(ctx context.Context, job T) {
	if !job.Schedule() {
		delete(r.all, job.GetKey())
		job.SetStatus(JobRemoved)
		return
	}
	r.delayQueue.Enqueue(ctx, job)
	job.SetStatus(JobDelay)
}

//func (r *Runner[T]) Schedule(ctx context.Context, job T) {
//	r.mutex.Lock()
//	defer r.mutex.Unlock()
//
//	switch job.GetStatus() {
//	case JobUnknown:
//	case JobDelay:
//		r.delayQueue.Remove(ctx, job.GetKey())
//	case JobWaiting:
//		r.waiting.Remove(job)
//	case JobRunning:
//		// 标记为 removed, 任务执行完成后再删除
//		return
//	case JobRemoved:
//		return
//	}
//	r.schedule(ctx, job)
//}

func (r *Runner[T]) Add(ctx context.Context, job T) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, ok := r.all[job.GetKey()]; ok {
		return nil
	}
	r.schedule(ctx, job)
	return nil
}

func (r *Runner[T]) UpdateOrAdd(ctx context.Context, job T) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if old, ok := r.all[job.GetKey()]; ok {
		old.Update(job)
		job = old
	}
	r.schedule(ctx, job)
	return nil
}

func (r *Runner[T]) StartNow(ctx context.Context, job T) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := job.GetKey()
	if old, ok := r.all[key]; ok {
		job = old
		if job.GetStatus() == JobDelay {
			r.delayQueue.Remove(ctx, key)
			r.waiting.Put(key, job)
			r.trigger()
		}
		return nil
	}
	r.all[key] = job
	r.waiting.Put(key, job)
	r.trigger()
	return nil
}

func (r *Runner[T]) trigger() {
	select {
	case r.signal <- struct{}{}:
	default:
	}
}

func (r *Runner[T]) Remove(ctx context.Context, key JobKey) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	job, ok := r.all[key]
	if !ok {
		return nil
	}
	switch job.GetStatus() {
	case JobUnknown:
		panic(fmt.Sprintf("invalid job status %v occurred after added", job.GetStatus()))
	case JobDelay:
		r.delayQueue.Remove(ctx, key)
	case JobWaiting:
		r.waiting.Remove(key)
	case JobRunning:
		// 标记为 removed, 任务执行完成后再删除
	case JobRemoved:
		return nil
	}
	delete(r.all, key)
	job.SetStatus(JobRemoved)
	return nil
}
