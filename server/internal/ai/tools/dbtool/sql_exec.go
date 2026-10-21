package dbtool

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cast"

	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/application/sqlexec"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type SQLExecParam struct {
	DbId   int64  `json:"dbId" jsonschema_description:"数据库ID。取值逻辑：1. 用户本次明确指定；2. 从前序工具的输入输出中继承已选定的数据库ID；3. 若均无，传0以触发参数补全。禁止凭空猜测。"`
	DbName string `json:"dbName" jsonschema_description:"数据库名称（可选）。取值逻辑：1. 用户本次明确指定；2. 从前序工具的输入输出中继承已选定的数据库名称；3. 留空则使用资产配置的默认库。禁止凭空猜测。"`
	SQL    string `json:"sql" jsonschema_description:"SQL语句" jsonschema:"required" `
	Remark string `json:"remark" jsonschema_description:"执行目的说明：简要描述为什么要执行这条SQL、预期达到什么效果（供用户审批与事后审计理解）" jsonschema:"required"`
}

type SQLExecOutput struct {
	DbId     int64  `json:"dbId" jsonschema_description:"数据库ID"`
	DbName   string `json:"dbName" jsonschema_description:"数据库名称"`
	DbType   string `json:"dbType" jsonschema_description:"数据库类型，如mysql、postgresql等"`
	Effected int64  `json:"effected" jsonschema_description:"影响的行数"`

	// Warnings 管理员策略给出的提醒（已渲染成当前语言）。不阻断执行，但必须让模型看见，
	// 否则它会向用户汇报「执行成功、一切正常」，而管理员配这一级别就是为了让人知道踩了哪条规则
	Warnings []string `json:"warnings,omitempty" jsonschema_description:"触发策略提醒：本次执行命中了管理员配置的提醒规则"`
}

// init 在包加载时声明入参形状（同 machinetool：必须早于任何一次许可下发）
func init() {
	tools.DeclareParamSchema(tools.ToolNameExecSql, SQLExecParam{})
}

