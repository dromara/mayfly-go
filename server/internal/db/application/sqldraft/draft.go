package sqldraft

// 生成管线：装配 → 生成 → 校验 → 有界自纠 → 结论。
//
// 管线只做编排，不含任何策略：召回归 Linker、材料归 Fragment、判定归 Checker、
// 模板与组合归 Intent。这样「多加一条校验」不会碰到「多加一种召回」的代码。

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"mayfly-go/internal/ai/pkg/textgen"
	"mayfly-go/internal/db/application/sqlutil"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
)

// DraftAppImpl SQL 草稿应用服务实现
type DraftAppImpl struct {
	conns    DbConnReader       `inject:"T"`
	tableAcl TableAcl           `inject:"T"`
	gate     SqlGate            `inject:"T"`
	savedSql SavedSqlExamples   `inject:"T"`
	sampler  ColumnValueSampler `inject:"T"`
	// terms 业务词典（语义层）：喂召回也喂上下文
	terms Terms `inject:"T"`
}

var _ Service = (*DraftAppImpl)(nil)

// Draft 生成一份 SQL 草稿
func (d *DraftAppImpl) Draft(ctx context.Context, req *Req) (*Draft, error) {
	if req == nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrSqlDraftPromptRequired)
	}
	normalizeReq(req)
	if req.DbId == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrSqlDraftPromptRequired)
	}
	spec, ok := specOf(req.Intent)
	if !ok {
		return nil, errorx.NewBizI(ctx, imsg.ErrSqlDraftIntentUnknown, "intent", req.Intent)
	}
	// 描述是否必填由意图决定：解释/优化/改写可以零输入（选中 SQL 直接点），
	// 而「按描述生成」没有描述就没有需求
	if strings.TrimSpace(req.Prompt) == "" && !spec.PromptOptional {
		return nil, errorx.NewBizI(ctx, imsg.ErrSqlDraftPromptRequired)
	}
	// 材料长度上界：拒绝而不是截断。截断后的需求会生成一份看起来正常、实际答非所问的草稿，
	// 用户无从察觉；明确报错则让用户自己去精简（前端同侧也做了校验，这里是权威闸门）
	if total := len([]rune(materialsOf(req))); total > maxMaterialRunes {
		return nil, errorx.NewBizI(ctx, imsg.ErrSqlDraftMaterialTooLong, "max", maxMaterialRunes)
	}
	// fail-closed：模型未配置就不猜、不降级成「空结果」，让界面给出明确的状态
	port, ok := textgen.Get()
	if !ok {
		return nil, errorx.NewBizI(ctx, imsg.ErrSqlDraftUnavailable)
	}
	if available, err := port.Available(ctx); err != nil || !available {
		return nil, errorx.NewBizI(ctx, imsg.ErrSqlDraftUnavailable)
	}

	conn, err := d.conns.GetDbConn(ctx, req.DbId, req.Db)
	if err != nil {
		return nil, err
	}
	snap, err := buildSnapshot(ctx, conn.Metadata(), conn.GetDialect(), conn.Info.Type, d.tableAcl, req.DbId, req.Db)
	if err != nil {
		return nil, err
	}

	run := &draftRun{
		req:         req,
		spec:        spec,
		snap:        snap,
		meta:        conn.Metadata(),
		dialect:     conn.GetDialect(),
		conn:        conn,
		acl:         d.tableAcl,
		gate:        d.gate,
		savedSql:    d.savedSql,
		sampler:     d.sampler,
		termSvc:     d.terms,
		sampleCache: make(map[string][]string, 8),
		gen:         newGenerator(port),
	}
	if spec.MaxOutputTokens > 0 {
		run.gen.maxTokens = spec.MaxOutputTokens
	}
	return run.execute(ctx)
}

