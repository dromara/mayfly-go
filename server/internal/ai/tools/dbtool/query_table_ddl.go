package dbtool

import (
	"context"
	"fmt"

	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/internal/db/application"
	"mayfly-go/pkg/i18n"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type QueryTableDDLParam struct {
	DbId       int64    `json:"dbId" jsonschema_description:"数据库ID。取值逻辑：1. 用户本次明确指定；2. 从前序工具的输入输出中继承已选定的数据库ID；3. 若均无，传0以触发参数补全。禁止凭空猜测。"`
	DbName     string   `json:"dbName" jsonschema_description:"数据库名称（可选）。取值逻辑：1. 用户本次明确指定；2. 从前序工具的输入输出中继承已选定的数据库名称；3. 留空则使用资产配置的默认库。禁止凭空猜测。"`
	TableNames []string `json:"tableNames" jsonschema_description:"表名列表，支持一次查询多个表的DDL" jsonschema:"required" `
}

type QueryTableDDLOutput struct {
	DbId   int64             `json:"dbId" jsonschema_description:"数据库ID"`
	DbName string            `json:"dbName" jsonschema_description:"数据库名称"`
	DbType string            `json:"dbType" jsonschema_description:"数据库类型，如mysql、postgresql等"`
	DDLS   map[string]string `json:"ddls" jsonschema_description:"各表的DDL，key为表名，value为DDL语句"`
}

func GetQueryTableDDL() (tool.InvokableTool, error) {
	return utils.InferTool("DbQueryTableDDL",
		i18n.T(imsg.DbQueryTableDDLToolInfo),
		func(ctx context.Context, param *QueryTableDDLParam) (*QueryTableDDLOutput, error) {
			toolDesc := i18n.TC(ctx, imsg.DbQueryTableDDLToolDesc)
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

			if len(param.TableNames) == 0 {
				return nil, tools.NewToolError(fmt.Errorf("%s", i18n.TC(ctx, imsg.MissingRequiredParams)), tools.RecoverRetry)
			}

			conn, err := application.GetDbApp().GetDbConn(ctx, uint64(param.DbId), param.DbName)
			if err != nil {
				return nil, tools.NewToolError(err, tools.RecoverRetry)
			}

			ddls := make(map[string]string, len(param.TableNames))
			md := conn.GetMetadata()
			for _, tableName := range param.TableNames {
				ddl, err := md.GetTableDDL(tableName, false)
				if err != nil {
					return nil, tools.NewToolError(err, tools.RecoverRetry)
				}
				ddls[tableName] = ddl
			}

			output := &QueryTableDDLOutput{
				DbId:   param.DbId,
				DbName: param.DbName,
				DbType: string(conn.Info.Type),
				DDLS:   ddls,
			}
			return output, nil
		},
	)
}
