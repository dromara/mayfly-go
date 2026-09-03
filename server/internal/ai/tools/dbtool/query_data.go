package dbtool

import (
	"context"
	"fmt"

	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/i18n"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type QueryDataParam struct {
	DbId   int64    `json:"dbId" jsonschema_description:"数据库ID。取值逻辑：1. 用户本次明确指定；2. 从前序工具的输入输出中继承已选定的数据库ID；3. 若均无，传0以触发参数补全。禁止凭空猜测。"`
	DbName string   `json:"dbName" jsonschema_description:"数据库名称（可选）。取值逻辑：1. 用户本次明确指定；2. 从前序工具的输入输出中继承已选定的数据库名称；3. 留空则使用资产配置的默认库。禁止凭空猜测。"`
	SQLs   []string `json:"sqls" jsonschema_description:"SQL语句数组，支持一次性查询多条SQL" jsonschema:"required" `
}

type SingleQueryResult struct {
	SQL     string             `json:"sql" jsonschema_description:"执行的SQL语句"`
	Columns []*dbi.QueryColumn `json:"columns" jsonschema_description:"查询结果列信息"`
	Rows    []map[string]any   `json:"rows" jsonschema_description:"查询结果数据"`
	Error   string             `json:"error,omitempty" jsonschema_description:"执行错误信息"`
}

type QueryDataOutput struct {
	DbId    int64               `json:"dbId" jsonschema_description:"数据库ID"`
	DbName  string              `json:"dbName" jsonschema_description:"数据库名称"`
	DbType  string              `json:"dbType" jsonschema_description:"数据库类型，如mysql、postgresql等"`
	Results []SingleQueryResult `json:"results" jsonschema_description:"多条SQL查询结果"`
}

func GetQueryData() (tool.InvokableTool, error) {
	return utils.InferTool("DbQueryData",
		i18n.T(imsg.DbQueryDataToolInfo),
		func(ctx context.Context, param *QueryDataParam) (*QueryDataOutput, error) {
			toolDesc := i18n.TC(ctx, imsg.DbQueryDataToolDesc)
			tools.TryApplyResumedParams(ctx, param)
			// 检查必要参数，触发参数完善（dbName 可选，留空使用资产默认库）
			if param.DbId == 0 {
				if err := tools.InterruptOrResumeParamCompletion(ctx, toolDesc, param, i18n.TC(ctx, imsg.DbInfoIncomplete), "db", []tools.CompletionParamInfo{
					{Param: "dbId", Name: "数据库ID"},
					{Param: "dbName", Name: "数据库名称（可选，留空使用默认库）"},
				}, queryDbOptions(ctx)); err != nil {
					return nil, err
				}
			}

			if len(param.SQLs) == 0 {
				return nil, fmt.Errorf("sqls parameter is required")
			}

			if len(param.SQLs) > 10 {
				return nil, fmt.Errorf("too many SQL statements, maximum allowed is 10, got %d", len(param.SQLs))
			}

			conn, err := application.GetDbApp().GetDbConn(ctx, uint64(param.DbId), param.DbName)
			if err != nil {
				return nil, tools.NewToolError(err, tools.RecoverRetry)
			}

			results := make([]SingleQueryResult, 0, len(param.SQLs))
			for _, sql := range param.SQLs {
				if sql == "" {
					continue
				}

				result := SingleQueryResult{SQL: sql}
				rows := make([]map[string]any, 0)
				columns, err := conn.WalkQueryRows(ctx, sql, func(row map[string]any, columns []*dbi.QueryColumn) error {
					rows = append(rows, row)
					if len(rows) > 1000 {
						return dbi.NewStopWalkQueryError("The maximum number of query rows is exceeded: 1000")
					}
					return nil
				})

				if err != nil {
					result.Error = err.Error()
				} else {
					result.Columns = columns
					result.Rows = rows
				}
				results = append(results, result)
			}

			output := &QueryDataOutput{
				DbId:    param.DbId,
				DbName:  param.DbName,
				DbType:  string(conn.Info.Type),
				Results: results,
			}
			return output, nil
		},
	)
}
