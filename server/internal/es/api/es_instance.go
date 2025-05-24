package api

import (
	"mayfly-go/internal/es/api/form"
	"mayfly-go/internal/es/api/vo"
	"mayfly-go/internal/es/application"
	"mayfly-go/internal/es/application/dto"
	"mayfly-go/internal/es/domain/entity"
	"mayfly-go/internal/es/esm/esi"
	"mayfly-go/internal/es/imsg"
	"mayfly-go/internal/pkg/consts"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"net/http"
	"net/url"
	"strings"

	"github.com/may-fly/cast"
)

type Instance struct {
	inst                application.Instance    `inject:"T"`
	tagApp              tagapp.TagTree          `inject:"T"`
	resourceAuthCertApp tagapp.ResourceAuthCert `inject:"T"`
}

func (d *Instance) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{

		// /es/instance 获取实例列表
		req.NewGet("", d.Instances),

		// /es/instance/test-conn 测试连接
		req.NewPost("/test-conn", d.TestConn),

		// /es/instance 添加实例
		req.NewPost("", d.SaveInstance).Log(req.NewLogSaveI(imsg.LogEsInstSave)),

		// /es/instance/:id 删除实例
		req.NewDelete(":instanceId", d.DeleteInstance).Log(req.NewLogSaveI(imsg.LogEsInstDelete)),

		// /es/instance/proxy 反向代理es接口请求
		req.NewAny("/proxy/:instanceId/*path", d.Proxy),
	}

	return req.NewConfs("/es/instance", reqs[:]...)
}

func (d *Instance) Instances(rc *req.Ctx) {
	queryCond := req.BindQuery[*entity.InstanceQuery](rc)

	// 只查询实例，兼容没有录入密码的实例
	instTags := d.tagApp.GetAccountTags(rc.GetLoginAccount().Id, &tagentity.TagTreeQuery{
		TypePaths:     collx.AsArray(tagentity.NewTypePaths(tagentity.TagTypeEsInstance)),
		CodePathLikes: collx.AsArray(queryCond.TagPath),
	})

	// 不存在可操作的数据库，即没有可操作数据
	if len(instTags) == 0 {
		rc.ResData = model.NewEmptyPageResult[any]()
		return
	}
	dbInstCodes := tagentity.GetCodesByCodePaths(tagentity.TagTypeEsInstance, instTags.GetCodePaths()...)
	queryCond.Codes = dbInstCodes

	res, err := d.inst.GetPageList(queryCond)
	biz.ErrIsNil(err)
	resVo := model.PageResultConv[*entity.EsInstance, *vo.InstanceListVO](res)
	instvos := resVo.List

	// 只查询标签
	certTags := d.tagApp.GetAccountTags(rc.GetLoginAccount().Id, &tagentity.TagTreeQuery{
		TypePaths:     collx.AsArray(tagentity.NewTypePaths(tagentity.TagTypeEsInstance, tagentity.TagTypeAuthCert)),
		CodePathLikes: collx.AsArray(queryCond.TagPath),
	})

	// 填充授权凭证信息
	d.resourceAuthCertApp.FillAuthCertByAcNames(tagentity.GetCodesByCodePaths(tagentity.TagTypeAuthCert, certTags.GetCodePaths()...), collx.ArrayMap(instvos, func(vos *vo.InstanceListVO) tagentity.IAuthCert {
		return vos
	})...)

	// 填充标签信息
	d.tagApp.FillTagInfo(tagentity.TagType(consts.ResourceTypeEsInstance), collx.ArrayMap(instvos, func(insvo *vo.InstanceListVO) tagentity.ITagResource {
		return insvo
	})...)

	rc.ResData = resVo
}

func (d *Instance) TestConn(rc *req.Ctx) {
	fm, instance := req.BindJsonAndCopyTo[*form.InstanceForm, *entity.EsInstance](rc)

	var ac *tagentity.ResourceAuthCert
	if len(fm.AuthCerts) > 0 {
		ac = fm.AuthCerts[0]
	}

	res, err := d.inst.TestConn(rc.MetaCtx, instance, ac)
	biz.ErrIsNil(err)
	rc.ResData = res
}
func (d *Instance) SaveInstance(rc *req.Ctx) {
	fm, instance := req.BindJsonAndCopyTo[*form.InstanceForm, *entity.EsInstance](rc)

	rc.ReqParam = fm
	id, err := d.inst.SaveInst(rc.MetaCtx, &dto.SaveEsInstance{
		EsInstance:   instance,
		AuthCerts:    fm.AuthCerts,
		TagCodePaths: fm.TagCodePaths,
	})
	biz.ErrIsNil(err)
	rc.ResData = id
}
func (d *Instance) DeleteInstance(rc *req.Ctx) {
	idsStr := rc.PathParam("instanceId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		biz.ErrIsNilAppendErr(d.inst.Delete(rc.MetaCtx, cast.ToUint64(v)), "delete db instance failed: %s")
	}
}
func (d *Instance) Proxy(rc *req.Ctx) {
	path := rc.PathParam("path")
	instanceId := getInstanceId(rc)
	// 去掉request中的 id 和 path参数，否则es会报错

	r := rc.GetRequest()
	_ = RemoveQueryParam(r, "id", "path")

	err := d.inst.DoConn(rc.MetaCtx, instanceId, func(conn *esi.EsConn) error {
		conn.Proxy(rc.GetWriter(), r, path)
		return nil
	})

	biz.ErrIsNil(err)
}

func RemoveQueryParam(req *http.Request, paramNames ...string) error {
	parsedURL, err := url.ParseRequestURI(req.RequestURI)
	if err != nil {
		return err
	}
	// Get the query parameters
	queryParams, err := url.ParseQuery(parsedURL.RawQuery)
	if err != nil {
		return err
	}
	// Remove the specified query parameter
	for i := range paramNames {
		delete(queryParams, paramNames[i])
	}
	// Reconstruct the query string
	parsedURL.RawQuery = queryParams.Encode()
	// Update the request URL
	req.URL = parsedURL
	req.RequestURI = parsedURL.String()
	return nil
}

func getInstanceId(rc *req.Ctx) uint64 {
	instanceId := rc.PathParamInt("instanceId")
	biz.IsTrue(instanceId > 0, "instanceId error")
	return uint64(instanceId)
}