// loadTerms 取本次请求可用的业务词典条目（一次请求内只查一趟）
func (r *draftRun) loadTerms(ctx context.Context) {
	// 未注册词典服务时安静跳过：语义层是增强材料，缺它等于回到没有登记口径的状态，
	// 不是错误——生成仍按表注释与词面通道进行，不该因此弹错
	if r.termSvc == nil {
		return
	}
	items, err := r.termSvc.ListForSchema(ctx, r.req.DbId, r.req.Db)
	if err != nil {
		// 读失败降级为「没有词典」并留痕：词典是可选增强，为一趟元数据读取失败而让整次
		// 生成失败，等于把配置面故障转嫁给使用者
		logx.WarnfContext(ctx, "[sqldraft] glossary terms unavailable, continue without it: %s", err.Error())
		return
	}
	r.terms = items
}

// promptTableSet 把「仅凭描述命中的表」摊平成集合（小写）
func promptTableSet(groups [][]candidate) map[string]struct{} {
	set := make(map[string]struct{}, 8)
	for _, group := range groups {
		for _, one := range group {
			set[strings.ToLower(one.Table)] = struct{}{}
		}
	}
	return set
}

// toContextTables 把选中表投影成对外展示的「依据」清单。
//
// 依据的连接符在这里拼而不是在召回策略里拼：策略只产出事实清单，展示语言（中文用「；」、
// 英文用 "; "）由持有 ctx 的投影层决定，避免把标点写死进召回逻辑
func toContextTables(ctx context.Context, picked []*tableInfo, reasons map[string][]string) []ContextTable {
	joiner := i18n.TC(ctx, imsg.SqlDraftReasonJoiner)
	tables := make([]ContextTable, 0, len(picked))
	for _, info := range picked {
		tables = append(tables, ContextTable{
			Name:    info.Name,
			Comment: info.Comment,
			Columns: len(info.Columns),
			Reason:  strings.Join(reasons[strings.ToLower(info.Name)], joiner),
		})
	}
	return tables
}

// usageOrNil 只有真的调用过模型（或有用量）时才回传用量，避免界面上出现「0 tokens」噪声
func usageOrNil(gen *generator) *textgen.Usage {
	if gen.calls == 0 && gen.usage.TotalTokens == 0 {
		return nil
	}
	usage := *gen.usage
	return &usage
}

