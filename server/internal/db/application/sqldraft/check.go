package sqldraft

// 确定性校验：把「模型说得像 SQL」变成「这份 SQL 在目标库上成立与否」。
//
// 每条校验只产出 Finding，不改 SQL、不互相阻断；是否自纠、是否允许写回由管线按
// Severity 统一裁决。这条分工很重要：校验器一旦自己决定「要不要重试」，就会各自
// 实现一套重试逻辑，最终没人说得清一轮请求到底调了几次模型。
//
// 权威边界：
//   - 「要不要审批」只来自 CheckSqlsWithoutTicket（与交互执行同源），不采信模型；
//   - 「有没有权限」只来自 AssertStatementAllowed，且错误文本收敛为通用码；
//   - 「列名是否存在」用结构化引用判（SELECT 列表/INSERT 列/UPDATE SET），
//     表达式里的整词只用于「是否触碰受限列」的提示，不用于判「不存在」——
//     否则 count/sum/now 这类函数名会被误报成不存在的列。

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"

	"mayfly-go/internal/db/application/sqlutil"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/i18n"
)

func init() {
	registerChecker(parseChecker{})
	registerChecker(valueChecker{})
	registerChecker(identifierChecker{})
	registerChecker(writeChecker{})
	registerChecker(targetTableChecker{})
	registerChecker(noSourceChecker{})
	registerChecker(policyChecker{})
	registerChecker(aclChecker{})
	registerChecker(planChecker{})
}

// classify 与执行期同源的语句分类（解析失败不报错，按关键字兜底）
func classify(dialect dbi.Dialect, sql string) (sqlstmt.Stmt, string) {
	return sqlutil.ClassifyStatement(dialect, sql)
}

// isReadOnlyType 只读语句类型（与执行路由的判定口径一致：select 与 other-read）
func isReadOnlyType(stmtType string) bool {
	return stmtType == string(sqlstmt.StmtTypeSelect) || stmtType == string(sqlstmt.StmtTypeOtherRead)
}

// parseChecker 解析失败即致命：连语句类型都判不出来，不能写回编辑器
type parseChecker struct{}

func (parseChecker) Id() string { return "parse" }

func (parseChecker) Check(ctx context.Context, cc *checkCtx) []Finding {
	if cc.stmt == nil {
		return []Finding{{
			Severity: SeverityInvalid,
			Code:     CodeParseFailed,
			Message:  i18n.TC(ctx, imsg.ErrSqlDraftParseFailed),
		}}
	}
	return nil
}

// identifierChecker 表/列存在性核对（幻觉标识符的主要拦截点）
type identifierChecker struct{}

func (identifierChecker) Id() string { return "identifier" }

