package sqlexec

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
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cast"
)

// 语句类型候选值，与执行器分派保持一致：
// select 为查询语句，read 为解析失败或无法归类的只读语句，merge 为 upsert 写语句，other 为其余语句
var dbStmtTypes = []string{"select", "read", "insert", "update", "delete", "merge", "ddl", "other"}

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
	dbCheckRowsExceeds         = "db.affected-rows-exceeds"
	dbCheckFullScan            = "db.full-scan"
	dbCheckBigTableDdl         = "db.ddl-on-large-table"
	dbCheckNoPrimaryKey        = "db.no-primary-key"
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
		// 以下四个都要查库，全部做成惰性派生：条件没引用到就不会付这次 EXPLAIN/内省的开销，
		// 未绑定流程定义的库更是完全不进入这条路径（见 evaluateSQLTrigger 的 procdef 空判）
		{
			Key: dbFieldEstimatedRows, TitleKey: "flow.field.estimatedRows", Group: "risk", Type: trigger.TypeNumber,
			Resolve: func(tc *trigger.Context) (any, error) {
				res, err := explainOnce(tc)
				if err != nil {
					return nil, err
				}
				return res.Rows, nil
			},
		},
		{
			Key: dbFieldEstimateKind, TitleKey: "flow.field.estimateKind", Group: "risk", Type: trigger.TypeString,
			Resolve: func(tc *trigger.Context) (any, error) {
				res, err := explainOnce(tc)
				if err != nil {
					return nil, err
				}
				return res.EstimateKind, nil
			},
		},
		{
			Key: dbFieldFullScan, TitleKey: "flow.field.fullScan", Group: "risk", Type: trigger.TypeBool,
			Resolve: func(tc *trigger.Context) (any, error) {
				res, err := explainOnce(tc)
				if err != nil {
					return nil, err
				}
				return res.FullScan, nil
			},
		},
		{
			Key: dbFieldTargetTableNoPk, TitleKey: "flow.field.targetTableNoPk", Group: "risk", Type: trigger.TypeBool,
			Resolve: func(tc *trigger.Context) (any, error) {
				missing, err := noPrimaryKeyTables(tc.GoContext(), tc)
				if err != nil {
					return nil, err
				}
				return len(missing) > 0, nil
			},
		},
		{
			Key: dbFieldTargetTableRows, TitleKey: "flow.field.targetTableRows", Group: "target", Type: trigger.TypeNumber,
			Resolve: func(tc *trigger.Context) (any, error) {
				return maxTargetTableRows(tc.GoContext(), tc)
			},
		},
	}
}

// attachDbFacts 附加库定位信息，供上面这些要查库的派生字段按需取连接。
//
// 只放标量（dbId + 库名）而不放连接对象：事实集合会被写进日志与试算回显，
// 放连接会让序列化出问题，也让「谁能取消谁的执行」这类判定误以为拿着连接就等于有权限
func attachDbFacts(attributes map[string]any, dbConn *dbi.DbConn) {
	if dbConn == nil || dbConn.Info == nil {
		return
	}
	attributes[dbAttrDbId] = dbConn.Info.Id
	attributes[dbAttrDbName] = dbConn.Info.Database
}

// derivedNumber 取数值型派生字段。
//
// 取不到（方言不支持、连接不可用、字段未提供）一律上抛 error 而不是返回 0：
// 返回 0 会让「估算行数超阈值需审批」在算不出来的库上永远不命中，即配了规则却静默不生效
func derivedNumber(tc *trigger.Context, key string) (float64, error) {
	field, ok := trigger.FieldOf(DbSQLExecFlowBizType, key)
	if !ok {
		return 0, errorx.NewBizf("trigger field %s is not registered", key)
	}
	value, found, err := tc.FieldValue(field)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, errorx.NewBizf("trigger field %s has no value for this statement", key)
	}
	return cast.ToFloat64(value), nil
}

// derivedBool 取布尔型派生字段，语义同 derivedNumber：算不出来就是无法判定
func derivedBool(tc *trigger.Context, key string) (bool, error) {
	field, ok := trigger.FieldOf(DbSQLExecFlowBizType, key)
	if !ok {
		return false, errorx.NewBizf("trigger field %s is not registered", key)
	}
	value, found, err := tc.FieldValue(field)
	if err != nil {
		return false, err
	}
	if !found {
		return false, errorx.NewBizf("trigger field %s has no value for this statement", key)
	}
	return trigger.ParamBool(map[string]any{"value": value}, "value"), nil
}

