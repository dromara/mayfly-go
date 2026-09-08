package mask

import (
	"context"
	"strings"
	"sync"
	"time"

	"mayfly-go/internal/db/config"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	sqlbase "mayfly-go/internal/db/dbm/sqlparser/base"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"
	masksvc "mayfly-go/internal/db/domain/mask"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/imsg"
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/stringx"
)

type MaskApp interface {
	base.App[*entity.DbMaskRule]

	// GetRulePageList 分页获取脱敏规则
	GetRulePageList(condition *entity.MaskRuleQuery, orderBy ...string) (*model.PageResult[*entity.DbMaskRule], error)

	// SaveRule 保存脱敏规则
	SaveRule(ctx context.Context, rule *entity.DbMaskRule) error

	// DeleteRule 删除脱敏规则
	DeleteRule(ctx context.Context, id uint64) error

	// GetTagPageList 分页获取列标签
	GetTagPageList(condition *entity.MaskColumnQuery, orderBy ...string) (*model.PageResult[*entity.DbMaskColumn], error)

	// GetTagById 根据id获取列标签
	GetTagById(id uint64) (*entity.DbMaskColumn, error)

	// SaveTag 保存列标签
	SaveTag(ctx context.Context, tag *entity.DbMaskColumn) error

	// DeleteTag 删除列标签
	DeleteTag(ctx context.Context, id uint64) error

	// BuildStmtRowMasker 根据解析后的stmt与查询结果列构建行脱敏器
	// 未启用脱敏/账号豁免/无规则命中时返回(nil, nil)；
	// 脱敏计划构建失败时按MaskFailClosed配置：阻断查询返回错误（fail-close）或降级为不脱敏（fail-open）
	BuildStmtRowMasker(ctx context.Context, dbConn *dbi.DbConn, stmt sqlstmt.Stmt, columns []*dbi.QueryColumn) (*masksvc.RowMasker, error)

	// BuildQueryRowMasker 根据sql与查询结果列构建行脱敏器（内部解析sql），供AI工具等未预解析场景使用
	// 错误语义同BuildStmtRowMasker
	BuildQueryRowMasker(ctx context.Context, dbConn *dbi.DbConn, sql string, columns []*dbi.QueryColumn) (*masksvc.RowMasker, error)
}

var _ MaskApp = (*MaskAppImpl)(nil)

type MaskAppImpl struct {
	base.AppImpl[*entity.DbMaskRule, repository.MaskRule]

	maskRuleRepo   repository.MaskRule   `inject:"T"`
	maskColumnRepo repository.MaskColumn `inject:"T"`
	roleApp        sysapp.Role           `inject:"T"`

	planCache planCache
}

// planCache 脱敏计划缓存：规则全局生效 + 列标签按实例生效
// 任一规则/列标签变更时主动失效，并设置TTL兜底（防直接改库绕过失效逻辑）
type planCache struct {
	mu      sync.RWMutex
	plans   map[uint64]*planCacheEntry
	exempts map[uint64]time.Time // 账号豁免判定缓存
	version int64                // 变更版本号
	ttl     time.Duration
}

type planCacheEntry struct {
	plan     *masksvc.Plan
	version  int64
	expireAt time.Time
}

const (
	planCacheTtl   = time.Minute
	exemptCacheTtl = 5 * time.Minute
)

func newPlanCache() planCache {
	return planCache{
		plans:   make(map[uint64]*planCacheEntry),
		exempts: make(map[uint64]time.Time),
		ttl:     planCacheTtl,
	}
}

// getPlan 获取指定实例的脱敏计划，优先读缓存。
// 构建失败时按MaskFailClosed配置：fail-close上抛错误阻断本次查询，
// fail-open（默认）降级为不脱敏，避免系统库抖动阻断正常查询
func (m *MaskAppImpl) getPlan(ctx context.Context, instanceId uint64) (*masksvc.Plan, error) {
	m.planCache.mu.RLock()
	entry, ok := m.planCache.plans[instanceId]
	version := m.planCache.version
	m.planCache.mu.RUnlock()

	if ok && entry.version == version && time.Now().Before(entry.expireAt) {
		return entry.plan, nil
	}

	plan, err := m.buildPlan(ctx, instanceId)
	if err != nil {
		if config.GetDbms().MaskFailClosed {
			return nil, errorx.NewBizf("mask plan build failed, query blocked by maskFailClosed: %s", err.Error())
		}
		logx.ErrorfContext(ctx, "build mask plan failed, masking disabled for instance %d: %s", instanceId, err.Error())
		return nil, nil
	}

	m.planCache.mu.Lock()
	// IOC以new()创建实例，planCache为零值，懒初始化防nil map写入panic
	if m.planCache.plans == nil {
		m.planCache.plans = make(map[uint64]*planCacheEntry)
	}
	m.planCache.plans[instanceId] = &planCacheEntry{plan: plan, version: version, expireAt: time.Now().Add(m.planCache.ttl)}
	m.planCache.mu.Unlock()
	return plan, nil
}

