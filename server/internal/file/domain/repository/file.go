package repository

import (
	"mayfly-go/internal/file/domain/entity"
	"mayfly-go/pkg/base"
)

type File interface {
	base.Repo[*entity.File]
}