// normalizeReq 归一化请求：缺省意图按 draft 处理，其余字段去掉首尾空白
func normalizeReq(req *Req) {
	req.Intent = strings.TrimSpace(req.Intent)
	if req.Intent == "" {
		req.Intent = "draft"
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.Selection = strings.TrimSpace(req.Selection)
	req.AnchorStatement = strings.TrimSpace(req.AnchorStatement)
	req.RefineOf = strings.TrimSpace(req.RefineOf)
}

// draftRun 一次生成的运行状态
type draftRun struct {
	req     *Req
	spec    Spec
	snap    *snapshot
	meta    metadataReader
	dialect dbi.Dialect
	conn    *dbi.DbConn
	acl     TableAcl
	gate    SqlGate
	// savedSql 同库已存脚本仓储（交给片段自取材料，见 fragCtx.savedSql）
	savedSql SavedSqlExamples
	// sampler 列取值采样（可空：离线测试与未配置时跳过采样）
	sampler ColumnValueSampler
	// sampleCache 本次请求内的采样缓存（同一列只查一次）
	sampleCache map[string][]string
	// sampled 本次进入上下文的采样结果（供片段渲染与生成后核对）
	sampled []sampledValue
	// termSvc 业务词典服务（可空：离线测试与未注册时为 nil，词典通道安静跳过）
	termSvc Terms
	// terms 本次可用的词典条目（一次请求内只取一次，见 loadTerms）
	terms []*entity.DbAiTerm
	// promptGroups 「仅凭需求描述」的提议结果（一次请求内只算一次，见 promptProposals）
	promptGroups [][]candidate
	// promptTables 仅凭需求描述命中的表（写语句目标表核对用）
	promptTables map[string]struct{}
	gen          *generator

	warnings []string
}

func (r *draftRun) execute(ctx context.Context) (*Draft, error) {
	text := materialsOf(r.req)
	// 词典在召回之前取：glossaryLinker 与 glossaryFragment 共用同一份快照
	r.loadTerms(ctx)
	// 写语句目标表核对的基准必须是「描述命中的表」，不含选择题结果：
	// 选择题是模型的判断，拿它当基准会让"模型自己挑的表"永远算"描述命中"，核对就此失效
	r.promptTables = promptTableSet(r.promptProposals(ctx))

	picked, reasons, warnings, err := r.resolveTables(ctx, text)
	r.warnings = append(r.warnings, warnings...)
	if err != nil {
		return nil, err
	}

	// 只装载选中表的列：大库上「先全量取列」会让一次生成退化成几秒的结构洪泛
	r.snap.ensureColumns(ctx, r.meta, picked)

	pickedInfo := make([]*tableInfo, 0, len(picked))
	for _, name := range picked {
		if info, ok := r.snap.find(name); ok {
			pickedInfo = append(pickedInfo, info)
		}
	}

	// 行内补全且没定位到表：直接给空结果，不花一次模型调用去生成注定不可用的语句
	if r.spec.AllowNoTables && len(picked) == 0 {
		return &Draft{
			Intent:     r.spec.Id,
			Statements: []Statement{},
			Checks:     []Finding{},
			Warnings:   dedupeStrings(append(r.warnings, i18n.TC(ctx, imsg.ErrSqlDraftNoTableLocated))),
			Dialect:    r.snap.dialectLabel(),
		}, nil
	}

	// 枚举列取值采样：在列明细装好之后、渲染上下文之前（片段要用）
	r.sampled = r.sampleColumns(ctx, collectSampleTargets(pickedInfo))

	contextBlock, ctxWarnings := r.renderContext(ctx, pickedInfo)
	r.warnings = append(r.warnings, ctxWarnings...)

	system, err := renderTemplate(r.spec.SystemTemplate, map[string]any{
		"Dialect": r.snap.dialectLabel(),
		"Quote":   r.snap.quotePrefix,
	})
	if err != nil {
		return nil, err
	}
	task, err := renderTemplate(r.spec.TaskTemplate, map[string]any{
		"Prompt":    r.req.Prompt,
		"Selection": r.req.Selection,
		"Materials": sqlMaterials(r.req),
		"Refine":    r.refineBlock(ctx),
		"Context":   contextBlock,
	})
	if err != nil {
		return nil, err
	}

	// 散文产物（explain）：只生成一次、不校验、不写回
	if r.spec.OutputKind == OutputKindText {
		output, genErr := r.gen.complete(ctx, modelCall{purpose: purposeDraft, system: system, user: task, timeout: callTimeout})
		if genErr != nil {
			return nil, modelError(ctx, genErr, callTimeout)
		}
		explanation, ok := parseExplanation(output)
		if !ok {
			return nil, errorx.NewBizI(ctx, imsg.ErrSqlDraftParseFailed)
		}
		logx.InfofContext(ctx, "[sqldraft] intent=%s dbId=%d explanation_chars=%d", r.spec.Id, r.req.DbId, len([]rune(explanation)))
		return &Draft{
			Intent: r.spec.Id,
			// 必须是空切片而不是 nil：序列化成 null 后，前端对 statements/checks 的数组操作
			// （map/length/join）会抛异常，整个结果卡片渲染中断、界面停在 loading。
			Statements:    []Statement{},
			Checks:        []Finding{},
			Explanation:   explanation,
			ContextTables: toContextTables(ctx, pickedInfo, reasons),
			Warnings:      dedupeStrings(r.warnings),
			Dialect:       r.snap.dialectLabel(),
			Usage:         usageOrNil(r.gen),
		}, nil
	}

	statements, checks, repairs, err := r.generateAndCheck(ctx, system, task)
	if err != nil {
		return nil, err
	}
	statements, checks, candidates := r.rerollCandidates(ctx, system, task, statements, checks)

	contextTables := toContextTables(ctx, pickedInfo, reasons)

	if statements == nil {
		statements = []Statement{}
	}
	if checks == nil {
		checks = []Finding{}
	}
	draft := &Draft{
		Intent:        r.spec.Id,
		Statements:    statements,
		Verification:  verificationOf(checks),
		Checks:        checks,
		ContextTables: contextTables,
		Warnings:      dedupeStrings(r.warnings),
		Repairs:       repairs,
		Candidates:    candidates,
		Dialect:       r.snap.dialectLabel(),
	}
	draft.Usage = usageOrNil(r.gen)
	logx.InfofContext(ctx, "[sqldraft] intent=%s dbId=%d picked=%d calls=%d repairs=%d verification=%s",
		r.spec.Id, r.req.DbId, len(pickedInfo), r.gen.calls, repairs, draft.Verification)
	return draft, nil
}

// resolveTables 表召回：提议阶段 → （必要时）模型选择题 → 扩展阶段 → 排序截断。
//
// 两阶段是这里的骨架，不是可选优化：扩展类策略（外键邻接、同族表）的依据是「已确定的表集合」，
// 而该集合只有选择题跑完才存在。把两者混在同一轮循环里，扩展策略拿到的永远是空集合，
// 于是静默失效——本包真的踩过这个坑（外键通道上线后从未产出过一张表）。
func (r *draftRun) resolveTables(ctx context.Context, text string) ([]string, map[string][]string, []string, error) {
	reasons := make(map[string][]string, maxTables)
	// 解释类：跳过召回（材料自带表列信息）
	if r.spec.SkipRecall {
		return nil, reasons, nil, nil
	}
	warnings := make([]string, 0, 2)
	proposers, expanders := partitionLinkers(r.spec.Linkers)

	// ---- 提议阶段：只依据用户材料 ----
	lc := &linkCtx{text: text, prompt: r.req.Prompt, snap: r.snap, meta: r.meta, terms: r.terms, limit: maxTables}
	groups := r.proposeAll(ctx, lc, proposers, &warnings)
	merged := mergeCandidates(groups...)
	// chooserTried 记录是否走了语义召回通道（说明描述本身没能定位到表）
	chooserTried := false

	// 「要不要再问一次模型」只看需求描述本身的命中强度，不看编辑器里的材料。
	// 材料里的表名（光标所在语句引用的表）只是线索：它说明用户在编辑这张表，
	// 不代表这张表就是本次需求要用的表。曾因为把材料计入置信度，编辑器里正好停在
	// `UPDATE t_sys_log ...` 时「统计每个实例下有多少个库」被判定为已足够可信，
	// 于是不再走选择题，上下文里全是日志表、需求要的库表一张都没带进来。
	promptGroups := r.promptProposals(ctx)
	if bestScore(mergeCandidates(promptGroups...)) < confidentScore && strings.TrimSpace(text) != "" {
		chooserTried = true
		names, chooseWarnings, err := chooseTables(ctx, r.gen, r.snap, r.meta, text, maxTables)
		warnings = append(warnings, chooseWarnings...)
		switch {
		case err != nil:
			logx.WarnfContext(ctx, "[sqldraft] table chooser failed: %s", err.Error())
			warnings = append(warnings, i18n.TC(ctx, imsg.ErrSqlDraftChooserDegraded))
		default:
			reason := i18n.TC(ctx, imsg.SqlDraftChooseReason)
			chosen := make([]candidate, 0, len(names))
			for _, name := range names {
				chosen = append(chosen, candidate{Table: name, Score: confidentScore, Reasons: []string{reason}})
			}
			// 选择题结果优先于子串召回：模型看过全部表名与注释，判断依据比子串强
			merged = mergeCandidates(append(append([][]candidate{chosen}, promptGroups...), groups...)...)
		}
	}

	// 行内补全：没有可定位的表就返回空结果（界面不展示建议），不报错也不硬编
	if r.spec.AllowNoTables && len(merged) == 0 {
		return nil, reasons, warnings, nil
	}

	// fail-closed：描述没命中、语义召回又失败/没选出任何表时，不要再拿编辑器里的线索表硬编。
	// 真机实测的坏结局是：模型在没有相关表的上下文里编出一条读 t_sys_resource 的假语句，
	// 花了 114s + 一轮自纠才被校验拦下——比直接告诉用户"定位不到表"差得多。
	if chooserTried && len(merged) == 0 {
		return nil, nil, warnings, errorx.NewBizI(ctx, imsg.ErrSqlDraftNoTableLocated)
	}

	// ---- 扩展阶段：以「已确定集合」为基准补充（外键邻接 / 同族表）----
	if len(expanders) > 0 && len(merged) > 0 {
		base := candidateNames(merged)
		if len(base) > maxTables {
			base = base[:maxTables]
		}
		// 扩展阶段的 linkCtx 不带 text：这些策略不看描述，看了就会把提议阶段的口径混进来
		extra := r.proposeAll(ctx, &linkCtx{snap: r.snap, meta: r.meta, picked: base, limit: maxTables}, expanders, &warnings)
		if len(extra) > 0 {
			merged = mergeCandidates(append([][]candidate{merged}, extra...)...)
		}
	}

	picked := make([]string, 0, maxTables)
	for _, item := range merged {
		if len(picked) >= maxTables {
			warnings = append(warnings, i18n.TC(ctx, imsg.ErrSqlDraftTablesTruncated, "max", maxTables))
			break
		}
		picked = append(picked, item.Table)
		reasons[strings.ToLower(item.Table)] = item.Reasons
	}
	return picked, reasons, warnings, nil
}

// proposeAll 依次执行一批召回策略，收集结果并把降级事实追加到 warnings。
//
// 单路失败不终止整次生成：其余通道给出的表仍然可用，如实告知比整次失败更有用
func (r *draftRun) proposeAll(ctx context.Context, lc *linkCtx, list []Linker, warnings *[]string) [][]candidate {
	groups := make([][]candidate, 0, len(list))
	for _, linker := range list {
		items, err := linker.Propose(ctx, lc)
		if err != nil {
			logx.WarnfContext(ctx, "[sqldraft] linker %s failed: %s", linker.Id(), err.Error())
			*warnings = append(*warnings, i18n.TC(ctx, imsg.ErrSqlDraftLinkerDegraded, "linker", linker.Id()))
			continue
		}
		groups = append(groups, items)
	}
	return groups
}

// promptProposals 「仅凭需求描述」的提议结果，一次请求内只算一次。
//
// 两处消费者（写语句目标表核对的基准、选择题的置信度判定）必须口径一致，
// 而确定性召回在大库上并不便宜：曾经各算一次，等于每个请求把它跑了两遍。
// 用 memo 而不是由调用方传参，是为了让"只算一次"成为本类型的不变量，
// 而不是依赖每个调用点记得复用同一个变量。
func (r *draftRun) promptProposals(ctx context.Context) [][]candidate {
	if r.promptGroups != nil {
		return r.promptGroups
	}
	proposers, _ := partitionLinkers(r.spec.Linkers)
	r.promptGroups = r.proposeOnPrompt(ctx, strings.TrimSpace(r.req.Prompt), proposers)
	return r.promptGroups
}

// proposeOnPrompt 只用需求描述跑一遍提议阶段召回（不记告警）：
// 它的唯一用途是判断「描述本身能不能定位到表」，因此失败即视为没命中。
// 扩展类策略不参与——它们的依据是「已选表集合」，此刻还没有集合。
func (r *draftRun) proposeOnPrompt(ctx context.Context, prompt string, proposers []Linker) [][]candidate {
	if prompt == "" || len(proposers) == 0 {
		return nil
	}
	// terms 必须一起传：置信度判据看的是「仅凭描述能命中什么」，词典正是描述侧的语义来源，
	// 漏传会让登记过口径的库仍然去跑一次选表选择题（真机实测：参考表 12 张、多花 1.4k tokens）
	lc := &linkCtx{text: prompt, prompt: prompt, snap: r.snap, meta: r.meta, terms: r.terms, limit: maxTables}
	groups := make([][]candidate, 0, len(proposers))
	for _, linker := range proposers {
		items, err := linker.Propose(ctx, lc)
		if err != nil {
			continue
		}
		groups = append(groups, items)
	}
	return groups
}

// renderContext 按优先级装配片段，超预算时丢弃低优先片段并如实告知。
//
// 片段自己向 fragCtx 要材料（快照按需装载结构、端口取外部语料），管线不预知谁需要什么
func (r *draftRun) renderContext(ctx context.Context, picked []*tableInfo) (string, []string) {
	warnings := make([]string, 0, 1)
	fc := &fragCtx{snap: r.snap, picked: picked, req: r.req, meta: r.meta, savedSql: r.savedSql, sampled: r.sampled, terms: r.terms}

	type entry struct {
		fragment Fragment
		text     string
	}
	entries := make([]entry, 0, len(r.spec.Fragments))
	for _, id := range r.spec.Fragments {
		fragment, ok := fragmentOf(id)
		if !ok {
			continue
		}
		entries = append(entries, entry{fragment: fragment, text: fragment.Render(ctx, fc)})
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].fragment.Priority() < entries[j].fragment.Priority() })

	var sb strings.Builder
	used := 0
	for index, item := range entries {
		body := strings.TrimSpace(item.text)
		if body == "" {
			continue
		}
		cost := textgen.EstimateTokens(body)
		if used+cost > contextTokenBudget {
			dropped := make([]string, 0, len(entries)-index)
			for _, rest := range entries[index:] {
				dropped = append(dropped, rest.fragment.Id())
			}
			warnings = append(warnings, i18n.TC(ctx, imsg.ErrSqlDraftContextTruncated, "fragments", strings.Join(dropped, ", ")))
			break
		}
		used += cost
		sb.WriteString(body)
		sb.WriteString("\n\n")
	}
	return strings.TrimRight(sb.String(), "\n"), warnings
}