// invalidate 规则/列标签变更时使缓存失效
func (m *MaskAppImpl) invalidate() {
	m.planCache.mu.Lock()
	defer m.planCache.mu.Unlock()
	m.planCache.version++
	m.planCache.plans = make(map[uint64]*planCacheEntry)
	m.planCache.exempts = make(map[uint64]time.Time)
}

// buildPlan 加载规则与列标签并构建脱敏计划
func (m *MaskAppImpl) buildPlan(ctx context.Context, instanceId uint64) (*masksvc.Plan, error) {
	ruleEntities, err := m.maskRuleRepo.ListEnabled()
	if err != nil {
		return nil, err
	}
	tags, err := m.maskColumnRepo.ListByInstance(instanceId)
	if err != nil {
		return nil, err
	}

	// 权重降序排列，权重相同时按id升序保证稳定
	sortMaskRules(ruleEntities)

	rules := make([]*masksvc.Rule, 0, len(ruleEntities))
	for _, re := range ruleEntities {
		alg, err := masksvc.Get(re.Algorithm)
		if err != nil {
			// 算法未注册的规则跳过，不影响其他规则
			logx.ErrorfContext(ctx, "skip mask rule %s: %s", re.Name, err.Error())
			continue
		}
		params, err := parseMaskParams(re.Params)
		if err != nil {
			logx.ErrorfContext(ctx, "skip mask rule %s: %s", re.Name, err.Error())
			continue
		}
		rules = append(rules, &masksvc.Rule{
			Name:      re.Name,
			MatchType: masksvc.MatchType(re.MatchType),
			Pattern:   re.Pattern,
			Algorithm: alg,
			Params:    params,
		})
	}

	tagRules := make(map[uint64]*masksvc.Rule, 0)
	columnTags := make([]*masksvc.ColumnTag, 0, len(tags))
	for _, te := range tags {
		tag, err := m.toMaskColumnTag(ctx, te, tagRules)
		if err != nil {
			logx.ErrorfContext(ctx, "skip mask column tag %d: %s", te.Id, err.Error())
			continue
		}
		columnTags = append(columnTags, tag)
	}
	return masksvc.NewPlan(rules, columnTags)
}

// sortMaskRules 权重降序
func sortMaskRules(rules []*entity.DbMaskRule) {
	for i := 0; i < len(rules); i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[j].Weight > rules[i].Weight || (rules[j].Weight == rules[i].Weight && rules[j].Id < rules[i].Id) {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}
}

// toMaskColumnTag 实体转领域列标签，解析绑定的规则或直接指定的算法
func (m *MaskAppImpl) toMaskColumnTag(ctx context.Context, te *entity.DbMaskColumn, tagRules map[uint64]*masksvc.Rule) (*masksvc.ColumnTag, error) {
	tag := &masksvc.ColumnTag{
		DbName:     te.DbName,
		TableName:  te.MatchTable,
		ColumnName: te.ColumnName,
		Action:     masksvc.TagAction(te.Action),
	}
	if te.Action == entity.MaskColumnActionExempt {
		return tag, nil
	}

	// 解析算法：直接指定的算法优先，其次绑定规则
	params, err := parseMaskParams(te.Params)
	if err != nil {
		return nil, err
	}
	if te.Algorithm != "" {
		alg, err := masksvc.Get(te.Algorithm)
		if err != nil {
			return nil, err
		}
		tag.Algorithm = alg
		tag.Params = params
		return tag, nil
	}

	if te.RuleId > 0 {
		rule, ok := tagRules[te.RuleId]
		if !ok {
			re, err := m.maskRuleRepo.GetById(te.RuleId)
			if err != nil {
				return nil, err
			}
			alg, err := masksvc.Get(re.Algorithm)
			if err != nil {
				return nil, err
			}
			ruleParams, err := parseMaskParams(re.Params)
			if err != nil {
				return nil, err
			}
			rule = &masksvc.Rule{Name: re.Name, MatchType: masksvc.MatchType(re.MatchType), Pattern: re.Pattern, Algorithm: alg, Params: ruleParams}
			tagRules[te.RuleId] = rule
		}
		tag.Algorithm = rule.Algorithm
		// 标签自身参数优先于规则参数
		if len(params) > 0 {
			tag.Params = params
		} else {
			tag.Params = rule.Params
		}
		return tag, nil
	}

	return nil, errorx.NewBizf("mask column tag has neither algorithm nor rule")
}

