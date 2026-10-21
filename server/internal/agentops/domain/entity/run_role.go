package entity

import "strings"

// RunRole 运行的「形态」：同一种触发方式（TriggerType）下，这条运行在多阶段流程里扮演什么角色。
//
// 为什么不把形态写进标题前缀（旧做法 `[编排]`、`[评测回放]` …）：标题是**展示数据**，中文前缀
// 一旦落库，英文界面就永远看到中文（与 headless 会话标题曾犯的同一错误，见 headless.go 的
// 「不要按来源编中文标题」规则），而且任何拿 `title LIKE '[评测%'` 当分类谓词的地方（存量迁移）
// 都会因为改一次措辞而静默漏判。分类的唯一依据必须是结构化字段，展示文案由前端按 role 取 i18n。
//
// 与两个近邻概念的分工（三者不可互相派生）：
//   - TriggerType（本实体独立列）：谁发起的 —— cron/event/manual/orchestration，可查询可筛选；
//   - headless.RunKind（ai 侧）：无头运行的来源，与 TriggerType 同一词汇、显式映射
//     （见 application.triggerTypeToRunKind），**不是**本字段，别混用同名；
//   - RunRole（本字段）：流程内部的角色（规划/步骤/审查、执行/校验…），只在多阶段流程里有意义。
//     它从来不是查询/统计维度，因此按 docs/server/domain.md 收敛在 Extra，不加数据库列。
const (
	// RunRoleEvalReplay 离线评测：用诊断档重放原始输入（打分那轮不落 Run 行，故无对应角色）
	RunRoleEvalReplay = "eval_replay"
	// RunRoleOrchParent 编排父运行（承载最终报告，不做实际工作）
	RunRoleOrchParent = "orch_parent"
	// RunRoleOrchPlan 编排：规划阶段
	RunRoleOrchPlan = "orch_plan"
	// RunRoleOrchStep 编排：单个步骤执行
	RunRoleOrchStep = "orch_step"
	// RunRoleOrchReview 编排：审查阶段
	RunRoleOrchReview = "orch_review"
	// RunRoleExec 提案批准后的受控执行
	RunRoleExec = "exec"
	// RunRoleVerify 受控执行后的只读校验
	RunRoleVerify = "verify"
)

// LegacyRunRoleTitlePrefixes 历史数据里「标题前缀 → 形态」的一次性识别表。
//
// 只服务存量回填/展示兜底两处，**新代码不得再按标题分类**：写入侧已全部改成直接置 RunRole，
// 且标题不再带前缀。清单只列真实落过 Run 行的前缀（实测 `[评测打分]` 只作为无头会话标题存在、
// 从未落成 Run 行，因此不在表内——留着就是永远不会命中的死分支）。
//
// 顺序敏感：`[编排步骤 ` 必须在 `[编排] ` 之前匹配，否则步骤运行会被误判成父运行。
//
// StripThrough 处理「前缀里含可变内容」的形态：步骤标题历史形态是 `[编排步骤 <n>] 真实标题`，
// 只剥固定前缀会留下 `3] ` 碎片。非空时额外剥到首次出现该串之后（含该串与其后的一个空格）。
var LegacyRunRoleTitlePrefixes = []struct {
	Prefix       string
	Role         string
	StripThrough string
}{
	{Prefix: "[编排步骤 ", Role: RunRoleOrchStep, StripThrough: "] "},
	{Prefix: "[编排规划] ", Role: RunRoleOrchPlan},
	{Prefix: "[编排审查] ", Role: RunRoleOrchReview},
	{Prefix: "[编排] ", Role: RunRoleOrchParent},
	{Prefix: "[评测回放] ", Role: RunRoleEvalReplay},
	{Prefix: "[执行] ", Role: RunRoleExec},
	{Prefix: "[校验] ", Role: RunRoleVerify},
}

// DetectLegacyRunRoleTitle 从历史标题里识别形态，返回 (形态, 去掉前缀后的裸标题)。
//
// 未命中任何前缀时返回 ("", 原标题)。仅供处理存量行使用，业务写入点直接赋 RunRole。
func DetectLegacyRunRoleTitle(title string) (role, bareTitle string) {
	for _, it := range LegacyRunRoleTitlePrefixes {
		if !strings.HasPrefix(title, it.Prefix) {
			continue
		}
		bare := strings.TrimPrefix(title, it.Prefix)
		if it.StripThrough != "" {
			if idx := strings.Index(bare, it.StripThrough); idx >= 0 {
				bare = strings.TrimPrefix(bare[idx+len(it.StripThrough):], " ")
			}
		}
		return it.Role, bare
	}
	return "", title
}

// RunRole 与 Extra 的双向同步由 run.go 的 BeforeSave/AfterFind 完成（与 OwnerUsername 同一机制）：
// VO 按名拷贝，因此接口契约不变。写入点直接给字段赋值，不提供同名 Set/Get 方法
// ——有字段又配同名方法就是两个入口，正是本模块一路在消除的双真源形态。
