package repository

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
)

type ResourceOpLog interface {
	base.Repo[*entity.ResourceOpLog]
}
