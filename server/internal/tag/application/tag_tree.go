package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/internal/tag/application/dto"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/internal/tag/imsg"
	"mayfly-go/internal/tag/infrastructure/cache"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"slices"
	"strings"

	"github.com/may-fly/cast"
)

type TagTree interface {
	base.App[*entity.TagTree]

	ListByQuery(condition *entity.TagTreeQuery, toEntity any)

	SaveTag(ctx context.Context, pid uint64, tt *entity.TagTree) error

	// SaveResourceTag 保存资源类型标签
	SaveResourceTag(ctx context.Context, param *dto.SaveResourceTag) error

	// RelateTagsByCodeAndType 将指定标签数组关联至满足指定标签类型和标签code的标签下
	RelateTagsByCodeAndType(ctx context.Context, param *dto.RelateTagsByCodeAndType) error

	// UpdateTagName 根据标签类型与code更新对应标签名
	UpdateTagName(ctx context.Context, tagType entity.TagType, tagCode string, tagName string) error

	// ChangeParentTag 变更指定类型标签的父标签
	ChangeParentTag(ctx context.Context, tagType entity.TagType, tagCode string, parentTagType entity.TagType, newParentCode string) error

	// MovingTag 移动标签
	MovingTag(ctx context.Context, fromTagPath string, toTagPath string) error

	// DeleteTagByParam 删除标签，会删除该标签下所有子标签信息以及团队关联的标签信息
	DeleteTagByParam(ctx context.Context, param *dto.DelResourceTag) error

	Delete(ctx context.Context, id uint64) error

	// GetAccountTags 获取指定账号有权限操作的标签列表
	// @param accountId 账号id
	// @param query 查询条件
	GetAccountTags(accountId uint64, query *entity.TagTreeQuery) dto.SimpleTagTrees

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

	tagTreeRelateApp TagTreeRelate `inject:"T"`
}

var _ (TagTree) = (*tagTreeAppImpl)(nil)

