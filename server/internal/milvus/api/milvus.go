package api

import (
	"mayfly-go/internal/milvus/api/form"
	"mayfly-go/internal/milvus/api/vo"
	"mayfly-go/internal/milvus/application"
	"mayfly-go/internal/milvus/domain/entity"
	"mayfly-go/internal/milvus/imsg"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/spf13/cast"
)

// Milvus Milvus API
type Milvus struct {
	milvusApp           application.Milvus      `inject:"T"`
	tagTreeApp          tagapp.TagTree          `inject:"T"`
	resourceAuthCertApp tagapp.ResourceAuthCert `inject:"T"`
}

// ReqConfs 注册路由
func (m *Milvus) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 获取所有 milvus 列表
		req.NewGet("", m.Milvuses),

		// 测试连接
		req.NewPost("/test-conn", m.TestConn),

		// 保存
		req.NewPost("", m.Save).Log(req.NewLogSaveI(imsg.LogSave)),

		// 删除
		req.NewDelete(":id", m.DeleteById).Log(req.NewLogSaveI(imsg.LogDelete)),
	}

	return req.NewConfs("milvus", reqs[:]...)
}

// Milvuses 获取 Milvus 列表
func (m *Milvus) Milvuses(rc *req.Ctx) {
	queryCond := rc.BindQuery[entity.MilvusQuery]()

	// 不存在可访问标签 id，即没有可操作数据
	tags := m.tagTreeApp.GetAccountTags(rc.GetLoginAccount().Id, &tagentity.TagTreeQuery{
		TypePaths:     collx.AsArray(tagentity.NewTypePaths(tagentity.TagTypeMilvus, tagentity.TagTypeAuthCert)),
		CodePathLikes: collx.AsArray(queryCond.TagPath),
	})
	if len(tags) == 0 {
		rc.ResData = model.NewEmptyPageResult[any]()
		return
	}

	tagCodePaths := tags.GetCodePaths()
	milvusCodes := tagentity.GetCodesByCodePaths(tagentity.TagTypeMilvus, tagCodePaths...)
	queryCond.Codes = milvusCodes

	res, err := m.milvusApp.GetPageList(queryCond)
	biz.ErrIsNil(err)
	resVo := model.PageResultConv[*entity.Milvus, *vo.Milvus](res)

	// 填充授权凭证信息
	acNames := tagentity.GetCodesByCodePaths(tagentity.TagTypeAuthCert, tagCodePaths...)
	m.resourceAuthCertApp.FillAuthCertByAcNames(acNames, collx.ArrayMap(resVo.List, func(vos *vo.Milvus) tagentity.IAuthCert {
		return vos
	})...)

	rc.ResData = resVo
}

// TestConn 测试连接
func (m *Milvus) TestConn(rc *req.Ctx) {
	f := rc.BindJson[form.Milvus]()
	instance := &entity.Milvus{
		Host:               f.Host,
		SshTunnelMachineId: f.SshTunnelMachineId,
	}
	biz.ErrIsNilAppendErr(m.milvusApp.TestConn(rc.MetaCtx, instance, f.AuthCerts[0]), "connection error: %s")
}

// Save 保存
func (m *Milvus) Save(rc *req.Ctx) {
	f := rc.BindJson[form.Milvus]()
	instance := &entity.Milvus{
		Code:               f.Code,
		Name:               f.Name,
		Host:               f.Host,
		Database:           f.Database,
		SshTunnelMachineId: f.SshTunnelMachineId,
	}
	instance.Id = f.Id

	rc.ReqParam = form.Milvus{
		Name:     f.Name,
		Host:     f.Host,
		Database: f.Database,
	}

	biz.ErrIsNil(m.milvusApp.SaveMilvus(rc.MetaCtx, instance, f.AuthCerts, f.TagCodePaths...))
}

// DeleteById 删除
func (m *Milvus) DeleteById(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		m.milvusApp.Delete(rc.MetaCtx, cast.ToUint64(v))
	}
}
