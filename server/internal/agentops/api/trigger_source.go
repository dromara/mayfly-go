package api

import (
	"mayfly-go/internal/agentops/trigger"
	"mayfly-go/pkg/req"
)

// TriggerSource 触发源目录 API：表单「触发源」选项与列表筛选器的唯一数据源。
//
// 为什么要开这个读接口（与 Profile 目录同一理由）：trigger.Registry 是开放扩展点——新增一种
// 来源只需实现 TriggerSource 并向注册表 Install 一行，但若前端仍拿静态枚举装配下拉，新来源
// 就**根本无法被创建**（表单里没有它），已存在的行还会在列表里渲染成红色「-」。
// 那样注册表的可扩展性是假的：扩展一个来源必须改前端发版。
//
// 只回类型键、不回展示名：展示文案属于前端字典（后端写死中文会让英文界面永远看到中文，
// 与 docs/server/i18n.md 同一口径）；前端按 `agentops.source<驼峰>` 命中与否决定展译文
// 还是原始键，新来源未配文案时仍可选、仍可读。
type TriggerSource struct{}

func (t *TriggerSource) ReqConfs() *req.Confs {
	// 触发源目录是纯元信息（类型标识 + 展示名），不含租户数据，按平台惯例列表 GET 不挂权限码
	reqs := [...]*req.Conf{
		req.NewGet("", t.List),
	}

	return req.NewConfs("/agentops/trigger-sources", reqs[:]...)
}

// List 返回全部已注册触发源的类型标识（稳定排序，见 Registry.Types）。
//
// 注册表当前一定非空（alert/webhook 随 init 装配），但仍返回空数组而非报错：
// 装配缺失时下拉退化为不可选，比整页 500 更可解释
func (t *TriggerSource) List(rc *req.Ctx) {
	types := trigger.Default.Types()
	if types == nil {
		types = []string{}
	}
	rc.ResData = types
}

var _ req.RouterApi = (*TriggerSource)(nil)
