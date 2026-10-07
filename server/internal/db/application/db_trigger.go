package application

import (
	"context"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/imsg"
	flowapp "mayfly-go/internal/flow/application"
	flowdto "mayfly-go/internal/flow/application/dto"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
	flowimsg "mayfly-go/internal/flow/imsg"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"strconv"
)

// 语句类型候选值，与执行器分派保持一致：
// select 为查询语句，read 为解析失败或无法归类的只读语句，other 为其余语句
var dbStmtTypes = []string{"select", "read", "insert", "update", "delete", "ddl", "other"}

// 破坏性 DDL 关键字默认集
var dbDestructiveKeywords = []string{"drop", "truncate", "rename"}

// 触发策略字段 key
const (
	dbFieldStmtType     = "stmtType"
	dbFieldSQL          = "sql"
	dbFieldSQLHasWhere  = "sqlHasWhere"
	dbFieldTargetTables = "targetTables"
	dbFieldSQLLength    = "sqlLength"
)

// 内置检查项 key
const (
	dbCheckDmlRequiresApproval = "db.dml-requires-approval"
	dbCheckDmlWithoutWhere     = "db.dml-without-where"
	dbCheckDestructiveDdl      = "db.destructive-ddl"
	dbCheckSqlSizeExceeds      = "db.sql-size-exceeds"
)

// init 注册 DBMS 执行 SQL 场景的触发策略能力。
//
// 元数据在包加载期注册，不依赖 IOC 实例；试算解析内部按需取用应用服务
func init() {
	flowapp.RegisterTriggerBiz(trigger.BizMeta{
		BizType: DbSQLExecFlowBizType,
		// 治理的资源类型：保存校验据此判断兜底级别在这些资源上落不落得了地
		GovernPaths:    [][]int8{{consts.ResourceTypeDbInstance, consts.ResourceTypeAuthCert, consts.ResourceTypeDbName}},
		Approvable:     true,
		Fields:         dbTriggerFields(),
		Checks:         dbTriggerChecks(),
		Presets:        dbTriggerPresets(),
		SimulateFields: dbSimulateInputs(),
		Simulate:       simulateDbTriggerInput,
	})
}

func dbTriggerFields() []trigger.TriggerField {
	return []trigger.TriggerField{
		{
			Key: dbFieldStmtType, TitleKey: "flow.field.stmtType", Group: "statement", Type: trigger.TypeEnum,
			Options: collx.ArrayMap(dbStmtTypes, func(item string) trigger.FieldOption {
				return trigger.FieldOption{Value: item}
			}),
		},
		{Key: dbFieldSQL, TitleKey: "flow.field.sql", Group: "statement", Type: trigger.TypeString},
		{
			Key: dbFieldSQLLength, TitleKey: "flow.field.sqlLength", Group: "statement", Type: trigger.TypeNumber,
			Resolve: func(tc *trigger.Context) (any, error) {
				sql, _ := tc.RawValue(dbFieldSQL)
				return len(sql), nil
			},
		},
		{Key: dbFieldSQLHasWhere, TitleKey: "flow.field.sqlHasWhere", Group: "risk", Type: trigger.TypeBool},
		{Key: dbFieldTargetTables, TitleKey: "flow.field.targetTables", Group: "target", Type: trigger.TypeStringList},
	}
}

