package dbtool

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/mask"
	"mayfly-go/pkg/i18n"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// explainAnalyzeRe 匹配 EXPLAIN ANALYZE/ANALYSE：ANALYZE 会真正执行被解释的语句（可能是写），只读通道不得放行
var explainAnalyzeRe = regexp.MustCompile(`(?i)\banalyz[es]\b`)

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
			tools.TryApplyResumedParams(ctx, param)
			// 检查必要参数，触发参数完善（dbName 可选，留空使用资产默认库）
			if param.DbId == 0 {
				if err := tools.InterruptOrResumeParamCompletion(ctx, "DbQueryData", param, i18n.TC(ctx, imsg.DbInfoIncomplete), "db", []tools.CompletionParamInfo{
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

			conn, err := ensureDbConn(ctx, param.DbId, param.DbName)
			if err != nil {
				return nil, err
			}

			parser := conn.GetDialect().GetSQLParser()
			splitter := conn.GetDialect().GetSQLSplitter()

			results := make([]SingleQueryResult, 0, len(param.SQLs))
			// 先做只读分类：非只读语句直接拒绝并回指 ExecSql，只读语句收集后统一过策略再执行
			readOnlySqls := make([]string, 0, len(param.SQLs))
			for _, sql := range param.SQLs {
				if sql == "" {
					continue
				}
				// 判定与执行都必须以「拆分后的单条语句」为单位：Classify 只解析到首个 ';'，
				// 若对整串判定，`SELECT 1; DELETE FROM t` 会按首句判只读、而 WalkQueryRows 执行整串
				// （PG/SQLite 等驱动会把分号后的写语句一并执行）——只读通道藏写，生产级漏洞
				stmts, splitErr := splitStatements(splitter, sql)
				if splitErr != nil {
					results = append(results, SingleQueryResult{SQL: sql, Error: fmt.Sprintf("SQL split failed: %v", splitErr)})
					continue
				}
				nonReadOnly := false
				for _, single := range stmts {
					if !isReadOnlyStatement(parser, splitter, single) {
						nonReadOnly = true
						break
					}
				}
				if nonReadOnly {
					results = append(results, SingleQueryResult{
						SQL:   sql,
						Error: "QueryData tool only allows read-only queries (SELECT/WITH/SHOW/EXPLAIN/DESC). Use ExecSql tool for data modification operations, which requires user approval.",
					})
					continue
				}
				readOnlySqls = append(readOnlySqls, stmts...)
			}

			// 触发策略与交互执行/ExecSql 同源：管理员对只读语句配置的规则对 Agent 同样生效（编辑器拦得住，Agent 也拦得住）；
			// 拒绝结论不可重试——退回给模型重试等于给它换写法绕过治理的机会。策略闸门置于任何执行之前
			if len(readOnlySqls) > 0 {
				if _, err := application.GetDbSQLExecApp().CheckSqlsWithoutTicket(ctx, conn, readOnlySqls); err != nil {
					return nil, tools.NewToolError(err, tools.RecoverNone)
				}
			}

			for _, sql := range readOnlySqls {
				result := SingleQueryResult{SQL: sql}
				rows := make([]map[string]any, 0)
				var rowMasker *mask.RowMasker
				columns, err := conn.WalkQueryRows(ctx, sql, func(row map[string]any, columns []*dbi.QueryColumn) error {
					rows = append(rows, row)
					if len(rows) > 1000 {
						return dbi.NewStopWalkQueryError("The maximum number of query rows is exceeded: 1000")
					}
					// AI查询结果同样执行服务端脱敏（首次行回调时构建脱敏器，未启用时为nil）
					if rowMasker == nil {
						var maskErr error
						rowMasker, maskErr = application.GetMaskEngine().BuildQueryRowMasker(ctx, conn, sql, columns)
						if maskErr != nil {
							// fail-close：脱敏计划不可用时阻断本次查询，避免敏感数据明文透出
							return maskErr
						}
					}
					if rowMasker != nil {
						rowMasker.MaskRow(row)
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

// isReadOnlyStatement 判定语句是否为「确认只读」：走 sqlparser.Classify 单一真源，只放行 StmtTypeSelect
// （select/with/table 等，后端走查询通道回结果集）。StmtTypeOther 是解析兜底大类，除只读元信息动词外
// 还混有 CALL/DO/SET/VACUUM 等带副作用语句，故仅按首词白名单放行 show/describe/desc 与非 ANALYZE 的 explain，
// 其余一律判非只读（导向 ExecSql 走审批），防止「以只读之名从这条无审批的查询通道执行写」。
// WITH 数据修改型 CTE（WITH x AS (DELETE..) SELECT）由 sqlparser 回填 WithStmt.MainType 判为写、
// 已从只读通道拦下（见 query_data_test）。仍不覆盖：SELECT 调副作用函数（lo_export/dblink_exec/
// nextval 等无法静态判定）、COPY TO/FROM——靠后端策略/工单/只读账号兜底。
func isReadOnlyStatement(parser sqlparser.SQLParser, splitter sqlparser.SQLSplitter, sql string) bool {
	stmt, parseErr := parser.Parse(sql)
	switch sqlparser.Classify(parser, splitter, sql, stmt, parseErr) {
	case sqlstmt.StmtTypeSelect:
		return true
	case sqlstmt.StmtTypeOther:
		switch splitter.LeadingKeyword(sql) {
		case "show", "describe", "desc":
			return true
		case "explain":
			// 纯 EXPLAIN 只出计划、不执行；EXPLAIN ANALYZE 会真正执行底层语句（可能是写），只读通道不得放行
			return !explainAnalyzeRe.MatchString(sql)
		}
	}
	return false
}

// splitStatements 用方言切割器把一段 SQL 拆成独立单语句。判定与执行都以单语句为单位，
// 杜绝「一条元素里塞多条、只判首句就整串执行」的多语句藏写旁路
func splitStatements(splitter sqlparser.SQLSplitter, sql string) ([]string, error) {
	var out []string
	err := splitter.SplitSQL(strings.NewReader(sql), func(single string) error {
		if strings.TrimSpace(single) != "" {
			out = append(out, single)
		}
		return nil
	})
	return out, err
}