func (identifierChecker) Check(ctx context.Context, cc *checkCtx) []Finding {
	snap := cc.snap
	if snap == nil || cc.stmt == nil {
		return nil
	}
	findings := make([]Finding, 0, 4)

	tables := sqlutil.StatementTables(cc.stmt)
	referenced := make(map[string]*tableInfo, len(tables))
	for _, name := range tables {
		info, ok := snap.find(name)
		if ok {
			referenced[strings.ToLower(name)] = info
			continue
		}
		if snap.isHidden(name) {
			// 表存在但当前账号不可见：只说「不在可见范围」，不回表名细节
			findings = append(findings, Finding{
				Severity: SeverityUnverified,
				Code:     CodeTableRestricted,
				Message:  i18n.TC(ctx, imsg.ErrSqlDraftTableRestricted),
			})
			continue
		}
		findings = append(findings, Finding{
			Severity:   SeverityInvalid,
			Code:       CodeUnknownTable,
			Message:    i18n.TC(ctx, imsg.ErrSqlDraftUnknownTable, "table", name),
			Identifier: name,
			Candidates: closestNames(name, snap.visibleNames(), 3),
		})
	}

	// 结构化列引用：判「列不存在」
	for _, ref := range strictColumnRefs(cc.stmt) {
		column, ok := plainIdentifier(ref.Column)
		if !ok {
			// 表达式形态（如 COUNT(d.id)）里的标识符可能带括号残留，无法安全判定存在性；
			// 不判 ≠ 判过，交给表达式通道去判「是否触碰受限列」
			continue
		}
		if !columnExists(ref.Table, column, referenced, snap) {
			if snap.isColumnDenied(column) {
				findings = append(findings, Finding{
					Severity: SeverityUnverified,
					Code:     CodeColumnRestricted,
					Message:  i18n.TC(ctx, imsg.ErrSqlDraftColumnRestricted),
				})
				continue
			}
			findings = append(findings, Finding{
				Severity:   SeverityInvalid,
				Code:       CodeUnknownColumn,
				Message:    i18n.TC(ctx, imsg.ErrSqlDraftUnknownColumn, "column", column),
				Identifier: column,
				Candidates: closestNames(column, columnPool(referenced), 3),
			})
		}
	}

	// 表达式整词引用：只判「是否触碰受限列」，不判存在性（避免把函数名当列名）
	for _, ref := range sqlutil.StatementColumnRefs(cc.stmt) {
		if ref.Column == "" || strings.HasPrefix(ref.Column, "*") {
			continue
		}
		if !snap.isColumnDenied(ref.Column) {
			continue
		}
		findings = append(findings, Finding{
			Severity: SeverityUnverified,
			Code:     CodeColumnRestricted,
			Message:  i18n.TC(ctx, imsg.ErrSqlDraftColumnRestricted),
		})
		break
	}
	return findings
}

// plainIdentifier 取纯标识符：去掉括号/空白等表达式残留，只对「一个词就是列名」的形态给出结论。
//
// 为什么需要：方言解析器对函数参数的 ColumnName 会带上右括号（实测 COUNT(d.id) → "id)"），
// 直接比对会报一条「列 id) 不存在」的假错误——把可用的草稿判成不可用。
func plainIdentifier(text string) (string, bool) {
	trimmed := strings.Trim(strings.TrimSpace(text), "() `\"'")
	if trimmed == "" {
		return "", false
	}
	for index, r := range trimmed {
		valid := r == '_' || r == '$' || unicode.IsLetter(r) || (index > 0 && unicode.IsDigit(r))
		if !valid {
			return "", false
		}
	}
	return trimmed, true
}

// columnExists 列是否存在于「语句引用到的表」里。
//
// 引用表没有列明细（读取失败/方言不支持）时返回 true：不知道 ≠ 不存在，
// 这种情况由「无列明细」的事实降级为不判，而不是据此把草稿判成错误。
func columnExists(tableAlias, column string, referenced map[string]*tableInfo, snap *snapshot) bool {
	alias := strings.ToLower(strings.TrimSpace(tableAlias))
	targets := make([]*tableInfo, 0, len(referenced))
	for key, info := range referenced {
		if alias != "" && alias != key && alias != strings.ToLower(info.Name) {
			continue
		}
		targets = append(targets, info)
	}
	if len(targets) == 0 {
		targets = append(targets, referencedValues(referenced)...)
	}
	if len(targets) == 0 {
		return true
	}
	checked := false
	for _, info := range targets {
		if len(info.Columns) == 0 {
			continue
		}
		checked = true
		for _, candidate := range info.Columns {
			if strings.EqualFold(candidate.Name, column) {
				return true
			}
		}
	}
	return !checked
}

func referencedValues(referenced map[string]*tableInfo) []*tableInfo {
	values := make([]*tableInfo, 0, len(referenced))
	for _, info := range referenced {
		values = append(values, info)
	}
	return values
}

// columnPool 候选列名池（用于给出「最相近的列」修复提示）
func columnPool(referenced map[string]*tableInfo) []string {
	pool := make([]string, 0, 32)
	for _, info := range referenced {
		for _, column := range info.Columns {
			pool = append(pool, column.Name)
		}
	}
	return pool
}

