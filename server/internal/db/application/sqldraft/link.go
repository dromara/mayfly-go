package sqldraft

// 表召回：把「一句中文描述」映射到几张真实的表。
//
// 这是生成准确率的头号瓶颈：中文描述里通常**不出现**任何真实表名/列名（"统计近 7 天
// 每个用户的下单金额"），而库里的表注释往往正是中文。因此召回分成两个阶段：
//
//	提议阶段（Propose）：只依据用户材料。确定性通道（描述里出现的标识符 / 注释词面相关）
//	                     + 语义通道（把「表名 + 注释」清单交给模型做一次选择题）
//	扩展阶段（Expand）：依据「已确定的表集合」补充——外键邻接（JOIN 的另一侧）、
//	                     同后缀同族表（中文"实例"对应两张 *_instance）
//
// 语义通道只在提议阶段的确定性通道没有任何强命中时才跑：有种子时省掉一次调用，
// 也避免模型把已经确定的表改掉。
//
// 阶段必须显式声明（Linker.Stage）：扩展类策略的依据是「已选表集合」，而该集合
// 只有选择题跑完才存在。曾把关系召回排在提议阶段、且 picked 取自一个只在循环结束后
// 才写入的 map，导致它恒为空、整条外键通道从未生效过——两阶段模型让这类错误无法再写出来。

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/i18n"
)

func init() {
	registerLinker(identifierLinker{})
	registerLinker(commentLinker{})
	registerLinker(glossaryLinker{})
	registerLinker(relationLinker{})
	registerLinker(siblingLinker{})
}

// identifierLinker 描述/选区里直接出现了表名
type identifierLinker struct{}

func (identifierLinker) Id() string { return "identifier" }

func (identifierLinker) Stage() LinkStage { return LinkStagePropose }

func (identifierLinker) Propose(ctx context.Context, lc *linkCtx) ([]candidate, error) {
	text := strings.ToLower(lc.text)
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	// 命中来源要如实标注：描述里出现表名、与编辑器选区/光标语句里出现表名是两回事，
	// 后者只说明光标正停在这张表的语句上。真机坐实过反例：描述里没有任何表名，
	// 命中来自光标语句，理由却仍写「描述中出现了该表名」——解释性表面不能说假话
	prompt := strings.ToLower(lc.prompt)
	descReason := i18n.TC(ctx, imsg.SqlDraftIdentifierReason)
	materialReason := i18n.TC(ctx, imsg.SqlDraftMaterialIdentifierReason)
	out := make([]candidate, 0, 4)
	for _, table := range lc.snap.visibleTables() {
		key := strings.ToLower(table.Name)
		if key == "" || !strings.Contains(text, key) {
			continue
		}
		reason := materialReason
		if prompt != "" && strings.Contains(prompt, key) {
			reason = descReason
		}
		// 名字越长越可信：短名（如 t、id）极易是普通词的一部分
		out = append(out, candidate{Table: table.Name, Score: 10 + float64(len([]rune(key)))/10, Reasons: []string{reason}})
	}
	return out, nil
}

// glossaryLinker 业务词典命中：描述里出现登记过的术语，就把该术语登记的表带进候选。
//
// 这是语义层最值钱的一半（另一半是 glossaryFragment）：存量库的表注释普遍为空，
// 「活跃实例」「坏账率」这类口径在 schema 里根本不存在，词面通道与模型选择题都无从判断，
// 真机实测就会挑错同义表。人工登记的口径因此要有明确权重：
//   - 分数 9：高于注释词面（门槛 3 字，最高约 9 分且需长公共串），低于「描述里直接写了表名」(10+)；
//   - 足以跨过 confidentScore：登记过口径的库不必再花一次模型选择题，既准又快。
//
// 术语短于 2 字（如「库」）不参与匹配：单字在中文描述里遍地命中，等于把整本词典的表都拖进上下文。
//
// 分数刻意压在 confidentScore 之下（真机回退教训）：词典命中只证明「描述里这个术语对应
// 这几张表」，不证明「描述只需要这几张表」。「统计每个数据库实例近 30 天的指标采样平均值」
// 命中了「数据库实例」，若据此跳过选表选择题，指标采样表根本进不了上下文，模型转而输出一条
// 把错误文本当数据返回的放弃式语句（`SELECT 'ERROR: No metric sampling table found ...'`）。
// 与 identifier 通道（10+ 分，用户在描述里直接点名表）的区别正在这里：
// 点名一张表是「需求已定位」的完整信号，解释一个术语不是。
// glossaryScore 词典命中的权重：高于外键邻接（6）、低于 confidentScore（8）。
//
// 低于门槛是有意的：词典不能单独宣称「描述只需要这些表」（见类型注释里的真机回退）。
const glossaryScore = 7