func dbTriggerChecks() []trigger.CheckDef {
	return []trigger.CheckDef{
		{
			Key: dbCheckDmlRequiresApproval, TitleKey: "flow.check.dmlRequiresApproval", Summary: imsg.TriggerReasonDmlRequiresApproval,
			BizTypes: []string{DbSQLExecFlowBizType}, Default: flowentity.SeverityRequired,
			Params: []trigger.CheckParam{{
				Key: "stmtTypes", TitleKey: "flow.checkParam.stmtTypes", Type: trigger.TypeEnum, Required: true,
				Default: []string{"insert", "update", "delete", "ddl"},
				Options: collx.ArrayMap(dbStmtTypes, func(item string) trigger.FieldOption {
					return trigger.FieldOption{Value: item}
				}),
			}},
			Evaluate: func(ctx context.Context, tc *trigger.Context, params map[string]any) (bool, error) {
				stmtType, _ := tc.RawValue(dbFieldStmtType)
				return trigger.OneOf(stmtType, trigger.ParamStrings(params, "stmtTypes")), nil
			},
		},
		{
			Key: dbCheckDmlWithoutWhere, TitleKey: "flow.check.dmlWithoutWhere", Summary: imsg.TriggerReasonDmlWithoutWhere,
			DescriptionKey: "flow.checkDesc.dmlWithoutWhere",
			BizTypes:       []string{DbSQLExecFlowBizType}, Default: flowentity.SeverityRequired,
			Evaluate: func(ctx context.Context, tc *trigger.Context, params map[string]any) (bool, error) {
				stmtType, _ := tc.RawValue(dbFieldStmtType)
				if stmtType != string(sqlstmt.StmtTypeUpdate) && stmtType != string(sqlstmt.StmtTypeDelete) {
					return false, nil
				}
				// 取值缺失说明调用方未提供解析结果，按存在无 WHERE 风险处理
				hasWhere, provided := tc.Attribute(dbFieldSQLHasWhere)
				return !provided || !trigger.ParamBool(map[string]any{"value": hasWhere}, "value"), nil
			},
		},
		{
			Key: dbCheckDestructiveDdl, TitleKey: "flow.check.destructiveDdl", Summary: imsg.TriggerReasonDestructiveDdl,
			BizTypes: []string{DbSQLExecFlowBizType}, Default: flowentity.SeverityRequired,
			Params: []trigger.CheckParam{{
				Key: "keywords", TitleKey: "flow.checkParam.keywords", Type: trigger.TypeStringList, Required: true,
				Default: dbDestructiveKeywords,
			}},
			Evaluate: func(ctx context.Context, tc *trigger.Context, params map[string]any) (bool, error) {
				stmtType, _ := tc.RawValue(dbFieldStmtType)
				if stmtType != string(sqlstmt.StmtTypeDDL) {
					return false, nil
				}
				sql, _ := tc.RawValue(dbFieldSQL)
				for _, keyword := range trigger.ParamStrings(params, "keywords") {
					if trigger.ContainsFold(sql, keyword) {
						return true, nil
					}
				}
				return false, nil
			},
		},
		{
			Key: dbCheckSqlSizeExceeds, TitleKey: "flow.check.sqlSizeExceeds", Summary: imsg.TriggerReasonSqlSize,
			DescriptionKey: "flow.checkDesc.sqlSizeExceeds",
			BizTypes:       []string{DbSQLExecFlowBizType}, Default: flowentity.SeverityWarning,
			Params: []trigger.CheckParam{{
				Key: "maxKb", TitleKey: "flow.checkParam.maxKb", Type: trigger.TypeNumber, Required: true,
				Default: float64(2048), Min: ptrFloat(1), Max: ptrFloat(1024 * 1024),
			}},
			Evaluate: func(ctx context.Context, tc *trigger.Context, params map[string]any) (bool, error) {
				sql, _ := tc.RawValue(dbFieldSQL)
				maxBytes := trigger.ParamFloat(params, "maxKb", 2048) * 1024
				return float64(len(sql)) > maxBytes, nil
			},
		},
	}
}

// dbTriggerPresets 常见意图的预置组合，用户一键套用而不用自己拼条件
func dbTriggerPresets() []trigger.PolicyPreset {
	return []trigger.PolicyPreset{
		{Key: "standard", TitleKey: "flow.policy.presetStandard", CheckKeys: []string{dbCheckDmlRequiresApproval}},
		{Key: "strict", TitleKey: "flow.policy.presetStrict", CheckKeys: []string{dbCheckDmlRequiresApproval, dbCheckDmlWithoutWhere, dbCheckDestructiveDdl}},
		{Key: "loose", TitleKey: "flow.policy.presetLoose", CheckKeys: []string{dbCheckDestructiveDdl}},
	}
}

// dbSimulateInputs 试算需要目标库与 SQL：语句分类依赖方言解析器，
// 不指定目标库就试算等于用另一套分类口径，结论可能与真实执行不一致
func dbSimulateInputs() []trigger.SimulateInput {
	return []trigger.SimulateInput{
		// 目标库让管理员从资源树里选：库里自增主键没人背得出来，手填 id 的表单等于不可用；
		// 选中后库名一并带出（NameKey），仍可改成同一资产下的其它库
		{Key: "dbId", TitleKey: "flow.simulate.dbId", Type: trigger.TypeNumber, EditorKey: "db-select", NameKey: "db", Required: true},
		{Key: "db", TitleKey: "flow.simulate.db", Type: trigger.TypeString},
		{Key: "sql", TitleKey: "flow.simulate.sql", Type: trigger.TypeString, PlaceholderKey: "flow.simulate.sqlPlaceholder", Required: true, Multiline: true},
	}
}

// ptrFloat 便于在参数定义里内联声明边界值
func ptrFloat(value float64) *float64 {
	return &value
}