// strictColumnRefs 结构化列引用（SELECT 列表 / INSERT 列 / UPDATE SET）。
//
// 只取 AST 里有明确字段的这些位置：它们不会混入函数名与关键字，因此可以承担
// 「不存在」这种强结论。表达式文本里的整词比对留给受限列提示（见本文件头）。
func strictColumnRefs(stmt sqlstmt.Stmt) []sqlutil.ColumnRef {
	switch typed := stmt.(type) {
	case *sqlstmt.SelectStmt:
		refs := make([]sqlutil.ColumnRef, 0, len(typed.Items))
		// 递归闭包必须先声明再赋值（Go 的 := 在闭包体里引用自身会得到未定义的符号）
		var collect func(one *sqlstmt.SelectStmt)
		collect = func(one *sqlstmt.SelectStmt) {
			if one == nil {
				return
			}
			for _, item := range one.Items {
				if item.IsStar() || item.ColumnName == "" {
					continue
				}
				refs = append(refs, sqlutil.ColumnRef{Table: item.TableAlias, Column: item.ColumnName})
			}
			for _, union := range one.Unions {
				collect(union.Select)
			}
		}
		collect(typed)
		return refs
	case *sqlstmt.InsertStmt:
		refs := make([]sqlutil.ColumnRef, 0, len(typed.Columns))
		for _, column := range typed.Columns {
			refs = append(refs, sqlutil.ColumnRef{Table: typed.Table.Name, Column: column})
		}
		return refs
	case *sqlstmt.UpdateStmt:
		refs := make([]sqlutil.ColumnRef, 0, len(typed.Set))
		table := ""
		if len(typed.Tables) > 0 {
			table = typed.Tables[0].Name
		}
		for _, assignment := range typed.Set {
			refs = append(refs, sqlutil.ColumnRef{Table: table, Column: assignment.Column})
		}
		return refs
	}
	return nil
}

// valueChecker 枚举取值核对：语句里与列比较的字面量必须出现在该列的采样结果里。
//
// 判定为 unverified 而不是 invalid：采样只读了有限行，不在采样集里不等于不存在。
// 但它足以提醒用户「这个取值在库里没见过」，而这正是结构校验完全看不到的一类错误。
type valueChecker struct{}

func (valueChecker) Id() string { return "value" }

func (valueChecker) Check(ctx context.Context, cc *checkCtx) []Finding {
	if len(cc.sampled) == 0 || cc.sql == "" {
		return nil
	}
	findings := make([]Finding, 0, 2)
	for _, sample := range cc.sampled {
		known := make(map[string]struct{}, len(sample.Values))
		for _, value := range sample.Values {
			known[strings.ToLower(value)] = struct{}{}
		}
		for _, literal := range literalsForColumn(cc.sql, sample.Column) {
			if _, ok := known[strings.ToLower(literal)]; ok {
				continue
			}
			candidates := sample.Values
			if len(candidates) > 5 {
				candidates = candidates[:5]
			}
			findings = append(findings, Finding{
				Severity:   SeverityUnverified,
				Code:       CodeValueNotSampled,
				Message:    i18n.TC(ctx, imsg.ErrSqlDraftValueNotSampled, "value", literal, "column", sample.Column),
				Identifier: sample.Column,
				Candidates: candidates,
			})
		}
	}
	return findings
}

// quotedLiteralPattern 「'字面量'」（两个连续单引号是转义）：与列名无关，编译一次常驻
var quotedLiteralPattern = regexp.MustCompile(`'((?:[^']|'')*)'`)

// columnLiteralPatterns 一个列名对应的两种比较形态
type columnLiteralPatterns struct {
	equal *regexp.Regexp
	in    *regexp.Regexp
}

// literalPatternCache 列名 → 已编译形态。
//
// 校验按「语句 × 采样列」双重循环调用，每次重新 MustCompile 会把正则编译成本放大到
// 上百次，而这纯粹是 CPU 浪费（模式只由列名决定）。列名来自目标库 schema，规模有限；
// 仍设上限，避免「超多列的库 + 长期运行」把缓存变成无界增长。
var (
	literalPatternCache     sync.Map
	literalPatternCacheSize atomic.Int64
)

