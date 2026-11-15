package sqldraft

// 扩展点：意图 / 召回 / 片段 / 校验四个接缝的注册表。
//
// 为什么做成注册表而不是在管线里写分支：这四个轴的变化原因彼此独立（新增一种意图、
// 换一种召回策略、多一段上下文、多一条校验），写进分支会让管线随功能线性变长；
// 注册表让「新增一个形态 = 注册一行」，管线只负责装配与执行。
//
// 门槛：抽象只在已有 ≥2 个真实实现时成立。当前意图 5 个、召回 4 个、片段 5 个、
// 校验 8 个，均为真实形态；不要为「将来可能需要」再造空接缝。
//
// 反面约束（本包踩过两次）：策略知识不得留在编排层。
//   - 「同后缀同族表补齐」曾写成 draftRun 的方法并无条件执行，新增一种召回就得改管线；
//   - 「示例片段需要先取数」曾写成管线里的 `if id == "examples"`，新增一种带材料的片段同理。
// 两者都已回归接缝：召回策略自声明阶段，片段自取材料。

import (
	"context"
	"sort"
	"sync"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"
)

// Spec 意图声明：一个意图 = 一套提示词模板 + 一组召回/片段/校验 + 预算
type Spec struct {
	// Id 意图稳定标识（前端入口与请求参数都用它）
	Id string
	// SystemTemplate 系统提示词模板文件名（prompts/ 下）
	SystemTemplate string
	// TaskTemplate 任务提示词模板文件名（prompts/ 下）
	TaskTemplate string
	// Linkers 参与表召回的策略 id（顺序即同阶段内的执行顺序；阶段由策略自己声明）
	Linkers []string
	// Fragments 参与上下文装配的片段
	Fragments []string
	// Checkers 参与校验的 checkers（顺序即执行顺序）
	Checkers []string
	// MaxOutputTokens 输出上限（0 取默认）
	MaxOutputTokens int
	// MaxRepairs 自纠轮数上限（0 取默认；-1 表示不纠）
	MaxRepairs int
	// ExtraCandidates 首轮（含自纠）未通过校验时额外补采的候选数，0 表示不补采。
	//
	// 条件触发而不是无条件多采：多候选是准确率与延迟/成本的直接交换，而已通过校验的请求
	// 占多数，为它们再花一次模型不划算；真正需要的是结构与取值对不上的那部分
	ExtraCandidates int
	// OutputKind 产物形态：sql（默认，走校验与写回）| text（散文，不校验不写回）
	OutputKind string
	// AllowNoTables 允许在没定位到表时继续（行内补全宁可不给建议，也不要报错弹窗）
	AllowNoTables bool
	// PromptOptional 需求描述可以为空（只要材料里有 SQL 就能干活）。
	//
	// 面向「选中一段 SQL 直接点解释/优化」这条零输入路径：业界同类交互都不要求先打字，
	// 而本项目的 explain/optimize/fix 意图其实现阶段并不依赖描述（explain 模板此前甚至
	// 没有描述槽位）。刻意留在意图声明里而不是前端判据，是因为「能不能空」属于生成侧事实：
	// 前端放宽了、后端仍报错，用户会看到「点了没反应」
	PromptOptional bool
	// SkipRecall 完全跳过表召回，也不做表选择题。
	//
	// 解释类意图用：待解释的 SQL 就在材料里，表和列的信息随之而来；再为它跑一次
	// 「从描述里找表」既慢（真机实测多花 45s）又可能因为描述里没有表名而空转。
	SkipRecall bool
}

// 产物形态
const (
	OutputKindSQL  = "sql"
	OutputKindText = "text"
)

// normalize 填默认值：让「没写」与「写了默认值」在管线里表现一致，
// 也避免消费方到处判 `if spec.OutputKind == "" || spec.OutputKind == OutputKindSQL`
func (s Spec) normalize() Spec {
	if s.OutputKind == "" {
		s.OutputKind = OutputKindSQL
	}
	return s
}