// generateAndCheck 生成 → 校验 → 有界自纠
func (r *draftRun) generateAndCheck(ctx context.Context, system, task string) ([]Statement, []Finding, int, error) {
	originalTask := task
	output, err := r.gen.complete(ctx, modelCall{purpose: purposeDraft, system: system, user: task, timeout: callTimeout})
	if err != nil {
		return nil, nil, 0, modelError(ctx, err, callTimeout)
	}
	statements, ok := parseContract(output)
	if !ok {
		statements = nil
	}

	checks := r.checkStatements(ctx, statements, !ok)
	repairs := 0
	maxRepairs := r.spec.MaxRepairs
	if maxRepairs == 0 {
		maxRepairs = maxRepairRounds
	}
	for repairs < maxRepairs && hasInvalid(checks) {
		repairPrompt, renderErr := renderTemplate("repair_task.md", map[string]any{
			"Original": originalTask,
			"Findings": renderFindings(checks),
		})
		if renderErr != nil {
			return statements, checks, repairs, nil
		}
		repairs++
		output, err = r.gen.complete(ctx, modelCall{purpose: purposeRepair, system: system, user: repairPrompt, timeout: callTimeout})
		if err != nil {
			// 自纠失败保留上一版结果：用户至少能看到一份带问题的草稿与失败原因
			logx.WarnfContext(ctx, "[sqldraft] repair round %d failed: %s", repairs, err.Error())
			break
		}
		repaired, ok := parseContract(output)
		if !ok {
			// 自纠后反而解析不出来：保留原结果并记录，不用更差的结果覆盖
			checks = append(checks, Finding{
				Severity: SeverityUnverified,
				Code:     CodeParseFailed,
				Message:  i18n.TC(ctx, imsg.ErrSqlDraftRepairFailed),
			})
			break
		}
		statements = repaired
		checks = r.checkStatements(ctx, statements, false)
	}
	return statements, checks, repairs, nil
}

