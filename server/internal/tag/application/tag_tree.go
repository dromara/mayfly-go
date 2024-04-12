package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"strings"
)

// 标签接口，实现了该接口的结构体默认都可以当成标签树的一种标签
type ITag interface {
	// 资源标签code
	GetCode() string

	// 资源标签名
	GetName() string
}

// 保存标签参数
type SaveResourceTagParam struct {
	Code string
	Name string
	Type entity.TagType

	ParentTagIds []uint64 // 关联标签，空数组则为删除该资源绑定的标签
}

type RelateTagsByCodeAndTypeParam struct {
	ParentTagCode string         // 父标签编号
	ParentTagType entity.TagType // 父标签类型

	TagType entity.TagType // 要关联的标签类型
	Tags    []ITag         // 要关联的标签数组
}

type DelResourceTagParam struct {
	Id           uint64
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

	// 获取指定账号有权限操作的标签列表
	// @param accountId 账号id
	// @param query 查询条件
	GetAccountTags(accountId uint64, query *entity.TagTreeQuery) []*entity.TagTree

	// 获取指定账号有权限操作的标签codes
	GetAccountTagCodes(accountId uint64, resourceType int8, tagPath string) []string

	// SaveResourceTag 保存资源类型标签
	SaveResourceTag(ctx context.Context, param *SaveResourceTagParam) error

	// RelateTagsByCodeAndType 将指定标签数组关联至满足指定标签类型和标签code的标签下
	RelateTagsByCodeAndType(ctx context.Context, param *RelateTagsByCodeAndTypeParam) error

	// DeleteTagByParam 删除标签，会删除该标签下所有子标签信息以及团队关联的标签信息
	DeleteTagByParam(ctx context.Context, param *DelResourceTagParam) error

	// 根据标签类型和标签code获取对应的标签路径列表
	ListTagPathByTypeAndCode(resourceType int8, resourceCode string) []string

	// 根据账号id获取其可访问标签信息
	ListTagByAccountId(accountId uint64) []string

	// 账号是否有权限访问该标签关联的资源信息
	CanAccess(accountId uint64, tagPath ...string) error

	// 填充资源的标签信息
	FillTagInfo(resourceTagType entity.TagType, resources ...entity.ITagResource)
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
		tag.Type = entity.TagTypeTag
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

func (p *tagTreeAppImpl) GetAccountTags(accountId uint64, query *entity.TagTreeQuery) []*entity.TagTree {
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
			queryPaths := collx.ArrayFilter[string](tagPaths, func(tagPath string) bool {
				for _, acPath := range accountTagPaths {
					// 查询条件： a/b/  有权的：a/  查询结果应该是  a/b/
					if strings.HasPrefix(tagPath, acPath) {
						return true
					}
				}
				return false
			})

			acPaths := collx.ArrayFilter[string](accountTagPaths, func(acPath string) bool {
				for _, tagPath := range tagPaths {
					// 查询条件： a/  有权的：a/b/  查询结果应该是  a/b/
					if strings.HasPrefix(acPath, tagPath) {
						return true
					}
				}
				return false
			})

			accountTagPaths = append(queryPaths, acPaths...)
		}
	}

	tagResourceQuery.Codes = query.Codes
	tagResourceQuery.CodePathLikes = accountTagPaths
	p.ListByQuery(tagResourceQuery, &tagResources)
	return tagResources
}

func (p *tagTreeAppImpl) GetAccountTagCodes(accountId uint64, resourceType int8, tagPath string) []string {
	tagResources := p.GetAccountTags(accountId, &entity.TagTreeQuery{Type: entity.TagType(resourceType), CodePathLikes: []string{tagPath}})
	// resouce code去重
	code2Resource := collx.ArrayToMap[*entity.TagTree, string](tagResources, func(val *entity.TagTree) string {
		return val.Code
	})

	return collx.MapKeys(code2Resource)
}

