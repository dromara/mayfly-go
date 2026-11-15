package sqldraft

// 上下文片段：把结构事实渲染成给模型的材料。
//
// 片段是按「预算内的可选材料」建模的，而不是一段拼接大字符串：
// 每段自带优先级与渲染职责，超预算时由管线按优先级丢弃并如实告知——
// 这样新增一种材料（如示例、索引、统计信息）不需要改任何已有片段。
//
// 材料由片段**自取**：结构类走快照的按需 ensure（快照负责去重与权限裁剪），
// 外部语料走 fragCtx 上的端口。管线不预知「哪个片段需要什么」，因此新增一种
// 带材料的片段只需注册一行——这正是曾经 `if id == "examples"` 分支要消除的东西。

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"mayfly-go/pkg/logx"
)

// 示例片段的取数与裁剪参数。
//
// 放在片段自己的文件里而不是 types.go 的全局预算区：它们只服务这一个片段，
// 与「上下文总预算」不是一个层级的旋钮
const (
	// examplePoolSize 从仓储取的候选脚本数（先取一批，再按相关度挑）
	examplePoolSize = 8
	// exampleLimit 最终进上下文的示例数
	exampleLimit = 2
	// exampleMinRunes 短于此长度的脚本没有参考价值（多半是 `select 1` 之类）
	exampleMinRunes = 8
	// exampleMaxRunes 单条示例的长度上限，超出截断并标注
	exampleMaxRunes = 400
)

func init() {
	registerFragment(dialectFragment{})
	registerFragment(tablesFragment{})
	registerFragment(valuesFragment{})
	registerFragment(relationFragment{})
	registerFragment(glossaryFragment{})
	registerFragment(examplesFragment{})
}

// dialectFragment 方言事实（引用符与版本决定语法正确性，必须由服务端给出而不是让模型猜）
type dialectFragment struct{}

func (dialectFragment) Id() string { return "dialect" }

func (dialectFragment) Priority() int { return 10 }

func (dialectFragment) Render(_ context.Context, fc *fragCtx) string {
	snap := fc.snap
	var sb strings.Builder
	sb.WriteString("## 目标库\n")
	sb.WriteString("方言：")
	sb.WriteString(snap.dialectLabel())
	sb.WriteString("\n库名：")
	sb.WriteString(snap.db)
	if snap.quotePrefix != "" {
		sb.WriteString("\n标识符引用符：")
		sb.WriteString(snap.quotePrefix)
		sb.WriteString("（库名/表名/列名含关键字、空格或大小写敏感时用它包裹）")
	}
	return sb.String()
}

// tablesFragment 选中表的列结构（注释是模型理解业务语义的主要线索）
type tablesFragment struct{}

func (tablesFragment) Id() string { return "tables" }

func (tablesFragment) Priority() int { return 20 }