// refineBlock 多轮细化的提示块：有上一版语句时渲染 prompts/refine_block.md，否则空串。
//
// 文案留在模板文件里而不是拼在 Go 字符串里：调这段措辞与调其他提示词一样频繁，
// 散进代码会让 prompts/ 不再是提示词的唯一真源（见 prompt.go 的约定）
func (r *draftRun) refineBlock(ctx context.Context) string {
	previous := strings.TrimSpace(r.req.RefineOf)
	if previous == "" {
		return ""
	}
	block, err := renderTemplate("refine_block.md", map[string]any{"Previous": previous})
	if err != nil {
		logx.WarnfContext(ctx, "[sqldraft] refine block unavailable, continue without it: %s", err.Error())
		return ""
	}
	return block
}

// rerollCandidates 未通过校验时的候选补采与重排选优。
//
// 与自纠的分工：自纠是「把你指出的问题改掉」（有依据、定向），补采是「换一份答案」
// （无依据、发散）。前者先跑，因为它更便宜也更准；只有补采完仍不对，才把问题原样交给用户。
//
// 为什么平手保留已有结果：换一份「同样好」的草稿对用户没有收益，只会让他困惑
// 「我刚才看到的那版去哪了」。
func (r *draftRun) rerollCandidates(ctx context.Context, system, task string, statements []Statement, checks []Finding) ([]Statement, []Finding, int) {
	if r.spec.ExtraCandidates <= 0 || verificationOf(checks) == verificationVerified {
		return statements, checks, 0
	}
	bestStatements, bestChecks := statements, checks
	taken := 0
	temperature := candidateTemperature
	for index := 0; index < r.spec.ExtraCandidates; index++ {
		output, err := r.gen.complete(ctx, modelCall{
			purpose:     purposeCandidate,
			system:      system,
			user:        task,
			timeout:     candidateTimeout,
			temperature: &temperature,
		})
		if err != nil {
			// 补采失败不动已有结果：首轮那份带问题的草稿仍然有用，也不该因为增强失败而整体报错
			logx.WarnfContext(ctx, "[sqldraft] candidate %d unavailable, keep current result: %s", index+1, err.Error())
			break
		}
		parsed, ok := parseContract(output)
		if !ok {
			continue
		}
		taken++
		parsedChecks := r.checkStatements(ctx, parsed, false)
		if isBetterCandidate(parsedChecks, bestChecks) {
			bestStatements, bestChecks = parsed, parsedChecks
		}
	}
	if taken > 0 {
		logx.InfofContext(ctx, "[sqldraft] candidate reroll taken=%d verification=%s", taken, verificationOf(bestChecks))
	}
	return bestStatements, bestChecks, taken
}

