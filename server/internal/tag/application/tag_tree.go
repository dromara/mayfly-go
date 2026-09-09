package application

import (
	"context"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/internal/tag/application/dto"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/internal/tag/imsg"
	"mayfly-go/internal/tag/infra/cache"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"slices"
	"strings"

	"github.com/spf13/cast"
)

type TagTree interface {
	base.App[*entity.TagTree]

	TagTreeReader
	TagTreeChecker
	TagTreeManager

	// ListTagByAccountId 根据账号id获取其可访问标签信息（tag 域内部使用）
	ListTagByAccountId(accountId uint64) []string
}

// TagTreeReader 标签树读侧接口：查询账号可访问标签、资源标签路径等
// 外部模块仅按需注入此接口，避免依赖整个聚合接口（ISP）
type TagTreeReader interface {
	// GetAccountTags 获取指定账号有权限操作的标签列表
	//  -  accountId 账号id
	//  -  query 查询条件
	GetAccountTags(accountId uint64, query *entity.TagTreeQuery) dto.SimpleTagTrees

	// ListByQuery 根据条件查询标签
	ListByQuery(condition *entity.TagTreeQuery, toEntity any) error

	// 根据标签类型和标签code获取对应的标签路径列表
	ListTagPathByTypeAndCode(resourceType int8, resourceCode string) []string

	// ListResourceTagByCode 根据资源code获取资源关联的标签
	ListResourceTagByCode(resourceCode string) []*entity.ResourceTag

	// GetAccountResourceCodes 获取账号可操作的指定类型资源code集合
	//   - tagPath 标签路径过滤条件，可为空
	//   - types 资源类型路径，如 [TagTypeMachine, TagTypeAuthCert]，返回首个类型的资源code
	GetAccountResourceCodes(accountId uint64, tagPath string, types ...entity.TagType) []string
}

// TagTreeChecker 标签权限校验接口：资源模块鉴权的最小依赖面
type TagTreeChecker interface {
	// CanAccess 账号是否有权限访问该标签关联的资源信息
	CanAccess(accountId uint64, tagPath ...string) error

	// CanAccessByCode 根据资源类型与资源编号校验当前登录账号是否有权操作该资源
	// 未获取到登录账号(系统内部任务调用)或管理员账号时不校验
	CanAccessByCode(ctx context.Context, resourceType int8, resourceCode string) error
}

// TagTreeManager 标签管理接口：标签及资源标签的保存/更新/删除/关联
type TagTreeManager interface {
	SaveTag(ctx context.Context, pid uint64, tt *entity.TagTree) error

	// SaveResourceTag 保存资源类型标签
	SaveResourceTag(ctx context.Context, param *dto.SaveResourceTag) error

	// RelateTagsByCodeAndType 将指定标签数组关联至满足指定标签类型和标签code的标签下
	RelateTagsByCodeAndType(ctx context.Context, param *dto.RelateTagsByCodeAndType) error

	// UpdateTagName 根据标签类型与code更新对应标签名
	UpdateTagName(ctx context.Context, tagType entity.TagType, tagCode string, tagName string) error

	// ChangeParentTag 变更指定类型标签的父标签
	ChangeParentTag(ctx context.Context, tagType entity.TagType, tagCode string, parentTagType entity.TagType, newParentCode string) error

	// DeleteTagByParam 删除标签，会删除该标签下所有子标签信息以及团队关联的标签信息
	DeleteTagByParam(ctx context.Context, param *dto.DelResourceTag) error

	Delete(ctx context.Context, id uint64) error
}

// TagTreeService 资源应用层常用组合接口：读 + 权限校验 + 资源标签管理，
// 仍窄于聚合接口（不含 base.App 的通用 CRUD 能力）
type TagTreeService interface {
	TagTreeReader
	TagTreeChecker
	TagTreeManager
}

type tagTreeAppImpl struct {
	base.AppImpl[*entity.TagTree, repository.TagTree]

	tagTreeRelateApp TagTreeRelate `inject:"T"`
}

var _ TagTree = (*tagTreeAppImpl)(nil)