func (p *tagTreeAppImpl) SaveResourceTag(ctx context.Context, param *SaveResourceTagParam) error {
	code := param.Code
	tagType := entity.TagType(param.Type)
	name := param.Name
	tagIds := param.ParentTagIds

	if code == "" {
		return errorx.NewBiz("资源编号不能为空")
	}
	if tagType == 0 {
		return errorx.NewBiz("资源类型不能为空")
	}

	// 如果tagIds为空数组，则为删除该资源标签
	if len(tagIds) == 0 {
		return p.DeleteTagByParam(ctx, &DelResourceTagParam{
			ResourceType: tagType,
			ResourceCode: code,
		})
	}

	if name == "" {
		name = code
	}

	// 该资源对应的旧资源标签信息
	var oldTagTree []*entity.TagTree
	p.ListByCond(&entity.TagTree{Type: tagType, Code: code}, &oldTagTree)

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
				Code:     code,
				Type:     tagType,
				Name:     name,
				CodePath: fmt.Sprintf("%s%d%s%s%s", tag.CodePath, tagType, entity.CodePathResourceSeparator, code, entity.CodePathSeparator), // tag1/tag2/1|resourceCode1/11|resourceCode2
			})

		}
		if err := p.BatchInsert(ctx, addTagResource); err != nil {
			return err
		}
	}

	if len(delTagIds) > 0 {
		for _, tagId := range delTagIds {
			if err := p.DeleteTagByParam(ctx, &DelResourceTagParam{
				ResourceType: tagType,
				ResourceCode: code,
				Pid:          tagId,
			}); err != nil {
				return err
			}

		}
	}

	return nil
}

