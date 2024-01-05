package repository

import (
	"context"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DbTask[T model.ModelI] interface {
	base.Repo[T]

	UpdateTaskStatus(ctx context.Context, task T) error
	UpdateEnabled(ctx context.Context, taskId uint64, enabled bool) error
	ListToDo() ([]T, error)
	ListRepeating() ([]T, error)
	AddTask(ctx context.Context, tasks ...T) error
}