// parseMaskParams 解析算法参数json
func parseMaskParams(paramsJson string) (masksvc.Params, error) {
	paramsJson = stringx.Trim(paramsJson)
	if paramsJson == "" {
		return masksvc.Params{}, nil
	}
	m, err := jsonx.ToMapByStr(paramsJson)
	if err != nil {
		return nil, errorx.NewBizf("invalid mask params json: %s", err.Error())
	}
	params := make(masksvc.Params, len(m))
	for k, v := range m {
		params[k] = v
	}
	return params, nil
}

// isExemptAccount 判断当前登录账号是否命中脱敏豁免角色（带缓存）
func (m *MaskAppImpl) isExemptAccount(ctx context.Context, accountId uint64) bool {
	exemptRoleIds := config.GetDbms().MaskExemptRoleIds
	if len(exemptRoleIds) == 0 || accountId == 0 {
		return false
	}

	m.planCache.mu.RLock()
	expireAt, ok := m.planCache.exempts[accountId]
	m.planCache.mu.RUnlock()
	if ok && time.Now().Before(expireAt) {
		return true
	}

	accountRoles, err := m.roleApp.GetAccountRoles(accountId)
	if err != nil {
		logx.ErrorfContext(ctx, "get account roles for mask exempt failed: %s", err.Error())
		return false
	}
	for _, ar := range accountRoles {
		if collx.ArrayContains(exemptRoleIds, ar.RoleId) {
			m.planCache.mu.Lock()
			if m.planCache.exempts == nil {
				m.planCache.exempts = make(map[uint64]time.Time)
			}
			m.planCache.exempts[accountId] = time.Now().Add(exemptCacheTtl)
			m.planCache.mu.Unlock()
			return true
		}
	}
	return false
}

// BuildStmtRowMasker 根据解析后的stmt与查询结果列构建行脱敏器
func (m *MaskAppImpl) BuildStmtRowMasker(ctx context.Context, dbConn *dbi.DbConn, stmt sqlstmt.Stmt, columns []*dbi.QueryColumn) (*masksvc.RowMasker, error) {
	if dbConn == nil || dbConn.Info == nil || len(columns) == 0 {
		return nil, nil
	}
	var selectStmt *sqlstmt.SelectStmt
	if s, ok := stmt.(*sqlstmt.SelectStmt); ok {
		selectStmt = s
	}
	return m.buildRowMasker(ctx, dbConn, selectStmt, columns)
}

// BuildQueryRowMasker 根据sql与查询结果列构建行脱敏器（内部解析sql）
func (m *MaskAppImpl) BuildQueryRowMasker(ctx context.Context, dbConn *dbi.DbConn, sql string, columns []*dbi.QueryColumn) (*masksvc.RowMasker, error) {
	if dbConn == nil || dbConn.Info == nil || len(columns) == 0 {
		return nil, nil
	}
	var selectStmt *sqlstmt.SelectStmt
	if stmt, err := dbConn.GetDialect().GetSQLParser().Parse(sql); err == nil {
		if s, ok := stmt.(*sqlstmt.SelectStmt); ok {
			selectStmt = s
		}
	}
	return m.buildRowMasker(ctx, dbConn, selectStmt, columns)
}

// buildRowMasker 构建行脱敏器核心逻辑
func (m *MaskAppImpl) buildRowMasker(ctx context.Context, dbConn *dbi.DbConn, selectStmt *sqlstmt.SelectStmt, columns []*dbi.QueryColumn) (*masksvc.RowMasker, error) {
	if !config.GetDbms().MaskEnabled {
		return nil, nil
	}

	// 账号命中豁免角色则不脱敏
	if account := contextx.GetLoginAccount(ctx); account != nil && m.isExemptAccount(ctx, account.Id) {
		return nil, nil
	}

	plan, err := m.getPlan(ctx, dbConn.Info.InstanceId)
	if err != nil {
		return nil, err
	}
	if plan == nil || plan.Empty() {
		return nil, nil
	}

	return buildRowMaskerFromPlan(plan, dbConn.Info.GetDatabase(), selectStmt, columns, dbConn.GetDialect().GetSQLParser()), nil
}

// maxMaskLineageDepth 派生表血缘递归解析的最大深度限制，防恶意深嵌套SQL过度解析
const maxMaskLineageDepth = 5

