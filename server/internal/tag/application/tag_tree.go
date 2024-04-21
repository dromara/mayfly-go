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

// 保存资源标签参数
type SaveResourceTagParam struct {
	ParentTagCodePaths []string // 关联标签，空数组则为删除该资源绑定的标签

	ResourceTag *ResourceTag // 资源标签信息
}

type ResourceTag struct {
	Code string
	Type entity.TagType
	Name string

	Children []*ResourceTag // 子资源标签
}

type RelateTagsByCodeAndTypeParam struct {
	ParentTagCode string         // 父标签编号
	ParentTagType entity.TagType // 父标签类型

	Tags []*ResourceTag // 要关联的标签数组
}

type DelResourceTagParam struct {
	Id           uint64
	ResourceCode string
	ResourceType entity.TagType

	// 要删除的子节点类型，若存在值，则为删除资源标签下的指定类型的子标签
	ChildType entity.TagType
}

type TagTree interface {
	base.App[*entity.TagTree]

	ListByQuery(condition *entity.TagTreeQuery, toEntity any)

	SaveTag(ctx context.Context, pid uint64, tt *entity.TagTree) error

	Delete(ctx context.Context, id uint64) error

	// GetAccountTags 获取指定账号有权限操作的标签列表
	// @param accountId 账号id
	// @param query 查询条件
	GetAccountTags(accountId uint64, query *entity.TagTreeQuery) []*entity.TagTree

	// GetAccountTagCodes 获取指定账号有权限操作的标签codes
	GetAccountTagCodes(accountId uint64, resourceType int8, tagPath string) []string

	// GetAccountTagCodePaths 获取指定账号有权限操作的codePaths
	GetAccountTagCodePaths(accountId uint64, tagType entity.TagType, tagPath string) []string

	// SaveResourceTag 保存资源类型标签
	SaveResourceTag(ctx context.Context, param *SaveResourceTagParam) error

	// RelateTagsByCodeAndType 将指定标签数组关联至满足指定标签类型和标签code的标签下
	RelateTagsByCodeAndType(ctx context.Context, param *RelateTagsByCodeAndTypeParam) error

	// UpdateTagName 根据标签类型与code更新对应标签名
	UpdateTagName(ctx context.Context, tagType entity.TagType, tagCode string, tagName string) error

	// ChangeParentTag 变更指定类型标签的父标签
	ChangeParentTag(ctx context.Context, tagType entity.TagType, tagCode string, parentTagType entity.TagType, newParentCode string) error

	// MovingTag 移动标签
	MovingTag(ctx context.Context, fromTagPath string, toTagPath string) error

	// DeleteTagByParam 删除标签，会删除该标签下所有子标签信息以及团队关联的标签信息
	DeleteTagByParam(ctx context.Context, param *DelResourceTagParam) error

	// 根据标签类型和标签code获取对应的标签路径列表
	ListTagPathByTypeAndCode(resourceType int8, resourceCode string) []string

	// ListTagByAccountId 根据账号id获取其可访问标签信息
	ListTagByAccountId(accountId uint64) []string

	// CanAccess 账号是否有权限访问该标签关联的资源信息
	CanAccess(accountId uint64, tagPath ...string) error

	// FillTagInfo 填充资源的标签信息
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

func (p *tagTreeAppImpl) SaveTag(ctx context.Context, pid uint64, tag *entity.TagTree) error {
	accountId := contextx.GetLoginAccount(ctx).Id
	// 新建项目树节点信息
	if tag.Id == 0 {
		if strings.Contains(tag.Code, entity.CodePathSeparator) {
			return errorx.NewBiz("标识符不能包含'/'")
		}
		if pid != 0 {
			parentTag, err := p.GetById(new(entity.TagTree), pid)
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

func (p *tagTreeAppImpl) GetAccountTagCodePaths(accountId uint64, tagType entity.TagType, tagPath string) []string {
	tagResources := p.GetAccountTags(accountId, &entity.TagTreeQuery{Type: tagType, CodePathLikes: []string{tagPath}})
	// resouce code去重
	code2Resource := collx.ArrayToMap[*entity.TagTree, string](tagResources, func(val *entity.TagTree) string {
		return val.CodePath
	})

	return collx.MapKeys(code2Resource)
}

func (p *tagTreeAppImpl) SaveResourceTag(ctx context.Context, param *SaveResourceTagParam) error {
	code := param.ResourceTag.Code
	tagType := param.ResourceTag.Type
	parentTagCodePaths := param.ParentTagCodePaths

	if code == "" {
		return errorx.NewBiz("保存资源标签失败: 资源编号不能为空")
	}
	if tagType == 0 {
		return errorx.NewBiz("保存资源标签失败:资源类型不能为空")
	}

	// 如果tagIds为空数组，则为删除该资源标签
	if len(parentTagCodePaths) == 0 {
		return p.DeleteTagByParam(ctx, &DelResourceTagParam{
			ResourceType: tagType,
			ResourceCode: code,
		})
	}

	// 获取所有关联的父标签
	var parentTags []*entity.TagTree
	p.ListByWheres(collx.M{"code_path in ?": parentTagCodePaths}, &parentTags)
	if len(parentTags) == 0 || len(parentTags) != len(parentTagCodePaths) {
		return errorx.NewBiz("保存资源标签失败: 存在错误的关联标签")
	}

	newTags := p.toTags(parentTags, param.ResourceTag)

	var oldParentTagTree []*entity.TagTree
	p.ListByCond(&entity.TagTree{Type: tagType, Code: code}, &oldParentTagTree)

	// 该资源对应的旧资源标签信息
	var oldChildrenTags []*entity.TagTree
	if len(oldParentTagTree) > 0 {
		// 获取所有旧的子标签
		p.ListByQuery(&entity.TagTreeQuery{
			CodePathLikes: collx.ArrayMap[*entity.TagTree, string](oldParentTagTree, func(val *entity.TagTree) string {
				return val.CodePath
			}),
		}, &oldChildrenTags)
	}

	// 旧的codePath -> tag
	oldCodePath2Tag := collx.ArrayToMap[*entity.TagTree, string](oldChildrenTags, func(val *entity.TagTree) string { return val.CodePath })
	// 新的codePath -> tag
	newCodePath2Tag := collx.ArrayToMap[*entity.TagTree, string](newTags, func(val *entity.TagTree) string { return val.CodePath })

	var addCodePaths, delCodePaths []string
	addCodePaths, delCodePaths, _ = collx.ArrayCompare(collx.MapKeys(newCodePath2Tag), collx.MapKeys(oldCodePath2Tag))

	if len(addCodePaths) > 0 {
		logx.DebugfContext(ctx, "SaveResourceTag-新增标签[%v]", addCodePaths)
		addTags := make([]*entity.TagTree, 0)
		for _, addCodePath := range addCodePaths {
			addTags = append(addTags, newCodePath2Tag[addCodePath])
		}
		if err := p.BatchInsert(ctx, addTags); err != nil {
			return err
		}
	}

	if len(delCodePaths) > 0 {
		logx.DebugfContext(ctx, "SaveResourceTag-删除标签[%v]", delCodePaths)

		var delTagIds []uint64
		for _, delCodePath := range delCodePaths {
			delTag := oldCodePath2Tag[delCodePath]
			if delTag != nil && delTag.Id != 0 {
				delTagIds = append(delTagIds, delTag.Id)
			}
		}

		return p.deleteByIds(ctx, delTagIds)
	}

	return nil
}

func (p *tagTreeAppImpl) RelateTagsByCodeAndType(ctx context.Context, param *RelateTagsByCodeAndTypeParam) error {
	parentTagCode := param.ParentTagCode
	parentTagType := param.ParentTagType

	// 获取满足指定编号与类型的所有标签信息
	var parentTags []*entity.TagTree
	p.ListByCond(&entity.TagTree{Type: parentTagType, Code: parentTagCode}, &parentTags)
	// 标签codePaths（相当于需要关联的标签数组的父tag）
	parentTagCodePaths := collx.ArrayMap(parentTags, func(tag *entity.TagTree) string {
		return tag.CodePath
	})

	if len(parentTagCodePaths) == 0 {
		return errorx.NewBiz("不存在满足[type=%d, code=%s]的标签", parentTagType, parentTagCode)
	}

	for _, tag := range param.Tags {
		if err := (p.SaveResourceTag(ctx, &SaveResourceTagParam{
			ResourceTag:        tag,
			ParentTagCodePaths: parentTagCodePaths,
		})); err != nil {
			return err
		}
	}
	return nil
}

func (p *tagTreeAppImpl) UpdateTagName(ctx context.Context, tagType entity.TagType, tagCode string, tagName string) error {
	return p.UpdateByWheres(ctx, &entity.TagTree{Name: tagName}, collx.Kvs("code = ?", tagCode, "type = ?", tagType))
}

func (p *tagTreeAppImpl) ChangeParentTag(ctx context.Context, tagType entity.TagType, tagCode string, parentTagType entity.TagType, newParentCode string) error {
	// 获取资源编号对应的资源标签信息
	var resourceTags []*entity.TagTree
	p.ListByCond(&entity.TagTree{Type: tagType, Code: tagCode}, &resourceTags)
	if len(resourceTags) == 0 {
		logx.WarnfContext(ctx, "ChangeParentTag-[%d-%s]标签信息不存在", tagType, tagCode)
		return nil
	}

	if p.CountByCond(&entity.TagTree{Type: parentTagType, Code: newParentCode}) == 0 {
		return errorx.NewBiz("该父标签不存在")
	}

	// 获取该资源编号对应的所有子资源标签信息
	var resourceChildrenTags []*entity.TagTree
	p.ListByQuery(&entity.TagTreeQuery{CodePathLikes: collx.ArrayMap(resourceTags, func(tag *entity.TagTree) string {
		return tag.CodePath
	})}, &resourceChildrenTags)

	// 更新父标签的codepath
	for _, tag := range resourceChildrenTags {
		pathSection := entity.GetTagPathSections(tag.CodePath)
		for i, ps := range pathSection {
			if ps.Type == tagType && ps.Code == tagCode {
				// 将父标签编号修改为对应的新编号与类型
				pathSection[i-1].Code = newParentCode
				pathSection[i-1].Type = parentTagType
			}
		}

		tag.CodePath = pathSection.ToCodePath()
		if err := p.UpdateById(ctx, tag); err != nil {
			return err
		}
	}

	return nil
}

func (p *tagTreeAppImpl) MovingTag(ctx context.Context, fromTagPath string, toTagPath string) error {
	fromTag := &entity.TagTree{CodePath: fromTagPath}
	if err := p.GetBy(fromTag); err != nil {
		return errorx.NewBiz("移动标签不存在")
	}

	toTag := &entity.TagTree{CodePath: toTagPath}
	if err := p.GetBy(toTag); err != nil {
		return errorx.NewBiz("目标标签不存在")
	}

	// 获取要移动标签的所有子标签
	var childrenTags []*entity.TagTree
	p.ListByQuery(&entity.TagTreeQuery{CodePathLike: fromTagPath}, &childrenTags)

	// 获取父路径, 若fromTagPath=tag1/tag2/1|xxx则返回 tag1/tag2/
	fromParentPath := entity.GetParentPath(fromTagPath, 0)
	for _, childTag := range childrenTags {
		// 替换path，若childPath = tag1/tag2/1|xxx/11|yyy, toTagPath=tag3/tag4则替换为tag3/tag4/1|xxx/11|yyy/
		childTag.CodePath = strings.Replace(childTag.CodePath, fromParentPath, toTagPath, 1)
		if err := p.UpdateById(ctx, childTag); err != nil {
			return err
		}
	}

	return nil
}

func (p *tagTreeAppImpl) DeleteTagByParam(ctx context.Context, param *DelResourceTagParam) error {
	// 获取资源编号对应的资源标签信息
	var resourceTags []*entity.TagTree
	cond := &entity.TagTree{Type: param.ResourceType, Code: param.ResourceCode}
	cond.Id = param.Id
	p.ListByCond(cond, &resourceTags)

	if len(resourceTags) == 0 {
		logx.DebugfContext(ctx, "TagTreeApp.DeleteTagByParam[%d-%s]不存在可删除的标签", param.ResourceType, param.ResourceCode)
		return nil
	}

	delTagType := param.ChildType
	for _, resourceTag := range resourceTags {
		// 获取所有关联的子标签
		var childrenTag []*entity.TagTree
		p.ListByWheres(collx.M{
			"code_path LIKE ?": resourceTag.CodePath + "%",
			"type = ?":         delTagType,
		}, &childrenTag)
		if len(childrenTag) == 0 {
			continue
		}

		childrenTagIds := collx.ArrayMap(childrenTag, func(item *entity.TagTree) uint64 {
			return item.Id
		})
		// 删除code_path下的所有子标签
		return p.deleteByIds(ctx, childrenTagIds)
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
		resourceCode2Resouce[tr.Code].SetTagInfo(entity.ResourceTag{CodePath: tr.GetTagPath()})
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

	return p.DeleteTagByParam(ctx, &DelResourceTagParam{
		Id: id,
	})
}

func (p *tagTreeAppImpl) toTags(parentTags []*entity.TagTree, param *ResourceTag) []*entity.TagTree {
	tags := make([]*entity.TagTree, 0)

	// 递归函数，将标签及其子标签展开为一个扁平数组
	var flattenTags func(parentTag *entity.TagTree, tag *ResourceTag)
	flattenTags = func(parentTag *entity.TagTree, resourceTagParam *ResourceTag) {
		if resourceTagParam == nil {
			return
		}

		tagType := resourceTagParam.Type
		tagCode := resourceTagParam.Code
		tagName := resourceTagParam.Name
		if tagName == "" {
			tagName = tagCode
		}

		tag := &entity.TagTree{
			Code:     tagCode,
			CodePath: fmt.Sprintf("%s%d%s%s%s", parentTag.CodePath, tagType, entity.CodePathResourceSeparator, tagCode, entity.CodePathSeparator), // tag1/tag2/1|resourceCode1/11|resourceCode2/
			Type:     tagType,
			Name:     tagName,
		}
		// 将当前标签加入数组
		tags = append(tags, tag)

		// 递归处理子标签
		for _, child := range resourceTagParam.Children {
			flattenTags(tag, child)
		}
	}

	for _, parentTag := range parentTags {
		// 开始展开标签
		flattenTags(parentTag, param)
	}

	return tags
}

func (p *tagTreeAppImpl) deleteByIds(ctx context.Context, tagIds []uint64) error {
	if err := p.DeleteByWheres(ctx, collx.M{
		"id in ?": tagIds,
	}); err != nil {
		return err
	}

	// 删除team关联的标签
	return p.tagTreeTeamRepo.DeleteByWheres(ctx, collx.M{"tag_id in ?": tagIds})
}