var (
	registryMu sync.RWMutex
	intents    = make(map[string]Spec)
	linkers    = make(map[string]Linker)
	fragments  = make(map[string]Fragment)
	checkers   = make(map[string]Checker)
)

// RegisterIntent 注册意图（后注册覆盖同 id；缺省字段在此归一）
func RegisterIntent(spec Spec) {
	spec = spec.normalize()
	registryMu.Lock()
	defer registryMu.Unlock()
	intents[spec.Id] = spec
}

// IntentIds 已注册意图 id 的有序清单（供契约测试与文档核对）
func IntentIds() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	ids := make([]string, 0, len(intents))
	for id := range intents {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func specOf(id string) (Spec, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	spec, ok := intents[id]
	return spec, ok
}

// ---------- 召回 ----------

// candidate 一张候选表及其命中依据（依据进界面「为什么带上这张表」）
type candidate struct {
	Table string
	Score float64
	// Reasons 命中依据清单。刻意不是拼好的句子：多路召回命中同一张表时依据要并列展示，
	// 而连接符（「；」/ "; "）属于展示语言，只能在有 ctx 的投影层拼（见 toContextTables）。
	// 策略内部一旦自己拼串，就等于把中文标点写死进了召回逻辑。
	Reasons []string
}

// addReason 追加一条依据（同一条不重复）
func (c *candidate) addReason(reason string) {
	if reason == "" {
		return
	}
	for _, exist := range c.Reasons {
		if exist == reason {
			return
		}
	}
	c.Reasons = append(c.Reasons, reason)
}

// linkCtx 召回输入
type linkCtx struct {
	// text 参与匹配的文本（描述 + 选区，已拼接）
	text string
	// prompt 需求描述原文（text 的子集）。提议阶段策略按 text 匹配，但「命中来自描述
	// 还是来自编辑器材料」是两回事，理由文案要如实标注来源就需要单独持有描述
	prompt string
	// snap 结构快照（只含当前账号可见的表）
	snap *snapshot
	// meta 元数据读取入口：扩展类策略要按需 ensure 外键等结构细节，
	// 由策略自己向快照要（快照负责去重与缓存），编排层不预知谁需要什么
	meta metadataReader
	// picked 已确定的表（扩展阶段的基准集合；提议阶段为空）
	picked []string
	// terms 该库的业务词典条目（管线一次取好，召回与片段共用同一份）。
	//
	// 由管线而不是策略自己取：词典同时服务 glossaryLinker 与 glossaryFragment，
	// 两边各查一次会让一次生成多打一趟库，且两遍取数之间词典被改过时二者依据不一致
	terms []*entity.DbAiTerm
	// limit 期望的候选上限
	limit int
}

// LinkStage 召回阶段
type LinkStage int

const (
	// LinkStagePropose 提议阶段：只依据用户材料（描述/选区/光标语句）提出候选表。
	// 它的结果决定「描述本身够不够可信」，进而决定要不要花一次模型调用做选择题
	LinkStagePropose LinkStage = iota
	// LinkStageExpand 扩展阶段：依据「已确定的表集合」补充候选（外键邻接、同族表）。
	// 必须跑在选择题之后：选择题的结果才是集合基准，跑在前面拿到的是空集合
	LinkStageExpand
)

// Linker 表召回策略
//
// 实现约定：只提议、不决策——排序截断与去重由管线统一做，避免每个策略各自实现一套
// 合并逻辑（那正是「多加一种召回就改一次排序」的来源）。
type Linker interface {
	Id() string
	// Stage 本策略所属阶段。阶段是策略的**内在属性**（依据来自描述还是来自已选表集合），
	// 因此声明在策略上而不是由意图指定：意图作者无法把扩展类策略误排到提议阶段
	Stage() LinkStage
	Propose(ctx context.Context, lc *linkCtx) ([]candidate, error)
}

func registerLinker(l Linker) {
	registryMu.Lock()
	defer registryMu.Unlock()
	linkers[l.Id()] = l
}

func linkerOf(id string) (Linker, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	l, ok := linkers[id]
	return l, ok
}

// partitionLinkers 按声明顺序把意图启用的策略分成「提议 / 扩展」两组。
//
// 未注册的 id 静默跳过：注册表允许意图引用尚不存在的策略会让一次拼写错误变成运行期 panic，
// 而契约测试（TestIntentRegistryConsistent）已经钉住「引用的策略必须都已注册」
func partitionLinkers(ids []string) (proposers, expanders []Linker) {
	proposers = make([]Linker, 0, len(ids))
	expanders = make([]Linker, 0, 2)
	for _, id := range ids {
		linker, ok := linkerOf(id)
		if !ok {
			continue
		}
		if linker.Stage() == LinkStageExpand {
			expanders = append(expanders, linker)
			continue
		}
		proposers = append(proposers, linker)
	}
	return proposers, expanders
}

// candidateNames 候选集合的表名（扩展阶段的 picked 基准）
func candidateNames(items []candidate) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Table)
	}
	return names
}

