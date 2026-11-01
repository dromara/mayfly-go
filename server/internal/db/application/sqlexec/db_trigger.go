package sqlexec

import (
	"context"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
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
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cast"
)

// selectKeywordPattern 判断写语句里是否含 SELECT（子查询/来源表），用于判定表引用可信度。
// 宁可对含字面量 "select" 的语句多拒一次（仅在配了 ACL 规则的库上），也不放过子查询绕过。
var selectKeywordPattern = regexp.MustCompile(`(?i)\bselect\b`)

// viewTableRefPattern 从视图定义里取 FROM/JOIN 后的表引用；括号开头（子查询）不匹配
// viewCacheTtl 视图清单缓存有效期：够短，新建视图很快就会生效；够长，翻页不会反复查系统表
const viewCacheTtl = 30 * time.Second

var viewTableRefPattern = regexp.MustCompile("(?is)\\b(?:from|join)\\s+([\\w.$`\"\\[\\]]+)")

// 语句类型候选值，与执行器分派保持一致：
// select 为查询语句，read 为解析失败或无法归类的只读语句，merge 为 upsert 写语句，other 为其余语句
// dbStmtTypes 策略「语句类型」可选值：来自解析器的语句注册表（单一真源），
// 新增语句类型只要在 base/sharedStatementSpecs 填好 PolicyType 即自动出现。
var dbStmtTypes = sqlparser.PolicyStatementTypes()

// 破坏性 DDL 关键字默认集
var dbDestructiveKeywords = []string{"drop", "truncate", "rename"}

// 触发策略字段 key
const (
	dbFieldStmtType     = "stmtType"
	dbFieldSQL          = "sql"
	dbFieldSQLHasWhere  = "sqlHasWhere"
	dbFieldTargetTables = "targetTables"
	dbFieldSQLLength    = "sqlLength"
	// dbFieldSelectStar SELECT 列表里用了 * （含 t.* 与 UNION 各分支）
	dbFieldSelectStar = "selectStar"
	// dbFieldDmlTableCount 写语句涉及的表数量（>1 即多表更新/删除）
	dbFieldDmlTableCount = "dmlTableCount"
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
	dbCheckSelectStar          = "db.select-star"
	dbCheckMultiTableDml       = "db.multi-table-dml"
	dbCheckTableInList         = "db.table-in-list"
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
		{Key: dbFieldSelectStar, TitleKey: "flow.field.selectStar", Group: "statement", Type: trigger.TypeBool},
		{Key: dbFieldDmlTableCount, TitleKey: "flow.field.dmlTableCount", Group: "statement", Type: trigger.TypeNumber},
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
		{
			Key: dbCheckSelectStar, TitleKey: "flow.check.selectStar", Summary: imsg.TriggerReasonSelectStar,
			DescriptionKey: "flow.checkDesc.selectStar",
			// 默认「仅提醒」：SELECT * 是性能与可维护性问题，不是权限边界。
			// 需要用它挡「大字段表全列拉取」时，管理员把它提级即可
			BizTypes: []string{DbSQLExecFlowBizType}, Default: flowentity.SeverityWarning,
			Evaluate: func(ctx context.Context, tc *trigger.Context, _ map[string]any) (bool, error) {
				stmtType, _ := tc.RawValue(dbFieldStmtType)
				if stmtType != string(sqlstmt.StmtTypeSelect) {
					return false, nil
				}
				// 事实缺失（只有关键字分类、没有 AST）时判不命中：这是风格规则，
				// 不能因为解析不出结构就拦人
				star, ok := tc.Attribute(dbFieldSelectStar)
				if !ok {
					return false, nil
				}
				matched, _ := star.(bool)
				return matched, nil
			},
		},
		{
			Key: dbCheckMultiTableDml, TitleKey: "flow.check.multiTableDml", Summary: imsg.TriggerReasonMultiTableDml,
			DescriptionKey: "flow.checkDesc.multiTableDml",
			// 多表更新/删除的旧值采集与回滚一律 fail-closed 跳过（无法按行定位），
			// 因此默认要求审批：让「改了什么、怎么退回去」在事前就有人看过
			BizTypes: []string{DbSQLExecFlowBizType}, Default: flowentity.SeverityRequired,
			Evaluate: func(ctx context.Context, tc *trigger.Context, _ map[string]any) (bool, error) {
				stmtType, _ := tc.RawValue(dbFieldStmtType)
				if stmtType != string(sqlstmt.StmtTypeUpdate) && stmtType != string(sqlstmt.StmtTypeDelete) {
					return false, nil
				}
				raw, ok := tc.Attribute(dbFieldDmlTableCount)
				if !ok {
					return false, nil
				}
				count, _ := raw.(int)
				return count > 1, nil
			},
		},
		{
			Key: dbCheckTableInList, TitleKey: "flow.check.tableInList", Summary: imsg.TriggerReasonTableInList,
			DescriptionKey: "flow.checkDesc.tableInList",
			// 把「指定表需审批」变成开箱可用的检查项：此前只能靠自定义条件手填目标表，
			// 而自定义条件不区分合规意图，配置期也看不出这条策略在保护什么
			BizTypes: []string{DbSQLExecFlowBizType}, Default: flowentity.SeverityRequired,
			Params: []trigger.CheckParam{{
				Key: "tables", TitleKey: "flow.checkParam.tables", Type: trigger.TypeStringList, Required: true,
			}},
			Evaluate: func(ctx context.Context, tc *trigger.Context, params map[string]any) (bool, error) {
				configured := trigger.ParamStrings(params, "tables")
				if len(configured) == 0 {
					// 名单为空 = 没有要保护的表，不命中（而不是拦全部）
					return false, nil
				}
				for _, target := range targetTablesOf(tc) {
					for _, want := range configured {
						// 表名大小写随方言与建表方式变化：不区分大小写比对，
						// 否则把表名写成大写就绕过了保护名单
						if strings.EqualFold(target, want) {
							return true, nil
						}
					}
				}
				return false, nil
			},
		},
	}
}

