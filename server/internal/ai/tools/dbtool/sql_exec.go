package dbtool

import (
	"context"
	"fmt"
	"strings"

	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	flowapp "mayfly-go/internal/flow/application"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"

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

			conn, err := ensureDbConn(ctx, param.DbId, param.DbName)
			if err != nil {
				return nil, err
			}

			// 用户审批（始终需要，Agent 执行 SQL 的基本安全关卡）
			if err := tools.InterruptOrResumeApproval(ctx, toolDesc, param, i18n.TC(ctx, imsg.SqlExecApprovalReason)); err != nil {
				return nil, err
			}

			// 获取流程定义（用于检查管理员配置的 SQL 审批策略）
			procdef := flowapp.GetProcdefApp().GetProcdefByCodePath(ctx, conn.Info.CodePath...)

			// 使用方言切割器拆分多语句 SQL，逐条检查
			splitter := conn.GetDialect().GetSQLSplitter()
			var sqlStatements []string
			if splitErr := splitter.SplitSQL(strings.NewReader(param.SQL), func(s string) error {
				sqlStatements = append(sqlStatements, s)
				return nil
			}); splitErr != nil {
				return nil, tools.NewToolError(fmt.Errorf("SQL split failed: %w", splitErr), tools.RecoverRetry)
			}
			if len(sqlStatements) == 0 {
				return nil, tools.NewToolError(fmt.Errorf("no SQL statements to execute"), tools.RecoverRetry)
			}

			// 逐条解析 SQL，检查流程引擎是否额外要求审批（管理员配置的策略）
			// 注：用户已通过上方审批，此处仅用于策略合规性校验与日志记录
			sp := conn.GetDialect().GetSQLParser()
			for _, s := range sqlStatements {
				stmtType := ""
				if stmt, parseErr := sp.Parse(s); parseErr == nil && stmt != nil {
					switch stmt.(type) {
					case *sqlstmt.SelectStmt, *sqlstmt.WithStmt:
						stmtType = "select"
					case *sqlstmt.UpdateStmt:
						stmtType = "update"
					case *sqlstmt.DeleteStmt:
						stmtType = "delete"
					case *sqlstmt.InsertStmt:
						stmtType = "insert"
					case *sqlstmt.DdlStmt:
						stmtType = "ddl"
					case *sqlstmt.OtherStmt:
						stmtType = "read"
					default:
						stmtType = "other"
					}
				} else {
					// 解析失败，按关键字兜底分类
					kind := splitter.LeadingKeyword(s)
					if kind == "" && len(s) >= 10 {
						kind = strings.ToLower(s[:10])
					} else if kind == "" {
						kind = strings.ToLower(s)
					}
					switch {
					case strings.Contains(kind, "select"), strings.Contains(kind, "with"),
						strings.Contains(kind, "show"), strings.Contains(kind, "explain"):
						stmtType = "select"
					case strings.Contains(kind, "update"):
						stmtType = "update"
					case strings.Contains(kind, "delete"):
						stmtType = "delete"
					case strings.Contains(kind, "insert"):
						stmtType = "insert"
					case strings.Contains(kind, "create"), strings.Contains(kind, "alter"),
						strings.Contains(kind, "drop"), strings.Contains(kind, "truncate"),
						strings.Contains(kind, "rename"):
						stmtType = "ddl"
					default:
						stmtType = "other"
					}
				}

				// 记录流程引擎策略匹配结果（供审计日志参考）
				if procdef != nil && procdef.MatchCondition(application.DbSqlExecFlowBizType, collx.Kvs("stmtType", stmtType)) {
					logx.InfofContext(ctx, "[AgentSqlExec] flow engine requires approval for stmtType=%s, user approval already obtained", stmtType)
				}
			}

			// 逐条执行 SQL
			var totalEffected int64
			for _, s := range sqlStatements {
				res, execErr := conn.ExecContext(ctx, s)
				if execErr != nil {
					return nil, tools.NewToolError(execErr, tools.RecoverRetry)
				}
				totalEffected += res
			}

			// 审计日志：记录 SQL 执行操作（操作人、数据库、SQL 内容、执行目的），供事后审计追溯
			if la := contextx.GetLoginAccount(ctx); la != nil {
				logx.InfofContext(ctx, "[AgentSqlExec] operator=%s(%d), db=%s(%d), sql=%s, remark=%s",
					la.Username, la.Id, conn.Info.Name, param.DbId, param.SQL, param.Remark)
			}

			return &SqlExecOutput{
				DbId:     param.DbId,
				DbName:   param.DbName,
				DbType:   string(conn.Info.Type),
				Effected: totalEffected,
			}, nil
		},
	)
}