// stmtWritesRows 语句是否会改动已有数据：只有这三类 DML 存在「影响行数」这个量。
//
// DDL 与查询没有影响行数：把按行数的规则套在 ALTER 上，要么根本取不到计划（mysql 的 EXPLAIN 不接受 DDL）
// 而按 fail-closed 报成高危，要么报个 0 行，两种结论都没信息量；结构变更的风险由
// ddl-on-large-table 承担，两条规则各管一段才不会互相抵消
func stmtWritesRows(stmtType string) bool {
	switch stmtType {
	case string(sqlstmt.StmtTypeInsert), string(sqlstmt.StmtTypeUpdate), string(sqlstmt.StmtTypeDelete):
		return true
	}
	return false
}

// stmtLocatesRows 语句是否需要在既有行上定位：只有 UPDATE/DELETE 存在「表没主键就无从按行回滚与审计」的问题，
// 往无主键表 INSERT 是建模问题，不该伪装成一条变更风险
func stmtLocatesRows(stmtType string) bool {
	return stmtType == string(sqlstmt.StmtTypeUpdate) || stmtType == string(sqlstmt.StmtTypeDelete)
}

// stmtHasPlan 语句能否拿到执行计划（与各方言 EXPLAIN 的支持范围一致）。
//
// 拿不到计划的语句（DDL、SHOW 等）不该因为「没有计划」被报一句提醒
func stmtHasPlan(stmtType string) bool {
	return stmtWritesRows(stmtType) || stmtType == string(sqlstmt.StmtTypeSelect)
}