// checkSQLTrigger 按已绑定流程定义的触发策略判定该语句能否直接执行。
//
// 语句类型取自执行器已完成的分类，解析出的结构化事实一并透传，
// 避免在求值路径上重复解析，也保证分类口径与执行、审计一致
func (d *dbSQLExecAppImpl) checkSQLTrigger(ctx context.Context, sqlExecParam *sqlExecParam, stmtType string) error {
	decision, err := d.evaluateSQLTrigger(ctx, sqlExecParam, stmtType)
	if err != nil || decision == nil {
		return err
	}
	// 阻断语义（含错误码与命中原因）由 flow 应用层统一给出，本模块只声明自己的话术
	if err := flowapp.NewBlockError(ctx, decision, dbBlockSpeech(sqlExecParam.RequireWarnAck), sqlExecParam.WarnAcknowledged); err != nil {
		return err
	}
	// 「仅提醒」不阻断执行，但结论必须回到操作者眼前，否则这一级别只进服务端日志等于没配
	sqlExecParam.Notices = append(sqlExecParam.Notices, policyNotices(decision)...)
	return nil
}

// evaluateSQLTrigger 求值这条 SQL 的触发策略结论；未绑定流程定义时返回 nil 表示无需治理
func (d *dbSQLExecAppImpl) evaluateSQLTrigger(ctx context.Context, sqlExecParam *sqlExecParam, stmtType string) (*trigger.Decision, error) {
	procdef := sqlExecParam.Procdef
	if procdef == nil {
		return nil, nil
	}
	return d.flowProcdefApp.CheckProcdefTrigger(ctx, procdef, &flowdto.TriggerRequest{
		BizType:    DbSQLExecFlowBizType,
		Raw:        map[string]string{dbFieldStmtType: stmtType, dbFieldSQL: sqlExecParam.SQL},
		Attributes: sqlTriggerFacts(sqlExecParam.SQL, sqlExecParam.Stmt),
	})
}

// dbBlockSpeech 数据库入口给阻断判定器的话术与能力声明，交互执行与代执行共用一份
func dbBlockSpeech(requireWarnAck bool) flowapp.BlockSpeech {
	return flowapp.BlockSpeech{
		Forbidden:      flowimsg.ErrOperationForbidden,
		Approval:       imsg.ErrNeedSubmitWorkTicket,
		RequireWarnAck: requireWarnAck,
	}
}

// CheckSqlsWithoutTicket 判定「拿不到工单通道」的入口（AI Agent、数据导入）能否执行这批 SQL，返回与入参等长的提醒数组（无提醒为空串）。
// 任何一条的结论是「需审批」或「禁止执行」都直接返回 error，调用方不得继续执行。
//
// 语句分类、事实抽取与话术都复用交互执行那条链路，不让调用方另判一套：两处口径不一致就会出现
// 「界面拦得住、Agent 放过去」这种从日志里都看不出原因的偏差。流程定义按连接解析一次，
// 多条语句不会退化成每条两趟库查询。
//
// 与交互执行只差两点：① 提醒不追问（Agent 现场没有能点确认框的人，只回显）；
// ② 这里拿不到工单通道，因此「需审批」结论即拒绝——用户在 Agent 里点的「允许本次执行」
// 放行的只是这次工具调用，不等于管理员策略要求的工单审批，两者不能互相顶替
func (d *dbSQLExecAppImpl) CheckSqlsWithoutTicket(ctx context.Context, conn *dbi.DbConn, sqls []string) ([]string, error) {
	return d.checkSqlsWithoutTicket(ctx, conn.Info.CodePath, conn.GetDialect(), sqls)
}

// checkSqlsWithoutTicket 判定本体：把「从连接取信息」与「判定」分开，前者是 I/O、后者可单测
func (d *dbSQLExecAppImpl) checkSqlsWithoutTicket(ctx context.Context, codePaths []string, dialect dbi.Dialect, sqls []string) ([]string, error) {
	notices := make([]string, len(sqls))
	procdef := d.flowProcdefApp.GetProcdefByCodePath(ctx, codePaths...)
	if procdef == nil {
		// 未绑定流程定义即无需治理，也不必为每条语句付解析开销
		return notices, nil
	}
	sp, splitter := dialect.GetSQLParser(), dialect.GetSQLSplitter()
	for i, sql := range sqls {
		stmt, parseErr := sp.Parse(sql)
		sqlExec := &sqlExecParam{SQL: sql, Stmt: stmt, Procdef: procdef}
		// 分类走 sqlparser.Classify：与交互执行、审计同一口径
		decision, err := d.evaluateSQLTrigger(ctx, sqlExec, string(sqlparser.Classify(sp, splitter, sql, stmt, parseErr)))
		if err != nil {
			return nil, err
		}
		if err := flowapp.NewBlockError(ctx, decision, dbBlockSpeech(false), false); err != nil {
			return nil, err
		}
		notices[i] = flowapp.WarnNotice(ctx, decision)
	}
	return notices, nil
}