func (p *tagTreeAppImpl) SaveTag(ctx context.Context, pid uint64, tag *entity.TagTree) error {
	// 新建资源树节点信息
	if tag.Id == 0 {
		tag.Code = stringx.Rand(12)
		if pid != 0 {
			parentTag, err := p.GetById(pid)
			if err != nil {
				return errorx.NewBiz("parent tag not found")
			}

			tag.CodePath = parentTag.CodePath + tag.Code + entity.CodePathSeparator

			account := contextx.GetLoginAccount(ctx)
			if account == nil {
				return errorx.NewBiz("login account not found")
			}
			if p.CanAccess(account.Id, tag.CodePath) != nil {
				return errorx.NewBizI(ctx, imsg.ErrNoPermissionCreateTag)
			}
		} else {
			tag.CodePath = tag.Code + entity.CodePathSeparator
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

	// 校验当前操作者是否拥有待关联的标签路径，防止越权将资源挂载至他人标签路径下
	if err := p.checkTagPathOwner(ctx, parentTagCodePaths...); err != nil {
		return err
	}

	// 获取所有关联的父标签
	parentTags, err := p.ListByCond(model.NewCond().In("code_path", parentTagCodePaths))
	if err != nil {
		return err
	}
	if len(parentTags) == 0 || len(parentTags) != len(parentTagCodePaths) {
		// 存在错误的关联标签
		return errorx.NewBiz("save resource tag failed: There is an incorrect relate tag")
	}

	newTags := p.toTags(parentTags, param.ResourceTag)

	oldParentTagTree, err := p.ListByCond(&entity.TagTree{Type: tagType, Code: code})
	if err != nil {
		return err
	}

	// 该资源对应的旧资源标签信息
	var oldChildrenTags []*entity.TagTree
	if len(oldParentTagTree) > 0 {
		// 获取所有旧的子标签
		if err := p.ListByQuery(&entity.TagTreeQuery{
			CodePathLikes: collx.ArrayMap(oldParentTagTree, func(val *entity.TagTree) string {
				return val.CodePath
			}),
		}, &oldChildrenTags); err != nil {
			return err
		}
	}
	// 旧的codePath -> tag
	oldCodePath2Tag := collx.ArrayToMap(oldChildrenTags, func(val *entity.TagTree) string { return val.CodePath })
	// 新的codePath -> tag
	newCodePath2Tag := collx.ArrayToMap(newTags, func(val *entity.TagTree) string { return val.CodePath })

	var addCodePaths, delCodePaths []string
	addCodePaths, delCodePaths, _ = collx.ArrayCompare(collx.MapKeys(newCodePath2Tag), collx.MapKeys(oldCodePath2Tag))

	if len(addCodePaths) == 0 && len(delCodePaths) == 0 {
		return nil
	}

	// 增删在同一事务中执行，避免部分成功导致资源标签重复挂载等脏数据
	return p.Tx(ctx, func(ctx context.Context) error {
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
			if len(delTagIds) > 0 {
				if err := p.DeleteById(ctx, delTagIds...); err != nil {
					return err
				}
				// 删除与标签有关联信息的记录(如团队关联的标签等)
				if err := p.tagTreeRelateApp.DeleteByCond(ctx, model.NewCond().In("tag_id", delTagIds)); err != nil {
					return err
				}
			}
		}

		// 资源标签已变更，失效涉及资源的标签路径短缓存
		for _, addCodePath := range addCodePaths {
			if t := newCodePath2Tag[addCodePath]; t != nil {
				cache.DelResourceTagPaths(int8(t.Type), t.Code)
			}
		}
		for _, delCodePath := range delCodePaths {
			if t := oldCodePath2Tag[delCodePath]; t != nil {
				cache.DelResourceTagPaths(int8(t.Type), t.Code)
			}
		}
		return nil
	})
}

func (p *tagTreeAppImpl) RelateTagsByCodeAndType(ctx context.Context, param *dto.RelateTagsByCodeAndType) error {
	parentTagCode := param.ParentTagCode
	parentTagType := param.ParentTagType

	// 获取满足指定编号与类型的所有标签信息
	parentTags, err := p.ListByCond(&entity.TagTree{Type: parentTagType, Code: parentTagCode})
	if err != nil {
		return err
	}
	// 标签codePaths（相当于需要关联的标签数组的父tag）
	parentTagCodePaths := collx.ArrayMap(parentTags, func(tag *entity.TagTree) string {
		return tag.CodePath
	})

	if len(parentTagCodePaths) == 0 {
		// 不满足满足条件的标签
		return errorx.NewBizf("There is no tag that satisfies [type=%d, code=%s]", parentTagType, parentTagCode)
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
	resourceTags, err := p.ListByCond(&entity.TagTree{Type: tagType, Code: tagCode})
	if err != nil {
		return err
	}
	if len(resourceTags) == 0 {
		logx.WarnfContext(ctx, "ChangeParentTag - [%d-%s] tag not found", tagType, tagCode)
		return nil
	}

	if p.CountByCond(&entity.TagTree{Type: parentTagType, Code: newParentCode}) == 0 {
		return errorx.NewBiz("parent tag not found")
	}

	// 获取该资源编号对应的所有子资源标签信息
	var resourceChildrenTags []*entity.TagTree
	if err := p.ListByQuery(&entity.TagTreeQuery{CodePathLikes: collx.ArrayMap(resourceTags, func(tag *entity.TagTree) string {
		return tag.CodePath
	})}, &resourceChildrenTags); err != nil {
		return err
	}

	// 统一在事务中变更路径，避免部分更新成功导致路径不一致
	return p.Tx(ctx, func(ctx context.Context) error {
		for _, tag := range resourceChildrenTags {
			pathSections := entity.CodePath(tag.CodePath).GetPathSections()
			for i, ps := range pathSections {
				if ps.Type == tagType && ps.Code == tagCode {
					// 防御：资源段前必须存在父标签段，防止越界 panic
					if i == 0 {
						return errorx.NewBizf("tag[%s] has no parent tag to change", tag.CodePath)
					}
					// 将父标签编号修改为对应的新编号与类型
					pathSections[i-1].Code = newParentCode
					pathSections[i-1].Type = parentTagType
				}
			}

			tag.CodePath = pathSections.ToCodePath()
			if err := p.UpdateById(ctx, tag); err != nil {
				return err
			}
			// 路径已变更，失效该资源的标签路径短缓存
			cache.DelResourceTagPaths(int8(tag.Type), tag.Code)
		}
		return nil
	})
}

func (p *tagTreeAppImpl) DeleteTagByParam(ctx context.Context, param *dto.DelResourceTag) error {
	// 资源标签删除时校验操作者是否有权操作该资源，防止越权删除他人资源的标签关联信息
	if param.ResourceType != 0 && param.ResourceCode != "" {
		if err := p.CanAccessByCode(ctx, int8(param.ResourceType), param.ResourceCode); err != nil {
			return err
		}
	}

	// 获取资源编号对应的资源标签信息
	cond := &entity.TagTree{Type: param.ResourceType, Code: param.ResourceCode}
	cond.Id = param.Id
	resourceTags, err := p.ListByCond(cond)
	if err != nil {
		return err
	}

	if len(resourceTags) == 0 {
		logx.DebugfContext(ctx, "TagTreeApp.DeleteTagByParam[%d-%s] - There are no deletable tags", param.ResourceType, param.ResourceCode)
		return nil
	}

	delTagType := param.ChildType
	var childrenTagIds []uint64
	// codePath -> 子标签列表（用于失效路径缓存）
	childrenTagsCache := make(map[string][]*entity.TagTree, 0)
	for _, resourceTag := range resourceTags {
		// 获取所有关联的子标签
		childrenTag, err := p.ListByCond(model.NewCond().RLike("code_path", resourceTag.CodePath).Eq("type", delTagType))
		if err != nil {
			return err
		}
		if len(childrenTag) == 0 {
			continue
		}
		childrenTagsCache[resourceTag.CodePath] = childrenTag

		childrenTagIds = append(childrenTagIds, collx.ArrayMap(childrenTag, func(item *entity.TagTree) uint64 {
			return item.Id
		})...)
	}

	if len(childrenTagIds) == 0 {
		return nil
	}
	// 删除code_path下的所有子标签
	if err := p.deleteByIds(ctx, collx.ArrayDeduplicate(childrenTagIds)); err != nil {
		return err
	}

	// 失效被删除资源标签的路径短缓存（含子标签自身的类型与编码）
	for _, resourceTag := range resourceTags {
		cache.DelResourceTagPaths(int8(resourceTag.Type), resourceTag.Code)
	}
	for _, resourceTag := range resourceTags {
		for _, child := range childrenTagsCache[resourceTag.CodePath] {
			cache.DelResourceTagPaths(int8(child.Type), child.Code)
		}
	}
	return nil
}

func (p *tagTreeAppImpl) ListByQuery(condition *entity.TagTreeQuery, toEntity any) error {
	return p.GetRepo().SelectByCondition(condition, toEntity)
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
		tagTypeTags, err := p.ListByCond(&entity.TagTree{Type: entity.TagTypeTag}, "code_path")
		if err != nil {
			// 查询失败时按无可操作数据返回(fail-closed)，避免越权数据被展示
			logx.Errorf("GetAccountTags - query admin root tags error: %v", err)
			return tagResources
		}
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
		var appendTypes []entity.TagType
		codePathLikes, needFilterAccountTagPaths, appendTypes = buildTypePathFilter(accountTagPaths, typePaths, query.GetAllChildren)
		if !query.GetAllChildren {
			tagResourceQuery.Types = append(tagResourceQuery.Types, appendTypes...)
		}
	}

	// 账号权限经过处理为空，则说明没有用户可以操作的标签，直接返回即可
	if len(codePathLikes) == 0 {
		return tagResources
	}

	tagResourceQuery.Codes = query.Codes
	tagResourceQuery.CodePathLikes = codePathLikes
	if err := p.ListByQuery(tagResourceQuery, &tagResources); err != nil {
		// 查询失败时按无可操作数据返回(fail-closed)，避免越权数据被展示
		logx.Errorf("GetAccountTags - query resource tags error: %v", err)
		return nil
	}

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
	// 该方法在每次资源操作鉴权（CanAccessByCode）时都会调用，加短 TTL 缓存减少 DB 往返，
	// 资源标签变更时已主动失效
	if paths, err := cache.GetResourceTagPaths(resourceType, resourceCode); err == nil {
		return paths
	}
	trs, err := p.ListByCond(&entity.TagTree{Type: entity.TagType(resourceType), Code: resourceCode})
	if err != nil {
		// 查询失败时返回空(fail-closed)，避免将无标签路径的资源判定为可访问
		logx.Errorf("ListTagPathByTypeAndCode[%d-%s] error: %v", resourceType, resourceCode, err)
		return nil
	}
	paths := collx.ArrayMap(trs, func(tr *entity.TagTree) string {
		return tr.CodePath
	})
	if err := cache.SaveResourceTagPaths(resourceType, resourceCode, paths); err != nil {
		logx.Errorf("ListTagPathByTypeAndCode[%d-%s] save cache error: %v", resourceType, resourceCode, err)
	}
	return paths
}

func (p *tagTreeAppImpl) ListResourceTagByCode(resourceCode string) []*entity.ResourceTag {
	// 获取资源code关联的标签列表信息
	var tagResources []*entity.TagTree
	if err := p.ListByQuery(&entity.TagTreeQuery{Codes: collx.AsArray(resourceCode)}, &tagResources); err != nil {
		logx.Errorf("ListResourceTagByCode[%s] error: %v", resourceCode, err)
		return nil
	}
	return collx.ArrayMap(tagResources, func(tt *entity.TagTree) *entity.ResourceTag {
		return &entity.ResourceTag{TagId: tt.Id, CodePath: string(entity.CodePath(tt.CodePath).GetTag())}
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

func (p *tagTreeAppImpl) Delete(ctx context.Context, id uint64) error {
	la := contextx.GetLoginAccount(ctx)
	if la == nil {
		return errorx.NewBiz("login account not found")
	}
	tag, err := p.GetById(id)
	if err != nil {
		return errorx.NewBiz("tag not found")
	}
	if err := p.CanAccess(la.Id, tag.CodePath); err != nil {
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
			Code: tagCode,
			// 统一走领域方法构建资源段，避免协议格式散落
			CodePath: string(entity.CodePath(parentTag.CodePath).AppendResource(tagType, tagCode)),
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
	queryPaths := collx.ArrayFilter(tagPaths, func(tagPath string) bool {
		for _, acPath := range accountTagPaths {
			// 查询条件： a/b/  有权的：a/  查询结果应该是: a/b/
			if strings.HasPrefix(tagPath, acPath) {
				return true
			}
		}
		return false
	})

	acPaths := collx.ArrayFilter(accountTagPaths, func(acPath string) bool {
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
// 与路径顺序无关，任意两条路径存在祖先/后代(或重复)关系均判定为冲突
func hasConflictPath(codePaths []string) bool {
	if len(codePaths) == 0 {
		return false
	}
	for i, str := range codePaths {
		for j, other := range codePaths {
			if i == j {
				continue
			}
			if strings.HasPrefix(other, str) {
				return true
			}
		}
	}
	return false
}

// buildTypePathFilter 根据账号拥有的标签路径与待查询的资源类型路径，构建资源标签的codePath模糊查询条件
// 返回值：
//   - codePathLikes: 资源标签codePath模糊匹配条件
//   - needFilter:    GetAllChildren模式下需对查询结果进行二次过滤的映射(实际匹配codePath -> 账号拥有的标签路径)
//   - appendTypes:   需要追加至查询条件中的资源类型(即各类型路径的最后一段类型)，GetAllChildren时为空
func buildTypePathFilter(accountTagPaths []string, typePaths []entity.TypePath, getAllChildren bool) ([]string, map[string][]string, []entity.TagType) {
	codePathLikes := make([]string, 0)
	needFilter := make(map[string][]string, 0)
	var appendTypes []entity.TagType

	for _, typePath := range typePaths {
		childOrderTypes := typePath.ToTagTypes()
		// 如果不是获取所有子节点，则需要追加Type进行过滤
		if !getAllChildren {
			appendTypes = append(appendTypes, childOrderTypes[len(childOrderTypes)-1])
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

			// tag1/tag2/type1|%/type2|%/
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
				needFilter[actualMatchCodePath] = append(needFilter[actualMatchCodePath], accountTag)
				codePathLikes = append(codePathLikes, actualMatchCodePath)
			}
		}
	}

	// 去重处理
	return collx.ArrayDeduplicate(codePathLikes), needFilter, appendTypes
}

// CanAccessByCode 根据资源类型与资源编号校验当前登录账号是否有权操作该资源
// 未获取到登录账号(系统内部任务调用)或管理员账号时不校验
func (p *tagTreeAppImpl) CanAccessByCode(ctx context.Context, resourceType int8, resourceCode string) error {
	la := contextx.GetLoginAccount(ctx)
	if la == nil || la.Id == consts.AdminId {
		return nil
	}
	return p.CanAccess(la.Id, p.ListTagPathByTypeAndCode(resourceType, resourceCode)...)
}

// checkTagPathOwner 校验当前登录账号是否拥有指定标签路径的操作权限
// 未获取到登录账号(系统内部任务调用)或管理员账号时不校验
func (p *tagTreeAppImpl) checkTagPathOwner(ctx context.Context, tagCodePaths ...string) error {
	la := contextx.GetLoginAccount(ctx)
	if la == nil || la.Id == consts.AdminId {
		return nil
	}
	return p.CanAccess(la.Id, tagCodePaths...)
}

// GetAccountResourceCodes 获取账号可操作的指定类型资源code集合
//   - tagPath 标签路径过滤条件，可为空
//   - types 资源类型路径，如 [TagTypeMachine, TagTypeAuthCert]，返回首个类型的资源code
func (p *tagTreeAppImpl) GetAccountResourceCodes(accountId uint64, tagPath string, types ...entity.TagType) []string {
	if len(types) == 0 {
		return nil
	}
	tags := p.GetAccountTags(accountId, &entity.TagTreeQuery{
		TypePaths:     collx.AsArray(entity.NewTypePaths(types...)),
		CodePathLikes: collx.AsArray(tagPath),
	})
	if len(tags) == 0 {
		return nil
	}
	return entity.GetCodesByCodePaths(types[0], tags.GetCodePaths()...)
}