// queryLineage 一条SELECT语句的列级血缘：描述各结果列到源表/源列的映射关系
// （含派生表别名到其输出列血缘的映射，支持外层限定列跨层归属）
type queryLineage struct {
	refs         map[string]exprColRef            // 血缘条目：结果列key -> 源引用
	tables       []string                         // FROM/JOIN解析出的真实表名（含星号展开的通配来源表）
	aliasToTable map[string]string                // 表别名/表名(小写) -> 真实表名
	derivedOf    map[string]map[string]exprColRef // 派生表别名(小写) -> 该表输出列血缘
	derivedStar  map[string][]string              // 派生表别名(小写) -> 其星号展开的来源表列表（输出列=来源表同名列）
	starTables   []string                         // 本层星号展开的来源表列表（本层作为派生表被引用时供外层建立通配血缘）
}

// buildRowMaskerFromPlan 基于给定脱敏计划构建行脱敏器（纯函数，不依赖配置/账号/规则库，便于跨方言运行时验证）
func buildRowMaskerFromPlan(plan *masksvc.Plan, database string, selectStmt *sqlstmt.SelectStmt, columns []*dbi.QueryColumn, parser sqlparser.SqlParser) *masksvc.RowMasker {
	resultKeys := make([]string, 0, len(columns))
	for _, col := range columns {
		resultKeys = append(resultKeys, col.Key)
	}
	lin := buildQueryLineage(selectStmt, resultKeys, plan, database, parser, 0)
	if lin == nil {
		lin = &queryLineage{refs: map[string]exprColRef{}, aliasToTable: map[string]string{}, derivedOf: map[string]map[string]exprColRef{}, derivedStar: map[string][]string{}}
	}
	// 血缘条目转换为NewRowMasker所需映射
	srcColumnOf := make(map[string]string, len(lin.refs))
	tableOf := make(map[string]string)
	for key, ref := range lin.refs {
		srcColumnOf[key] = ref.colName
		if ref.locked {
			tableOf[key] = ref.table
		}
	}
	// 仅按名字兑底的模式下，重名列场景（行数据key是col.Key，连接查询重名列会被改名）别名映射补充按Name查找；
	// 位置对应模式下Key已直接映射，不做补充（避免表达式项被邻近同名列的错误归属）
	if selectStmt != nil && len(selectStmt.Items) != len(columns) {
		for _, col := range columns {
			if src, ok := srcColumnOf[col.Name]; ok {
				if _, exists := srcColumnOf[col.Key]; !exists {
					srcColumnOf[col.Key] = src
				}
			}
		}
	}

	return masksvc.NewRowMasker(plan, database, lin.tables, tableOf, resultKeys, srcColumnOf, func(colName string) {
		for _, col := range columns {
			if col.Key == colName || col.Name == colName {
				col.Masked = true
				return
			}
		}
	})
}