// ---------- 片段 ----------

// fragCtx 片段渲染输入
//
// 片段可以持有**能力端口**（savedSql / sampler）并自取材料，而不是等编排层把材料填好：
// 编排层一旦要填，就得知道「哪个片段需要什么」，于是长出 `if id == "examples"` 这类分支。
// 快照负责结构类材料的按需装载与去重，端口负责外部语料，两者都不需要编排层预知。
type fragCtx struct {
	snap   *snapshot
	picked []*tableInfo
	req    *Req
	// meta 元数据读取入口：片段按需 ensure 结构细节（如关系片段要外键）
	meta metadataReader
	// savedSql 同库已存脚本仓储（可空：离线测试与未配置时为 nil，片段自行判空）
	savedSql SavedSqlExamples
	// sampled 枚举列采样值（由管线一次取好，片段不直接访问数据库）
	sampled []sampledValue
	// terms 业务词典条目（与召回共用同一份，见 linkCtx.terms）
	terms []*entity.DbAiTerm
}

// Fragment 上下文片段
//
// Priority 越小越先装配；预算不足时从大到小丢弃（先丢示例、再丢关系，保住表结构）。
type Fragment interface {
	Id() string
	Priority() int
	Render(ctx context.Context, fc *fragCtx) string
}

func registerFragment(f Fragment) {
	registryMu.Lock()
	defer registryMu.Unlock()
	fragments[f.Id()] = f
}

func fragmentOf(id string) (Fragment, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	f, ok := fragments[id]
	return f, ok
}

// ---------- 校验 ----------

// checkCtx 校验输入
type checkCtx struct {
	snap    *snapshot
	sql     string
	stmt    sqlstmt.Stmt
	stmtTyp string
	meta    metadataReader
	dialect dbi.Dialect
	// conn 仅审批闸门需要（CheckSqlsWithoutTicket 的入参）；离线测试里可为空
	conn *dbi.DbConn
	dbId uint64
	// gate 执行闸门（审批事实来源，与交互执行同源）
	gate SqlGate
	// acl 表列权限（语句级 dry 判定）
	acl TableAcl
	// includePlans 是否取执行计划（只读校验，不带 ANALYZE）
	includePlans bool
	// sampled 本次采到的枚举取值（生成后核对字面量用）
	sampled []sampledValue
	// promptTables 仅凭「需求描述」就能命中的表（小写）。用于核对写语句的目标表是否跑题：
	// 候选表里常有同义表（xxx_instance / plugin_instance），模型可能写向错误的那张
	promptTables map[string]struct{}
}

// Checker 一条确定性校验
//
// 实现约定：只产出 Finding，不修改 SQL、不阻断其他校验——是否自纠/是否允许写回
// 由管线按 Severity 统一裁决。
type Checker interface {
	Id() string
	Check(ctx context.Context, cc *checkCtx) []Finding
}

func registerChecker(c Checker) {
	registryMu.Lock()
	defer registryMu.Unlock()
	checkers[c.Id()] = c
}

func checkerOf(id string) (Checker, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	c, ok := checkers[id]
	return c, ok
}