type glossaryLinker struct{}

func (glossaryLinker) Id() string { return "glossary" }

func (glossaryLinker) Stage() LinkStage { return LinkStagePropose }

func (glossaryLinker) Propose(ctx context.Context, lc *linkCtx) ([]candidate, error) {
	text := strings.ToLower(lc.text)
	if strings.TrimSpace(text) == "" || len(lc.terms) == 0 {
		return nil, nil
	}
	out := make([]candidate, 0, 4)
	for _, item := range lc.terms {
		term := strings.TrimSpace(item.Term)
		if len([]rune(term)) < 2 || !strings.Contains(text, strings.ToLower(term)) {
			continue
		}
		reason := i18n.TC(ctx, imsg.SqlDraftGlossaryReason, "term", term)
		for _, table := range item.Tables {
			// 词典不能越过权限：登记的表若对当前账号不可见（或已被删）就跳过。
			// 快照是可见性的唯一真源，这里只查不装载
			if _, ok := lc.snap.find(table); !ok {
				continue
			}
			out = append(out, candidate{Table: table, Score: glossaryScore, Reasons: []string{reason}})
		}
	}
	return out, nil
}

// commentLinker 表名/注释与描述的词面相关（CJK 场景主要靠它）
type commentLinker struct{}

func (commentLinker) Id() string { return "comment" }

func (commentLinker) Stage() LinkStage { return LinkStagePropose }

func (commentLinker) Propose(ctx context.Context, lc *linkCtx) ([]candidate, error) {
	text := strings.ToLower(lc.text)
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	reason := i18n.TC(ctx, imsg.SqlDraftCommentReason)
	out := make([]candidate, 0, 8)
	for _, table := range lc.snap.visibleTables() {
		// 只比注释，**不比表名**：表名走 identifierLinker 的整词匹配。
		// 曾用最长公共子串比表名，`t_sys_file` 与描述里出现的 `t_sys_log` 因共享
		// 前缀 `t_sys_` 拿到 6 分而被当成相关表——表名共享前缀是常态，子串相似度
		// 在标识符上没有区分力，只会把同前缀的一整族表都拖进上下文。
		comment := strings.ToLower(strings.TrimSpace(table.Comment))
		if comment == "" {
			continue
		}
		best := longestCommonSubstring(comment, text)
		// 门槛 3：中文两字重合（「统计」「数据」）在业务描述里遍地都是，
		// 用它召回等于把整个库都带上；三字以上才具备业务含义
		if best < 3 {
			continue
		}
		score := 2 + float64(best)
		if best >= 5 {
			score += 2
		}
		out = append(out, candidate{Table: table.Name, Score: score, Reasons: []string{reason}})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > lc.limit {
		out = out[:lc.limit]
	}
	return out, nil
}

// relationLinker 已选表的外键邻接（JOIN 需要的那一侧）
type relationLinker struct{}

func (relationLinker) Id() string { return "relation" }

// Stage 扩展阶段：依据是「已选表集合」，而集合只有选择题跑完才存在
func (relationLinker) Stage() LinkStage { return LinkStageExpand }

func (relationLinker) Propose(ctx context.Context, lc *linkCtx) ([]candidate, error) {
	if len(lc.picked) == 0 {
		return nil, nil
	}
	// 外键按需向快照要：召回阶段是它的第一个消费者，关系片段是第二个，
	// 快照负责去重（同一张表只查一次）与「方言不支持」的短路
	lc.snap.ensureRelations(ctx, lc.meta, lc.picked)
	if len(lc.snap.fks) == 0 {
		return nil, nil
	}
	picked := make(map[string]struct{}, len(lc.picked))
	for _, name := range lc.picked {
		picked[strings.ToLower(name)] = struct{}{}
	}
	reason := i18n.TC(ctx, imsg.SqlDraftRelationReason)
	out := make([]candidate, 0, 4)
	for _, edge := range lc.snap.fks {
		from, to := strings.ToLower(edge.FromTable), strings.ToLower(edge.ToTable)
		_, fromPicked := picked[from]
		_, toPicked := picked[to]
		switch {
		case fromPicked && !toPicked:
			if _, visible := lc.snap.find(edge.ToTable); visible {
				out = append(out, candidate{Table: edge.ToTable, Score: 6, Reasons: []string{reason}})
			}
		case toPicked && !fromPicked:
			if _, visible := lc.snap.find(edge.FromTable); visible {
				out = append(out, candidate{Table: edge.FromTable, Score: 6, Reasons: []string{reason}})
			}
		}
	}
	return out, nil
}

// siblingLinker 把与已选表「末段词根相同」的可见表补进候选（如 *_instance）。
//
// 只做一层、有上限：同后缀同族表在需求里往往无法区分（中文"实例"对应两个英文后缀相同的表），
// 与其让模型凭一张表硬写，不如把同族表一起给它，由列结构与系统提示里的"目标表须与需求主体一致"裁决。
// 真机踩到过：需求说"实例"，模型挑了 t_ai_plugin_instance，而 t_db_instance 根本没进上下文。
type siblingLinker struct{}

func (siblingLinker) Id() string { return "sibling" }

func (siblingLinker) Stage() LinkStage { return LinkStageExpand }

func (siblingLinker) Propose(ctx context.Context, lc *linkCtx) ([]candidate, error) {
	if len(lc.picked) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(lc.picked)+4)
	suffixes := make(map[string]struct{}, 4)
	for _, name := range lc.picked {
		seen[strings.ToLower(name)] = struct{}{}
		if suffix := tableSuffix(name); suffix != "" {
			suffixes[suffix] = struct{}{}
		}
	}
	if len(suffixes) == 0 {
		return nil, nil
	}
	reason := i18n.TC(ctx, imsg.SqlDraftSiblingReason)
	out := make([]candidate, 0, 3)
	for _, table := range lc.snap.visibleTables() {
		if len(out) >= maxSiblingTables || len(lc.picked)+len(out) >= lc.limit {
			break
		}
		key := strings.ToLower(table.Name)
		if _, ok := seen[key]; ok {
			continue
		}
		if _, ok := suffixes[tableSuffix(table.Name)]; !ok {
			continue
		}
		out = append(out, candidate{Table: table.Name, Score: 3, Reasons: []string{reason}})
		seen[key] = struct{}{}
	}
	return out, nil
}

