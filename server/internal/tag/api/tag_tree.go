package api

import (
	"fmt"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/tag/api/form"
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
	tagType := entity.TagType(rc.QueryInt("type"))
	accountTags := p.TagTreeApp.GetAccountTags(rc.GetLoginAccount().Id, &entity.TagTreeQuery{Type: tagType})
	if len(accountTags) == 0 {
		rc.ResData = []any{}
		return
	}

	allTags := p.complteTags(accountTags)
	tagTrees := make(vo.TagTreeVOS, 0)
	for _, tag := range allTags {
		tagTrees = append(tagTrees, tag)
	}
	rc.ResData = tagTrees.ToTrees(0)
}

// complteTags 补全标签信息，使其能构造为树结构
func (p *TagTree) complteTags(resourceTags []*entity.TagTree) []*entity.TagTree {
	codePath2Tag := collx.ArrayToMap(resourceTags, func(tag *entity.TagTree) string {
		return tag.CodePath
	})

	// 如tagPath = tag1/tag2/tag3/ 需要转为该路径所关联的所有标签路径即 tag1/  tag1/tag2/  tag1/tag2/tag3/三个相关联标签，才可以构造成一棵树
	allTagPaths := make([]string, 0)
	for _, tagPath := range collx.MapKeys(codePath2Tag) {
		allTagPaths = append(allTagPaths, entity.GetAllCodePath(tagPath)...)
	}
	allTagPaths = collx.ArrayDeduplicate(allTagPaths)

	notExistCodePaths := make([]string, 0)
	for _, tagPath := range allTagPaths {
		if _, ok := codePath2Tag[tagPath]; !ok {
			notExistCodePaths = append(notExistCodePaths, tagPath)
		}
	}
	// 未存在需要补全的标签信息，则返回
	if len(notExistCodePaths) == 0 {
		return resourceTags
	}

	var tags []*entity.TagTree
	p.TagTreeApp.ListByQuery(&entity.TagTreeQuery{CodePaths: notExistCodePaths}, &tags)
	// 完善需要补充的标签信息
	return append(resourceTags, tags...)
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
	tagForm := &form.TagTree{}
	tagTree := req.BindJsonAndCopyTo(rc, tagForm, new(entity.TagTree))

	rc.ReqParam = fmt.Sprintf("tagTreeId: %d, tagName: %s, code: %s", tagTree.Id, tagTree.Name, tagTree.Code)

	biz.ErrIsNil(p.TagTreeApp.SaveTag(rc.MetaCtx, tagForm.Pid, tagTree))
}

func (p *TagTree) DelTagTree(rc *req.Ctx) {
	biz.ErrIsNil(p.TagTreeApp.Delete(rc.MetaCtx, uint64(rc.PathParamInt("id"))))
}

func (p *TagTree) MovingTag(rc *req.Ctx) {
	movingForm := &form.MovingTag{}
	req.BindJsonAndValid(rc, movingForm)
	rc.ReqParam = movingForm
	biz.ErrIsNil(p.TagTreeApp.MovingTag(rc.MetaCtx, movingForm.FromPath, movingForm.ToPath))
}

// 获取用户可操作的标签路径
func (p *TagTree) TagResources(rc *req.Ctx) {
	resourceType := int8(rc.PathParamInt("rtype"))
	accountId := rc.GetLoginAccount().Id
	tagResources := p.TagTreeApp.GetAccountTags(accountId, &entity.TagTreeQuery{Type: entity.TagType(resourceType)})

	tagPath2Resource := collx.ArrayToMap[*entity.TagTree, string](tagResources, func(tagResource *entity.TagTree) string {
		return tagResource.GetTagPath()
	})

	tagPaths := collx.MapKeys(tagPath2Resource)
	sort.Strings(tagPaths)
	rc.ResData = tagPaths
}

// 统计当前用户指定标签下关联的资源数量
func (p *TagTree) CountTagResource(rc *req.Ctx) {
	tagPath := rc.Query("tagPath")
	accountId := rc.GetLoginAccount().Id

	machineCodes := entity.GetCodeByPath(entity.TagTypeMachine, p.TagTreeApp.GetAccountTagCodePaths(accountId, entity.TagTypeMachineAuthCert, tagPath)...)
	dbCodes := entity.GetCodeByPath(entity.TagTypeDb, p.TagTreeApp.GetAccountTagCodePaths(accountId, entity.TagTypeDbName, tagPath)...)

	rc.ResData = collx.M{
		"machine": len(machineCodes),
		"db":      len(dbCodes),
		"redis":   len(p.TagTreeApp.GetAccountTagCodes(accountId, consts.ResourceTypeRedis, tagPath)),
		"mongo":   len(p.TagTreeApp.GetAccountTagCodes(accountId, consts.ResourceTypeMongo, tagPath)),
	}
}
