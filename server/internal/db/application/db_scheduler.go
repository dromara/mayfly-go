package application

import (
	"context"
	"errors"
	"fmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/queue"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/timex"
	"sync"
	"time"
)

const sleepAfterError = time.Minute

type dbScheduler[T entity.DbTask] struct {
	mutex     sync.Mutex
	waitGroup sync.WaitGroup
	queue     *queue.DelayQueue[T]
	context   context.Context
	cancel    context.CancelFunc
	RunTask   func(ctx context.Context, task T) error
	taskRepo  repository.DbTask[T]
}

type dbSchedulerOption[T entity.DbTask] func(*dbScheduler[T])

func newDbScheduler[T entity.DbTask](taskRepo repository.DbTask[T], opts ...dbSchedulerOption[T]) (*dbScheduler[T], error) {
	ctx, cancel := context.WithCancel(context.Background())
	scheduler := &dbScheduler[T]{
		taskRepo: taskRepo,
		queue:    queue.NewDelayQueue[T](0),
		context:  ctx,
		cancel:   cancel,
	}
	for _, opt := range opts {
		opt(scheduler)
	}
	if scheduler.RunTask == nil {
		return nil, errors.New("数据库任务调度器没有设置 RunTask")
	}
	if err := scheduler.loadTask(context.Background()); err != nil {
		return nil, err
	}
	scheduler.waitGroup.Add(1)
	go scheduler.run()
	return scheduler, nil
}

func (s *dbScheduler[T]) updateTaskStatus(ctx context.Context, status entity.TaskStatus, lastErr error, task T) error {
	base := task.TaskBase()
	base.LastStatus = status
	var result = task.TaskResult(status)
	if lastErr != nil {
		result = fmt.Sprintf("%v: %v", result, lastErr)
	}
	base.LastResult = stringx.TruncateStr(result, entity.LastResultSize)
	base.LastTime = timex.NewNullTime(time.Now())
	return s.taskRepo.UpdateTaskStatus(ctx, task)
}

func (s *dbScheduler[T]) UpdateTask(ctx context.Context, task T) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.taskRepo.UpdateById(ctx, task); err != nil {
		return err
	}

	oldTask, ok := s.queue.Remove(ctx, task.GetId())
	if !ok {
		return errors.New("任务不存在")
	}
	oldTask.Update(task)
	if !oldTask.Schedule() {
		return nil
	}
	if !s.queue.Enqueue(ctx, oldTask) {
		return errors.New("任务入队失败")
	}
	return nil
}

func (s *dbScheduler[T]) run() {
	defer s.waitGroup.Done()

	for !s.closed() {
		time.Sleep(time.Second)

		s.mutex.Lock()
		task, ok := s.queue.TryDequeue()
		if !ok {
			s.mutex.Unlock()
			continue
		}
		if err := s.updateTaskStatus(s.context, entity.TaskReserved, nil, task); err != nil {
			s.mutex.Unlock()
			timex.SleepWithContext(s.context, sleepAfterError)
			continue
		}
		s.mutex.Unlock()

		errRun := s.RunTask(s.context, task)
		taskStatus := entity.TaskSuccess
		if errRun != nil {
			taskStatus = entity.TaskFailed
		}
		s.mutex.Lock()
		if err := s.updateTaskStatus(s.context, taskStatus, errRun, task); err != nil {
			s.mutex.Unlock()
			timex.SleepWithContext(s.context, sleepAfterError)
			continue
		}
		task.Schedule()
		if !task.Finished() {
			s.queue.Enqueue(s.context, task)
		}
		s.mutex.Unlock()

		if errRun != nil {
			timex.SleepWithContext(s.context, sleepAfterError)
		}
	}
}

func (s *dbScheduler[T]) Close() {
	s.cancel()
	s.waitGroup.Wait()
}

func (s *dbScheduler[T]) closed() bool {
	return s.context.Err() != nil
}

func (s *dbScheduler[T]) loadTask(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tasks, err := s.taskRepo.ListToDo()
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if !task.Schedule() {
			continue
		}
		s.queue.Enqueue(ctx, task)
	}
	return nil
}

func (s *dbScheduler[T]) AddTask(ctx context.Context, tasks ...T) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, task := range tasks {
		if err := s.taskRepo.AddTask(ctx, task); err != nil {
			return err
		}
		if !task.Schedule() {
			continue
		}
		s.queue.Enqueue(ctx, task)
	}
	return nil
}

func (s *dbScheduler[T]) DeleteTask(ctx context.Context, taskId uint64) error {
	// todo: 删除数据库备份历史文件
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.taskRepo.DeleteById(ctx, taskId); err != nil {
		return err
	}
	s.queue.Remove(ctx, taskId)
	return nil
}

func (s *dbScheduler[T]) EnableTask(ctx context.Context, taskId uint64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.taskRepo.UpdateEnabled(ctx, taskId, true); err != nil {
		return err
	}
	s.queue.Remove(ctx, taskId)

	task := anyx.DeepZero[T]()
	if err := s.taskRepo.GetById(task, taskId); err != nil {
		return err
	}
	if !task.Schedule() {
		return nil
	}
	s.queue.Enqueue(ctx, task)
	return nil
}

func (s *dbScheduler[T]) DisableTask(ctx context.Context, taskId uint64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.taskRepo.UpdateEnabled(ctx, taskId, false); err != nil {
		return err
	}
	s.queue.Remove(ctx, taskId)
	return nil
}
