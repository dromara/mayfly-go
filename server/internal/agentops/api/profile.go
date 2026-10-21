package api

import (
	"mayfly-go/internal/ai/pkg/headless"
	"mayfly-go/pkg/req"
)

// Profile Agent 装配档目录 API：表单档位选项与列表筛选器的唯一数据源。
//
// 为什么要开这个读接口：后端 RegisterProfile 是开放扩展点（业务可注册自定义档），而前端曾把
// 四档写死成静态枚举——新增一档就得改前端发版，且保存时无人校验 profileId 是否存在，任意字符串
// 都能入库。目录由 ai/application 注册到中立契约 headless.ProfileCatalog，本接口只做透传。
type Profile struct{}

func (p *Profile) ReqConfs() *req.Confs {
	// 档位目录是纯元信息（id / 名称 / 说明 / 是否需一次性许可），不含任何租户数据，
	// 按平台惯例列表 GET 不挂权限码
	reqs := [...]*req.Conf{
		req.NewGet("", p.List),
	}

	return req.NewConfs("/agentops/profiles", reqs[:]...)
}

// List 返回全部已注册档位（按 id 升序）。
//
// ai 侧尚未装配时返回空数组而非报错：档位下拉退化为不可选，比整页 500 更可解释
func (p *Profile) List(rc *req.Ctx) {
	catalog := headless.Profiles()
	if catalog == nil {
		rc.ResData = []headless.ProfileInfo{}
		return
	}
	rc.ResData = catalog.List()
}

var _ req.RouterApi = (*Profile)(nil)