// literalPatternCacheMax 缓存条目上限（超出后不再新增，只走即时编译）
const literalPatternCacheMax = 512

// identifierTailPattern 列名后可能紧跟的标识符引用符（反引号 / 双引号 / MSSQL 右方括号）。
//
// 这一条是修真机缺陷时加上的：MySQL、PG 等方言生成的语句默认带引用符（`type` = 'x'），
// 而原正则在列名之后直接要求 `\s*=`，遇到收尾的引用符就永不匹配 —— 于是 valueChecker
// 对每一条真实草稿都是空转，界面照样报「已通过校验」。
// 列名**前面**的引用符不用管：`\b` 在「引用符 → 字母」之间本身就是词边界。
const identifierTailPattern = `["` + "`" + `\]]?`

func literalPatternsFor(column string) columnLiteralPatterns {
	if cached, ok := literalPatternCache.Load(column); ok {
		return cached.(columnLiteralPatterns)
	}
	name := regexp.QuoteMeta(column)
	patterns := columnLiteralPatterns{
		equal: regexp.MustCompile(`(?is)\b` + name + `\b` + identifierTailPattern + `\s*=\s*'((?:[^']|'')*)'`),
		in:    regexp.MustCompile(`(?is)\b` + name + `\b` + identifierTailPattern + `\s+in\s*\(([^)]*)\)`),
	}
	if literalPatternCacheSize.Load() < literalPatternCacheMax {
		if _, loaded := literalPatternCache.LoadOrStore(column, patterns); !loaded {
			literalPatternCacheSize.Add(1)
		}
	}
	return patterns
}

// literalsForColumn 取「列 = '字面量'」与「列 IN ('a','b')」两种比较形态里的字面量。
//
// 只认这两种形态：把整句里的所有字符串都拿去比对，会把日期、中文说明等无关字面量
// 一并判成"取值不存在"，制造大量噪声，反而让用户不再信任校验结论。
func literalsForColumn(sql, column string) []string {
	patterns := literalPatternsFor(column)
	literals := make([]string, 0, 4)
	for _, match := range patterns.equal.FindAllStringSubmatch(sql, -1) {
		if len(match) >= 2 {
			literals = append(literals, strings.ReplaceAll(match[1], "''", "'"))
		}
	}
	for _, match := range patterns.in.FindAllStringSubmatch(sql, -1) {
		if len(match) < 2 {
			continue
		}
		for _, one := range quotedLiteralPattern.FindAllStringSubmatch(match[1], -1) {
			if len(one) == 2 {
				literals = append(literals, strings.ReplaceAll(one[1], "''", "'"))
			}
		}
	}
	return literals
}

// noSourceChecker 读语句没有引用任何表时给出「无法核对」结论。
//
// 真机证据：模型在上下文里找不到需要的表时不会说「我不知道」，而是输出一条
// `SELECT 'ERROR: No metric sampling table found in context' AS error_message, NULL AS ...`
// —— 把放弃的话术当成查询结果返回。它语法合法、没有引用任何不存在的列，
// 于是被前面的校验一路放行成「已通过校验」。空表引用是这类产物的稳定特征：
// 任何需要查库的答案都必须至少引用一张表或一个视图。
//
// 定为 unverified 而不是 invalid：`SELECT VERSION()` 这类 legitimately 无表的语句确实存在，
// 界面要说的只是「这一条没核对到任何东西」，而不是「这条不能用」。
type noSourceChecker struct{}

func (noSourceChecker) Id() string { return "no_source" }

func (noSourceChecker) Check(ctx context.Context, cc *checkCtx) []Finding {
	// 只管 SELECT：OtherRead（SHOW/EXPLAIN 等）本来就不以表引用为形态
	if cc.stmt == nil || cc.stmtTyp != string(sqlstmt.StmtTypeSelect) {
		return nil
	}
	if len(sqlutil.StatementTables(cc.stmt)) > 0 {
		return nil
	}
	// 解析器拿不到表引用不等于语句真的没表（CTE 内层表不可静态抽取）：
	// 文本里出现过 from 就交给 identifier 去标 uncheckable，不在这里重复下结论
	if strings.Contains(strings.ToLower(cc.sql), "from") {
		return nil
	}
	return []Finding{{
		Severity: SeverityUnverified,
		Code:     CodeNoTableSource,
		Message:  i18n.TC(ctx, imsg.ErrSqlDraftNoTableSource),
	}}
}