func (p *tagTreeAppImpl) SaveTag(ctx context.Context, pid uint64, tag *entity.TagTree) error {
	accountId := contextx.GetLoginAccount(ctx).Id
	// 新建项目树节点信息
	if tag.Id == 0 {
		if strings.Contains(tag.Code, entity.CodePathSeparator) {
			return errorx.NewBizI(ctx, imsg.ErrTagCodeInvalid)
		}
		if pid != 0 {
			parentTag, err := p.GetById(pid)
			if err != nil {
				return errorx.NewBiz("parent tag not found")
			}
			// if p.tagResourceApp.CountByCond(&entity.TagResource{TagId: tag.Pid}) > 0 {
			// 	return errorx.NewBiz("该父标签已关联资源, 无法添加子标签")
			// }

			tag.CodePath = parentTag.CodePath + tag.Code + entity.CodePathSeparator
		} else {
			if accountId != consts.AdminId {
				return errorx.NewBizI(ctx, imsg.ErrNoAdminCreateTag)
			}
			tag.CodePath = tag.Code + entity.CodePathSeparator
		}
		if p.CanAccess(accountId, tag.CodePath) != nil {
			return errorx.NewBizI(ctx, imsg.ErrNoPermissionCreateTag)
		}

		// 判断该路径是否存在
		var hasLikeTags []entity.TagTree
		p.GetRepo().SelectByCondition(&entity.TagTreeQuery{CodePathLikes: []string{tag.CodePath}}, &hasLikeTags)
		if len(hasLikeTags) > 0 {
			return errorx.NewBizI(ctx, imsg.ErrTagCodePathLikeExist)
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

func (p *tagTreeAppImpl) SaveResourceTag(ctx context.Context, param *dto.SaveResourceTag) error {
	code := param.ResourceTag.Code
	tagType := param.ResourceTag.Type
	parentTagCodePaths := param.ParentTagCodePaths

	if code == "" {
		return errorx.NewBiz("save resource tag failed: resource code can not be empty")
	}
	if tagType == 0 {
		return errorx.NewBiz("save resource tag failed: resource type can not be empty")
	}

	// 如果tagIds为空数组，则为删除该资源标签
	if len(parentTagCodePaths) == 0 {
		return p.DeleteTagByParam(ctx, &dto.DelResourceTag{
			ResourceType: tagType,
			ResourceCode: code,
		})
	}

	// 获取所有关联的父标签
	parentTags, _ := p.ListByCond(model.NewCond().In("code_path", parentTagCodePaths))
	if len(parentTags) == 0 || len(parentTags) != len(parentTagCodePaths) {
		// 存在错误的关联标签
		return errorx.NewBiz("save resource tag failed: There is an incorrect relate tag")
	}

	newTags := p.toTags(parentTags, param.ResourceTag)

	oldParentTagTree, _ := p.ListByCond(&entity.TagTree{Type: tagType, Code: code})

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
		logx.DebugfContext(ctx, "SaveResourceTag - add tag[%v]", addCodePaths)
		addTags := make([]*entity.TagTree, 0)
		for _, addCodePath := range addCodePaths {
			addTags = append(addTags, newCodePath2Tag[addCodePath])
		}
		if err := p.BatchInsert(ctx, addTags); err != nil {
			return err
		}
	}

	if len(delCodePaths) > 0 {
		logx.DebugfContext(ctx, "SaveResourceTag - delete tag[%v]", delCodePaths)

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

func (p *tagTreeAppImpl) RelateTagsByCodeAndType(ctx context.Context, param *dto.RelateTagsByCodeAndType) error {
	parentTagCode := param.ParentTagCode
	parentTagType := param.ParentTagType

	// 获取满足指定编号与类型的所有标签信息
	parentTags, _ := p.ListByCond(&entity.TagTree{Type: parentTagType, Code: parentTagCode})
	// 标签codePaths（相当于需要关联的标签数组的父tag）
	parentTagCodePaths := collx.ArrayMap(parentTags, func(tag *entity.TagTree) string {
		return tag.CodePath
	})

	if len(parentTagCodePaths) == 0 {
		// 不满足满足条件的标签
		return errorx.NewBiz("There is no tag that satisfies [type=%d, code=%s]", parentTagType, parentTagCode)
	}

	for _, tag := range param.Tags {
		if err := (p.SaveResourceTag(ctx, &dto.SaveResourceTag{
			ResourceTag:        tag,
			ParentTagCodePaths: parentTagCodePaths,
		})); err != nil {
			return err
		}
	}

	return nil
}

func (p *tagTreeAppImpl) UpdateTagName(ctx context.Context, tagType entity.TagType, tagCode string, tagName string) error {
	return p.UpdateByCond(ctx, &entity.TagTree{Name: tagName}, &entity.TagTree{Type: tagType, Code: tagCode})
}

func (p *tagTreeAppImpl) ChangeParentTag(ctx context.Context, tagType entity.TagType, tagCode string, parentTagType entity.TagType, newParentCode string) error {
	// 获取资源编号对应的资源标签信息
	resourceTags, _ := p.ListByCond(&entity.TagTree{Type: tagType, Code: tagCode})
	if len(resourceTags) == 0 {
		logx.WarnfContext(ctx, "ChangeParentTag - [%d-%s] tag not found", tagType, tagCode)
		return nil
	}

	if p.CountByCond(&entity.TagTree{Type: parentTagType, Code: newParentCode}) == 0 {
		return errorx.NewBiz("parent tag not found")
	}

	// 获取该资源编号对应的所有子资源标签信息
	var resourceChildrenTags []*entity.TagTree
	p.ListByQuery(&entity.TagTreeQuery{CodePathLikes: collx.ArrayMap(resourceTags, func(tag *entity.TagTree) string {
		return tag.CodePath
	})}, &resourceChildrenTags)

	// 更新父标签的codepath
	for _, tag := range resourceChildrenTags {
		pathSections := entity.CodePath(tag.CodePath).GetPathSections()
		for i, ps := range pathSections {
			if ps.Type == tagType && ps.Code == tagCode {
				// 将父标签编号修改为对应的新编号与类型
				pathSections[i-1].Code = newParentCode
				pathSections[i-1].Type = parentTagType
			}
		}

		tag.CodePath = pathSections.ToCodePath()
		if err := p.UpdateById(ctx, tag); err != nil {
			return err
		}
	}

	return nil
}

func (p *tagTreeAppImpl) MovingTag(ctx context.Context, fromTagPath string, toTagPath string) error {
	fromTag := &entity.TagTree{CodePath: fromTagPath}
	if err := p.GetByCond(fromTag); err != nil {
		return errorx.NewBiz("move tag not found")
	}

	toTag := &entity.TagTree{CodePath: toTagPath}
	if err := p.GetByCond(toTag); err != nil {
		return errorx.NewBiz("target tag not found")
	}

	// 获取要移动标签的所有子标签
	var childrenTags []*entity.TagTree
	p.ListByQuery(&entity.TagTreeQuery{CodePathLikes: []string{fromTagPath}}, &childrenTags)

	// 获取父路径, 若fromTagPath=tag1/tag2/1|xxx则返回 tag1/tag2/
	fromParentPath := string(entity.CodePath(fromTagPath).GetParent(0))
	for _, childTag := range childrenTags {
		// 替换path，若childPath = tag1/tag2/1|xxx/11|yyy, toTagPath=tag3/tag4则替换为tag3/tag4/1|xxx/11|yyy/
		childTag.CodePath = strings.Replace(childTag.CodePath, fromParentPath, toTagPath, 1)
		if err := p.UpdateById(ctx, childTag); err != nil {
			return err
		}
	}

	return nil
}

func (p *tagTreeAppImpl) DeleteTagByParam(ctx context.Context, param *dto.DelResourceTag) error {
	// 获取资源编号对应的资源标签信息
	cond := &entity.TagTree{Type: param.ResourceType, Code: param.ResourceCode}
	cond.Id = param.Id
	resourceTags, _ := p.ListByCond(cond)

	if len(resourceTags) == 0 {
		logx.DebugfContext(ctx, "TagTreeApp.DeleteTagByParam[%d-%s] - There are no deletable tags", param.ResourceType, param.ResourceCode)
		return nil
	}

	delTagType := param.ChildType
	var childrenTagIds []uint64
	for _, resourceTag := range resourceTags {
		// 获取所有关联的子标签
		childrenTag, _ := p.ListByCond(model.NewCond().RLike("code_path", resourceTag.CodePath).Eq("type", delTagType))
		if len(childrenTag) == 0 {
			continue
		}

		childrenTagIds = append(childrenTagIds, collx.ArrayMap(childrenTag, func(item *entity.TagTree) uint64 {
			return item.Id
		})...)
	}

	if len(childrenTagIds) == 0 {
		return nil
	}
	// 删除code_path下的所有子标签
	return p.deleteByIds(ctx, collx.ArrayDeduplicate(childrenTagIds))
}

func (p *tagTreeAppImpl) ListByQuery(condition *entity.TagTreeQuery, toEntity any) {
	p.GetRepo().SelectByCondition(condition, toEntity)
}

func (p *tagTreeAppImpl) GetAccountTags(accountId uint64, query *entity.TagTreeQuery) dto.SimpleTagTrees {
	types := query.Types
	tagResourceQuery := &entity.TagTreeQuery{
		Types: types,
	}

	var tagResources []*dto.SimpleTagTree
	var accountTagPaths []string

	if accountId == consts.AdminId {
		// admin账号，获取所有root tag进行查找过滤
		tagTypeTags, _ := p.ListByCond(&entity.TagTree{Type: entity.TagTypeTag}, "code_path")
		accountTagPaths = collx.ArrayFilter(collx.ArrayMap(tagTypeTags, func(item *entity.TagTree) string {
			return item.CodePath
		}), func(path string) bool {
			return len(entity.CodePath(path).GetPathSections()) == 1
		})
	} else {
		// 获取账号有权限操作的标签路径列表
		accountTagPaths = p.ListTagByAccountId(accountId)
	}

	if len(accountTagPaths) == 0 {
		return tagResources
	}

	// 去除空字符串标签
	tagPaths := collx.ArrayRemoveBlank(query.CodePathLikes)
	// 如果需要查询指定标签下的资源标签，则需要与用户拥有的权限进行过滤，避免越权
	if len(tagPaths) > 0 {
		accountTagPaths = filterCodePaths(accountTagPaths, tagPaths)
	}

	codePathLikes := accountTagPaths
	needFilterAccountTagPaths := make(map[string][]string, 0)
	typePaths := query.TypePaths
	if len(typePaths) > 0 {
		codePathLikes = []string{}

		for _, typePath := range typePaths {
			childOrderTypes := typePath.ToTagTypes()
			// 如果不是获取所有子节点，则需要追加Type进行过滤
			if !query.GetAllChildren {
				tagResourceQuery.Types = append(tagResourceQuery.Types, childOrderTypes[len(childOrderTypes)-1])
			}

			// 资源类型模糊匹配，若childTypes = [machineType, authcertType]  => machineType|%/authcertType|%/
			// 标签加上路径即可过滤出需要的标签，-> tag1/tag2/machineType|%/authcertType|%/
			childOrderTypesMatch := strings.Join(collx.ArrayMap(childOrderTypes, func(tt entity.TagType) string {
				return cast.ToString(int8(tt)) + entity.CodePathResourceSeparator + "%"
			}), entity.CodePathSeparator) + entity.CodePathSeparator

			// 根据用户拥有的标签路径，赋值要过滤匹配的标签类型路径
			for _, accountTag := range accountTagPaths {
				accountTagCodePath := entity.CodePath(accountTag)
				// 标签路径，不包含资源段，如tag1/tag2/1|xxx => tag1/tag2/
				tagPath := accountTagCodePath.GetTag()
				// 纯纯的标签类型(不包含资源段)，则直接在该标签路径上补上对应的子资源类型匹配表达式
				if tagPath == accountTagCodePath {
					// 查询标签类型为标签时，特殊处理
					if len(childOrderTypes) == 1 && childOrderTypes[0] == entity.TagTypeTag {
						codePathLikes = append(codePathLikes, accountTag)
						continue
					}

					// 纯标签可能还有其他子标签的纯标签，故需要多加一个匹配，如tagPath = tag1/，而系统还有 tag1/tag2/  tag1/tag2/tag3等，故需要多一个tag模糊匹配，即tag1/%/xxx
					codePathLikes = append(codePathLikes, accountTag+childOrderTypesMatch, accountTag+"%"+entity.CodePathSeparator+childOrderTypesMatch)
					continue
				}

				// 将用户有权限操作的标签如  tag1/tag2/type|code 替换为tag1/tag2/type|%，并与需要查询的资源类型进行匹配
				accountTagCodePathSections := accountTagCodePath.GetPathSections()
				for _, section := range accountTagCodePathSections {
					if section.Type == entity.TagTypeTag {
						continue
					}
					section.Code = "%"
				}

				// tag1/tag2/type1|%/type2|%
				codePathLike := string(tagPath) + childOrderTypesMatch
				accountMatchPath := accountTagCodePathSections.ToCodePath()
				// 用户有权限操作该标签则直接添加即可
				if entity.CodePath(accountMatchPath).CanAccess(codePathLike) {
					codePathLikes = append(codePathLikes, accountTag)
					continue
				}

				// 如用户分配了: "default/type1|code1/type2|code2/type3|code3/",即accountMathPath=default/type1|%/type2|%/type3/%/, 需要查询的codePathLike为: default/type1|%/type2|%/，即用户分配的标签路径是查询的子节点。
				// 则codePathLike 使用default/type1|code1/type2|code2/去查
				if strings.HasPrefix(accountMatchPath, codePathLike) {
					actualMatchCodePath := accountTagCodePathSections[len(entity.CodePath(codePathLike).GetPathSections())-1].Path
					needFilterAccountTagPaths[actualMatchCodePath] = append(needFilterAccountTagPaths[actualMatchCodePath], accountTag)
					codePathLikes = append(codePathLikes, actualMatchCodePath)
				}
			}
		}

		// 去重处理
		codePathLikes = collx.ArrayDeduplicate(codePathLikes)
	}

	// 账号权限经过处理为空，则说明没有用户可以操作的标签，直接返回即可
	if len(codePathLikes) == 0 {
		return tagResources
	}

	tagResourceQuery.Codes = query.Codes
	tagResourceQuery.CodePathLikes = codePathLikes
	p.ListByQuery(tagResourceQuery, &tagResources)

	// 获取所有子节点，并且存在需要过滤的路径，则进行过滤处理
	if query.GetAllChildren && len(needFilterAccountTagPaths) > 0 {
		tagResources = collx.ArrayFilter(tagResources, func(tr *dto.SimpleTagTree) bool {
			for codePathLike, accountTags := range needFilterAccountTagPaths {
				if strings.HasPrefix(tr.CodePath, codePathLike) {
					return slices.ContainsFunc(accountTags, func(accountTag string) bool {
						return entity.CodePath(accountTag).CanAccess(tr.CodePath)
					})
				}
			}
			return true
		})
	}

	return tagResources
}

func (p *tagTreeAppImpl) ListTagPathByTypeAndCode(resourceType int8, resourceCode string) []string {
	trs, _ := p.ListByCond(&entity.TagTree{Type: entity.TagType(resourceType), Code: resourceCode})
	return collx.ArrayMap(trs, func(tr *entity.TagTree) string {
		return tr.CodePath
	})
}

func (p *tagTreeAppImpl) ListTagByAccountId(accountId uint64) []string {
	tagPaths, err := cache.GetAccountTagPaths(accountId)
	if err != nil {
		tagPaths = p.tagTreeRelateApp.GetTagPathsByAccountId(accountId)
		cache.SaveAccountTagPaths(accountId, tagPaths)
	}
	return tagPaths
}

func (p *tagTreeAppImpl) CanAccess(accountId uint64, tagPath ...string) error {
	if accountId == consts.AdminId {
		return nil
	}
	tagPaths := p.ListTagByAccountId(accountId)
	// 判断该资源标签是否为该账号拥有的标签或其子标签
	for _, v := range tagPaths {
		accountTagCodePath := entity.CodePath(v)
		for _, tp := range tagPath {
			if accountTagCodePath.CanAccess(tp) {
				return nil
			}
		}
	}

	return errorx.NewBizI(context.Background(), imsg.ErrNoPermissionOpResource)
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
	p.ListByQuery(&entity.TagTreeQuery{Codes: collx.MapKeys(resourceCode2Resouce), Types: []entity.TagType{resourceTagType}}, &tagResources)

	for _, tr := range tagResources {
		// 赋值标签信息
		resource := resourceCode2Resouce[tr.Code]
		if resource != nil {
			resource.SetTagInfo(entity.ResourceTag{TagId: tr.Id, CodePath: string(entity.CodePath(tr.CodePath).GetTag())})
		}
	}
}

func (p *tagTreeAppImpl) Delete(ctx context.Context, id uint64) error {
	accountId := contextx.GetLoginAccount(ctx).Id
	tag, err := p.GetById(id)
	if err != nil {
		return errorx.NewBiz("tag not found")
	}
	if err := p.CanAccess(accountId, tag.CodePath); err != nil {
		return errorx.NewBizI(ctx, imsg.ErrNoPermissionDeleteTag)
	}

	return p.DeleteTagByParam(ctx, &dto.DelResourceTag{
		Id: id,
	})
}

func (p *tagTreeAppImpl) toTags(parentTags []*entity.TagTree, param *dto.ResourceTag) []*entity.TagTree {
	tags := make([]*entity.TagTree, 0)

	// 递归函数，将标签及其子标签展开为一个扁平数组
	var flattenTags func(parentTag *entity.TagTree, tag *dto.ResourceTag)
	flattenTags = func(parentTag *entity.TagTree, resourceTagParam *dto.ResourceTag) {
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
	return p.Tx(ctx, func(ctx context.Context) error {
		return p.DeleteById(ctx, tagIds...)
	}, func(ctx context.Context) error {
		// 删除与标签有关联信息的记录(如团队关联的标签等)
		return p.tagTreeRelateApp.DeleteByCond(ctx, model.NewCond().In("tag_id", tagIds))
	})
}

// filterCodePaths 根据账号拥有的标签路径以及指定的标签路径，过滤出符合查询条件的标签路径
func filterCodePaths(accountTagPaths []string, tagPaths []string) []string {
	var res []string
	queryPaths := collx.ArrayFilter[string](tagPaths, func(tagPath string) bool {
		for _, acPath := range accountTagPaths {
			// 查询条件： a/b/  有权的：a/  查询结果应该是: a/b/
			if strings.HasPrefix(tagPath, acPath) {
				return true
			}
		}
		return false
	})

	acPaths := collx.ArrayFilter[string](accountTagPaths, func(acPath string) bool {
		for _, tagPath := range tagPaths {
			// 查询条件： a/  有权的：a/b/  查询结果应该是: a/b/，如果以a/去查可能会查出无权的 a/c/相关联的数据
			if strings.HasPrefix(acPath, tagPath) {
				return true
			}
		}
		return false
	})

	res = append(queryPaths, acPaths...)
	return collx.ArrayDeduplicate(res)
}

// hasConflictPath 判断标签路径中是否存在冲突路径，如不能同时存在tag1/tag2/tag3  tag1/  tag1/tag2等，因为拥有父级标签则拥有所有子标签资源等信息
func hasConflictPath(codePaths []string) bool {
	if len(codePaths) == 0 {
		return false
	}
	seen := make(map[string]bool)
	for _, str := range codePaths {
		parts := strings.Split(str, entity.CodePathSeparator)
		var prefix string
		for _, part := range parts {
			prefix += part + entity.CodePathSeparator
			if seen[prefix] {
				return true
			}
		}
		seen[str] = true
	}
	return false
}
