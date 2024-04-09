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

// 保存资源标签参数
type SaveResourceTagParam struct {
	ResourceCode string
	ResourceName string
	ResourceType entity.TagType

	TagIds []uint64 // 关联标签，相当于父标签 pid，空数组则为删除该资源绑定的标签
}

type DelResourceTagParam struct {
	ResourceCode string
	ResourceType entity.TagType

	Pid uint64 //父标签 pid

	// 要删除的子节点类型，若存在值，则为删除资源标签下的指定类型的子标签
	ChildType entity.TagType
}

type TagTree interface {
	base.App[*entity.TagTree]

	ListByQuery(condition *entity.TagTreeQuery, toEntity any)

	Save(ctx context.Context, tt *entity.TagTree) error

	Delete(ctx context.Context, id uint64) error

	// 获取指定账号有权限操作的资源信息列表
	// @param accountId 账号id
	// @param resourceType 资源类型
	// @param tagPath 访问指定的标签路径下关联的资源
	GetAccountTagResources(accountId uint64, query *entity.TagTreeQuery) []*entity.TagTree

	// 获取指定账号有权限操作的资源codes
	GetAccountResourceCodes(accountId uint64, resourceType int8, tagPath string) []string

	// SaveResource 保存资源标签
	SaveResource(ctx context.Context, req *SaveResourceTagParam) error

	// DeleteResource 删除资源标签，会删除该资源下所有子节点信息
	DeleteResource(ctx context.Context, param *DelResourceTagParam) error

	// 根据资源信息获取对应的标签路径列表
	ListTagPathByResource(resourceType int8, resourceCode string) []string

	// 根据账号id获取其可访问标签信息
	ListTagByAccountId(accountId uint64) []string

	// 账号是否有权限访问该标签关联的资源信息
	CanAccess(accountId uint64, tagPath ...string) error

	// 填充资源的标签信息
	FillTagInfo(resources ...entity.ITagResource)
}