// candidateRank 一份候选的排序键（各分量越小越好）：先比结论等级，再比 invalid 与
// unverified 的条数。做成纯函数是为了让「谁更好」这条判据能被单测钉住，
// 而不是藏在补采循环里
func candidateRank(findings []Finding) [3]int {
	rank := [3]int{0, 0, 0}
	switch verificationOf(findings) {
	case verificationInvalid:
		rank[0] = 2
	case verificationUnverified:
		rank[0] = 1
	}
	for _, finding := range findings {
		switch finding.Severity {
		case SeverityInvalid:
			rank[1]++
		case SeverityUnverified:
			rank[2]++
		}
	}
	return rank
}

// isBetterCandidate 候选是否严格优于当前结果
func isBetterCandidate(candidate, current []Finding) bool {
	a, b := candidateRank(candidate), candidateRank(current)
	for index := range a {
		if a[index] != b[index] {
			return a[index] < b[index]
		}
	}
	return false
}

// checkStatements 对每条语句跑意图声明的校验器
func (r *draftRun) checkStatements(ctx context.Context, statements []Statement, unparsed bool) []Finding {
	findings := make([]Finding, 0, 8)
	if unparsed {
		findings = append(findings, Finding{
			Severity: SeverityInvalid,
			Code:     CodeParseFailed,
			Message:  i18n.TC(ctx, imsg.ErrSqlDraftParseFailed),
		})
		return findings
	}
	for index, statement := range statements {
		stmt, stmtType := classify(r.dialect, statement.SQL)
		// 列核对的前提是「被引用表的列已装载」：上下文只装了召回表的列，
		// 而模型完全可能引用一张存在但没进上下文的表。此时若不补装，
		// columnExists 只能按「不知道」放行，草稿就会被标成「已通过校验」——
		// 那是把观测缺口说成好消息。
		// 快照负责去重，因此重复引用同一张表不会重复查询；不可见的表由快照自行跳过
		r.snap.ensureColumns(ctx, r.meta, sqlutil.StatementTables(stmt))
		cc := &checkCtx{
			snap:    r.snap,
			sql:     statement.SQL,
			stmt:    stmt,
			stmtTyp: stmtType,
			meta:    r.meta,
			dialect: r.dialect,
			conn:    r.conn,
			dbId:    r.req.DbId,
			gate:    r.gate,
			acl:     r.acl,
			// 计划只对前两条取：一条语句一次 EXPLAIN，多语句脚本逐条取会把响应拖到十几秒
			includePlans: r.req.IncludePlans && index < 2,
			sampled:      r.sampled,
			promptTables: r.promptTables,
		}
		for _, id := range r.spec.Checkers {
			checker, ok := checkerOf(id)
			if !ok {
				continue
			}
			findings = append(findings, checker.Check(ctx, cc)...)
		}
	}
	return dedupeFindings(findings)
}

