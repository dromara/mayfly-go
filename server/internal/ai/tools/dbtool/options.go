package dbtool

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"mayfly-go/internal/ai/application/resource"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/logx"
)

// maxDatabasesPerAsset 单个资产最多展开的库选项数量，防止库过多时选项列表过长
const maxDatabasesPerAsset = 50

// queryDbOptions 查询当前用户有权限操作的数据库资产并展开为可选项
// （数据源为统一资源查询服务，经「数据库实例+授权凭证+数据库」标签做账号级权限过滤；
// ask_user options 模式）
func queryDbOptions(ctx context.Context) []tools.CompletionOption {
	return tools.SafeOptionsFn(ctx, "queryDbOptions", func() []tools.CompletionOption {
		return listDbOptions(ctx)
	})
}

// dbOptionItem 单个库选项：dbName 为连接所需的真实库名
type dbOptionItem struct {
	dbName string
	label  string
}

func listDbOptions(ctx context.Context) []tools.CompletionOption {
	app := resource.GetApp()
	if app == nil {
		return nil
	}
	la := contextx.GetLoginAccount(ctx)
	if la == nil {
		logx.WarnfContext(ctx, "[queryDbOptions] no login account in context")
		return nil
	}

	resources, err := app.List(ctx, la.Id, &resource.Query{Types: []string{resource.TypeDb}})
	if err != nil {
		logx.WarnfContext(ctx, "[queryDbOptions] failed to list dbs: %v", err)
		return nil
	}

	var options []tools.CompletionOption
	for _, r := range resources {
		id, err := strconv.ParseInt(r.Id, 10, 64)
		if err != nil {
			logx.WarnfContext(ctx, "[queryDbOptions] invalid db resource id %q of asset %s", r.Id, r.Name)
			continue
		}

		// 注意：资产名不能作为连接的数据库名，必须为每个资产展开可连接的真实库名列表
		for _, item := range listDbNamesOfAsset(ctx, r) {
			payload := map[string]any{
				"dbId":   id,
				"dbName": item.dbName,
			}
			valueBytes, err := json.Marshal(payload)
			if err != nil {
				continue
			}
			options = append(options, tools.CompletionOption{
				Label: item.label,
				Value: string(valueBytes),
			})
		}
	}

	return options
}

// listDbNamesOfAsset 展开单个数据库资产的可连接库名列表（资产信息取自统一资源 Extra）
// - 指定库名模式：使用资产配置的 Database（空格分隔多个库名）
// - 自动获取模式：实时查询实例下的所有库名（否则空 dbName 连接后无默认库，查表结果为空）
func listDbNamesOfAsset(ctx context.Context, r *resource.Resource) []dbOptionItem {
	assetName := r.Name
	authCertName, _ := r.Extra[resource.ExtraKeyAuthCertName].(string)
	database, _ := r.Extra[resource.ExtraKeyDatabase].(string)
	mode, _ := r.Extra[resource.ExtraKeyGetDatabaseMode].(entity.DbGetDatabaseMode)

	// 指定库名模式
	if mode == entity.DbGetDatabaseModeAssign && database != "" {
		var items []dbOptionItem
		for _, name := range strings.Fields(database) {
			items = append(items, dbOptionItem{dbName: name, label: assetName + " / " + name})
		}
		if len(items) > 0 {
			return items
		}
		return []dbOptionItem{{dbName: "", label: assetName}}
	}

	// 自动获取模式：实时列库
	if authCertName != "" {
		if instApp := application.GetDbInstanceApp(); instApp != nil {
			dbNames, err := instApp.GetDatabasesByAc(ctx, authCertName)
			if err != nil {
				logx.WarnfContext(ctx, "[queryDbOptions] failed to list databases of asset %s: %v", assetName, err)
			} else {
				items := make([]dbOptionItem, 0, len(dbNames))
				for _, name := range dbNames {
					if name == "" {
						continue
					}
					items = append(items, dbOptionItem{dbName: name, label: assetName + " / " + name})
					if len(items) >= maxDatabasesPerAsset {
						break
					}
				}
				if len(items) > 0 {
					return items
				}
			}
		}
	}

	// 兜底：无法确定库名时传空串（由连接层决定默认库）
	return []dbOptionItem{{dbName: "", label: assetName}}
}
