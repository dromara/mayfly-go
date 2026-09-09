package api

import (
	"fmt"
	"mayfly-go/internal/tag/api/form"
	"mayfly-go/internal/tag/api/vo"
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/application/dto"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"sort"
	"strings"
)

type TagTree struct {
	tagTreeApp       application.TagTree       `inject:"T"`
	tagTreeRelateApp application.TagTreeRelate `inject:"T"`
}

func (t *TagTree) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 获取标签树列表
		req.NewGet("", t.GetTagTree),

		// 根据条件获取标签
		req.NewGet("query", t.ListByQuery),

		req.NewPost("", t.SaveTagTree).Log(req.NewLogSaveI(imsg.LogTagSave)).RequiredPermissionCode("tag:save"),

		req.NewDelete(":id", t.DelTagTree).Log(req.NewLogSaveI(imsg.LogTagDelete)).RequiredPermissionCode("tag:del"),

		req.NewGet("/resource-tags", t.ListResourceTags),

		req.NewGet("/resources/tag-paths", t.TagResources),

		req.NewGet("/resources/count", t.CountTagResource),

		// 获取关联的标签id列表
		req.NewGet("/relate/:relateType/:relateId", t.GetRelateTagIds),
	}

	return req.NewConfs("/tag-trees", reqs[:]...)
}

func (p *TagTree) GetTagTree(rc *req.Ctx) {
	tagTypesStr := rc.Query("type")

	var typePaths []entity.TypePath
	if tagTypesStr != "" {
		typePaths = collx.ArrayMap(strings.Split(tagTypesStr, ","), func(val string) entity.TypePath {
			return entity.TypePath(val)
		})
	}

	accountTags := p.tagTreeApp.GetAccountTags(rc.GetLoginAccount().Id, &entity.TagTreeQuery{TypePaths: typePaths})
	if len(accountTags) == 0 {
		rc.ResData = []any{}
		return
	}

	allTags, err := p.complteTags(accountTags)
	biz.ErrIsNil(err)
	tagTrees := make(vo.TagTreeVOS, 0)
	for _, tag := range allTags {
		tagTrees = append(tagTrees, tag)
	}

	rc.ResData = tagTrees.ToTrees(0)
}

// complteTags 补全标签信息，使其能构造为树结构
func (p *TagTree) complteTags(resourceTags []*dto.SimpleTagTree) ([]*dto.SimpleTagTree, error) {
	codePath2Tag := collx.ArrayToMap(resourceTags, func(tag *dto.SimpleTagTree) string {
		return tag.CodePath
	})

	// 如tagPath = tag1/tag2/tag3/ 需要转为该路径所关联的所有标签路径即 tag1/  tag1/tag2/  tag1/tag2/tag3/三个相关联标签，才可以构造成一棵树
	allTagPaths := make([]string, 0)
	for _, tagPath := range collx.MapKeys(codePath2Tag) {
		allTagPaths = append(allTagPaths, entity.CodePath(tagPath).GetAllPath()...)
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
		return resourceTags, nil
	}

	var tags []*dto.SimpleTagTree
	if err := p.tagTreeApp.ListByQuery(&entity.TagTreeQuery{CodePaths: notExistCodePaths}, &tags); err != nil {
		return nil, err
	}
	// 完善需要补充的标签信息
	return append(resourceTags, tags...), nil
}

func (p *TagTree) ListByQuery(rc *req.Ctx) {
	cond := new(entity.TagTreeQuery)

	cond.Id = uint64(rc.QueryInt("id"))

	if tagPaths := rc.Query("tagPaths"); tagPaths != "" {
		cond.CodePaths = strings.Split(tagPaths, ",")
	}
	if tagType := rc.QueryInt("type"); tagType > 0 {
		cond.Types = collx.AsArray(entity.TagType(tagType))
	}
	if codes := rc.Query("codes"); codes != "" {
		cond.Codes = strings.Split(codes, ",")
	}

	var tagTrees []entity.TagTree
	biz.ErrIsNil(p.tagTreeApp.ListByQuery(cond, &tagTrees))
	rc.ResData = tagTrees
}

func (p *TagTree) SaveTagTree(rc *req.Ctx) {
	tagForm, tagTree := rc.BindJsonAndCopyTo[form.TagTree, entity.TagTree]()

	rc.ReqParam = fmt.Sprintf("tagTreeId: %d, tagName: %s, code: %s", tagTree.Id, tagTree.Name, tagTree.Code)

	biz.ErrIsNil(p.tagTreeApp.SaveTag(rc.MetaCtx, tagForm.Pid, tagTree))
}

func (p *TagTree) DelTagTree(rc *req.Ctx) {
	biz.ErrIsNil(p.tagTreeApp.Delete(rc.MetaCtx, uint64(rc.PathParamInt("id"))))
}

func (p *TagTree) ListResourceTags(rc *req.Ctx) {
	resourceCode := rc.Query("resourceCode")
	biz.NotEmpty(resourceCode, "resourceCode cannot be empty")
	rc.ResData = p.tagTreeApp.ListResourceTagByCode(resourceCode)
}

// 获取用户可操作的标签路径
func (p *TagTree) TagResources(rc *req.Ctx) {
	resourceType := rc.Query("resourceType")
	biz.NotEmpty(resourceType, "resourceType cannot be empty")
	tagResources := p.tagTreeApp.GetAccountTags(rc.GetLoginAccount().Id, &entity.TagTreeQuery{TypePaths: collx.AsArray(entity.TypePath(resourceType))})

	tagPath2Resource := collx.ArrayToMap(tagResources, func(tagResource *dto.SimpleTagTree) string {
		return string(entity.CodePath(tagResource.CodePath).GetTag())
	})

	tagPaths := collx.MapKeys(tagPath2Resource)
	sort.Strings(tagPaths)
	rc.ResData = tagPaths
}

// 统计当前用户指定标签下关联的资源数量
func (p *TagTree) CountTagResource(rc *req.Ctx) {
	tagPath := rc.Query("tagPath")
	accountId := rc.GetLoginAccount().Id

	// 资源类型计数配置：key为返回字段名，Types为资源类型路径(机器/数据库实例需关联授权凭证，故路径含两段)
	countResourceTypes := []struct {
		Key   string
		Types []entity.TagType
	}{
		{"machine", []entity.TagType{entity.TagTypeMachine, entity.TagTypeAuthCert}},
		{"db", []entity.TagType{entity.TagTypeDbInstance, entity.TagTypeAuthCert}},
		{"es", []entity.TagType{entity.TagTypeEsInstance}},
		{"redis", []entity.TagType{entity.TagTypeRedis}},
		{"mongo", []entity.TagType{entity.TagTypeMongo}},
		{"container", []entity.TagType{entity.TagTypeContainer}},
		{"kafka", []entity.TagType{entity.TagTypeMqKafka}},
		{"milvus", []entity.TagType{entity.TagTypeMilvus}},
	}

	counts := make(collx.M, len(countResourceTypes))
	for _, crt := range countResourceTypes {
		counts[crt.Key] = len(p.tagTreeApp.GetAccountResourceCodes(accountId, tagPath, crt.Types...))
	}
	rc.ResData = counts
}

// 获取关联的标签id
func (p *TagTree) GetRelateTagIds(rc *req.Ctx) {
	rc.ResData = p.tagTreeRelateApp.GetTagPathsByRelate(entity.TagRelateType(rc.PathParamInt("relateType")), uint64(rc.PathParamInt("relateId")))
}
