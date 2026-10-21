// 可观测工具的授权面：指标与告警历史同样是账号级资源，必须按运行身份收敛可见面。
//
// 为什么必须在这里显式判定：HTTP 侧的告警事件列表靠「调用者持有 alert:event 权限码」把关，
// 那背后是一个坐在界面前的人类操作者；Agent 工具这条链没有权限码，只有运行 owner
// （headless 经 contextx 注入的登录账号）。少这一步，任意一次派生运行都能读全库告警
// 与任意资源的实时指标——属于 fail-open 的越权读取面。
package monitor

import (
	"context"
	"errors"
	"fmt"

	"mayfly-go/internal/ai/application/resource"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/contextx"

	"github.com/spf13/cast"
)

// resourceAppLoader 统一资源查询服务加载器（与 ListResources 工具、参数补全选项同源）
var resourceAppLoader = resource.GetApp

// SetResourceAppLoader 替换资源查询服务加载器（测试注入扩展点，对齐 resourcetool.SetAppLoader 模式）
func SetResourceAppLoader(loader func() resource.App) {
	resourceAppLoader = loader
}

// runAccountId 取本次运行的归属账号：工具权限收敛的唯一依据，缺失即拒绝（不猜测为管理员）
func runAccountId(ctx context.Context) (uint64, error) {
	la := contextx.GetLoginAccount(ctx)
	if la == nil {
		return 0, tools.NewToolError(errors.New("no login account in run context"), tools.RecoverNone)
	}
	return la.Id, nil
}

// visibleResourceIds 该账号在指定资源类型键名下可见的资源 id 集合。
//
// 走统一资源查询服务的 List，而不是给它新加一个「单点可见性」方法：可见面判定（标签树授权 +
// 资源提供者）已经在那一处收口，再开一条判定路径就是第二个真源，两处一旦分叉就会出现
// 「列表里看得见、工具说无权访问」这类无法解释的行为差异
func visibleResourceIds(ctx context.Context, accountId uint64, typeKey string) (map[uint64]bool, error) {
	app := resourceAppLoader()
	if app == nil {
		return nil, tools.NewToolError(errors.New("resource app not initialized"), tools.RecoverRetry)
	}
	resources, err := app.List(ctx, accountId, &resource.Query{Types: []string{typeKey}})
	if err != nil {
		return nil, tools.NewToolError(fmt.Errorf("list visible resources failed: %w", err), tools.RecoverRetry)
	}
	ids := make(map[uint64]bool, len(resources))
	for _, r := range resources {
		id := cast.ToUint64(r.Id)
		if id == 0 {
			// 非数值 id 不参与判定：留零值会让「未指定资源」被误判成命中可见集合
			continue
		}
		ids[id] = true
	}
	return ids, nil
}

// resolveVisibleType 解析资源类型键名并取回该账号在该类型下的可见资源集合。
// 类型未登记键名（凭证/库名等非资源主体）直接拒绝，不退回数字串去查可见面
func resolveVisibleType(ctx context.Context, resourceType int8) (typeKey string, ids map[uint64]bool, err error) {
	accountId, err := runAccountId(ctx)
	if err != nil {
		return "", nil, err
	}
	typeKey, ok := consts.ResourceTypeKey(resourceType)
	if !ok {
		return "", nil, tools.NewToolError(fmt.Errorf("unknown resource type %d", resourceType), tools.RecoverNone)
	}
	ids, err = visibleResourceIds(ctx, accountId, typeKey)
	if err != nil {
		return "", nil, err
	}
	return typeKey, ids, nil
}

// requireResourceAccess 校验目标资源对运行身份可见：无身份、类型不可辨识、资源不可见一律拒绝。
//
// 供「必须指定单个资源」的工具（QueryMetrics）使用——这类工具没有「退化为全量」的余地
func requireResourceAccess(ctx context.Context, resourceType int8, resourceId uint64) error {
	if resourceId == 0 {
		return tools.NewToolError(errors.New("resourceId is required"), tools.RecoverNone)
	}
	typeKey, ids, err := resolveVisibleType(ctx, resourceType)
	if err != nil {
		return err
	}
	if !ids[resourceId] {
		return tools.NewToolError(
			fmt.Errorf("resource %s:%d is not accessible by the run owner", typeKey, resourceId), tools.RecoverNone)
	}
	return nil
}
