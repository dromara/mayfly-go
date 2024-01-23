package api

import (
	"fmt"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/tag/api/vo"
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"sort"
	"strings"
)

type TagTree struct {
	TagTreeApp     application.TagTree     `inject:""`
	TagResourceApp application.TagResource `inject:""`
}

func (p *TagTree) GetTagTree(rc *req.Ctx) {
	// 超管返回所有标签树
	if rc.GetLoginAccount().Id == consts.AdminId {
		var tagTrees vo.TagTreeVOS
		p.TagTreeApp.ListByQuery(new(entity.TagTreeQuery), &tagTrees)
		rc.ResData = tagTrees.ToTrees(0)
		return
	}

	// 获取用户可以操作访问的标签路径
	tagPaths := p.TagTreeApp.ListTagByAccountId(rc.GetLoginAccount().Id)

	rootTag := make(map[string][]string, 0)
	for _, accountTagPath := range tagPaths {
		root := strings.Split(accountTagPath, "/")[0] + entity.CodePathSeparator
		tags := rootTag[root]
		tags = append(tags, accountTagPath)
		rootTag[root] = tags

	}

	// 获取所有以root标签开头的子标签
	tags := p.TagTreeApp.ListTagByPath(collx.MapKeys(rootTag)...)
	tagTrees := make(vo.TagTreeVOS, 0)
	for _, tag := range tags {
		tagPath := tag.CodePath
		root := strings.Split(tagPath, "/")[0] + entity.CodePathSeparator
		// 获取用户可操作的标签路径列表
		accountTagPaths := rootTag[root]
		for _, accountTagPath := range accountTagPaths {
			if strings.HasPrefix(tagPath, accountTagPath) || strings.HasPrefix(accountTagPath, tagPath) {
				tagTrees = append(tagTrees, tag)
				break
			}
		}
	}

	rc.ResData = tagTrees.ToTrees(0)
}

func (p *TagTree) ListByQuery(rc *req.Ctx) {
	cond := new(entity.TagTreeQuery)
	tagPaths := rc.GinCtx.Query("tagPaths")
	cond.CodePaths = strings.Split(tagPaths, ",")
	var tagTrees vo.TagTreeVOS
	p.TagTreeApp.ListByQuery(cond, &tagTrees)
	rc.ResData = tagTrees
}

func (p *TagTree) SaveTagTree(rc *req.Ctx) {
	tagTree := &entity.TagTree{}
	ginx.BindJsonAndValid(rc.GinCtx, tagTree)

	rc.ReqParam = fmt.Sprintf("tagTreeId: %d, tagName: %s, code: %s", tagTree.Id, tagTree.Name, tagTree.Code)

	biz.ErrIsNil(p.TagTreeApp.Save(rc.MetaCtx, tagTree))
}

func (p *TagTree) DelTagTree(rc *req.Ctx) {
	biz.ErrIsNil(p.TagTreeApp.Delete(rc.MetaCtx, uint64(ginx.PathParamInt(rc.GinCtx, "id"))))
}

// 获取用户可操作的资源标签路径
func (p *TagTree) TagResources(rc *req.Ctx) {
	resourceType := int8(ginx.PathParamInt(rc.GinCtx, "rtype"))
	tagResources := p.TagTreeApp.GetAccountTagResources(rc.GetLoginAccount().Id, resourceType, "")
	tagPath2Resource := collx.ArrayToMap[entity.TagResource, string](tagResources, func(tagResource entity.TagResource) string {
		return tagResource.TagPath
	})

	tagPaths := collx.MapKeys(tagPath2Resource)
	sort.Strings(tagPaths)
	rc.ResData = tagPaths
}

// 资源标签关联信息查询
func (p *TagTree) QueryTagResources(rc *req.Ctx) {
	var trs []*entity.TagResource
	p.TagResourceApp.ListByQuery(ginx.BindQuery(rc.GinCtx, new(entity.TagResourceQuery)), &trs)
	rc.ResData = trs
}