// buildQueryLineage 构建SELECT语句的列级血缘。
//   - resultKeys为期望结果列列表：其长度与select项数一致且无星号展开时按位置精确对应
//     （覆盖别名/无别名/限定名/重名列全部场景）；否则退化为按结果列名/别名映射兑底
//   - resultKeys为nil表示派生表内层血缘：按输出列名记录（含别名与纯列引用），
//     星号展开时内层输出列未知，整体放弃
//   - plan/database用于校验表达式项token引用是否命中脱敏计划（内层血缘传nil）
//   - parser用于递归解析派生表内层SQL以建立表级血缘，depth限制递归深度
func buildQueryLineage(sel *sqlstmt.SelectStmt, resultKeys []string, plan *masksvc.Plan, database string, parser sqlparser.SqlParser, depth int) *queryLineage {
	if sel == nil || depth > maxMaskLineageDepth {
		return nil
	}
	lin := &queryLineage{
		refs:         make(map[string]exprColRef),
		aliasToTable: make(map[string]string),
		derivedOf:    make(map[string]map[string]exprColRef),
		derivedStar:  make(map[string][]string),
	}
	// 数据源引用按出现顺序收集（纯*展开即按此顺序）
	sourceRefs := make([]sqlstmt.TableRef, 0, len(sel.From)+len(sel.Joins))
	addTableRef := func(ref sqlstmt.TableRef) {
		if ref.Name == "" {
			return
		}
		// 派生表（括号子查询文本）：递归解析内层SQL建立输出列血缘
		if strings.Contains(ref.Name, "(") {
			lin.addDerived(ref.Alias, ref.Name, parser, depth)
			return
		}
		table := stripTableName(ref.Name)
		lin.aliasToTable[strings.ToLower(table)] = table
		if ref.Alias != "" {
			lin.aliasToTable[strings.ToLower(stripTableName(ref.Alias))] = table
		}
		lin.tables = append(lin.tables, table)
	}
	for _, ref := range sel.From {
		sourceRefs = append(sourceRefs, ref)
		addTableRef(ref)
	}
	for _, join := range sel.Joins {
		sourceRefs = append(sourceRefs, join.Table)
		addTableRef(join.Table)
	}
	// 星号展开的来源表即输出列的潜在来源：记录为通配血缘（本层作为派生表时供外层归属），
	// 并入tables供无归属列按表顺序兜底（与直接JOIN多表同名列兜底语义一致）
	starTables := lin.collectStarSources(sel, sourceRefs)
	for _, t := range starTables {
		if !collx.ArrayContains(lin.tables, t) {
			lin.tables = append(lin.tables, t)
		}
	}
	if len(starTables) > 0 {
		lin.starTables = starTables
	}
	positional := resultKeys != nil && len(resultKeys) == len(sel.Items)
	for _, item := range sel.Items {
		if item.IsStar() {
			positional = false
			break
		}
	}
	if positional {
		// 先收集查询内已知列（列名 -> 源引用），供表达式项token匹配
		colRefs := make(map[string]exprColRef, len(sel.Items))
		for _, item := range sel.Items {
			if item.IsStar() || isExprItem(item) || item.ColumnName == "" {
				continue
			}
			ref := exprColRef{colName: item.ColumnName}
			if q := item.TableAlias; q != "" {
				lin.qualifyRef(&ref, q, item.ColumnName)
			}
			colRefs[strings.ToLower(item.ColumnName)] = ref
		}
		for i, item := range sel.Items {
			key := resultKeys[i]
			// 表达式/函数项（MAX(u.phone)/CONCAT(phone,..)/"PHONE"||'-x'等）：
			// 先提取限定名对(qualifier.col)精确定位来源（真实表或派生表血缘），
			// 再退化为裸标识符token匹配；仅当以(表,列)命中脱敏计划时才记录，
			// 函数名/关键字无命中不会误设；token提取复用base词法能力，字符串字面量/注释自动跳过
			if isExprItem(item) {
				lin.resolveExprItem(key, stripExprAliasSuffix(item.Text), colRefs, plan, database)
				continue
			}
			if item.ColumnName == "" {
				continue
			}
			ref := exprColRef{colName: item.ColumnName}
			// 限定名（u.phone/u.phone AS p）的表限定符确定唯一来源
			if q := item.TableAlias; q != "" {
				lin.qualifyRef(&ref, q, item.ColumnName)
			}
			lin.refs[key] = ref
		}
	} else if resultKeys == nil {
		// 派生表内层血缘（按输出列名记录）：带别名/纯列引用的项可建立血缘；
		// 星号展开项由collectStarSources记录的通配血缘覆盖（输出列=来源表同名列），不在此处理
		for _, item := range sel.Items {
			if item.IsStar() {
				continue
			}
			outCol, srcCol := "", ""
			switch {
			case item.Alias != "":
				outCol = stripTableName(item.Alias)
				if item.ColumnName != "" && !strings.ContainsAny(item.ColumnName, "( ") {
					srcCol = stripTableAlias(item.ColumnName)
				}
			case !isExprItem(item) && item.ColumnName != "" && !strings.ContainsAny(item.ColumnName, "( "):
				outCol, srcCol = item.ColumnName, item.ColumnName
			default:
				continue
			}
			if outCol == "" {
				continue
			}
			if srcCol == "" {
				// 别名的表达式项：从表达式文本解析源引用（内层无计划，仅接受可解析的限定名对）
				lin.resolveExprItem(outCol, stripExprAliasSuffix(item.Text), nil, nil, "")
				continue
			}
			ref := exprColRef{colName: srcCol, locked: true}
			if q := item.TableAlias; q != "" {
				lin.qualifyRef(&ref, q, srcCol)
			}
			// 数据源唯一（单一真实表或唯一单表星号通配的派生表）且无UNION时，无限定列也可精确归属
			if ref.table == "" {
				if t := lin.singleSourceTable(sel); t != "" {
					ref.table = t
				}
			}
			lin.refs[outCol] = ref
		}
	} else {
		// 外层按名字兑底：仅带别名的普通列按结果列名映射（星号展开等场景）
		for _, item := range sel.Items {
			if item.Alias == "" || item.ColumnName == "" || strings.ContainsAny(item.ColumnName, "( ") {
				continue
			}
			// 方言解析器输出的alias可能带引用符（如 "p"、`p`），需去除后才能与结果列名匹配
			lin.refs[stripTableName(item.Alias)] = exprColRef{colName: stripTableAlias(item.ColumnName)}
		}
	}
	return lin
}