func GetSQLExec() (tool.InvokableTool, error) {

	return utils.InferTool(tools.ToolNameExecSql,
		i18n.T(imsg.ExecSQLToolInfo),
		func(ctx context.Context, param *SQLExecParam) (*SQLExecOutput, error) {
			tools.TryApplyResumedParams(ctx, param)
			// 检查必要参数，触发参数完善（dbName 可选，留空使用资产默认库）
			if param.DbId == 0 {
				if err := tools.InterruptOrResumeParamCompletion(ctx, tools.ToolNameExecSql, param, i18n.TC(ctx, imsg.DbInfoIncomplete), "db", []tools.CompletionParamInfo{
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

			// 使用方言切割器拆分多语句 SQL，逐条判定后再执行
			splitter := conn.GetDialect().GetSQLSplitter()
			var sqlStatements []string
			if splitErr := splitter.SplitSQL(strings.NewReader(param.SQL), func(stmt string) error {
				sqlStatements = append(sqlStatements, stmt)
				return nil
			}); splitErr != nil {
				return nil, tools.NewToolError(fmt.Errorf("SQL split failed: %w", splitErr), tools.RecoverRetry)
			}
			if len(sqlStatements) == 0 {
				return nil, tools.NewToolError(fmt.Errorf("no SQL statements to execute"), tools.RecoverRetry)
			}

			// 原子批次判定：全部语句都是数据写（INSERT/UPDATE/DELETE）才整批走事务——
			// 全成功才提交、任一条失败整批回滚；混入 DDL（隐式提交、不可回滚）或只读语句
			// 则退化为逐条执行，不给「可回滚」的假承诺。判据经 sqlexec.CanAtomicBatch
			// 与交互链路 atomicClasses 同源，不得在 Agent 侧另造一份
			atomicBatch := decideAtomicBatch(conn.GetDialect().GetSQLParser(), splitter, sqlStatements)

			// 管理员配置的触发策略：判定与交互执行共用同一条链路（语句分类、事实抽取、话术都同源），
			// 否则会出现「编辑器里拦得住、交给 Agent 就过去了」。拒绝结论必须终止本次执行：
			// 用户在 Agent 里点的「允许执行」放行的只是这次工具调用，不能顶替策略要求的工单审批
			agentNotices, err := application.GetDbSQLExecApp().CheckSqlsWithoutTicket(ctx, conn, sqlStatements)
			if err != nil {
				// 治理结论不属于「可重试的失败」：把错误退回给模型重试，等于再给它一次绕过策略的机会
				return nil, tools.NewToolError(err, tools.RecoverNone)
			}

			// 用户审批（策略通过后才请求，避免「管理员本就禁止的语句先弹一次审批再失败」，与机器命令通道同口径）
			if err := tools.InterruptOrResumeApproval(ctx, tools.ToolNameExecSql, param, i18n.TC(ctx, imsg.SQLExecApprovalReason)); err != nil {
				return nil, err
			}

			// 复用交互执行链路 DbSQLExecApp.Exec 执行写语句：与编辑器同源，获得按类型正确路由
			// （doUpdate/doDelete 采集旧值）、DbSQLExec 执行记录（事后可审计、可生成回滚 SQL），
			// 以及整批原子事务（全数据写时 Atomic=true：全成功才提交、任一条失败整批回滚）
			execResults, execErr := application.GetDbSQLExecApp().Exec(ctx, &dto.DbSQLExecReq{
				DbId:   uint64(param.DbId),
				Db:     conn.Info.Name,
				SQL:    param.SQL,
				Remark: param.Remark,
				DbConn: conn,
				Atomic: atomicBatch,
			})
			if execErr != nil {
				return nil, tools.NewToolError(execErr, tools.RecoverRetry)
			}
			totalEffected, outcomeErr := agentExecOutcome(execResults)
			if outcomeErr != nil {
				return nil, tools.NewToolError(outcomeErr, tools.RecoverRetry)
			}

			// 审计日志（Exec 内部已落 DbSQLExec 记录，这里补一条 Agent 视角文本日志便于关联会话与操作人）
			if la := contextx.GetLoginAccount(ctx); la != nil {
				logx.InfofContext(ctx, "[AgentSqlExec] operator=%s(%d), db=%s(%d), sql=%s, remark=%s",
					la.Username, la.Id, conn.Info.Name, param.DbId, param.SQL, param.Remark)
			}

			return &SQLExecOutput{
				DbId:     param.DbId,
				DbName:   param.DbName,
				DbType:   string(conn.Info.Type),
				Effected: totalEffected,
				// 提醒不阻断执行，但模型得知道自己踩了哪条提醒，否则它下一句就会说「一切正常」
				Warnings: collx.ArrayFilter(agentNotices, func(notice string) bool { return notice != "" }),
			}, nil
		},
	)
}

// decideAtomicBatch 判定拆分后的整批语句能否走原子事务：每条都过 sqlexec.CanAtomicBatch
// （与交互链路 atomicClasses 同源）才可原子；混入 DDL/只读/兜底大类则退化逐条执行。
// 空批次不可原子（调用方已挡，这里兜底 false）
func decideAtomicBatch(parser sqlparser.SQLParser, splitter sqlparser.SQLSplitter, statements []string) bool {
	if len(statements) == 0 {
		return false
	}
	for _, single := range statements {
		stmt, parseErr := parser.Parse(single)
		if !sqlexec.CanAtomicBatch(sqlparser.Classify(parser, splitter, single, stmt, parseErr)) {
			return false
		}
	}
	return true
}

// agentExecOutcome 把交互执行链路的逐条结果折叠成 Agent 工具输出：任一条失败即整体判失败
// （绝不把部分成功当成功），累加写语句的 rowsAffected。抽成纯函数便于单测，不依赖真实连接
func agentExecOutcome(results []*dto.DbSQLExecRes) (int64, error) {
	var effected int64
	for _, r := range results {
		if r == nil {
			continue
		}
		if r.ErrorMsg != "" {
			return 0, fmt.Errorf("SQL %q failed: %s", stringx.Truncate(r.SQL, 120, 50, "..."), r.ErrorMsg)
		}
		if len(r.Res) > 0 {
			effected += cast.ToInt64(r.Res[0]["rowsAffected"])
		}
	}
	return effected, nil
}
