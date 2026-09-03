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

type SqlExecParam struct {
	DbId   int64  `json:"dbId" jsonschema_description:"数据库ID。取值逻辑：1. 用户本次明确指定；2. 从前序工具的输入输出中继承已选定的数据库ID；3. 若均无，传0以触发参数补全。禁止凭空猜测。"`
	DbName string `json:"dbName" jsonschema_description:"数据库名称（可选）。取值逻辑：1. 用户本次明确指定；2. 从前序工具的输入输出中继承已选定的数据库名称；3. 留空则使用资产配置的默认库。禁止凭空猜测。"`
	SQL    string `json:"sql" jsonschema_description:"SQL语句" jsonschema:"required" `
	Remark string `json:"remark" jsonschema_description:"执行目的说明：简要描述为什么要执行这条SQL、预期达到什么效果（供用户审批与事后审计理解）" jsonschema:"required"`
}

type SqlExecOutput struct {
	DbId     int64  `json:"dbId" jsonschema_description:"数据库ID"`
	DbName   string `json:"dbName" jsonschema_description:"数据库名称"`
	DbType   string `json:"dbType" jsonschema_description:"数据库类型，如mysql、postgresql等"`
	Effected int64  `json:"effected" jsonschema_description:"影响的行数"`
}

func GetSqlExec() (tool.InvokableTool, error) {
	return utils.InferTool("ExecSql",
		i18n.T(imsg.ExecSqlToolInfo),
		func(ctx context.Context, param *SqlExecParam) (*SqlExecOutput, error) {
			toolDesc := i18n.TC(ctx, imsg.ExecSqlToolDesc)
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

			// 执行目的必填：供用户审批与事后审计理解（缺失时要求模型重试补充）
			if param.Remark == "" {
				return nil, tools.NewToolError(fmt.Errorf("remark parameter is required: describe the purpose of this SQL"), tools.RecoverRetry)
			}

			if err := tools.InterruptOrResumeApproval(ctx, toolDesc, param, i18n.TC(ctx, imsg.SqlExecApprovalReason)); err != nil {
				return nil, err
			}

			conn, err := application.GetDbApp().GetDbConn(ctx, uint64(param.DbId), param.DbName)
			if err != nil {
				return nil, tools.NewToolError(err, tools.RecoverRetry)
			}

			res, err := conn.ExecContext(ctx, param.SQL)
			if err != nil {
				return nil, tools.NewToolError(err, tools.RecoverRetry)
			}

			return &SqlExecOutput{
				DbId:     param.DbId,
				DbName:   param.DbName,
				DbType:   string(conn.Info.Type),
				Effected: res,
			}, nil
		},
	)
}