// addDerived 递归解析派生表SQL，建立其别名到"输出列 -> 内层源表/源列"血缘的映射
func (l *queryLineage) addDerived(alias, rawText string, parser sqlparser.SqlParser, depth int) {
	if alias == "" || parser == nil {
		return
	}
	innerSql := extractParenInner(rawText)
	if innerSql == "" {
		return
	}
	stmt, err := parser.Parse(innerSql)
	if err != nil {
		return
	}
	sel, ok := stmt.(*sqlstmt.SelectStmt)
	if !ok {
		return
	}
	inner := buildQueryLineage(sel, nil, nil, "", parser, depth+1)
	if inner == nil {
		return
	}
	aliasKey := strings.ToLower(stripTableName(alias))
	if len(inner.refs) > 0 {
		l.derivedOf[aliasKey] = inner.refs
	}
	// 内层星号展开：输出列即来源表同名列，记录通配血缘供外层限定列/结果列归属
	if len(inner.starTables) > 0 {
		l.derivedStar[aliasKey] = inner.starTables
	}
}

// collectStarSources 收集select项星号展开涉及的来源表（去重、保持出现顺序）。
// 星号展开语义为"输出列=来源表同名列"，故来源表可作为通配血缘；
// t.*取该表别名归属（真实表或派生表通配来源），纯*展开全部数据源（含UNION各分支与派生表通配来源）
func (l *queryLineage) collectStarSources(sel *sqlstmt.SelectStmt, sourceRefs []sqlstmt.TableRef) []string {
	hasStar := false
	for _, item := range sel.Items {
		if item.IsStar() {
			hasStar = true
			break
		}
	}
	if !hasStar {
		return nil
	}
	tables := make([]string, 0)
	add := func(t string) {
		if t == "" || collx.ArrayContains(tables, t) {
			return
		}
		tables = append(tables, t)
	}
	addRefSources := func(refs []sqlstmt.TableRef) {
		for _, ref := range refs {
			if ref.Name == "" {
				continue
			}
			if strings.Contains(ref.Name, "(") {
				// 派生表数据源：展开其通配来源
				for _, t := range l.derivedStar[strings.ToLower(stripTableName(ref.Alias))] {
					add(t)
				}
				continue
			}
			add(stripTableName(ref.Name))
		}
	}
	for _, item := range sel.Items {
		if !item.IsStar() {
			continue
		}
		if ta := item.TableAlias; ta != "" {
			q := strings.ToLower(ta)
			if t, ok := l.aliasToTable[q]; ok {
				add(t)
				continue
			}
			for _, t := range l.derivedStar[q] {
				add(t)
			}
			continue
		}
		addRefSources(sourceRefs)
		for _, u := range sel.Unions {
			if u.Select == nil {
				continue
			}
			refs := make([]sqlstmt.TableRef, 0, len(u.Select.From)+len(u.Select.Joins))
			refs = append(refs, u.Select.From...)
			for _, j := range u.Select.Joins {
				refs = append(refs, j.Table)
			}
			addRefSources(refs)
		}
	}
	return tables
}

// singleSourceTable 若本层数据源唯一（单一真实表，或唯一且单表星号通配的派生表）且无UNION，返回该真实表名
func (l *queryLineage) singleSourceTable(sel *sqlstmt.SelectStmt) string {
	if len(sel.Unions) > 0 {
		return ""
	}
	if len(l.tables)+len(l.derivedOf)+len(l.derivedStar) != 1 {
		return ""
	}
	if len(l.tables) == 1 {
		return l.tables[0]
	}
	for _, ws := range l.derivedStar {
		if len(ws) == 1 {
			return ws[0]
		}
	}
	return ""
}

// qualifierResolve 表限定符解析结果
type qualifierResolve struct {
	table     string                // 真实表名（来源唯一确定时）
	derived   map[string]exprColRef // 派生表列血缘
	ambiguous bool                  // 派生表星号通配多来源表：同名列归属歧义
	ok        bool
}

// resolveQualifier 解析表限定符：真实表别名返回真实表名；派生表别名返回其列血缘或星号通配来源
func (l *queryLineage) resolveQualifier(q string) qualifierResolve {
	q = strings.ToLower(q)
	if t, ok := l.aliasToTable[q]; ok {
		return qualifierResolve{table: t, ok: true}
	}
	if d, ok := l.derivedOf[q]; ok {
		return qualifierResolve{derived: d, ok: true}
	}
	// 派生表星号通配血缘：单一来源表时输出列即该表同名列，可精确归属；
	// 多来源表（JOIN/UNION展开）同名列归属歧义，标记ambiguous由调用方仅以全局规则解析
	if ws, ok := l.derivedStar[q]; ok {
		if len(ws) == 1 {
			return qualifierResolve{table: ws[0], ok: true}
		}
		return qualifierResolve{ambiguous: true, ok: true}
	}
	return qualifierResolve{}
}

