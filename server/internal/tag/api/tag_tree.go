package api

import (
	"fmt"
	"mayfly-go/internal/tag/api/vo"
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"strings"
)

type TagTree struct {
	TagTreeApp application.TagTree
}

func (p *TagTree) GetAccountTags(rc *req.Ctx) {
	tagPaths := p.TagTreeApp.ListTagByAccountId(rc.LoginAccount.Id)
	allTagPath := make([]string, 0)
	if len(tagPaths) > 0 {
		tags := p.TagTreeApp.ListTagByPath(tagPaths...)
		for _, v := range tags {
			allTagPath = append(allTagPath, v.CodePath)
		}
	}
	rc.ResData = allTagPath
}

func (p *TagTree) GetTagTree(rc *req.Ctx) {
	var tagTrees vo.TagTreeVOS
	p.TagTreeApp.ListByQuery(new(entity.TagTreeQuery), &tagTrees)
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

	loginAccount := rc.LoginAccount
	tagTree.SetBaseInfo(loginAccount)
	rc.ReqParam = fmt.Sprintf("tagTreeId: %d, tagName: %s, codePath: %s", tagTree.Id, tagTree.Name, tagTree.CodePath)

	biz.ErrIsNil(p.TagTreeApp.Save(tagTree))
}

func (p *TagTree) DelTagTree(rc *req.Ctx) {
	biz.ErrIsNil(p.TagTreeApp.Delete(uint64(ginx.PathParamInt(rc.GinCtx, "id"))))
}