// policyNotices 把不阻断的策略结论转成可随执行结果回传的提醒。
// 标题是前端 i18n key，后端不下发文案
func policyNotices(decision *trigger.Decision) []*dto.PolicyNotice {
	notices := decision.Notices()
	result := make([]*dto.PolicyNotice, 0, len(notices))
	for _, notice := range notices {
		result = append(result, &dto.PolicyNotice{Title: notice.Title, Detail: notice.Detail})
	}
	return result
}

// sqlTriggerFacts 由语句文本与解析出的 AST 计算触发策略所需的结构化事实。
//
// 真实执行路径与策略试算共用本函数，避免出现「试算说会拦、实际没拦」的判定漂移
func sqlTriggerFacts(sql string, stmt sqlstmt.Stmt) map[string]any {
	attributes := map[string]any{"sqlLength": len(sql)}
	switch typed := stmt.(type) {
	case *sqlstmt.UpdateStmt:
		attributes["sqlHasWhere"] = typed.Where != nil
		attributes["targetTables"] = tableRefNames(typed.Tables)
	case *sqlstmt.DeleteStmt:
		attributes["sqlHasWhere"] = typed.Where != nil
		attributes["targetTables"] = tableRefNames(typed.Tables)
	case *sqlstmt.InsertStmt:
		attributes["targetTables"] = []string{typed.Table.Name}
	case *sqlstmt.SelectStmt:
		attributes["sqlHasWhere"] = typed.Where != nil
		// 查询语句同样要给出目标表：「只读语句不许碰客户表」「某些表必须走工单」这类策略
		// 对 SELECT 一样成立。此前只在 DML/DDL 分支解析，条件里的 targetTables
		// 对查询恒不命中，等于这类策略配了却完全不生效
		if tables := selectTargetTables(typed); len(tables) > 0 {
			attributes["targetTables"] = tables
		}
	}
	return attributes
}

// selectTargetTables 收集一条查询真正读到的表：FROM、JOIN，以及 UNION 系列的每个子查询。
// 子查询里的表同样算，因为「碰了哪张表」的判定不看它出现在哪一层
func selectTargetTables(stmt *sqlstmt.SelectStmt) []string {
	names := make([]string, 0, len(stmt.From)+len(stmt.Joins))
	seen := make(map[string]bool)
	collect := func(refs ...sqlstmt.TableRef) {
		for _, name := range tableRefNames(refs) {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}

	collect(stmt.From...)
	for _, join := range stmt.Joins {
		collect(join.Table)
	}
	for _, union := range stmt.Unions {
		if union.Select != nil {
			for _, name := range selectTargetTables(union.Select) {
				collect(sqlstmt.TableRef{Name: name})
			}
		}
	}
	return names
}

func tableRefNames(tables []sqlstmt.TableRef) []string {
	names := make([]string, 0, len(tables))
	for _, table := range tables {
		if table.Name != "" {
			names = append(names, table.FullName())
		}
	}
	return names
}

// simulateDbTriggerInput 解析管理员粘贴的 SQL 用于策略试算。
//
// 语句分类依赖方言解析器，因此试算必须指定目标库；不猜测方言是为了不让试算结论与真实执行不一致
func simulateDbTriggerInput(ctx context.Context, raw map[string]string) (*trigger.SimulatedFacts, error) {
	sql := raw["sql"]
	if sql == "" {
		return nil, errorx.NewBizf("the simulated sql is required")
	}
	dbId, err := strconv.ParseUint(raw["dbId"], 10, 64)
	if err != nil || dbId == 0 {
		return nil, errorx.NewBizf("a target database is required to simulate a sql trigger policy")
	}

	dbConn, connErr := ioc.Get[Db]().GetDbConn(ctx, dbId, raw["db"])
	if connErr != nil {
		return nil, connErr
	}

	parser := dbConn.GetDialect().GetSQLParser()
	stmt, parseErr := parser.Parse(sql)
	stmtType := sqlparser.Classify(parser, dbConn.GetDialect().GetSQLSplitter(), sql, stmt, parseErr)
	if parseErr != nil {
		logx.WarnfContext(ctx, "flow trigger simulate: failed to parse the simulated sql, classified by keyword: %s", parseErr.Error())
	}
	return &trigger.SimulatedFacts{
		Raw:        map[string]string{"sql": sql, "stmtType": string(stmtType)},
		Attributes: sqlTriggerFacts(sql, stmt),
	}, nil
}