func (p *tagTreeAppImpl) RelateTagsByCodeAndType(ctx context.Context, param *RelateTagsByCodeAndTypeParam) error {
	parentTagCode := param.ParentTagCode
	parentTagType := param.ParentTagType
	tagType := param.TagType

	// 如果资源为，则表示清楚关联
	if len(param.Tags) == 0 {
		// 删除该资源下的所有指定类型的资源
		return p.DeleteTagByParam(ctx, &DelResourceTagParam{
			ResourceCode: parentTagCode,
			ResourceType: param.ParentTagType,
			ChildType:    tagType,
		})
	}

	// 获取满足指定编号与类型的所有标签信息
	var parentTags []*entity.TagTree
	p.ListByCond(&entity.TagTree{Type: parentTagType, Code: parentTagCode}, &parentTags)
	// 标签id（相当于需要关联的标签数组的父tag id）
	parentTagIds := collx.ArrayMap(parentTags, func(tag *entity.TagTree) uint64 {
		return tag.Id
	})

	if len(parentTagIds) == 0 {
		return errorx.NewBiz("不存在满足[type=%d, code=%s]的标签", parentTagType, parentTagCode)
	}

	var oldChildrenTags []*entity.TagTree
	// 获取该资源的所有旧的该类型的子标签
	p.ListByQuery(&entity.TagTreeQuery{
		CodePathLikes: collx.ArrayMap[*entity.TagTree, string](parentTags, func(val *entity.TagTree) string { return val.CodePath }),
		Type:          tagType,
	}, &oldChildrenTags)

	// 组合新的授权凭证资源标签
	newTags := make([]*entity.TagTree, 0)
	for _, resourceTag := range parentTags {
		for _, resource := range param.Tags {
			tagCode := resource.GetCode()
			newTags = append(newTags, &entity.TagTree{
				Pid:      resourceTag.Id,
				Type:     tagType,
				Code:     tagCode,
				CodePath: fmt.Sprintf("%s%d%s%s%s", resourceTag.CodePath, tagType, entity.CodePathResourceSeparator, tagCode, entity.CodePathSeparator),
				Name:     resource.GetName(),
			})
		}
	}

	// 旧的codePath -> tag
	oldCodePath2Tag := collx.ArrayToMap[*entity.TagTree, string](oldChildrenTags, func(val *entity.TagTree) string { return val.CodePath })
	// 新的codePath -> tag
	newCodePath2Tag := collx.ArrayToMap[*entity.TagTree, string](newTags, func(val *entity.TagTree) string { return val.CodePath })

	var addCodePaths, delCodePaths []string
	addCodePaths, delCodePaths, _ = collx.ArrayCompare(collx.MapKeys(newCodePath2Tag), collx.MapKeys(oldCodePath2Tag))

	if len(addCodePaths) > 0 {
		logx.DebugfContext(ctx, "RelateTags2CodeAndType[%d-%s]-新增标签[%v]", parentTagType, parentTagCode, addCodePaths)

		addTags := make([]*entity.TagTree, 0)
		for _, addCodePath := range addCodePaths {
			addTags = append(addTags, newCodePath2Tag[addCodePath])
		}
		if err := p.BatchInsert(ctx, addTags); err != nil {
			return err
		}
	}

	if len(delCodePaths) > 0 {
		logx.DebugfContext(ctx, "RelateTags2CodeAndType[%d-%s]-删除标签[%v]", parentTagType, parentTagCode, delCodePaths)

		for _, delCodePath := range delCodePaths {
			oldTag := oldCodePath2Tag[delCodePath]
			if oldTag == nil {
				continue
			}
			if err := p.DeleteTagByParam(ctx, &DelResourceTagParam{
				Id: oldTag.Id,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *tagTreeAppImpl) DeleteTagByParam(ctx context.Context, param *DelResourceTagParam) error {
	// 获取资源编号对应的资源标签信息
	var resourceTags []*entity.TagTree
	cond := &entity.TagTree{Type: param.ResourceType, Code: param.ResourceCode, Pid: param.Pid}
	cond.Id = param.Id
	p.ListByCond(cond, &resourceTags)

	if len(resourceTags) == 0 {
		logx.DebugfContext(ctx, "TagTreeApp.DeleteResource[%d-%s]不存在可删除的标签", param.ResourceType, param.ResourceCode)
		return nil
	}

	delTagType := param.ChildType
	for _, resourceTag := range resourceTags {
		// 获取所有关联的子标签
		var childrenTag []*entity.TagTree
		p.Repo.ListByWheres(collx.M{
			"code_path LIKE ?": resourceTag.CodePath + "%",
			"type = ?":         delTagType,
		}, &childrenTag)
		if len(childrenTag) == 0 {
			continue
		}

		childrenTagIds := collx.ArrayMap(childrenTag, func(item *entity.TagTree) uint64 {
			return item.Id
		})
		// 删除所有code_path下的子标签
		if err := p.DeleteByWheres(ctx, collx.M{
			"id in ?": childrenTagIds,
		}); err != nil {
			return err
		}

		// 删除team关联的标签
		return p.tagTreeTeamRepo.DeleteByWheres(ctx, collx.M{"tag_id in ?": childrenTagIds})
	}

	return nil
}

func (p *tagTreeAppImpl) ListTagPathByTypeAndCode(resourceType int8, resourceCode string) []string {
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

func (p *tagTreeAppImpl) FillTagInfo(resourceTagType entity.TagType, resources ...entity.ITagResource) {
	if len(resources) == 0 {
		return
	}

	// 资源编号 -> 资源
	resourceCode2Resouce := collx.ArrayToMap(resources, func(rt entity.ITagResource) string {
		return rt.GetCode()
	})

	// 获取所有资源code关联的标签列表信息
	var tagResources []*entity.TagTree
	p.ListByQuery(&entity.TagTreeQuery{Codes: collx.MapKeys(resourceCode2Resouce), Type: resourceTagType}, &tagResources)

	for _, tr := range tagResources {
		// 赋值标签信息
		resourceCode2Resouce[tr.Code].SetTagInfo(entity.ResourceTag{TagId: tr.Pid, TagPath: tr.GetTagPath()})
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
