package application

import (
	"context"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/collx"
	"strings"
)

type TagTree interface {
	base.App[*entity.TagTree]

	ListByQuery(condition *entity.TagTreeQuery, toEntity any)

	Save(ctx context.Context, tt *entity.TagTree) error

	Delete(ctx context.Context, id uint64) error

	// 获取指定账号有权限操作的资源信息列表
	// @param accountId 账号id
	// @param resourceType 资源类型
	// @param tagPath 访问指定的标签路径下关联的资源
	GetAccountTagResources(accountId uint64, resourceType int8, tagPath string) []entity.TagResource

	// 获取指定账号有权限操作的资源codes
	GetAccountResourceCodes(accountId uint64, resourceType int8, tagPath string) []string

	// 关联资源
	// @resourceCode 资源唯一编号
	// @resourceType 资源类型
	// @tagIds 资源关联的标签
	RelateResource(ctx context.Context, resourceCode string, resourceType int8, tagIds []uint64) error

	// 根据资源信息获取对应的标签路径列表
	ListTagPathByResource(resourceType int8, resourceCode string) []string

	// 根据tagPath获取自身及其所有子标签信息
	ListTagByPath(tagPath ...string) []*entity.TagTree

	// 根据账号id获取其可访问标签信息
	ListTagByAccountId(accountId uint64) []string

	// 账号是否有权限访问该标签关联的资源信息
	CanAccess(accountId uint64, tagPath ...string) error
}

type tagTreeAppImpl struct {
	base.AppImpl[*entity.TagTree, repository.TagTree]

	tagTreeTeamRepo repository.TagTreeTeam `inject:"TagTreeTeamRepo"`
	tagResourceApp  TagResource            `inject:"TagResourceApp"`
}

// 注入TagTreeRepo
func (p *tagTreeAppImpl) InjectTagTreeRepo(tagTreeRepo repository.TagTree) {
	p.Repo = tagTreeRepo
}

func (p *tagTreeAppImpl) Save(ctx context.Context, tag *entity.TagTree) error {
	accountId := contextx.GetLoginAccount(ctx).Id
	// 新建项目树节点信息
	if tag.Id == 0 {
		if strings.Contains(tag.Code, entity.CodePathSeparator) {
			return errorx.NewBiz("标识符不能包含'/'")
		}
		if tag.Pid != 0 {
			parentTag, err := p.GetById(new(entity.TagTree), tag.Pid)
			if err != nil {
				return errorx.NewBiz("父节点不存在")
			}
			if p.tagResourceApp.CountByCond(&entity.TagResource{TagId: tag.Pid}) > 0 {
				return errorx.NewBiz("该父标签已关联资源, 无法添加子标签")
			}

			tag.CodePath = parentTag.CodePath + tag.Code + entity.CodePathSeparator
		} else {
			if accountId != consts.AdminId {
				return errorx.NewBiz("非管理员无法添加根标签")
			}
			tag.CodePath = tag.Code + entity.CodePathSeparator
		}
		if p.CanAccess(accountId, tag.CodePath) != nil {
			return errorx.NewBiz("无权添加该标签")
		}

		// 判断该路径是否存在
		var hasLikeTags []entity.TagTree
		p.GetRepo().SelectByCondition(&entity.TagTreeQuery{CodePathLike: tag.CodePath}, &hasLikeTags)
		if len(hasLikeTags) > 0 {
			return errorx.NewBiz("已存在该标签路径开头的标签, 请修改该标识code")
		}

		return p.Insert(ctx, tag)
	}

	// 防止误传导致被更新
	tag.Code = ""
	tag.CodePath = ""
	return p.UpdateById(ctx, tag)
}

func (p *tagTreeAppImpl) ListByQuery(condition *entity.TagTreeQuery, toEntity any) {
	p.GetRepo().SelectByCondition(condition, toEntity)
}

func (p *tagTreeAppImpl) GetAccountTagResources(accountId uint64, resourceType int8, tagPath string) []entity.TagResource {
	tagResourceQuery := &entity.TagResourceQuery{
		ResourceType: resourceType,
	}

	var tagResources []entity.TagResource
	var accountTagPaths []string

	if accountId != consts.AdminId {
		// 获取账号有权限操作的标签路径列表
		accountTagPaths = p.ListTagByAccountId(accountId)
		if len(accountTagPaths) == 0 {
			return tagResources
		}
	}

	tagResourceQuery.TagPath = tagPath
	tagResourceQuery.TagPathLikes = accountTagPaths
	p.tagResourceApp.ListByQuery(tagResourceQuery, &tagResources)
	return tagResources
}

