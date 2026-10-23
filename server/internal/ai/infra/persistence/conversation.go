package persistence

import (
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/domain/repository"
	"mayfly-go/pkg/base"
)

type conversationRepoImpl struct {
	base.RepoImpl[*entity.Conversation]
}

var _ repository.Conversation = (*conversationRepoImpl)(nil)
