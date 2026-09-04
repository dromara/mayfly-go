// Package resourcetool AI 资源清单工具
//
// 提供 ListResources 工具：查询当前用户有权限操作的全部资源（机器、数据库等），
// 供 LLM 在用户未指定目标资源或询问资源清单/整体状态前获取资源定位信息
// （id/code/ip 等），再调用具体资源工具。数据源为 ai/application/resource
// 统一资源查询服务（账号级权限过滤，参数补全选项构建共用同一数据源）。
package resourcetool

import (
	"context"
	"fmt"
	"strings"

	"mayfly-go/internal/ai/application/resource"
	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/i18n"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// maxOutputResources 工具单次输出上限，防止资源过多撑爆上下文
// （截断时输出 Truncated 标记与完整 Total，由 LLM 自行按 Keyword 分批检索）
const maxOutputResources = 200

// appLoader 资源查询服务加载器（默认取全局默认 App，测试可替换）
var appLoader = resource.GetApp

// SetAppLoader 替换资源查询服务加载器（测试注入扩展点，对齐 mcpext.SetServerLoader 模式）
func SetAppLoader(loader func() resource.App) {
	appLoader = loader
}

// ListResourcesParam 资源清单查询参数
type ListResourcesParam struct {
	ResourceType string `json:"resourceType" jsonschema_description:"资源类型过滤：machine=机器、db=数据库。留空返回全部类型。"`
	Keyword      string `json:"keyword" jsonschema_description:"关键词过滤（可选），对资源名称/编码/描述模糊匹配，如机器 IP、机器名或数据库名。"`
}

// ResourceItem 统一资源条目
type ResourceItem struct {
	ResourceType string            `json:"resourceType" jsonschema_description:"资源类型"`
	Id           string            `json:"id" jsonschema_description:"资源ID（作为机器/数据库等工具调用的定位参数）"`
	Code         string            `json:"code" jsonschema_description:"资源编码（机器的 code 即授权凭证名 authCertName）"`
	Name         string            `json:"name" jsonschema_description:"资源名称"`
	Description  string            `json:"description" jsonschema_description:"辅助描述（机器为 ip:port，数据库为编码）"`
	Detail       map[string]string `json:"detail,omitempty" jsonschema_description:"类型相关明细：机器为 ip/port，数据库为可连接库名 databases"`
}

// ListResourcesOutput 资源清单查询输出
type ListResourcesOutput struct {
	Total     int            `json:"total" jsonschema_description:"符合条件的资源总数"`
	Truncated bool           `json:"truncated" jsonschema_description:"结果是否因超过单次输出上限被截断，截断时请细化 resourceType 或 keyword 后重试"`
	Resources []ResourceItem `json:"resources" jsonschema_description:"资源列表"`
}

// GetResourceList 创建资源清单查询工具
func GetResourceList() (tool.InvokableTool, error) {
	return utils.InferTool("ListResources",
		i18n.T(imsg.ResourceListToolInfo),
		func(ctx context.Context, param *ListResourcesParam) (*ListResourcesOutput, error) {
			la := contextx.GetLoginAccount(ctx)
			if la == nil {
				return nil, tools.NewToolError(fmt.Errorf("no login account in context"), tools.RecoverNone)
			}

			app := appLoader()
			if app == nil {
				return nil, tools.NewToolError(fmt.Errorf("resource app not initialized"), tools.RecoverRetry)
			}

			resources, err := app.List(ctx, la.Id, &resource.Query{
				Types:   typeFilter(param.ResourceType),
				Keyword: param.Keyword,
			})
			if err != nil {
				return nil, tools.NewToolError(err, tools.RecoverRetry)
			}

			output := &ListResourcesOutput{
				Total:     len(resources),
				Resources: make([]ResourceItem, 0, len(resources)),
			}
			for i, r := range resources {
				if i >= maxOutputResources {
					output.Truncated = true
					break
				}
				output.Resources = append(output.Resources, toResourceItem(r))
			}
			return output, nil
		},
	)
}

// typeFilter 解析资源类型参数为过滤条件（空值返回 nil=不过滤；未知类型由 App 跳过，结果为空）
func typeFilter(resourceType string) []string {
	resourceType = strings.TrimSpace(resourceType)
	if resourceType == "" {
		return nil
	}
	return []string{resourceType}
}

// toResourceItem 转换为输出条目（nil 资源丢弃）
func toResourceItem(r *resource.Resource) ResourceItem {
	item := ResourceItem{
		ResourceType: r.Type,
		Id:           r.Id,
		Code:         r.Code,
		Name:         r.Name,
		Description:  r.Description,
	}
	if len(r.Detail) > 0 {
		item.Detail = r.Detail
	}
	return item
}
