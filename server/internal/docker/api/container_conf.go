package api

import (
	"mayfly-go/internal/docker/api/form"
	"mayfly-go/internal/docker/api/vo"
	"mayfly-go/internal/docker/application"
	"mayfly-go/internal/docker/application/dto"
	"mayfly-go/internal/docker/dkm"
	"mayfly-go/internal/docker/domain/entity"
	"mayfly-go/internal/pkg/consts"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/spf13/cast"
)

type ContainerConf struct {
	containerApp application.Container `inject:"T"`
	tagTreeApp   tagapp.TagTree        `inject:"T"`
}

func (cc *ContainerConf) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("/page", cc.GetContainerPage),
		req.NewPost("/save", cc.Save),
		req.NewDelete("/del/:ids", cc.Delete),
	}

	return req.NewConfs("docker/container-conf", reqs[:]...)
}

func (cc *ContainerConf) GetContainerPage(rc *req.Ctx) {
	condition := req.BindQuery[*entity.ContainerQuery](rc)

	tags := cc.tagTreeApp.GetAccountTags(rc.GetLoginAccount().Id, &tagentity.TagTreeQuery{
		TypePaths:     collx.AsArray(tagentity.NewTypePaths(tagentity.TagTypeContainer)),
		CodePathLikes: collx.AsArray(condition.TagPath),
	})
	// 不存在，即没有可操作数据
	if len(tags) == 0 {
		rc.ResData = model.NewEmptyPageResult[any]()
		return
	}

	tagCodePaths := tags.GetCodePaths()
	containerCodes := tagentity.GetCodesByCodePaths(tagentity.TagTypeContainer, tagCodePaths...)
	condition.Codes = collx.ArrayDeduplicate(containerCodes)

	res, err := cc.containerApp.GetContainerPage(condition)
	biz.ErrIsNil(err)
	if res.Total == 0 {
		rc.ResData = res
		return
	}

	resVo := model.PageResultConv[*entity.Container, *vo.ContainerConf](res)
	containerVos := resVo.List
	cc.tagTreeApp.FillTagInfo(tagentity.TagType(consts.ResourceTypeContainer), collx.ArrayMap(containerVos, func(cvo *vo.ContainerConf) tagentity.ITagResource {
		return cvo
	})...)

	rc.ResData = resVo
}

func (c *ContainerConf) Save(rc *req.Ctx) {
	machineForm, container := req.BindJsonAndCopyTo[*form.ContainerSave, *entity.Container](rc)
	rc.ReqParam = machineForm

	biz.ErrIsNil(c.containerApp.SaveContainer(rc.MetaCtx, &dto.SaveContainer{
		Container:    container,
		TagCodePaths: machineForm.TagCodePaths,
	}))
}

func (c *ContainerConf) Delete(rc *req.Ctx) {
	idsStr := rc.PathParam("ids")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		biz.ErrIsNil(c.containerApp.DeleteContainer(rc.MetaCtx, cast.ToUint64(v)))
	}
}

func GetCli(rc *req.Ctx) *dkm.Client {
	id := rc.PathParamInt("id")
	biz.IsTrue(id > 0, "id error")
	cli, err := application.GetContainerApp().GetContainerCli(rc.MetaCtx, uint64(id))
	biz.ErrIsNil(err)
	return cli
}