// writeChecker 写语句提示（结论来自语句类型，不来自模型）
type writeChecker struct{}

func (writeChecker) Id() string { return "write" }

func (writeChecker) Check(ctx context.Context, cc *checkCtx) []Finding {
	if cc.stmt == nil || isReadOnlyType(cc.stmtTyp) {
		return nil
	}
	return []Finding{{
		Severity: SeverityNote,
		Code:     CodeWriteStatement,
		Message:  i18n.TC(ctx, imsg.ErrSqlDraftWriteHint, "type", cc.stmtTyp),
	}}
}

// targetTableChecker 写语句的目标表必须来自「需求描述直接命中的表」。
//
// 真机踩到过：需求是「把每个实例的备注更新为实例名称」，模型写向 t_ai_plugin_instance
// （同义表：xxx_instance），语句结构完全合法，但方向错了——这类错只有把目标表与需求
// 对齐才看得出来。判定为 unverified 而不是 invalid：命中间隔允许同义词缺失（描述里
// 只写"实例"而表名是 db_instance 时由注释命中，注释缺失就会误报）。
type targetTableChecker struct{}

func (targetTableChecker) Id() string { return "target_table" }

func (targetTableChecker) Check(ctx context.Context, cc *checkCtx) []Finding {
	if cc.stmt == nil || len(cc.promptTables) == 0 || isReadOnlyType(cc.stmtTyp) {
		return nil
	}
	targets := sqlutil.StatementTables(cc.stmt)
	findings := make([]Finding, 0, len(targets))
	for _, target := range targets {
		if _, ok := cc.promptTables[strings.ToLower(target)]; ok {
			continue
		}
		findings = append(findings, Finding{
			Severity:   SeverityUnverified,
			Code:       CodeTargetUnconfirmed,
			Message:    i18n.TC(ctx, imsg.ErrSqlDraftTargetUnconfirmed, "table", target),
			Identifier: target,
			Candidates: promptTableCandidates(cc.promptTables, 3),
		})
	}
	return findings
}

// promptTableCandidates 把命中的表名转成候选提示（稳定顺序，便于界面展示与回归）
func promptTableCandidates(tables map[string]struct{}, limit int) []string {
	names := make([]string, 0, len(tables))
	for name := range tables {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) > limit {
		names = names[:limit]
	}
	return names
}

// policyChecker 审批事实：走与交互执行同源的闸门判定，只作提示不拦截
type policyChecker struct{}

func (policyChecker) Id() string { return "policy" }

func (policyChecker) Check(ctx context.Context, cc *checkCtx) []Finding {
	if cc.gate == nil || cc.conn == nil {
		return nil
	}
	hints, err := cc.gate.CheckSqlsWithoutTicket(ctx, cc.conn, []string{cc.sql})
	if err != nil {
		// 「需审批 / 需确认」是闸门给出的**结论**，不是读策略失败：它必须显示成一句可执行的提示。
		// 曾把两者混为一谈，写语句的卡片于是显示「无法读取审批策略」——把明确的拦截说成系统故障。
		switch cc.gate.PolicyBlockKind(err) {
		case policyBlockApproval:
			return []Finding{{Severity: SeverityNote, Code: CodePolicyTicket, Message: i18n.TC(ctx, imsg.ErrSqlDraftNeedApproval)}}
		case policyBlockWarnAck:
			return []Finding{{Severity: SeverityNote, Code: CodePolicyTicket, Message: i18n.TC(ctx, imsg.ErrSqlDraftNeedWarnAck)}}
		}
		// 判定不了不等于「不需要审批」：如实标为无法确认
		return []Finding{{
			Severity: SeverityUnverified,
			Code:     CodeUncheckable,
			Message:  i18n.TC(ctx, imsg.ErrSqlDraftPolicyUnavailable),
		}}
	}
	findings := make([]Finding, 0, len(hints))
	for _, hint := range hints {
		if strings.TrimSpace(hint) == "" {
			continue
		}
		findings = append(findings, Finding{Severity: SeverityNote, Code: CodePolicyTicket, Message: hint})
	}
	return findings
}

