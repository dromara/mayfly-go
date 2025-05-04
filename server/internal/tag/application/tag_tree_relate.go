package application

import (
	"context"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/internal/tag/imsg"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
)

type TagTreeRelate interface {
	base.App[*entity.TagTreeRelate]

	// RelateTag 关联标签
	RelateTag(ctx context.Context, relateType entity.TagRelateType, relateId uint64, tagCodePaths ...string) error

	// GetRelateIds 根据标签路径获取对应关联的id
	GetRelateIds(ctx context.Context, relateType entity.TagRelateType, tagPaths ...string) ([]uint64, error)

	// GetTagPathsByAccountId 根据账号id获取该账号可操作的标签code路径
	GetTagPathsByAccountId(accountId uint64) []string

	// GetTagPathsByRelate 根据关联信息获取关联的标签codePaths
	GetTagPathsByRelate(relateType entity.TagRelateType, relateId uint64) []string

	// FillTagInfo 填充关联的标签信息
	FillTagInfo(relateType entity.TagRelateType, relates ...entity.IRelateTag)
}

type tagTreeRelateAppImpl struct {
	base.AppImpl[*entity.TagTreeRelate, repository.TagTreeRelate]

	tagTreeApp TagTree `inject:"T"`
}

var _ (TagTreeRelate) = (*tagTreeRelateAppImpl)(nil)

func (tr *tagTreeRelateAppImpl) RelateTag(ctx context.Context, relateType entity.TagRelateType, relateId uint64, tagCodePaths ...string) error {
	if hasConflictPath(tagCodePaths) {
		return errorx.NewBizI(ctx, imsg.ErrConflictingCodePath)
	}

	var tags []*entity.TagTree
	if len(tagCodePaths) > 0 {
		tr.tagTreeApp.ListByQuery(&entity.TagTreeQuery{CodePaths: tagCodePaths}, &tags)
		if len(tags) != len(tagCodePaths) {
			return errorx.NewBiz("There is an error tag path")
		}
	}

	oldRelates, _ := tr.ListByCond(&entity.TagTreeRelate{RelateType: relateType, RelateId: relateId})
	oldTagIds := collx.ArrayMap[*entity.TagTreeRelate, uint64](oldRelates, func(val *entity.TagTreeRelate) uint64 {
		return val.TagId
	})
	newTagIds := collx.ArrayMap[*entity.TagTree, uint64](tags, func(val *entity.TagTree) uint64 {
		return val.Id
	})
	addTagIds, delTagIds, _ := collx.ArrayCompare(newTagIds, oldTagIds)

	if len(addTagIds) > 0 {
		trs := make([]*entity.TagTreeRelate, 0)
		for _, tagId := range addTagIds {
			trs = append(trs, &entity.TagTreeRelate{
				TagId:      tagId,
				RelateType: relateType,
				RelateId:   relateId,
			})
		}
		if err := tr.BatchInsert(ctx, trs); err != nil {
			return err
		}
	}

	if len(delTagIds) > 0 {
		if err := tr.DeleteByCond(ctx, model.NewCond().Eq("relate_type", relateType).Eq("relate_id", relateId).In("tag_id", delTagIds)); err != nil {
			return err
		}
	}

	return nil
}

func (tr *tagTreeRelateAppImpl) GetRelateIds(ctx context.Context, relateType entity.TagRelateType, tagPaths ...string) ([]uint64, error) {
	la := contextx.GetLoginAccount(ctx)
	canAccessTagPaths := tagPaths
	if la != nil && la.Id != consts.AdminId {
		canAccessTagPaths = filterCodePaths(tr.tagTreeApp.ListTagByAccountId(la.Id), tagPaths)
	}

	poisibleTagPaths := make([]string, 0)
	for _, tagPath := range canAccessTagPaths {
		// 追加可能关联的标签路径，如tagPath = tag1/tag2/1|xxx/，需要获取所有关联的自身及父标签（tag1/  tag1/tag2/ tag1/tag2/1|xxx）
		poisibleTagPaths = append(poisibleTagPaths, entity.CodePath(tagPath).GetAllPath()...)
	}
	return tr.GetRepo().SelectRelateIdsByTagPaths(relateType, poisibleTagPaths...)
}

func (tr *tagTreeRelateAppImpl) GetTagPathsByAccountId(accountId uint64) []string {
	return tr.GetRepo().SelectTagPathsByAccountId(accountId)
}

func (tr *tagTreeRelateAppImpl) GetTagPathsByRelate(relateType entity.TagRelateType, relateId uint64) []string {
	return tr.GetRepo().SelectTagPathsByRelate(relateType, relateId)
}

func (tr *tagTreeRelateAppImpl) FillTagInfo(relateType entity.TagRelateType, relates ...entity.IRelateTag) {
	if len(relates) == 0 {
		return
	}

	// 关联id -> 关联信息
	relateIds2Relate := collx.ArrayToMap(relates, func(rt entity.IRelateTag) uint64 {
		return rt.GetRelateId()
	})

	relateTags, _ := tr.ListByCond(model.NewCond().Eq("relate_type", relateType).In("relate_id", collx.MapKeys(relateIds2Relate)))

	tagIds := collx.ArrayMap(relateTags, func(rt *entity.TagTreeRelate) uint64 {
		return rt.TagId
	})
	tags, _ := tr.tagTreeApp.GetByIds(tagIds)

	tagId2Tag := collx.ArrayToMap(tags, func(t *entity.TagTree) uint64 {
		return t.Id
	})
	for _, rt := range relateTags {
		relate := relateIds2Relate[rt.RelateId]
		tag := tagId2Tag[rt.TagId]
		if relate != nil && tag != nil {
			// 赋值标签信息
			relate.SetTagInfo(entity.ResourceTag{CodePath: tag.CodePath, TagId: tag.Id})
		}
	}
}