// tableSuffix 取表名末段词根（`t_db_instance` → instance）；过短或无下划线返回空，
// 避免 `_id`/`_v2` 这类无意义尾段把无关表都拖进来
func tableSuffix(name string) string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(name)), "_")
	if len(parts) < 2 {
		return ""
	}
	suffix := parts[len(parts)-1]
	if len(suffix) < 4 {
		return ""
	}
	return suffix
}

// chooseTables 语义通道：把「表名 + 注释」清单交给模型做一次选择题。
//
// 返回空切片表示模型没给出可识别的表名（解析失败或全部不可见），由管线按「无表可用」
// 处理——这里不做关键词兜底猜测，猜错比不给更糟。
//
// meta 用于给「注释为空」的候选补一段列名预览：本项目真实库（表注释普遍为空）里
// t_db_instance 与 t_ai_plugin_instance 光看名字无从区分，模型只能瞎挑一张；
// 列出前几列名后选择题的区分力显著提升。只对注释缺失的卡片补，避免无谓的元数据往返。
func chooseTables(ctx context.Context, gen *generator, snap *snapshot, meta metadataReader, prompt string, limit int) ([]string, []string, error) {
	tables := snap.visibleTables()
	warnings := make([]string, 0, 1)
	if len(tables) == 0 {
		return nil, warnings, nil
	}
	// 先按描述相关性预筛，再截断：直接取前 N 张会让「相关的那张表恰好排在后面」时选不出来
	cards := rankTablesForPrompt(tables, prompt)
	if len(cards) > maxTableCards {
		warnings = append(warnings, i18n.TC(ctx, imsg.ErrSqlDraftChooserTruncated, "total", len(tables), "max", maxTableCards))
		cards = cards[:maxTableCards]
	}
	previewColumns(ctx, snap, meta, cards)

	var sb strings.Builder
	for index, table := range cards {
		sb.WriteString(fmt.Sprintf("%d. %s", index+1, table.Name))
		if table.Comment != "" {
			sb.WriteString(" -- ")
			sb.WriteString(table.Comment)
		} else if len(table.Columns) > 0 {
			sb.WriteString(" cols: ")
			sb.WriteString(columnPreview(table.Columns))
		}
		sb.WriteString("\n")
	}

	rendered, err := renderTemplate("choose_tables.md", map[string]any{
		"Prompt": prompt,
		"Cards":  sb.String(),
		"Limit":  limit,
	})
	if err != nil {
		return nil, warnings, err
	}
	text, err := gen.complete(ctx, modelCall{purpose: purposeChoose, system: chooseSystemPrompt(), user: rendered, timeout: chooseTimeout})
	if err != nil {
		return nil, warnings, err
	}

	names := parseStringArray(text)
	valid := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := snap.find(name); ok {
			valid = append(valid, name)
		}
	}
	return valid, warnings, nil
}

