package resource

import (
	"context"

	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/logx"
)

// 本文件收敛「取服务 + 取登录态 + 按类型列资源」这段在每个工具包里各抄一遍的样板。
//
// 只收取数，不收判定：pickUniqueCert / nonEmptyDbNames / isUnknownDatabaseError 这类
// **安全判据**留在各自包内且保持纯函数（可表驱动测试、不外泄给别的调用方误用）

// ListVisible 按类型返回**当前登录账号可见**的资源清单。
//
// 无登录态或未装配服务时返回空清单而非错误：调用方都是「拿不到就退回不纠正/不提示」的
// fail-safe 语义，这里返回错误反而会被各调用方各自吞掉，形成第二套兜底
func ListVisible(ctx context.Context, types ...string) ([]*Resource, error) {
	app := GetApp()
	la := contextx.GetLoginAccount(ctx)
	if app == nil || la == nil {
		return nil, nil
	}
	resources, err := app.List(ctx, la.Id, &Query{Types: types})
	if err != nil {
		logx.WarnfContext(ctx, "[resource] list visible resources failed, types=%v, err=%v", types, err)
		return nil, err
	}
	return resources, nil
}

// FindVisibleById 在可见清单里按类型与资源 id 定位单条资源。
//
// id 是字符串承载的资源主键（与 Resource.Id 同口径）；非正数 id 直接判未命中，
// 避免调用方各写一遍 ParseInt 又各写一遍「0 当合法 id」的疏漏
func FindVisibleById(ctx context.Context, typ, id string) (*Resource, bool) {
	resources, err := ListVisible(ctx, typ)
	if err != nil || resources == nil {
		return nil, false
	}
	for _, r := range resources {
		if r != nil && r.Id == id {
			return r, true
		}
	}
	return nil, false
}

// CertNamesByMachine 机器 code → 该机器可见的授权凭证名列表（一台机器可挂多张凭证）。
//
// 由 machine_provider 从标签路径解析出的真实凭证 name 汇总而成；调用方判「唯一凭证」时
// 必须用 value 长度，不能靠条目数量猜（同机器多凭证会展开成多条资源）
func CertNamesByMachine(ctx context.Context) (map[string][]string, error) {
	resources, err := ListVisible(ctx, TypeMachine)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]string, len(resources))
	for _, r := range resources {
		if r == nil || r.Code == "" {
			continue
		}
		certName, _ := r.Extra[ExtraKeyAuthCertName].(string)
		if certName == "" {
			continue
		}
		out[r.Code] = append(out[r.Code], certName)
	}
	return out, nil
}
