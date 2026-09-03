package dbtool

import (
	"context"
	"encoding/json"
	"strings"

	"mayfly-go/internal/ai/tools"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/logx"
)

// maxDatabasesPerAsset 单个资产最多展开的库选项数量，防止库过多时选项列表过长
const maxDatabasesPerAsset = 50

// queryDbOptions 查询可用数据库列表并格式化为可选项（对齐 tokhub ask_user options 模式）
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
	dbApp := application.GetDbApp()
	if dbApp == nil {
		return nil
	}

	// 查询所有数据库注册信息（最多100条）
	pageResult, err := dbApp.GetPageList(&entity.DbQuery{}, "id DESC")
	if err != nil {
		logx.WarnfContext(ctx, "[queryDbOptions] failed to list dbs: %v", err)
		return nil
	}

	var options []tools.CompletionOption
	for _, db := range pageResult.List {
		if db == nil || db.Id == nil || db.Name == nil {
			continue
		}

		// 注意：db.Name 是资产显示名，不能作为连接的数据库名，
		// 必须为每个资产解析出可连接的真实库名列表。
		for _, item := range listDbNamesOfAsset(ctx, db) {
			payload := map[string]any{
				"dbId":   *db.Id,
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

// listDbNamesOfAsset 解析单个数据库资产的可连接库名列表。
// - 指定库名模式：使用资产配置的 Database（空格分隔多个库名）
// - 自动获取模式：实时查询实例下的所有库名（否则空 dbName 连接后无默认库，查表结果为空）
func listDbNamesOfAsset(ctx context.Context, db *entity.DbListPO) []dbOptionItem {
	assetName := *db.Name

	// 指定库名模式
	if db.GetDatabaseMode == entity.DbGetDatabaseModeAssign && db.Database != nil && *db.Database != "" {
		var items []dbOptionItem
		for _, name := range strings.Fields(*db.Database) {
			items = append(items, dbOptionItem{dbName: name, label: assetName + " / " + name})
		}
		if len(items) > 0 {
			return items
		}
		return []dbOptionItem{{dbName: "", label: assetName}}
	}

	// 自动获取模式：实时列库
	if db.AuthCertName != "" {
		if instApp := application.GetDbInstanceApp(); instApp != nil {
			dbNames, err := instApp.GetDatabasesByAc(ctx, db.AuthCertName)
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