// modelError 把端口返回的原始错误收敛成用户可读的业务错误。
//
// 不能把 `complete: failed to create chat completion: Post "https://...": context
// deadline exceeded` 直接抛给界面：它既不告诉用户该怎么办，还暴露了上游地址。
// 超时与其它失败分开表达——前者用户的正确动作是"稍后重试/把需求写具体"，后者才是查配置。
func modelError(ctx context.Context, err error, timeout time.Duration) error {
	if isTimeoutError(err) {
		// 不报具体秒数：真实失败可能来自模型配置里的客户端超时（比我们的 budget 更小），
		// 报一个对不上的数字会让用户以为是我们这边卡住
		_ = timeout
		return errorx.NewBizI(ctx, imsg.ErrSqlDraftModelTimeout)
	}
	return errorx.NewBizI(ctx, imsg.ErrSqlDraftModelFailed, "reason", sqlutil.BriefReason(err))
}

// isTimeoutError 超时判定：覆盖 ctx 截止（我们的 Timeout）、HTTP 客户端超时与网关超时
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	text := strings.ToLower(err.Error())
	for _, mark := range []string{"context deadline exceeded", "client.timeout", "timeout awaiting headers", "i/o timeout", "gateway timeout", "504"} {
		if strings.Contains(text, mark) {
			return true
		}
	}
	return false
}

