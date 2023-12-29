package service

import (
	"context"
	"errors"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/queue"
	"sync"
	"time"
)

type Scheduler[T entity.DbTask] struct {
	mutex            sync.Mutex
	wg               sync.WaitGroup
	queue            *queue.DelayQueue[T]
	closed           bool
	curTask          T
	curTaskContext   context.Context
	curTaskCancel    context.CancelFunc
	UpdateTaskStatus func(ctx context.Context, status entity.TaskStatus, lastErr error, task T) error
	RunTask          func(ctx context.Context, task T) error
}

type SchedulerOption[T entity.DbTask] func(*Scheduler[T])

func NewScheduler[T entity.DbTask](opts ...SchedulerOption[T]) (*Scheduler[T], error) {
	scheduler := &Scheduler[T]{
		queue: queue.NewDelayQueue[T](0),
	}
	for _, opt := range opts {
		opt(scheduler)
	}
	if scheduler.RunTask == nil || scheduler.UpdateTaskStatus == nil {
		return nil, errors.New("调度器没有设置 RunTask 或 UpdateTaskStatus")
	}
	scheduler.wg.Add(1)
	go scheduler.run()
	return scheduler, nil
}

func (m *Scheduler[T]) PushTask(ctx context.Context, task T) bool {
	if !task.Schedule() {
		return false
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.queue.Enqueue(ctx, task)
}

func (m *Scheduler[T]) UpdateTask(ctx context.Context, task T) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if task.GetId() == m.curTask.GetId() {
		return m.curTask.Update(task)
	}
	oldTask, ok := m.queue.Remove(ctx, task.GetId())
	if ok {
		if !oldTask.Update(task) {
			return false
		}
	} else {
		oldTask = task
	}
	if !oldTask.Schedule() {
		return false
	}
	return m.queue.Enqueue(ctx, oldTask)
}

func (m *Scheduler[T]) updateCurTask(status entity.TaskStatus, lastErr error, task T) bool {
	seconds := []time.Duration{time.Second * 1, time.Second * 8, time.Second * 64}
	for _, second := range seconds {
		if m.closed {
			return false
		}
		ctx, cancel := context.WithTimeout(context.Background(), second)
		err := m.UpdateTaskStatus(ctx, status, lastErr, task)
		cancel()
		if err != nil {
			logx.Errorf("保存任务失败: %v", err)
			time.Sleep(second)
			continue
		}
		return true
	}
	return false
}

func (m *Scheduler[T]) run() {
	defer m.wg.Done()

	var ctx context.Context
	var cancel context.CancelFunc
	for !m.closed {
		m.mutex.Lock()
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond)
		task, ok := m.queue.Dequeue(ctx)
		cancel()
		if !ok {
			m.mutex.Unlock()
			time.Sleep(time.Second)
			continue
		}
		m.curTask = task
		m.updateCurTask(entity.TaskReserved, nil, task)
		m.curTaskContext, m.curTaskCancel = context.WithCancel(context.Background())
		m.mutex.Unlock()

		err := m.RunTask(m.curTaskContext, task)

		m.mutex.Lock()
		taskStatus := entity.TaskSuccess
		if err != nil {
			taskStatus = entity.TaskFailed
		}
		m.updateCurTask(taskStatus, err, task)
		m.cancelCurTask()
		task.Schedule()
		if !task.IsFinished() {
			ctx, cancel = context.WithTimeout(context.Background(), time.Second)
			m.queue.Enqueue(ctx, task)
			cancel()
		}
		m.mutex.Unlock()
	}
}

func (m *Scheduler[T]) Close() {
	if m.closed {
		return
	}
	m.mutex.Lock()
	m.cancelCurTask()
	m.closed = true
	m.mutex.Unlock()

	m.wg.Wait()
}

func (m *Scheduler[T]) RemoveTask(taskId uint64) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.queue.Remove(context.Background(), taskId)
	if taskId == m.curTask.GetId() {
		m.cancelCurTask()
	}
	return true
}

func (m *Scheduler[T]) cancelCurTask() {
	if m.curTaskCancel != nil {
		m.curTaskCancel()
		m.curTaskCancel = nil
	}
}
