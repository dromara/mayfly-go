package api

import (
	"fmt"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/tag/api/vo"
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"sort"
	"strings"
)

type TagTree struct {
	TagTreeApp application.TagTree `inject:""`
}

func (p *TagTree) GetTagTree(rc *req.Ctx) {
	tagType := rc.QueryInt("type")
	// 超管返回所有标签树
	if rc.GetLoginAccount().Id == consts.AdminId {
		var tagTrees vo.TagTreeVOS
		p.TagTreeApp.ListByQuery(&entity.TagTreeQuery{Type: int8(tagType)}, &tagTrees)
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
	var tags []*entity.TagTree
	p.TagTreeApp.ListByQuery(&entity.TagTreeQuery{CodePathLikes: collx.MapKeys(rootTag), Type: int8(tagType)}, &tags)

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
	tagPaths := rc.Query("tagPaths")
	cond.CodePaths = strings.Split(tagPaths, ",")
	var tagTrees vo.TagTreeVOS
	p.TagTreeApp.ListByQuery(cond, &tagTrees)
	rc.ResData = tagTrees
}

func (p *TagTree) SaveTagTree(rc *req.Ctx) {
	tagTree := &entity.TagTree{}
	req.BindJsonAndValid(rc, tagTree)

	rc.ReqParam = fmt.Sprintf("tagTreeId: %d, tagName: %s, code: %s", tagTree.Id, tagTree.Name, tagTree.Code)

	biz.ErrIsNil(p.TagTreeApp.Save(rc.MetaCtx, tagTree))
}

func (p *TagTree) DelTagTree(rc *req.Ctx) {
	biz.ErrIsNil(p.TagTreeApp.Delete(rc.MetaCtx, uint64(rc.PathParamInt("id"))))
}

// 获取用户可操作的资源标签路径
func (p *TagTree) TagResources(rc *req.Ctx) {
	resourceType := int8(rc.PathParamInt("rtype"))
	tagResources := p.TagTreeApp.GetAccountTagResources(rc.GetLoginAccount().Id, resourceType, "")
	tagPath2Resource := collx.ArrayToMap[entity.TagTree, string](tagResources, func(tagResource entity.TagTree) string {
		return tagResource.GetParentPath()
	})

	tagPaths := collx.MapKeys(tagPath2Resource)
	sort.Strings(tagPaths)
	rc.ResData = tagPaths
}

// 统计当前用户指定标签下关联的资源数量
func (p *TagTree) CountTagResource(rc *req.Ctx) {
	tagPath := rc.Query("tagPath")
	accountId := rc.GetLoginAccount().Id
	rc.ResData = collx.M{
		"machine": len(p.TagTreeApp.GetAccountResourceCodes(accountId, consts.TagResourceTypeMachine, tagPath)),
		"db":      len(p.TagTreeApp.GetAccountResourceCodes(accountId, consts.TagResourceTypeDb, tagPath)),
		"redis":   len(p.TagTreeApp.GetAccountResourceCodes(accountId, consts.TagResourceTypeRedis, tagPath)),
		"mongo":   len(p.TagTreeApp.GetAccountResourceCodes(accountId, consts.TagResourceTypeMongo, tagPath)),
	}
}