// dbTriggerPresets 常见意图的预置组合，用户一键套用而不用自己拼条件
func dbTriggerPresets() []trigger.PolicyPreset {
	return []trigger.PolicyPreset{
		{Key: "standard", TitleKey: "flow.policy.presetStandard", CheckKeys: []string{dbCheckDmlRequiresApproval}},
		{Key: "strict", TitleKey: "flow.policy.presetStrict", CheckKeys: []string{
			dbCheckDmlRequiresApproval, dbCheckDmlWithoutWhere, dbCheckDestructiveDdl,
			dbCheckMultiTableDml, dbCheckSelectStar,
		}},
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

// tableAclEnabled 该库是否配置了表访问控制规则
func (d *DbSQLExecAppImpl) tableAclEnabled(ctx context.Context, conn *dbi.DbConn) (bool, error) {
	if d.tableAcl == nil || conn == nil || conn.Info == nil {
		return false, nil
	}
	return d.tableAcl.HasRules(ctx, conn.Info.Id)
}

// assertTableAcl 表级权限闸门：语句的目标表命中保护名单即拒绝。
//
// 目标表取自与触发策略同一份事实（sqlTriggerFacts）：两处各判一套必然出现
// 「策略看到表 A、权限看不到」的偏差。未注入实现（单测直接构造）视为未配置规则。
func (d *DbSQLExecAppImpl) assertTableAcl(ctx context.Context, sqlExecParam *sqlExecParam) error {
	if d.tableAcl == nil || sqlExecParam == nil || sqlExecParam.DbConn == nil || sqlExecParam.DbConn.Info == nil {
		return nil
	}
	tables, resolvable := d.aclScopeFor(ctx, sqlExecParam.DbConn, sqlExecParam.SQL, sqlExecParam.Stmt)
	dbId := sqlExecParam.DbConn.Info.Id
	// 一条语句一次判定：可信度 + 表级 + 列级（列级与表级是两套独立规则，
	// 表没被整体挡住不等于列可以随便读）
	return d.tableAcl.AssertStatementAllowed(ctx, dbId, tables, resolvable, statementColumnRefs(sqlExecParam.Stmt))
}

// aclTableScope 从解析结果里取「表引用」并给出「这份清单是否可信」。
//
// 触发策略那侧的 targetTables 是给规则匹配用的粗口径，直接拿来做权限判定会漏人：
//   - 派生表在 AST 里只是名为 "(SELECT ...)" 的表引用，内层真实表根本不在清单里；
//   - INSERT…SELECT 只有写入目标，来源表缺失；
//   - 条件子查询（WHERE ... IN (SELECT ... FROM t_secret)）完全不进清单；
//   - 解析失败时 Stmt 为 nil，清单为空。
//
// 这些形态一律返回 resolvable=false，由 ACL 层决定「有规则就拒绝」。
// OTHER 类（SHOW / EXPLAIN / CALL）保持可信但无表清单：否则 SHOW TABLES 都会被挡死，
// 而存储过程内部的表引用本来就无法静态判定（记为已知边界，不靠猜）。
func aclTableScope(sql string, stmt sqlstmt.Stmt) (tables []string, resolvable bool) {
	if stmt == nil {
		return nil, false
	}
	tables, _ = sqlTriggerFacts(sql, stmt)[dbFieldTargetTables].([]string)
	for _, table := range tables {
		if strings.HasPrefix(strings.TrimSpace(table), "(") {
			return tables, false
		}
	}
	switch stmt.StmtKind() {
	case sqlstmt.KindSelect:
		// 条件子查询 / 标量子查询里的表**不在** targetTables 里（清单只收 FROM/JOIN/UNION）。
		// 语句里出现第二个 SELECT 就说明清单不完整：`select * from t_db where id in (select id from t_secret)`
		// 此前按「清单可信」放行，等于用一条 IN 子查询即可读到被保护表。
		if len(selectKeywordPattern.FindAllStringIndex(sql, -1)) > 1 {
			return tables, false
		}
		return tables, true
	case sqlstmt.KindInsert, sqlstmt.KindUpdate, sqlstmt.KindDelete, sqlstmt.KindMerge:
		// 写语句里出现 SELECT 就意味着存在来源/子查询表，而它们不在清单里
		if selectKeywordPattern.MatchString(sql) {
			return tables, false
		}
		return tables, true
	case sqlstmt.KindDdl:
		return tables, true
	case sqlstmt.KindWith:
		// CTE：内层真实表不一定出现在清单里
		return tables, false
	}
	// OTHER 类要再分一层：SHOW/EXPLAIN/SET 这类不读业务表，放行才能保住基本可用性；
	// CALL/EXECUTE 的过程体里访问哪些表在客户端无法判定——配了规则的库上按不可判定拒绝，
	// 否则存储过程就是一条稳定绕过保护名单的通道（此前正因如此被整体放行）。
	// EXPLAIN 是特殊形态：纯 EXPLAIN 只出计划（只读），但 EXPLAIN ANALYZE 会**真正执行**
	// 内层语句。其 StmtKind 也是 KindOther，若走下面的 EXPLAIN 白名单就会被当成只读放行，
	// 使 `EXPLAIN ANALYZE DELETE FROM t` 绕过表级 ACL。这里先拦：只要带 ANALYZE
	// （或已识别出内层写类型），清单不可信 → 有规则的库一律拒绝。
	if explain, ok := stmt.(*sqlstmt.ExplainStmt); ok && (explain.Analyze || explain.Inner != "") {
		return nil, false
	}
	if otherStmtReadsNoTable(sql) {
		return nil, true
	}
	return nil, false
}

// otherStmtReadsNoTable 判断 OTHER 类语句是否确定不访问业务表（导航/事务/会话类）。
//
// 采用**白名单**：只有列出的形态才算确定安全，新增形态默认按「不可判定」处理——
// 黑名单在这里是危险的，漏一个就等于漏一条绕过通道。
func otherStmtReadsNoTable(sql string) bool {
	// 白名单单一真源在 tokenizer：ACL 与语句分类必须共用同一份名单，
	// 否则会出现「分类说只读、ACL 说不可判定」的自相矛盾，或一处加词另一处漏加。
	return tokenizer.IsReadOnlyLeadingKeyword(leadingKeyword(sql))
}

// leadingKeyword 取语句首个关键字（跳过前导空白、行注释与块注释），统一大写。
func leadingKeyword(sql string) string {
	text := sql
	for {
		text = strings.TrimLeft(text, " \t\r\n")
		switch {
		case strings.HasPrefix(text, "--"), strings.HasPrefix(text, "#"):
			if idx := strings.IndexAny(text, "\r\n"); idx >= 0 {
				text = text[idx+1:]
				continue
			}
			return ""
		case strings.HasPrefix(text, "/*"):
			if idx := strings.Index(text, "*/"); idx >= 0 {
				text = text[idx+2:]
				continue
			}
			return ""
		default:
			end := strings.IndexAny(text, " \t\r\n(")
			if end < 0 {
				return strings.ToUpper(text)
			}
			return strings.ToUpper(text[:end])
		}
	}
}

// viewNames 取某库的视图名集合（短 TTL 缓存）。
//
// 缓存键是实例内的库 id：视图清单变化频率远低于查询频率，而 ACL 判定是逐语句的，
// 不缓存等于给每次查询加两次系统表往返。
func (d *DbSQLExecAppImpl) viewNames(ctx context.Context, conn *dbi.DbConn) (map[string]bool, error) {
	dbId := conn.Info.Id
	d.viewCacheMu.Lock()
	entry, ok := d.viewCache[dbId]
	d.viewCacheMu.Unlock()
	if ok && time.Since(entry.at) < viewCacheTtl {
		return entry.names, nil
	}
	views, err := conn.Metadata().ListObjects(ctx, "", dbi.KindView)
	if err != nil {
		return nil, err
	}
	names := make(map[string]bool, len(views))
	for _, view := range views {
		names[aclTableKey(view.Name)] = true
	}
	d.viewCacheMu.Lock()
	if d.viewCache == nil {
		d.viewCache = make(map[uint64]viewNamesEntry)
	}
	d.viewCache[dbId] = viewNamesEntry{names: names, at: time.Now()}
	d.viewCacheMu.Unlock()
	return names, nil
}

// aclScopeFor 语句的 ACL 作用域：表清单 + 「清单是否可信」。
//
// 断言（checkSQLTrigger）与结果裁剪（doQuery 星号兜底）必须用同一份作用域：
// 两处各自从 sqlTriggerFacts 取清单，会出现「判定时展开了视图、裁剪时没展开」的偏差，
// 表现为视图里的被保护列被原样返回。
func (d *DbSQLExecAppImpl) aclScopeFor(ctx context.Context, conn *dbi.DbConn, sql string, stmt sqlstmt.Stmt) ([]string, bool) {
	tables, resolvable := aclTableScope(sql, stmt)
	if d.tableAcl == nil || conn == nil || conn.Info == nil {
		return tables, resolvable
	}
	// 视图是解析盲区：语句里只有视图名。只有该库配了规则才付「列视图 + 取定义」的代价
	if hasRules, err := d.tableAcl.HasRules(ctx, conn.Info.Id); err == nil && hasRules {
		var viewResolvable bool
		tables, viewResolvable = d.expandAclViews(ctx, conn, tables)
		if !viewResolvable {
			resolvable = false
		}
	}
	return tables, resolvable
}

// expandAclViews 把目标表里的视图展开成其定义中引用的表（只展开一层）。
//
// 返回 (展开后的表清单, 是否可解析)。三种情况：
//   - 方言列不出视图（ErrUnsupportedKind 等）：保持原清单，视为可解析——不能因为平台读不到视图
//     元数据就把该库所有查询挡死；
//   - 目标表是视图且能取到定义：把定义里的表并进清单一起判定；
//   - 是视图但取不到定义/提取不出表：不可解析，由 ACL 层按「有规则即拒绝」处理。
func (d *DbSQLExecAppImpl) expandAclViews(ctx context.Context, conn *dbi.DbConn, tables []string) ([]string, bool) {
	if len(tables) == 0 {
		return tables, true
	}
	viewNames, err := d.viewNames(ctx, conn)
	if err != nil {
		// 列不出视图（方言不支持/权限不足）：保持原清单，不能因此把整库查询挡死
		return tables, true
	}
	expanded := append([]string{}, tables...)
	for _, table := range tables {
		if !viewNames[aclTableKey(table)] {
			continue
		}
		ddl, ddlErr := conn.Metadata().ObjectDDL(ctx, "", dbi.KindView, table)
		if ddlErr != nil {
			return tables, false
		}
		refs := viewDefinitionTables(ddl)
		if len(refs) == 0 {
			return tables, false
		}
		expanded = append(expanded, refs...)
	}
	return expanded, true
}

// viewDefinitionTables 从视图定义（CREATE VIEW ... AS SELECT ... FROM/JOIN ...）里取引用的表名。
//
// 只做「一层、纯文本关键字」提取：视图定义是数据库自己存的规范文本，不是用户输入；
// 提取不到就返回空，由上层判为不可解析（宁拒不放）。
func viewDefinitionTables(ddl string) []string {
	matches := viewTableRefPattern.FindAllStringSubmatch(ddl, -1)
	names := make([]string, 0, len(matches))
	seen := make(map[string]bool, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := aclTableKey(match[1])
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}

// checkSQLTrigger 按已绑定流程定义的触发策略判定该语句能否直接执行。
//
// 语句类型取自执行器已完成的分类，解析出的结构化事实一并透传，
// 避免在求值路径上重复解析，也保证分类口径与执行、审计一致
func (d *DbSQLExecAppImpl) checkSQLTrigger(ctx context.Context, sqlExecParam *sqlExecParam, stmtType string) error {
	// 表级权限先于触发策略：它是「这张表能不能碰」，不是「要不要人批」。
	// 放在这里而不是各入口自己判，是为了让编辑器 / 网格 / 脚本执行共用同一口径
	if err := d.assertTableAcl(ctx, sqlExecParam); err != nil {
		return err
	}
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
	// 表级权限与触发策略相互独立：未绑定流程定义也要判，否则「配了保护名单但没绑工单」
	// 的库上，导出 / 导入 / AI 三个入口会全部绕过表级权限
	aclEnabled, err := d.tableAclEnabled(ctx, conn)
	if err != nil {
		return nil, err
	}
	if procdef == nil && !aclEnabled {
		// 既无策略也无保护名单：不必为每条语句付解析开销
		return notices, nil
	}
	sp, splitter := dialect.GetSQLParser(), dialect.GetSQLSplitter()
	for i, sql := range sqls {
		stmt, parseErr := sp.Parse(sql)
		// 必须带上连接：没有库定位就没有「要查哪个库」，估算行数/主键/表规模这类事实一律算不出来，
		// 而按无法判定处理会让导入与 Agent 入口在配了相应策略的库上全部被拦
		sqlExec := &sqlExecParam{DbConn: conn, SQL: sql, Stmt: stmt, Procdef: procdef}
		if err := d.assertTableAcl(ctx, sqlExec); err != nil {
			return nil, err
		}
		if procdef == nil {
			continue
		}
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
		attributes[dbFieldDmlTableCount] = len(typed.Tables)
	case *sqlstmt.DeleteStmt:
		attributes["sqlHasWhere"] = typed.Where != nil
		attributes["targetTables"] = tableRefNames(typed.Tables)
		attributes[dbFieldDmlTableCount] = len(typed.Tables)
	case *sqlstmt.InsertStmt:
		attributes["targetTables"] = []string{typed.Table.Name}
		attributes[dbFieldDmlTableCount] = 1
	case *sqlstmt.SelectStmt:
		attributes["sqlHasWhere"] = typed.Where != nil
		attributes[dbFieldSelectStar] = selectHasStar(typed)
		// 查询语句同样要给出目标表：「只读语句不许碰客户表」「某些表必须走工单」这类策略
		// 对 SELECT 一样成立。此前只在 DML/DDL 分支解析，条件里的 targetTables
		// 对查询恒不命中，等于这类策略配了却完全不生效
		if tables := selectTargetTables(typed); len(tables) > 0 {
			attributes["targetTables"] = tables
		}
	case *sqlstmt.DdlStmt:
		// 目标表以 AST 的 DdlStmt.Target 为准（解析器在骨架层提取，单一真源）；
		// ddlTargetTables 的文本抽取只作为**兜底**保留：它额外覆盖 `RENAME TABLE a TO b` 这类
		// AST 尚未建模的改名形态，两处结果合并去重，不丢既有能力。
		tables := ddlTargetTables(sql)
		if typed.Target != "" && !slices.ContainsFunc(tables, func(t string) bool { return strings.EqualFold(t, typed.Target) }) {
			tables = append([]string{typed.Target}, tables...)
		}
		if len(tables) > 0 {
			attributes["targetTables"] = tables
		}
	case *sqlstmt.MergeStmt:
		// MERGE INTO <target>：目标表由 AST 给出，使「指定表」策略与表级访问控制覆盖 upsert
		if typed.Target != "" {
			attributes["targetTables"] = []string{typed.Target}
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

// selectHasStar 判断一条查询是否在结果列里用了 *（含 t.*）。
//
// 必须是「结构判定」而不是扫文本：字符串字面量里的 '*'（如 LIKE '%*%'）与
// COUNT(*) 里的星号都不是「取全部列」，按文本判会给出误报。
// UNION 的每个分支单独判：外层没有星号不代表分支里没有。
func selectHasStar(stmt *sqlstmt.SelectStmt) bool {
	if stmt == nil {
		return false
	}
	for _, item := range stmt.Items {
		if item.IsStar() {
			return true
		}
	}
	for _, union := range stmt.Unions {
		if selectHasStar(union.Select) {
			return true
		}
	}
	return false
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
