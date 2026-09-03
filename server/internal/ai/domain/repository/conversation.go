package repository

import (
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/pkg/base"
)

type Conversation interface {
	base.Repo[*entity.Conversation]
}
