package dbtool

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"mayfly-go/internal/ai/application/resource"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/logx"
)

// maxDatabasesPerAsset 单个资产最多展开的库选项数量，防止库过多时选项列表过长
const maxDatabasesPerAsset = 50

// ensureDbConn 数据库工具公共连接获取模板：
// 校验 DbId 非零后获取数据库连接。
// 各工具在调用本函数前应已完成 TryApplyResumedParams 和参数补全中断。
func ensureDbConn(ctx context.Context, dbId int64, dbName string) (*dbi.DbConn, error) {
	if dbId == 0 {
		return nil, tools.NewToolError(fmt.Errorf("dbId is required"), tools.RecoverRetry)
	}
	// 空 dbName 不再静默去连「默认库」：实测当资产下有多个可连库时，它会连到一个没选库的上下文，
	// 结果是 DbQueryTables 返回 success + tables:[]、DbQueryData 返回 success 但体内裹着
	// `Error 1046 No database selected`——模型据此误判「这个库是空的」，白跑一轮，
	// 工具失败率统计也跟着失真（假成功）。直接报缺少 dbName 并列出候选，一次就能选对。
	// 只有一个可连库的资产不拦（候选只有一个时默认连它就是正确行为）
	if strings.TrimSpace(dbName) == "" {
		if candidates := candidateDbNames(ctx, dbId); len(candidates) > 1 {
			return nil, tools.NewToolError(fmt.Errorf(
				"dbName is required: this instance has %d connectable databases [%s]; pick the real schema name (the resource display name is not a schema name)",
				len(candidates), strings.Join(candidates, ", ")), tools.RecoverRetry)
		}
	}
	conn, err := application.GetDbApp().GetDbConn(ctx, uint64(dbId), dbName)
	if err != nil {
		// 把失败变成一次可自纠的调用：模型常把资源显示名（如「本地库」「ali」）当 schema 名传入。
		//
		// 旧提示只说「请先连默认库执行 SHOW DATABASES 获取真实库名」，但它正是因为连不上才失败——
		// 那是一条走不通的路，只能退化成继续瞎猜（实测这类占工具调用失败的多数）。
		// 现在直接把该实例可连接的真实库名列出来，一次失败即可纠正
		if dbName != "" && isUnknownDatabaseError(err) {
			// 文案由纯函数构造（语义可表驱动断言，见 dbUnknownNameFailure）
			err = dbUnknownNameFailure(err, candidateDbNames(ctx, dbId))
		}
		if hint := connLevelFailureHint(dbName, err); hint != nil {
			err = hint
		}
		return nil, tools.NewToolError(err, tools.RecoverRetry)
	}
	return conn, nil
}

// isUnknownDatabaseError 判定“连不上是因为库名/库权限”这一类失败。
//
// 必须覆盖四种表述：
//   - MySQL 1049 Unknown database：库名不存在
//   - MySQL 1044 Access denied ... to database：实测把实例显示名（mayfly-go）当 schema 传入时
//     报的是这个而不是 1049——只匹配 1049 会让这类错完全拿不到提示（真机实测：模型连撞 2 次
//     因为没有任何可执行信息）
//   - PostgreSQL 的 `pq: database "x" does not exist`
//   - SQLite 的「数据库文件不存在」
//
// 不匹配连接本身失败（如 invalid connection）：那种情形给库名提示只会误导
func isUnknownDatabaseError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "1049") || strings.Contains(msg, "Unknown database") ||
		strings.Contains(msg, "1044") || strings.Contains(msg, "Access denied") ||
		strings.Contains(msg, "does not exist") || strings.Contains(msg, "数据库文件不存在")
}

// isConnectionLevelError 判定「连不上是连接本身的问题，不是库名问题」。
//
// MySQL 驱动在首次建连失败时会把服务端错误（如 1044 Access denied）吞成 ErrBadConn，
// 上抛的只剩 invalid connection——不区分它就会让这类失败什么提示都拿不到。
// 与 isUnknownDatabaseError 互斥：前者给库名候选，后者给「先留空再试」这条可执行动作
func isConnectionLevelError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "invalid connection") || strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") || strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "bad file descriptor")
}

// candidateDbNames 列出指定数据库实例资产上可连接的真实库名（按当前账号可见面）。
// 拿不到（无登录态/资产已删/无权）返回空，调用方退回通用提示，不因取候选失败而遮蔽原错
func candidateDbNames(ctx context.Context, dbId int64) []string {
	// 取数走 resource 的统一入口（可见面过滤、登录态与 ParseInt 都在那里一处实现）
	r, ok := resource.FindVisibleById(ctx, resource.TypeDb, strconv.FormatInt(dbId, 10))
	if !ok {
		return nil
	}
	return nonEmptyDbNames(listDbNamesOfAsset(ctx, r))
}

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
	// 取数统一走 resource.ListVisible（服务可用性、登录态、可见面过滤只有一处实现）
	resources, err := resource.ListVisible(ctx, resource.TypeDb)
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