func (p *tagTreeAppImpl) GetAccountResourceCodes(accountId uint64, resourceType int8, tagPath string) []string {
	tagResources := p.GetAccountTagResources(accountId, resourceType, tagPath)
	// resouce code去重
	code2Resource := collx.ArrayToMap[entity.TagResource, string](tagResources, func(val entity.TagResource) string {
		return val.ResourceCode
	})

	return collx.MapKeys(code2Resource)
}

func (p *tagTreeAppImpl) RelateResource(ctx context.Context, resourceCode string, resourceType int8, tagIds []uint64) error {
	if resourceCode == "" {
		return errorx.NewBiz("资源编号不能为空")
	}
	// 如果tagIds为空数组，则为解绑该标签资源关联关系
	if len(tagIds) == 0 {
		return p.tagResourceApp.DeleteByCond(ctx, &entity.TagResource{
			ResourceCode: resourceCode,
			ResourceType: resourceType,
		})
	}

	var oldTagResources []*entity.TagResource
	p.tagResourceApp.ListByQuery(&entity.TagResourceQuery{ResourceType: resourceType, ResourceCode: resourceCode}, &oldTagResources)

	var addTagIds, delTagIds []uint64
	if len(oldTagResources) == 0 {
		addTagIds = tagIds
	} else {
		oldTagIds := collx.ArrayMap[*entity.TagResource, uint64](oldTagResources, func(tr *entity.TagResource) uint64 {
			return tr.TagId
		})
		addTagIds, delTagIds, _ = collx.ArrayCompare[uint64](tagIds, oldTagIds)
	}

	if len(addTagIds) > 0 {
		addTagResource := make([]*entity.TagResource, 0)
		for _, tagId := range addTagIds {
			tag, err := p.GetById(new(entity.TagTree), tagId)
			if err != nil {
				return errorx.NewBiz("存在错误标签id")
			}
			addTagResource = append(addTagResource, &entity.TagResource{
				ResourceCode: resourceCode,
				ResourceType: resourceType,
				TagId:        tagId,
				TagPath:      tag.CodePath,
			})
		}
		if err := p.tagResourceApp.BatchInsert(ctx, addTagResource); err != nil {
			return err
		}
	}

	if len(delTagIds) > 0 {
		for _, tagId := range delTagIds {
			cond := &entity.TagResource{ResourceCode: resourceCode, ResourceType: resourceType, TagId: tagId}
			if err := p.tagResourceApp.DeleteByCond(ctx, cond); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *tagTreeAppImpl) ListTagPathByResource(resourceType int8, resourceCode string) []string {
	var trs []*entity.TagResource
	p.tagResourceApp.ListByQuery(&entity.TagResourceQuery{ResourceType: resourceType, ResourceCode: resourceCode}, &trs)
	return collx.ArrayMap(trs, func(tr *entity.TagResource) string {
		return tr.TagPath
	})
}

func (p *tagTreeAppImpl) ListTagByPath(tagPaths ...string) []*entity.TagTree {
	var tags []*entity.TagTree
	p.GetRepo().SelectByCondition(&entity.TagTreeQuery{CodePathLikes: tagPaths}, &tags)
	return tags
}

func (p *tagTreeAppImpl) ListTagByAccountId(accountId uint64) []string {
	return p.tagTreeTeamRepo.SelectTagPathsByAccountId(accountId)
}

func (p *tagTreeAppImpl) CanAccess(accountId uint64, tagPath ...string) error {
	if accountId == consts.AdminId {
		return nil
	}
	tagPaths := p.ListTagByAccountId(accountId)
	// 判断该资源标签是否为该账号拥有的标签或其子标签
	for _, v := range tagPaths {
		for _, tp := range tagPath {
			if strings.HasPrefix(tp, v) {
				return nil
			}
		}
	}

	return errorx.NewBiz("您无权操作该资源")
}

func (p *tagTreeAppImpl) Delete(ctx context.Context, id uint64) error {
	accountId := contextx.GetLoginAccount(ctx).Id
	tag, err := p.GetById(new(entity.TagTree), id)
	if err != nil {
		return errorx.NewBiz("该标签不存在")
	}
	if err := p.CanAccess(accountId, tag.CodePath); err != nil {
		return errorx.NewBiz("您无权删除该标签")
	}

	if p.tagResourceApp.CountByCond(&entity.TagResource{TagId: id}) > 0 {
		return errorx.NewBiz("请先移除该标签关联的资源")
	}

	return p.Tx(ctx, func(ctx context.Context) error {
		return p.DeleteById(ctx, id)
	}, func(ctx context.Context) error {
		// 删除该标签关联的团队信息
		return p.tagTreeTeamRepo.DeleteByCond(ctx, &entity.TagTreeTeam{TagId: id})
	})
}