// qualifyRef 对源引用应用表限定符解析：真实表别名锁定该表；派生表别名替换为该表对应列的血缘；
// 星号通配歧义时锁定为无来源表（仅全局规则解析，避免逐表猜测误归属）
func (l *queryLineage) qualifyRef(ref *exprColRef, qualifier, col string) {
	r := l.resolveQualifier(qualifier)
	if !r.ok {
		return
	}
	if r.derived != nil {
		if dref, hit := r.derived[strings.ToLower(stripTableAlias(col))]; hit {
			*ref = dref
		}
		return
	}
	if r.ambiguous {
		ref.table, ref.locked = "", true
		return
	}
	ref.table, ref.locked = r.table, true
}

// resolveExprItem 解析表达式/函数项的源引用并记录血缘条目：
// 先提取限定名对(qualifier.col)精确定位来源（真实表或派生表血缘），再退化为裸标识符token匹配查询内已知列；
// plan非nil时仅记录能命中脱敏计划的引用（函数名/关键字无命中不误设），nil时（内层血缘）仅接受可解析的限定名对
func (l *queryLineage) resolveExprItem(key, exprBody string, colRefs map[string]exprColRef, plan *masksvc.Plan, database string) {
	for _, pair := range sqlbase.ExtractExprQualifiedPairs(exprBody) {
		r := l.resolveQualifier(pair.Qualifier)
		if !r.ok {
			continue
		}
		colName, refTable := pair.Column, r.table
		if r.derived != nil {
			dref, hit := r.derived[strings.ToLower(pair.Column)]
			if !hit {
				continue
			}
			colName, refTable = dref.colName, dref.table
		} else if r.ambiguous {
			// 多表星号通配歧义：锁定为无来源表，仅全局规则解析，避免逐表猜测误归属
			refTable = ""
		}
		if plan != nil {
			if alg, _ := plan.Resolve(database, refTable, colName); alg == nil {
				continue
			}
		}
		l.refs[key] = exprColRef{colName: colName, table: refTable, locked: true}
		return
	}
	if colRefs == nil {
		return
	}
	for _, token := range sqlbase.ExtractExprIdentifiers(exprBody) {
		colName, refTable := token, ""
		if ref, known := colRefs[strings.ToLower(token)]; known {
			colName, refTable = ref.colName, ref.table
		}
		if plan != nil {
			if alg, _ := plan.Resolve(database, refTable, colName); alg == nil {
				continue
			}
		}
		// 表达式引用一律锁定归属（无法定位来源表时锁定空串仅全局规则），
		// 不逐表猜测避免多表JOIN时误归属其他表的标签
		l.refs[key] = exprColRef{colName: colName, table: refTable, locked: true}
		return
	}
}

// extractParenInner 从以'('开头的文本截取首个平衡括号对内的内容
// （方言解析器的派生表Name可能带别名/ON等尾巴，如"(SELECT ...) t"、"(...) y ON"），失败返回空串
func extractParenInner(text string) string {
	if !strings.HasPrefix(text, "(") {
		return ""
	}
	depth := 0
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '\'': // 跳过单引号字符串字面量（含''转义），避免字面量括号干扰配对
			for i++; i < len(text); i++ {
				if text[i] == '\'' {
					if i+1 < len(text) && text[i+1] == '\'' {
						i++
						continue
					}
					break
				}
			}
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return text[1:i]
			}
		}
	}
	return ""
}

// stripTableAlias 去除列名的表别名前缀与引用符，如 t.phone -> phone、t."PHONE" -> PHONE
func stripTableAlias(column string) string {
	if idx := lastDotIdx(column); idx >= 0 {
		column = column[idx+1:]
	}
	return strings.Trim(column, "`\"'")
}

// stripTableName 去除表名的库名前缀与引用符，如 `db`.`t_user` -> t_user
func stripTableName(table string) string {
	table = strings.Trim(table, "`\"'")
	if idx := lastDotIdx(table); idx >= 0 {
		table = table[idx+1:]
	}
	return table
}

func lastDotIdx(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}

// stripExprAliasSuffix 去除select项文本尾部的 AS 别名（大小写不敏感），返回表达式主体
func stripExprAliasSuffix(text string) string {
	if idx := strings.LastIndex(strings.ToUpper(text), " AS "); idx >= 0 {
		return strings.TrimSpace(text[:idx])
	}
	return strings.TrimSpace(text)
}