func (tablesFragment) Render(_ context.Context, fc *fragCtx) string {
	if len(fc.picked) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## 表结构\n")
	for _, table := range fc.picked {
		sb.WriteString("- ")
		sb.WriteString(table.Name)
		if table.Comment != "" {
			sb.WriteString("（")
			sb.WriteString(table.Comment)
			sb.WriteString("）")
		}
		sb.WriteString("\n")
		if len(table.Columns) == 0 {
			sb.WriteString("  列明细不可得（读取失败或无权限），只能引用表名\n")
			continue
		}
		for _, column := range table.Columns {
			sb.WriteString("  - ")
			sb.WriteString(column.Name)
			if column.Type != "" {
				sb.WriteString(" ")
				sb.WriteString(column.Type)
			}
			if column.PrimaryKey {
				sb.WriteString(" PK")
			}
			if column.Comment != "" {
				sb.WriteString("  // ")
				sb.WriteString(column.Comment)
			}
			sb.WriteString("\n")
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

// valuesFragment 枚举列的实际取值（消除「列对了、取值不对」这类语义错误）
//
// 真机证据：模型写 `metric = 'query_time'`，列存在、语句通过全部结构校验，
// 但库里该列的取值是 lock_wait_count/ping_fail/...，执行结果为 0 行。
type valuesFragment struct{}

func (valuesFragment) Id() string { return "values" }

func (valuesFragment) Priority() int { return 25 }

func (valuesFragment) Render(_ context.Context, fc *fragCtx) string {
	if len(fc.sampled) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## 枚举列的实际取值（采样，可能不完整）\n")
	for _, one := range fc.sampled {
		sb.WriteString("- ")
		sb.WriteString(one.Table)
		sb.WriteString(".")
		sb.WriteString(one.Column)
		sb.WriteString(": ")
		sb.WriteString(sampleLabel(one.Values))
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// relationFragment 选中表之间的外键关系（JOIN 条件的主要依据）
type relationFragment struct{}

func (relationFragment) Id() string { return "relations" }

func (relationFragment) Priority() int { return 30 }

func (relationFragment) Render(ctx context.Context, fc *fragCtx) string {
	if len(fc.picked) == 0 {
		return ""
	}
	names := make([]string, 0, len(fc.picked))
	for _, table := range fc.picked {
		names = append(names, table.Name)
	}
	// 自取外键：召回阶段的关系策略可能已经装载过一部分，快照按表去重，
	// 因此这里只会补上「被截断后才进入 picked」的那些表
	fc.snap.ensureRelations(ctx, fc.meta, names)
	if len(fc.snap.fks) == 0 {
		return ""
	}
	picked := make(map[string]struct{}, len(fc.picked))
	for _, table := range fc.picked {
		picked[strings.ToLower(table.Name)] = struct{}{}
	}
	lines := make([]string, 0, 4)
	for _, edge := range fc.snap.fks {
		if _, ok := picked[strings.ToLower(edge.FromTable)]; !ok {
			continue
		}
		if _, ok := picked[strings.ToLower(edge.ToTable)]; !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s.%s → %s.%s", edge.FromTable, edge.FromColumn, edge.ToTable, edge.ToColumn))
	}
	if len(lines) == 0 {
		return ""
	}
	return "## 表间关系\n" + strings.Join(lines, "\n")
}

// glossaryDetailRunes 单条口径说明进上下文的长度上限。
//
// 上限存在是因为词典由人写、条目数由登记决定，而这段材料每次生成都要付费：
// 一条贴了半页文档的条目不该拖慢所有请求。超出部分截断并标注，不静默丢弃整条
const glossaryDetailRunes = 200

// glossaryFragment 业务词典（语义层）：把人工登记的口径交给模型。
//
// 与 glossaryLinker 分工不同：召回只认「描述里出现的术语」（否则整本词典的表都会被拖进上下文），
// 而这里给全量条目——「金额单位是分」这类全局口径恰恰是用户不会在需求里重复说的那部分知识。
// 优先级 35：排在表结构/取值/关系之后、示例之前，即预算紧张时先丢示例再丢词典。
type glossaryFragment struct{}

func (glossaryFragment) Id() string { return "glossary" }

func (glossaryFragment) Priority() int { return 35 }

func (glossaryFragment) Render(_ context.Context, fc *fragCtx) string {
	if len(fc.terms) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, term := range fc.terms {
		name := strings.TrimSpace(term.Term)
		tables := make([]string, 0, len(term.Tables))
		for _, table := range term.Tables {
			if table = strings.TrimSpace(table); table != "" {
				tables = append(tables, table)
			}
		}
		details := strings.TrimSpace(term.Details)
		if name == "" || (len(tables) == 0 && details == "") {
			continue
		}
		if len([]rune(details)) > glossaryDetailRunes {
			details = string([]rune(details)[:glossaryDetailRunes]) + "…（已截断）"
		}
		if sb.Len() == 0 {
			sb.WriteString("## 业务词典（人工登记口径，与表名/注释的猜测冲突时以词典为准）\n")
		}
		sb.WriteString("- ")
		sb.WriteString(name)
		if len(tables) > 0 {
			sb.WriteString(" → ")
			sb.WriteString(strings.Join(tables, ", "))
		}
		if details != "" {
			sb.WriteString("：")
			sb.WriteString(details)
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// examplesFragment 同库已保存脚本作为写法示例（本项目自有语料，不用外部样本）
//
// 示例按「与本次选中表的重合度」挑选，而不是取最新两条：最新的脚本很可能属于
// 上一件事（用户刚写完另一张表的语句），与本次需求毫无关系；而表重合的脚本连
// 表名带写法都对得上，是有效得多的 few-shot。
type examplesFragment struct{}

func (examplesFragment) Id() string { return "examples" }

func (examplesFragment) Priority() int { return 40 }

func (examplesFragment) Render(ctx context.Context, fc *fragCtx) string {
	examples := loadExamples(ctx, fc.savedSql, fc.req.DbId, fc.picked)
	if len(examples) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## 同库既有写法示例（仅作风格参考，不要照抄其中的表与列）\n")
	for _, example := range examples {
		sb.WriteString("```sql\n")
		sb.WriteString(example)
		sb.WriteString("\n```\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// loadExamples 取同库已存脚本，按与选中表的重合度排序后取前 exampleLimit 条。
//
// 去掉字符串字面量：示例只教写法，不该把业务数据带给模型。
// 取数失败只记日志、返回空：示例是增强项，没有它照样能生成，不值得让整次失败。
func loadExamples(ctx context.Context, savedSql SavedSqlExamples, dbId uint64, picked []*tableInfo) []string {
	if savedSql == nil || dbId == 0 {
		return nil
	}
	raw, err := savedSql.Examples(ctx, dbId, examplePoolSize)
	if err != nil {
		logx.WarnfContext(ctx, "[sqldraft] load saved sql examples failed: %s", err.Error())
		return nil
	}
	tables := make(map[string]struct{}, len(picked))
	for _, table := range picked {
		tables[strings.ToLower(table.Name)] = struct{}{}
	}

	type scored struct {
		text string
		hits int
	}
	pool := make([]scored, 0, len(raw))
	for _, one := range raw {
		text := strings.TrimSpace(stripStringLiterals(one))
		if len([]rune(text)) < exampleMinRunes {
			continue
		}
		if runes := []rune(text); len(runes) > exampleMaxRunes {
			text = string(runes[:exampleMaxRunes]) + "\n-- ..."
		}
		pool = append(pool, scored{text: text, hits: countTableHits(text, tables)})
	}
	// 重合度优先，同重合度保持仓储给出的顺序（通常是「越近越前」）
	sort.SliceStable(pool, func(i, j int) bool { return pool[i].hits > pool[j].hits })

	out := make([]string, 0, exampleLimit)
	for _, one := range pool {
		if len(out) >= exampleLimit {
			break
		}
		out = append(out, one.text)
	}
	return out
}

// countTableHits 脚本里出现了几张本次选中的表（大小写不敏感）
func countTableHits(sql string, tables map[string]struct{}) int {
	if len(tables) == 0 {
		return 0
	}
	lower := strings.ToLower(sql)
	hits := 0
	for name := range tables {
		if strings.Contains(lower, name) {
			hits++
		}
	}
	return hits
}

// stripStringLiterals 把字符串字面量内容替换为空（保留引号骨架）
func stripStringLiterals(sql string) string {
	var sb strings.Builder
	runes := []rune(sql)
	for i := 0; i < len(runes); i++ {
		current := runes[i]
		if current != '\'' && current != '"' {
			sb.WriteRune(current)
			continue
		}
		quote := current
		sb.WriteRune(quote)
		for i++; i < len(runes); i++ {
			if runes[i] == quote {
				// 连续两个引号是转义，继续留在字面量里
				if i+1 < len(runes) && runes[i+1] == quote {
					i++
					continue
				}
				break
			}
		}
		sb.WriteRune(quote)
	}
	return sb.String()
}