func dbTriggerChecks() []trigger.CheckDef {
	return []trigger.CheckDef{
		{
			Key: dbCheckDmlRequiresApproval, TitleKey: "flow.check.dmlRequiresApproval", Summary: imsg.TriggerReasonDmlRequiresApproval,
			BizTypes: []string{DbSQLExecFlowBizType}, Default: flowentity.SeverityRequired,
			Params: []trigger.CheckParam{{
				Key: "stmtTypes", TitleKey: "flow.checkParam.stmtTypes", Type: trigger.TypeEnum, Required: true,
				Default: []string{"insert", "update", "delete", "merge", "ddl"},
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
		{
			Key: dbCheckRowsExceeds, TitleKey: "flow.check.affectedRowsExceeds", Summary: imsg.TriggerReasonRowsExceed,
			DescriptionKey: "flow.checkDesc.affectedRowsExceeds",
			BizTypes:       []string{DbSQLExecFlowBizType}, Default: flowentity.SeverityRequired,
			Params: []trigger.CheckParam{{
				Key: "maxRows", TitleKey: "flow.checkParam.maxRows", Type: trigger.TypeNumber, Required: true,
				Default: float64(100000), Min: ptrFloat(1),
			}},
			// 估算取不到时按命中处理（fail-closed）：这条规则的用途是「大改动必须有人看过」，
			// 算不出来就放行等于把最坏的情况留给自动通过
			Evaluate: func(ctx context.Context, tc *trigger.Context, params map[string]any) (bool, error) {
				stmtType, _ := tc.RawValue(dbFieldStmtType)
				if !stmtWritesRows(stmtType) {
					return false, nil
				}
				rows, err := derivedNumber(tc, dbFieldEstimatedRows)
				if err != nil {
					logx.WarnfContext(ctx, "db trigger: affected rows is unavailable, treated as hit: %s", err.Error())
					return true, nil
				}
				return rows > trigger.ParamFloat(params, "maxRows", 100000), nil
			},
		},
		{
			Key: dbCheckFullScan, TitleKey: "flow.check.fullScan", Summary: imsg.TriggerReasonFullScan,
			DescriptionKey: "flow.checkDesc.fullScan",
			// 默认「仅提醒」：这是性能提示，不是权限边界；且拿不到计划时选择不打扰用户
			// （安全兜底已由 affected-rows-exceeds 的 fail-closed 承担，两个检查各管一段才不会互相抵消）
			BizTypes: []string{DbSQLExecFlowBizType}, Default: flowentity.SeverityWarning,
			Evaluate: func(ctx context.Context, tc *trigger.Context, _ map[string]any) (bool, error) {
				stmtType, _ := tc.RawValue(dbFieldStmtType)
				if !stmtHasPlan(stmtType) {
					return false, nil
				}
				fullScan, err := derivedBool(tc, dbFieldFullScan)
				if err != nil {
					logx.WarnfContext(ctx, "db trigger: the execution plan is unavailable, skip the full-scan check: %s", err.Error())
					return false, nil
				}
				return fullScan, nil
			},
		},
		{
			Key: dbCheckBigTableDdl, TitleKey: "flow.check.ddlOnLargeTable", Summary: imsg.TriggerReasonBigTableDdl,
			DescriptionKey: "flow.checkDesc.ddlOnLargeTable",
			BizTypes:       []string{DbSQLExecFlowBizType}, Default: flowentity.SeverityRequired,
			Params: []trigger.CheckParam{{
				Key: "minRows", TitleKey: "flow.checkParam.minRows", Type: trigger.TypeNumber, Required: true,
				Default: float64(1000000), Min: ptrFloat(1),
			}},
			Evaluate: func(ctx context.Context, tc *trigger.Context, params map[string]any) (bool, error) {
				stmtType, _ := tc.RawValue(dbFieldStmtType)
				if stmtType != string(sqlstmt.StmtTypeDDL) {
					return false, nil
				}
				rows, err := derivedNumber(tc, dbFieldTargetTableRows)
				if err != nil {
					logx.WarnfContext(ctx, "db trigger: table statistics are unavailable, treated as hit: %s", err.Error())
					return true, nil
				}
				return rows >= trigger.ParamFloat(params, "minRows", 1000000), nil
			},
		},
		{
			Key: dbCheckNoPrimaryKey, TitleKey: "flow.check.noPrimaryKey", Summary: imsg.TriggerReasonNoPrimaryKey,
			DescriptionKey: "flow.checkDesc.noPrimaryKey",
			// 无主键表上 UPDATE/DELETE 无法按行定位（回滚与审计都失去依据），默认需审批
			BizTypes: []string{DbSQLExecFlowBizType}, Default: flowentity.SeverityRequired,
			Evaluate: func(ctx context.Context, tc *trigger.Context, _ map[string]any) (bool, error) {
				stmtType, _ := tc.RawValue(dbFieldStmtType)
				if !stmtLocatesRows(stmtType) {
					return false, nil
				}
				noPk, err := derivedBool(tc, dbFieldTargetTableNoPk)
				if err != nil {
					logx.WarnfContext(ctx, "db trigger: primary key information is unavailable, treated as hit: %s", err.Error())
					return true, nil
				}
				return noPk, nil
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
func (d *DbSQLExecAppImpl) checkSQLTrigger(ctx context.Context, sqlExecParam *sqlExecParam, stmtType string) error {
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
func (d *DbSQLExecAppImpl) evaluateSQLTrigger(ctx context.Context, sqlExecParam *sqlExecParam, stmtType string) (*trigger.Decision, error) {
	procdef := sqlExecParam.Procdef
	if procdef == nil {
		return nil, nil
	}
	return d.flowProcdefApp.CheckProcdefTrigger(ctx, procdef, &flowdto.TriggerRequest{
		BizType:    DbSQLExecFlowBizType,
		Raw:        map[string]string{dbFieldStmtType: stmtType, dbFieldSQL: sqlExecParam.SQL},
		Attributes: triggerAttributes(sqlExecParam),
	})
}

// triggerAttributes 构造本次求值的结构化事实：纯解析事实 + 库定位（供要查库的派生字段按需取用）
func triggerAttributes(sqlExecParam *sqlExecParam) map[string]any {
	attributes := sqlTriggerFacts(sqlExecParam.SQL, sqlExecParam.Stmt)
	attachDbFacts(attributes, sqlExecParam.DbConn)
	return attributes
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
func (d *DbSQLExecAppImpl) CheckSqlsWithoutTicket(ctx context.Context, conn *dbi.DbConn, sqls []string) ([]string, error) {
	return d.checkSqlsWithoutTicket(ctx, conn, conn.Info.CodePath, conn.GetDialect(), sqls)
}

// checkSqlsWithoutTicket 判定本体：把「从连接取信息」与「判定」分开，前者是 I/O、后者可单测
func (d *DbSQLExecAppImpl) checkSqlsWithoutTicket(ctx context.Context, conn *dbi.DbConn, codePaths []string, dialect dbi.Dialect, sqls []string) ([]string, error) {
	notices := make([]string, len(sqls))
	procdef, err := d.flowProcdefApp.GetProcdefByCodePath(ctx, codePaths...)
	if err != nil {
		// 定位失败按「不可判定」拒绝整批，不能当成未绑定
		return nil, err
	}
	if procdef == nil {
		// 未绑定流程定义即无需治理，也不必为每条语句付解析开销
		return notices, nil
	}
	sp, splitter := dialect.GetSQLParser(), dialect.GetSQLSplitter()
	for i, sql := range sqls {
		stmt, parseErr := sp.Parse(sql)
		// 必须带上连接：没有库定位就没有「要查哪个库」，估算行数/主键/表规模这类事实一律算不出来，
		// 而按无法判定处理会让导入与 Agent 入口在配了相应策略的库上全部被拦
		sqlExec := &sqlExecParam{DbConn: conn, SQL: sql, Stmt: stmt, Procdef: procdef}
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
	case *sqlstmt.DdlStmt:
		// DDL 的目标表由语句文本取：骨架解析只把语句包装成「这是一条 DDL」（列定义与约束不是
		// 触发策略要的信息，为它扩展四个方言的解析器不划算）。没有表名就没有「大表结构变更」
		// 也没有「指定表的结构变更需审批」，扫不出来时留空，下游按无法判定处理（fail-closed）
		if tables := ddlTargetTables(sql); len(tables) > 0 {
			attributes["targetTables"] = tables
		}
	}
	return attributes
}

// 被改动表名在 DDL 里的三种出现形态（详细取舍见 ddlTargetTables 注释）
var (
	// ddlTableRe TABLE/TRUNCATE 后的第一个标识符，容忍 TRUNCATE TABLE 的可选 TABLE 与 IF [NOT] EXISTS
	ddlTableRe = regexp.MustCompile(`(?i)\b(?:TABLE|TRUNCATE(?:\s+TABLE)?)\s+(?:IF\s+(?:NOT\s+)?EXISTS\s+)?([^\s(,;]+)`)
	// ddlIndexRe 只有 CREATE/DROP INDEX 的表名出现在 ON 之后；不先确认这个形态就扫 ON，
	// 外键约束里的 `REFERENCES users (id) ON DELETE CASCADE` 会把 DELETE 当成表名
	ddlIndexRe   = regexp.MustCompile(`(?i)\b(?:CREATE|DROP)\s+(?:UNIQUE\s+|FULLTEXT\s+|SPATIAL\s+)?(?:INDEX|KEY)\b`)
	ddlIndexOnRe = regexp.MustCompile(`(?i)\bON\s+([^\s(,;]+)`)
	// ddlRenameTableRe（MySQL 的 RENAME TABLE a TO b）与 renameToRe（ALTER TABLE t RENAME TO t2）：
	// 两者表名都在 TO 之后。单凭语句里含 RENAME 就扫 TO 会把 `RENAME COLUMN a TO b` 的列名 b 当成表
	ddlRenameTableRe = regexp.MustCompile(`(?i)\bRENAME\s+(?:IF\s+EXISTS\s+)?TABLE\b`)
	ddlRenameToRe    = regexp.MustCompile(`(?i)\bRENAME\s+TO\b`)
	ddlToNameRe      = regexp.MustCompile(`(?i)\bTO\s+([^\s(,;]+)`)
)

// ddlQuoteReplacer 标识符包裹符：` " [] 在各方言里只是引用差异，策略按表匹配与内省都用裸名
var ddlQuoteReplacer = strings.NewReplacer("`", "", "\"", "", "[", "", "]", "")

// ddlTargetTables 从 DDL 文本里取出被改动的表名。
//
// 宁少不多：误收一张无关的表会让「按表配置」的策略在错误的表上命中，比扫不出来更难排查。
// 因此只认三个固定形态，`RENAME TABLE a TO b, c TO d` 这类多对改名只能收到部分表名（已记录的
// 风险：此时表规模按已知表判定，而改名本身属于破坏性 DDL，由 destructive-ddl 规则覆盖）
func ddlTargetTables(sql string) []string {
	names := make([]string, 0, 2)
	seen := make(map[string]bool)
	collect := func(re *regexp.Regexp) {
		for _, sub := range re.FindAllStringSubmatch(sql, -1) {
			if len(sub) < 2 {
				continue
			}
			name := ddlQuoteReplacer.Replace(sub[1])
			if name == "" {
				continue
			}
			key := strings.ToLower(name)
			if seen[key] {
				continue
			}
			seen[key] = true
			names = append(names, name)
		}
	}

	collect(ddlTableRe)
	if ddlIndexRe.MatchString(sql) {
		collect(ddlIndexOnRe)
	}
	if ddlRenameTableRe.MatchString(sql) || ddlRenameToRe.MatchString(sql) {
		collect(ddlToNameRe)
	}
	return names
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

	dbConn, connErr := ioc.Get[DbConnReader]().GetDbConn(ctx, dbId, raw["db"])
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
		Raw: map[string]string{"sql": sql, "stmtType": string(stmtType)},
		Attributes: func() map[string]any {
			attributes := sqlTriggerFacts(sql, stmt)
			attachDbFacts(attributes, dbConn)
			return attributes
		}(),
	}, nil
}