// exprColRef 查询内列引用的血缘信息（供表达式项token匹配与派生表血缘传播）
type exprColRef struct {
	colName string // 源列名（保留原始大小写）
	table   string // 来源表（限定名场景）；空串且locked=true表示确定无来源表，仅全局规则解析
	locked  bool   // 血缘是否锁定（锁定后不再按tables列表顺序逐表尝试兑底）
}

// isExprItem 判断select项是否为表达式/函数项。
// 项类型分类已收敛到解析器（base.ClassifySelectItem，跨方言Kind语义一致），应用层仅消费Kind
func isExprItem(item sqlstmt.SelectItem) bool {
	return item.Kind == sqlstmt.SelectItemExpr || item.Kind == sqlstmt.SelectItemFunction
}

// GetRulePageList 分页获取脱敏规则
func (m *MaskAppImpl) GetRulePageList(condition *entity.MaskRuleQuery, orderBy ...string) (*model.PageResult[*entity.DbMaskRule], error) {
	return m.maskRuleRepo.GetPageList(condition, orderBy...)
}

// SaveRule 保存脱敏规则，校验算法与参数合法性
func (m *MaskAppImpl) SaveRule(ctx context.Context, rule *entity.DbMaskRule) error {
	if rule.MatchType != entity.MaskMatchTypeRegex && rule.MatchType != entity.MaskMatchTypeExact && rule.MatchType != entity.MaskMatchTypePrefix {
		return errorx.NewBizf("invalid mask rule match type: %d", rule.MatchType)
	}
	if _, err := masksvc.Get(rule.Algorithm); err != nil {
		return err
	}
	if _, err := parseMaskParams(rule.Params); err != nil {
		return err
	}
	if rule.MatchType == entity.MaskMatchTypeRegex && rule.Pattern != "" {
		if _, err := masksvc.NewPlan([]*masksvc.Rule{{MatchType: masksvc.MatchTypeRegex, Pattern: rule.Pattern}}, nil); err != nil {
			return err
		}
	}
	if err := m.maskRuleRepo.Save(ctx, rule); err != nil {
		return err
	}
	m.invalidate()
	return nil
}

// DeleteRule 删除脱敏规则
func (m *MaskAppImpl) DeleteRule(ctx context.Context, id uint64) error {
	if err := m.maskRuleRepo.DeleteById(ctx, id); err != nil {
		return err
	}
	m.invalidate()
	return nil
}

// GetTagPageList 分页获取列标签
func (m *MaskAppImpl) GetTagPageList(condition *entity.MaskColumnQuery, orderBy ...string) (*model.PageResult[*entity.DbMaskColumn], error) {
	return m.maskColumnRepo.GetPageList(condition, orderBy...)
}

// GetTagById 根据id获取列标签
func (m *MaskAppImpl) GetTagById(id uint64) (*entity.DbMaskColumn, error) {
	return m.maskColumnRepo.GetById(id)
}

// SaveTag 保存列标签，校验动作对应的算法/规则配置
func (m *MaskAppImpl) SaveTag(ctx context.Context, tag *entity.DbMaskColumn) error {
	if tag.InstanceId == 0 {
		return errorx.NewBizf("mask column tag instance id is required")
	}
	switch tag.Action {
	case entity.MaskColumnActionBind:
		if tag.Algorithm == "" && tag.RuleId == 0 {
			return errorx.NewBizI(ctx, imsg.ErrMaskTagNeedAlgoOrRule)
		}
		if tag.Algorithm != "" {
			if _, err := masksvc.Get(tag.Algorithm); err != nil {
				return err
			}
		} else if _, err := m.maskRuleRepo.GetById(tag.RuleId); err != nil {
			return err
		}
	case entity.MaskColumnActionExempt:
	default:
		return errorx.NewBizf("invalid mask column action: %d", tag.Action)
	}
	if _, err := parseMaskParams(tag.Params); err != nil {
		return err
	}
	if err := m.maskColumnRepo.Save(ctx, tag); err != nil {
		return err
	}
	m.invalidate()
	return nil
}

// DeleteTag 删除列标签
func (m *MaskAppImpl) DeleteTag(ctx context.Context, id uint64) error {
	if err := m.maskColumnRepo.DeleteById(ctx, id); err != nil {
		return err
	}
	m.invalidate()
	return nil
}

// GetMaskApp 获取脱敏应用门面
func GetMaskApp() MaskApp {
	return ioc.Get[MaskApp]()
}