// previewColumns 给注释为空的卡片装载列明细（有上限：这是召回阶段的额外开销，
// 不能为了选择题把半个库的列都拉一遍）。
//
// 装载走快照的 ensureColumns，因此选中后渲染上下文时不会重复查询。
func previewColumns(ctx context.Context, snap *snapshot, meta metadataReader, cards []*tableInfo) {
	targets := make([]string, 0, maxColumnPreviewTables)
	for _, table := range cards {
		if len(targets) >= maxColumnPreviewTables {
			break
		}
		if strings.TrimSpace(table.Comment) != "" || len(table.Columns) > 0 {
			continue
		}
		targets = append(targets, table.Name)
	}
	if len(targets) == 0 {
		return
	}
	snap.ensureColumns(ctx, meta, targets)
}

// columnPreview 列名预览（最多 maxColumnPreviewCols 列，超出以 ... 收尾）
func columnPreview(columns []columnInfo) string {
	names := make([]string, 0, maxColumnPreviewCols)
	for _, column := range columns {
		if len(names) >= maxColumnPreviewCols {
			names = append(names, "...")
			break
		}
		names = append(names, column.Name)
	}
	return strings.Join(names, ", ")
}

// bestScore 合并结果里的最高分（决定是否还需要一次语义召回）
func bestScore(candidates []candidate) float64 {
	best := 0.0
	for _, item := range candidates {
		if item.Score > best {
			best = item.Score
		}
	}
	return best
}

// rankTablesForPrompt 按与描述的词面相关性排序候选表（稳定排序，同分保持原顺序）。
//
// 只用于「交给模型选择题」的候选排序，不改变最终召回判定：目的是让被截断时留下的是
// 更可能有用的表，而不是让模型少看一堆无关表。
func rankTablesForPrompt(tables []*tableInfo, prompt string) []*tableInfo {
	text := strings.ToLower(prompt)
	if strings.TrimSpace(text) == "" {
		return tables
	}
	type scored struct {
		table *tableInfo
		score int
	}
	list := make([]scored, 0, len(tables))
	for _, table := range tables {
		score := 0
		name := strings.ToLower(table.Name)
		if strings.Contains(text, name) {
			score += 10
		}
		if comment := strings.ToLower(strings.TrimSpace(table.Comment)); comment != "" {
			if n := longestCommonSubstring(comment, text); n >= 2 {
				score += n
			}
		}
		list = append(list, scored{table: table, score: score})
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].score > list[j].score })
	out := make([]*tableInfo, 0, len(list))
	for _, one := range list {
		out = append(out, one.table)
	}
	return out
}

// mergeCandidates 合并多路召回：同名取最高分并合并依据（依据进界面「为什么带上这张表」）
func mergeCandidates(groups ...[]candidate) []candidate {
	merged := make(map[string]*candidate, 8)
	order := make([]string, 0, 8)
	for _, group := range groups {
		for _, item := range group {
			key := strings.ToLower(strings.TrimSpace(item.Table))
			if key == "" {
				continue
			}
			existing, ok := merged[key]
			if !ok {
				copyItem := candidate{Table: item.Table, Score: item.Score, Reasons: append([]string(nil), item.Reasons...)}
				merged[key] = &copyItem
				order = append(order, key)
				continue
			}
			if item.Score > existing.Score {
				existing.Score = item.Score
			}
			for _, reason := range item.Reasons {
				existing.addReason(reason)
			}
		}
	}
	out := make([]candidate, 0, len(order))
	for _, key := range order {
		out = append(out, *merged[key])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

// longestCommonSubstring 最长公共子串长度（只用于词面相关打分，输入规模很小）
func longestCommonSubstring(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 || len(rb) == 0 {
		return 0
	}
	prev := make([]int, len(rb)+1)
	best := 0
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		for j := 1; j <= len(rb); j++ {
			if ra[i-1] == rb[j-1] {
				cur[j] = prev[j-1] + 1
				if cur[j] > best {
					best = cur[j]
				}
			}
		}
		prev = cur
	}
	return best
}
