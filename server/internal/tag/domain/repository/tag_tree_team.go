package repository

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
)

type TagTreeTeam interface {
	base.Repo[*entity.TagTreeTeam]

	SelectTagPathsByAccountId(accountId uint64) []string
}
