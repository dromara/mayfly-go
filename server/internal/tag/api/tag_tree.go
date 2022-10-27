package api

import (
	"fmt"
	"mayfly-go/internal/tag/api/vo"
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/ctx"
	"mayfly-go/pkg/ginx"
)

type TagTree struct {
	TagTreeApp application.TagTree
}

func (p *TagTree) GetAccountTags(rc *ctx.ReqCtx) {
	rc.ResData = p.TagTreeApp.ListTagByAccountId(rc.LoginAccount.Id)
}

func (p *TagTree) GetTagTree(rc *ctx.ReqCtx) {
	var tagTrees vo.TagTreeVOS
	p.TagTreeApp.ListByQuery(new(entity.TagTreeQuery), &tagTrees)
	rc.ResData = tagTrees.ToTrees(0)
}

func (p *TagTree) SaveTagTree(rc *ctx.ReqCtx) {
	projectTree := &entity.TagTree{}
	ginx.BindJsonAndValid(rc.GinCtx, projectTree)

	loginAccount := rc.LoginAccount
	projectTree.SetBaseInfo(loginAccount)
	p.TagTreeApp.Save(projectTree)

	rc.ReqParam = fmt.Sprintf("tagTreeId: %d, tagName: %s, codePath: %s", projectTree.Id, projectTree.Name, projectTree.CodePath)
}

func (p *TagTree) DelTagTree(rc *ctx.ReqCtx) {
	p.TagTreeApp.Delete(uint64(ginx.PathParamInt(rc.GinCtx, "id")))
}
