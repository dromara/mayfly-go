package repository

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
)

type TagTreeRelate interface {
	base.Repo[*entity.TagTreeRelate]

	// SelectRelateIdsByTagPaths 根据标签路径查询相关联的id
	SelectRelateIdsByTagPaths(relateType entity.TagRelateType, tagPaths ...string) ([]uint64, error)

	// SelectTagPathsByAccountId 根据账号id获取该账号可访问操作的标签codePaths（该方法调用较频繁，故不使用下列方法获取）
	SelectTagPathsByAccountId(accountId uint64) []string

	// SelectTagPathsByRelate 根据关联信息查询对应的关联的标签路径
	SelectTagPathsByRelate(relateType entity.TagRelateType, relateId uint64) []string
}
