package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/tag/application/dto"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/internal/tag/infrastructure/cache"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"strings"
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
	GetAccountTags(accountId uint64, query *entity.TagTreeQuery) []*dto.SimpleTagTree

	// GetAccountTagCodes 获取指定账号有权限操作的标签codes
	GetAccountTagCodes(accountId uint64, resourceType int8, tagPath string) []string

	// GetAccountTagCodePaths 获取指定账号有权限操作的codePaths
	GetAccountTagCodePaths(accountId uint64, tagType entity.TagType, tagPath string) []string

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

	tagTreeRelateApp TagTreeRelate `inject:"TagTreeRelateApp"`
}

var _ (TagTree) = (*tagTreeAppImpl)(nil)

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
			parentTag, err := p.GetById(pid)
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

func (p *tagTreeAppImpl) SaveResourceTag(ctx context.Context, param *dto.SaveResourceTag) error {
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
		return p.DeleteTagByParam(ctx, &dto.DelResourceTag{
			ResourceType: tagType,
			ResourceCode: code,
		})
	}

	// 获取所有关联的父标签
	parentTags, _ := p.ListByCond(model.NewCond().In("code_path", parentTagCodePaths))
	if len(parentTags) == 0 || len(parentTags) != len(parentTagCodePaths) {
		return errorx.NewBiz("保存资源标签失败: 存在错误的关联标签")
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
		return errorx.NewBiz("不存在满足[type=%d, code=%s]的标签", parentTagType, parentTagCode)
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
	if err := p.GetByCond(fromTag); err != nil {
		return errorx.NewBiz("移动标签不存在")
	}

	toTag := &entity.TagTree{CodePath: toTagPath}
	if err := p.GetByCond(toTag); err != nil {
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

func (p *tagTreeAppImpl) DeleteTagByParam(ctx context.Context, param *dto.DelResourceTag) error {
	// 获取资源编号对应的资源标签信息
	cond := &entity.TagTree{Type: param.ResourceType, Code: param.ResourceCode}
	cond.Id = param.Id
	resourceTags, _ := p.ListByCond(cond)

	if len(resourceTags) == 0 {
		logx.DebugfContext(ctx, "TagTreeApp.DeleteTagByParam[%d-%s]不存在可删除的标签", param.ResourceType, param.ResourceCode)
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

func (p *tagTreeAppImpl) GetAccountTags(accountId uint64, query *entity.TagTreeQuery) []*dto.SimpleTagTree {
	tagResourceQuery := &entity.TagTreeQuery{
		Type:  query.Type,
		Types: query.Types,
	}

	var tagResources []*dto.SimpleTagTree
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
			accountTagPaths = filterCodePaths(accountTagPaths, tagPaths)
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
	code2Resource := collx.ArrayToMap[*dto.SimpleTagTree, string](tagResources, func(val *dto.SimpleTagTree) string {
		return val.Code
	})

	return collx.MapKeys(code2Resource)
}

func (p *tagTreeAppImpl) GetAccountTagCodePaths(accountId uint64, tagType entity.TagType, tagPath string) []string {
	tagResources := p.GetAccountTags(accountId, &entity.TagTreeQuery{Type: tagType, CodePathLikes: []string{tagPath}})
	// resouce code去重
	code2Resource := collx.ArrayToMap[*dto.SimpleTagTree, string](tagResources, func(val *dto.SimpleTagTree) string {
		return val.CodePath
	})

	return collx.MapKeys(code2Resource)
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
		resource := resourceCode2Resouce[tr.Code]
		if resource != nil {
			resource.SetTagInfo(entity.ResourceTag{TagId: tr.Id, CodePath: tr.GetTagPath()})
		}
	}
}

func (p *tagTreeAppImpl) Delete(ctx context.Context, id uint64) error {
	accountId := contextx.GetLoginAccount(ctx).Id
	tag, err := p.GetById(id)
	if err != nil {
		return errorx.NewBiz("该标签不存在")
	}
	if err := p.CanAccess(accountId, tag.CodePath); err != nil {
		return errorx.NewBiz("您无权删除该标签")
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
	if err := p.DeleteById(ctx, tagIds...); err != nil {
		return err
	}

	// 删除与标签有关联信息的记录(如团队关联的标签等)
	return p.tagTreeRelateApp.DeleteByCond(ctx, model.NewCond().In("tag_id", tagIds))
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