// renderFindings 把校验结论渲染成给模型看的修复清单（含候选标识符）
func renderFindings(findings []Finding) string {
	var sb strings.Builder
	for _, finding := range findings {
		if finding.Severity == SeverityNote {
			continue
		}
		sb.WriteString("- ")
		sb.WriteString(finding.Message)
		if finding.Identifier != "" {
			sb.WriteString("（标识符：")
			sb.WriteString(finding.Identifier)
			sb.WriteString("）")
		}
		if len(finding.Candidates) > 0 {
			sb.WriteString("；可用候选：")
			sb.WriteString(strings.Join(finding.Candidates, ", "))
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// verificationOf 结论由确定性校验推导：任一 invalid 即 invalid，否则任一 unverified 即 unverified
func verificationOf(findings []Finding) string {
	verification := verificationVerified
	for _, finding := range findings {
		switch finding.Severity {
		case SeverityInvalid:
			return verificationInvalid
		case SeverityUnverified:
			verification = verificationUnverified
		}
	}
	return verification
}

func hasInvalid(findings []Finding) bool {
	for _, finding := range findings {
		if finding.Severity == SeverityInvalid {
			return true
		}
	}
	return false
}

// dedupeFindings 同一结论（级别+码+标识符+文案）只保留一条
func dedupeFindings(findings []Finding) []Finding {
	seen := make(map[string]struct{}, len(findings))
	out := make([]Finding, 0, len(findings))
	for _, finding := range findings {
		key := string(finding.Severity) + "|" + finding.Code + "|" + finding.Identifier + "|" + finding.Message
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, finding)
	}
	return out
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