type tagTreeAppImpl struct {
	base.AppImpl[*entity.TagTree, repository.TagTree]

	tagTreeTeamRepo repository.TagTreeTeam `inject:"TagTreeTeamRepo"`
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
			// if p.tagResourceApp.CountByCond(&entity.TagResource{TagId: tag.Pid}) > 0 {
			// 	return errorx.NewBiz("该父标签已关联资源, 无法添加子标签")
			// }

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

		// 普通标签类型
		tag.Type = -1
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

func (p *tagTreeAppImpl) GetAccountTagResources(accountId uint64, query *entity.TagTreeQuery) []*entity.TagTree {
	tagResourceQuery := &entity.TagTreeQuery{
		Type: query.Type,
	}

	var tagResources []*entity.TagTree
	var accountTagPaths []string

	if accountId != consts.AdminId {
		// 获取账号有权限操作的标签路径列表
		accountTagPaths = p.ListTagByAccountId(accountId)
		if len(accountTagPaths) == 0 {
			return tagResources
		}
	}

	// 去除空字符串标签
	tagPaths := collx.ArrayRemoveBlank(query.CodePathLikes)
	// 如果需要查询指定标签下的资源标签，则需要与用户拥有的权限进行过滤，避免越权
	if len(tagPaths) > 0 {
		// admin 则直接赋值需要获取的标签
		if len(accountTagPaths) == 0 {
			accountTagPaths = tagPaths
		} else {
			accountTagPaths = collx.ArrayFilter[string](tagPaths, func(s string) bool {
				for _, v := range accountTagPaths {
					// 要过滤的权限需要在用户拥有的子标签下, accountTagPath: test/  tagPath: test/test1/ -> true
					if strings.HasPrefix(v, s) {
						return true
					}
				}
				return false
			})
		}
	}

	// tagResourceQuery.CodePathLike = tagPath
	tagResourceQuery.Codes = query.Codes
	tagResourceQuery.CodePathLikes = accountTagPaths
	p.ListByQuery(tagResourceQuery, &tagResources)
	return tagResources
}

func (p *tagTreeAppImpl) GetAccountResourceCodes(accountId uint64, resourceType int8, tagPath string) []string {
	tagResources := p.GetAccountTagResources(accountId, &entity.TagTreeQuery{Type: entity.TagType(resourceType), CodePathLikes: []string{tagPath}})
	// resouce code去重
	code2Resource := collx.ArrayToMap[*entity.TagTree, string](tagResources, func(val *entity.TagTree) string {
		return val.Code
	})

	return collx.MapKeys(code2Resource)
}

func (p *tagTreeAppImpl) SaveResource(ctx context.Context, req *SaveResourceTagParam) error {
	resourceCode := req.ResourceCode
	resourceType := entity.TagType(req.ResourceType)
	resourceName := req.ResourceName
	tagIds := req.TagIds

	if resourceCode == "" {
		return errorx.NewBiz("资源编号不能为空")
	}
	if resourceType == 0 {
		return errorx.NewBiz("资源类型不能为空")
	}

	// 如果tagIds为空数组，则为删除该资源标签
	if len(tagIds) == 0 {
		return p.DeleteResource(ctx, &DelResourceTagParam{
			ResourceType: resourceType,
			ResourceCode: resourceCode,
		})
	}

	if resourceName == "" {
		resourceName = resourceCode
	}

	// 该资源对应的旧资源标签信息
	var oldTagTree []*entity.TagTree
	p.ListByCond(&entity.TagTree{Type: resourceType, Code: resourceCode}, &oldTagTree)

	var addTagIds, delTagIds []uint64
	if len(oldTagTree) == 0 {
		addTagIds = tagIds
	} else {
		oldTagIds := collx.ArrayMap(oldTagTree, func(tag *entity.TagTree) uint64 {
			return tag.Pid
		})
		addTagIds, delTagIds, _ = collx.ArrayCompare[uint64](tagIds, oldTagIds)
	}

	if len(addTagIds) > 0 {
		addTagResource := make([]*entity.TagTree, 0)
		for _, tagId := range addTagIds {
			tag, err := p.GetById(new(entity.TagTree), tagId)
			if err != nil {
				return errorx.NewBiz("存在错误标签id")
			}
			addTagResource = append(addTagResource, &entity.TagTree{
				Pid:      tagId,
				Code:     resourceCode,
				Type:     resourceType,
				Name:     resourceName,
				CodePath: tag.CodePath + resourceCode + entity.CodePathSeparator,
			})
		}
		if err := p.BatchInsert(ctx, addTagResource); err != nil {
			return err
		}
	}

	if len(delTagIds) > 0 {
		for _, tagId := range delTagIds {
			if err := p.DeleteResource(ctx, &DelResourceTagParam{
				ResourceType: resourceType,
				ResourceCode: resourceCode,
				Pid:          tagId,
			}); err != nil {
				return err
			}

		}
	}

	return nil
}

func (p *tagTreeAppImpl) DeleteResource(ctx context.Context, param *DelResourceTagParam) error {
	// 获取资源编号对应的资源标签信息
	var resourceTags []*entity.TagTree
	p.ListByCond(&entity.TagTree{Type: param.ResourceType, Code: param.ResourceCode, Pid: param.Pid}, &resourceTags)
	if len(resourceTags) == 0 {
		return nil
	}

	delTagType := param.ChildType
	for _, resourceTag := range resourceTags {
		// 删除所有code_path下的子标签
		if err := p.DeleteByWheres(ctx, collx.M{
			"code_path LIKE ?": resourceTag.CodePath + "%",
			"type = ?":         delTagType,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (p *tagTreeAppImpl) ListTagPathByResource(resourceType int8, resourceCode string) []string {
	var trs []*entity.TagTree
	p.ListByCond(&entity.TagTree{Type: entity.TagType(resourceType), Code: resourceCode}, &trs)
	return collx.ArrayMap(trs, func(tr *entity.TagTree) string {
		return tr.CodePath
	})
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

func (p *tagTreeAppImpl) FillTagInfo(resources ...entity.ITagResource) {
	if len(resources) == 0 {
		return
	}

	// 资源编号 -> 资源
	resourceCode2Resouce := collx.ArrayToMap(resources, func(rt entity.ITagResource) string {
		return rt.GetCode()
	})

	// 获取所有资源code关联的标签列表信息
	var tagResources []*entity.TagTree
	p.ListByQuery(&entity.TagTreeQuery{Codes: collx.MapKeys(resourceCode2Resouce)}, &tagResources)

	for _, tr := range tagResources {
		// 赋值标签信息
		resourceCode2Resouce[tr.Code].SetTagInfo(entity.ResourceTag{TagId: tr.Pid, TagPath: tr.GetParentPath(0)})
	}
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

	if p.CountByCond(&entity.TagTree{Pid: id}) > 0 {
		return errorx.NewBiz("请先移除该标签关联的资源")
	}

	return p.Tx(ctx, func(ctx context.Context) error {
		return p.DeleteById(ctx, id)
	}, func(ctx context.Context) error {
		// 删除该标签关联的团队信息
		return p.tagTreeTeamRepo.DeleteByCond(ctx, &entity.TagTreeTeam{TagId: id})
	})
}