// aclChecker 权限事实：语句级 ACL 的 dry 判定（不执行），错误文本收敛为通用码
type aclChecker struct{}

func (aclChecker) Id() string { return "acl" }

func (aclChecker) Check(ctx context.Context, cc *checkCtx) []Finding {
	if cc.acl == nil || cc.dbId == 0 || cc.stmt == nil {
		return nil
	}
	tables := sqlutil.StatementTables(cc.stmt)
	resolvable := len(tables) > 0 && !isUnresolvableShape(cc.stmt)
	refs := sqlutil.StatementColumnRefs(cc.stmt)
	if err := cc.acl.AssertStatementAllowed(ctx, cc.dbId, tables, resolvable, refs); err != nil {
		// 原始错误可能含无权列名：这里刻意不透传，只回通用码
		return []Finding{{
			Severity: SeverityUnverified,
			Code:     CodeAclDenied,
			Message:  i18n.TC(ctx, imsg.ErrSqlDraftAclDenied),
		}}
	}
	return nil
}

// isUnresolvableShape 表清单不可信的语句形态（与语句级 ACL 的可信度判据同源语义）：
// CTE 与含子查询的语句无法在此静态枚举全部来源表
func isUnresolvableShape(stmt sqlstmt.Stmt) bool {
	switch typed := stmt.(type) {
	case *sqlstmt.WithStmt:
		return true
	case *sqlstmt.SelectStmt:
		return len(typed.Unions) > 0
	}
	return false
}

// planChecker 只读计划校验（EXPLAIN 一律不带 ANALYZE，看计划不执行）
type planChecker struct{}

func (planChecker) Id() string { return "plan" }

func (planChecker) Check(ctx context.Context, cc *checkCtx) []Finding {
	if !cc.includePlans || cc.meta == nil || cc.stmt == nil {
		return nil
	}
	result, err := cc.meta.Explain(ctx, cc.sql)
	if err != nil || result == nil {
		// 方言不支持/语句不可分析：如实说明「拿不到计划」，不当成缺陷也不当成安全
		return []Finding{{
			Severity: SeverityUnverified,
			Code:     CodePlanUnavailable,
			Message:  i18n.TC(ctx, imsg.ErrExplainUnavailable, "reason", sqlutil.BriefReason(err)),
		}}
	}
	if result.FullScan {
		return []Finding{{
			Severity: SeverityNote,
			Code:     CodePlanFullScan,
			Message:  i18n.TC(ctx, imsg.ErrSqlDraftPlanFullScan, "rows", int(result.Rows)),
		}}
	}
	return nil
}

// closestNames 从候选池里挑最相近的几个名字（给自纠与人工修正一个可执行的下一步）
func closestNames(target string, pool []string, limit int) []string {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" || len(pool) == 0 {
		return nil
	}
	type scored struct {
		name  string
		score int
	}
	list := make([]scored, 0, len(pool))
	for _, name := range pool {
		lower := strings.ToLower(name)
		if lower == target {
			continue
		}
		score := longestCommonSubstring(lower, target)
		if strings.HasPrefix(lower, target) || strings.Contains(lower, target) || strings.Contains(target, lower) {
			score += 3
		}
		if score < 2 {
			continue
		}
		list = append(list, scored{name: name, score: score})
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].score == list[j].score {
			return list[i].name < list[j].name
		}
		return list[i].score > list[j].score
	})
	if len(list) > limit {
		list = list[:limit]
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		out = append(out, item.name)
	}
	return out
}
