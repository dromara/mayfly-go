package repository

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/base"
)

type Execution interface {
	base.Repo[*entity.Execution]
}
