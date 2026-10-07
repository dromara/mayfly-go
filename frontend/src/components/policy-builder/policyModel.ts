/**
 * 流程触发策略模型：策略结构、按字段类型收敛的操作符、完整性判定、自然语句派生与保存前校验。
 *
 * 与后端 server/internal/flow/domain/entity/trigger_policy.go 及
 * server/internal/flow/domain/trigger/{validate,engine}.go 逐条对齐：
 * 结构、操作符语义、级别取最严、豁免优先于触发。两侧改一侧必须同步另一侧，
 * 防漂移靠 __tests__/policyModel.test.ts 与后端 validate_test.go 中重合的判定表。
 *
 * 组件只依赖本模块，不直接依赖后端注册表形状：字段字典与检查项清单由
 * GET /flow/procdefs/policy-schema 下发（见 types.ts 的 PolicySchema），
 * 因此新增业务场景（Mongo/ES/机器等）时本模块与组件零改动。
 */

/** 处置级别：0 不处置 / 1 仅提醒 / 2 必须审批 / 3 禁止执行，值越大约束越强 */
export type Severity = 0 | 1 | 2 | 3;

export const SEVERITY_DISABLED: Severity = 0;
export const SEVERITY_WARNING: Severity = 1;
export const SEVERITY_REQUIRED: Severity = 2;
export const SEVERITY_FORBIDDEN: Severity = 3;

/** 可配置到单条规则上的级别。「不处置」由「规则不出现在配置里」表达，避免同一语义两种写法 */
// 会真正改变处置结果的级别集合（不含「不处置」）。只用于校验策略级兜底是个有效级别；
// 某个场景能配哪些级别一律取 scenario.severities，不从这里的常量推

export const SEVERITY_LABEL_KEYS: Record<Severity, string> = {
    0: 'flow.severity.disabled',
    1: 'flow.severity.warning',
    2: 'flow.severity.required',
    3: 'flow.severity.forbidden',
};

// 级别徽标色：需审批=主色、仅提醒=橙、禁止执行=红都是「有处置」；不处置代表该场景未被策略纳管，
// 取中性灰而不是 success 绿——绿色会被读成「已确认安全」，与它的实际含义相反
export const SEVERITY_TAG_TYPES: Record<Severity, 'info' | 'warning' | 'primary' | 'danger'> = {
    0: 'info',
    1: 'warning',
    2: 'primary',
    3: 'danger',
};

/** 字段值类型，与后端 trigger.FieldType 一致；未列出的类型按文本降级处理 */
export type FieldKind = 'enum' | 'string' | 'number' | 'bool' | 'list<string>' | 'list<number>' | 'time' | 'duration';

/** 操作符期望值形态，决定值编辑器 */
export type ValueKind = 'none' | 'single' | 'multi' | 'range';

export type NodeKind = 'group' | 'condition' | 'segment';

export type GroupLogic = 'all' | 'any' | 'none';

// ── 后端下发的 schema ────────────────────────────────────────────

export interface PolicyOperator {
    name: string;
    labelKey: string;
    valueKind: ValueKind;
}

export interface PolicyField {
    key: string;
    titleKey: string;
    group: string;
    type: FieldKind | string;
    editorKey: string;
    /** 期望值是否按数值比较，与参数侧同源由后端类型注册表下发（前端不再按 type 自判） */
    numeric?: boolean;
    options?: string[];
    ops: PolicyOperator[];
    /**
     * 数值边界，来自检查项参数的 schema 声明。
     *
     * 必须传给控件：只在前端算一遍越界提示、控件却不限制输入，管理员要填完才发现越界；
     * 而后端才是边界的唯一真源，这里只是把它如实反映到界面上
     */
    min?: number;
    max?: number;
}

export interface PolicyCheckParam {
    key: string;
    titleKey: string;
    type: FieldKind | string;
    /** 值编辑器标识，与字段字典同源由后端下发 */
    editorKey?: string;
    /** 期望值是否按数值比较（min/max 边界随之生效），同样由后端类型注册表下发 */
    numeric?: boolean;
    default?: unknown;
    min?: number;
    max?: number;
    options?: string[];
    required?: boolean;
}

export interface PolicyCheck {
    key: string;
    titleKey: string;
    descriptionKey?: string;
    default: Severity;
    params?: PolicyCheckParam[];
}

/** 试算面板的一个输入项，由后端场景注册表声明 */
export interface PolicySimulateInput {
    key: string;
    titleKey: string;
    type: FieldKind | string;
    placeholderKey?: string;
    required?: boolean;
    /** 值编辑器标识：条件字段从类型注册表推导，试算输入可由场景直接声明资源选择控件 */
    editorKey?: string;
    /** 资源选择控件选中后回填资源显示名的输入项 key（如库名），由场景声明 */
    nameKey?: string;
    /** 是否需要多行控件，由声明该输入的场景给出，不再由前端按类型猜 */
    multiline?: boolean;
}

/** 预置规则包：该场景自带的一组检查项选择，由后端注册表下发 */
export interface PolicyPreset {
    key: string;
    titleKey: string;
    checkKeys: string[];
}

/** 可复用条件组的引用候选项：条件树里的 segment 节点按 ref 引用它 */
export interface PolicySegmentRef {
    ref: string;
    name: string;
    remark?: string;
}

export interface PolicyScenario {
    bizType: string;
    /** 该场景可配置到规则上的处置级别：没有审批通道的场景不含「需审批」 */
    severities?: Severity[];
    /** 该场景能落地的最严级别：兜底级别落不了地时会收敛到这里，由引擎给出而不是前端再算一遍 */
    failClosedSeverity?: Severity;
    fields: PolicyField[];
    checks: PolicyCheck[];
    presets?: PolicyPreset[];
    simulateFields?: PolicySimulateInput[];
    /** 该场景可引用的条件组，由后端条件组表下发 */
    segments?: PolicySegmentRef[];
    /** 条件树预置写法（如或签 / 会签），由后端与字段字典同处注册，避免前端硬编码字段 key */
    conditionPresets?: ConditionPreset[];
}

/** 一个条件树预置写法：短名 + 说明 + 可直接落库的条件树 */
export interface ConditionPreset {
    key: string;
    titleKey: string;
    descriptionKey: string;
    ruleNode: RuleNode;
}

export interface PolicySchema {
    /** 可作为触发策略配置的场景 */
    scenarios: PolicyScenario[];
    /** 只服务于流程内部条件（连线跳转、节点完成）的字段字典 */
    conditionScenarios?: PolicyScenario[];
    /**
     * 已注册场景的治理路径并集，「流程定义 → 生效资源」的标签树据此筛选可勾选节点。
     *
     * 不由前端写死：写死了就变成「后端注册了新场景，管理员在资源树里根本选不到它」，
     * 策略配得再全也永远不会命中（机器命令场景就这么被关在门外过）。
     * 给的是整条路径而不是末级类型，治理粒度（库层 / 索引层）才不必在前端再映射一次
     */
    governPaths?: number[][];
}

// ── 策略结构（与后端 TriggerPolicy 一一对应）─────────────────────

export const TRIGGER_POLICY_VERSION = 1;

export interface RuleNode {
    kind: NodeKind;
    logic?: GroupLogic;
    items?: RuleNode[];
    field?: string;
    op?: string;
    value?: unknown;
    /** segment 节点引用的可复用条件组标识 */
    ref?: string;
}

export interface CheckConfig {
    key: string;
    bizType: string;
    severity: Severity;
    params?: Record<string, unknown>;
}

export interface CustomCondition {
    bizType: string;
    severity: Severity;
    when?: RuleNode;
    unless?: RuleNode;
}

export interface TriggerPolicy {
    version: number;
    defaultSeverity: Severity;
    checks?: CheckConfig[];
    customs?: CustomCondition[];
}

export interface Finding {
    source: string;
    severity: Severity;
    title: string;
    detail?: Record<string, unknown>;
    skipped?: boolean;
}

export interface Decision {
    severity: Severity;
    findings: Finding[];
    unknown?: string[];
    exempted?: boolean;
    matched?: boolean;
    procdefId?: number;
    procdefName?: string;
}

export interface SimulateResult {
    decision: Decision;
    fields: Record<string, unknown>;
}

/** 按字段类型分派的值编辑器形态；后端新增未登记类型时由编辑器降级为文本输入 */
/** 按数值比较并受 min/max 约束的参数类型，与后端 validateParamValue 的分支一致 */
/**
 * 编辑器标识 -> 具体控件。
 *
 * 键只有编辑器标识一种：字段类型到编辑器标识的换算由后端注册表负责（editorKey 随 schema 下发），
 * 前端再存一份「类型名 -> 控件」就等于第二个真源，新增字段类型时必然两边不一致
 */
export const VALUE_EDITORS: Record<string, 'select' | 'number' | 'switch' | 'tags' | 'input'> = {
    select: 'select',
    number: 'number',
    switch: 'switch',
    tags: 'tags',
    input: 'input',
};

/** 把检查项参数映射成值编辑器可消费的字段/操作符对，参数表单与条件值共用一套编辑器 */
export function paramEditor(param: PolicyCheckParam): { field: PolicyField; operator: PolicyOperator } {
    const multi = Array.isArray(param.default) || param.type === 'list<string>' || param.type === 'list<number>';
    return {
        field: {
            key: param.key,
            titleKey: param.titleKey,
            group: '',
            type: param.type,
            editorKey: param.editorKey ?? 'input',
            options: param.options,
            min: param.min,
            max: param.max,
            ops: [],
        },
        operator: { name: 'eq', labelKey: param.titleKey, valueKind: multi ? 'multi' : 'single' },
    };
}

/**
 * 场景可配置的处置级别：只能来自后端声明。
 *
 * 这里不做「没下发就按通用集合」的兜底——那等于前端替场景发明能力：
 * 没有审批通道的场景一旦漏发该字段，就会重新出现「能配出需审批、批了也没人执行」的假配置
 */
export function severitiesOf(scenario: PolicyScenario | null | undefined): Severity[] {
    return scenario?.severities ?? [];
}

/**
 * 新建规则时的默认处置级别：优先「需审批」（新增条件最常见的意图），
 * 该场景没有审批通道时退到最轻的可执行级别。
 *
 * 不写成「取 severities 的第一个」：那是依赖后端数组顺序的隐式契约，顺序一变默认值就变
 */
export function defaultRuleSeverity(scenario: PolicyScenario | null | undefined): Severity {
    const available = severitiesOf(scenario);
    if (available.includes(SEVERITY_REQUIRED)) return SEVERITY_REQUIRED;
    return available[0] ?? SEVERITY_WARNING;
}

/** 新建策略：兜底为必须审批，用于「绑定了流程但还没配置任何规则」时宁可多审 */
export function createPolicy(defaultSeverity: Severity = SEVERITY_REQUIRED): TriggerPolicy {
    return { version: TRIGGER_POLICY_VERSION, defaultSeverity };
}

export function scenarioOf(schema: PolicySchema | null, bizType: string): PolicyScenario | null {
    // 两个注册面都要找：只供条件树使用的场景（如流程实例变量）只出现在 conditionScenarios 里，
    // 只查触发场景会让条件组编辑器永远停在「加载中」——字段字典拿到的是 null，一行条件都加不出来
    const scenarios = schema?.scenarios ?? [];
    const conditions = schema?.conditionScenarios ?? [];
    return scenarios.find((item) => item.bizType === bizType) ?? conditions.find((item) => item.bizType === bizType) ?? null;
}

/**
 * cloneRuleNode 深拷贝条件树。
 *
 * 编辑抽屉不能直接持有列表行里的树对象：那是同一个引用，加一行条件就等于改了列表，
 * 保存失败后取消，界面上会留下一个从未落库的条件组（看起来像保存成功了）
 */
export function cloneRuleNode(node?: RuleNode | null): RuleNode | null {
    if (!node) {
        return null;
    }
    // structuredClone 克隆不了 Vue 的 reactive 代理（会抛 DataCloneError），这里走 JSON 往返
    return JSON.parse(JSON.stringify(node)) as RuleNode;
}

export function fieldOf(scenario: PolicyScenario | null, key?: string): PolicyField | null {
    if (!scenario || !key) return null;
    return scenario.fields.find((item) => item.key === key) ?? null;
}

export function operatorOf(field: PolicyField | null, name?: string): PolicyOperator | null {
    if (!field || !name) return null;
    return field.ops.find((item) => item.name === name) ?? null;
}

/**
 * 按操作符的期望值形态与字段类型归出初始取值。
 *
 * 布尔与数值字段必须给类型正确的初值：留空串会让条件行「看着已填、实则不完整」，
 * 用户不动那个控件就无法保存
 */
export function emptyValueFor(valueKind: ValueKind, fieldType?: FieldKind | string): unknown {
    switch (valueKind) {
        case 'multi':
            return [];
        case 'range':
            return ['', ''];
        case 'none':
            return undefined;
        case 'single':
            if (fieldType === 'bool') return false;
            if (fieldType === 'number') return 0;
            return '';
        default:
            return '';
    }
}

export function createConditionNode(field?: PolicyField): RuleNode {
    const firstOperator = field?.ops[0];
    return {
        kind: 'condition',
        field: field?.key ?? '',
        op: firstOperator?.name ?? '',
        value: emptyValueFor(firstOperator?.valueKind ?? 'single', field?.type),
    };
}

export function createGroupNode(logic: GroupLogic = 'all'): RuleNode {
    return { kind: 'group', logic, items: [] };
}

export function createSegmentNode(ref: string): RuleNode {
    return { kind: 'segment', ref };
}

/** 场景可引用的条件组；后端未下发时返回空数组，构建器据此隐藏「引用条件组」入口 */
export function segmentsOf(scenario: PolicyScenario | null | undefined): PolicySegmentRef[] {
    return scenario?.segments ?? [];
}

export function segmentOf(scenario: PolicyScenario | null, ref?: string): PolicySegmentRef | null {
    if (!scenario || !ref) return null;
    return scenario.segments?.find((item) => item.ref === ref) ?? null;
}

// ── 检查项配置读写（勾选状态 = 配置是否存在）─────────────────────

export function checkConfigOf(policy: TriggerPolicy, bizType: string, checkKey: string): CheckConfig | null {
    return policy.checks?.find((item) => item.bizType === bizType && item.key === checkKey) ?? null;
}

/** 勾选检查项：按注册表默认级别与参数默认值生成配置 */
export function enableCheck(policy: TriggerPolicy, bizType: string, check: PolicyCheck): void {
    if (checkConfigOf(policy, bizType, check.key)) return;
    const params: Record<string, unknown> = {};
    for (const param of check.params ?? []) {
        if (param.default !== undefined && param.default !== null) params[param.key] = param.default;
    }
    const config: CheckConfig = { key: check.key, bizType, severity: check.default };
    if (Object.keys(params).length > 0) config.params = params;
    policy.checks = [...(policy.checks ?? []), config];
}

export function disableCheck(policy: TriggerPolicy, bizType: string, checkKey: string): void {
    policy.checks = (policy.checks ?? []).filter((item) => !(item.bizType === bizType && item.key === checkKey));
}

export function customOf(policy: TriggerPolicy, bizType: string): CustomCondition | null {
    return policy.customs?.find((item) => item.bizType === bizType) ?? null;
}

function groupOf(...items: RuleNode[]): RuleNode {
    return { kind: 'group', logic: 'all', items };
}

export function ensureCustom(policy: TriggerPolicy, bizType: string, scenario?: PolicyScenario | null): CustomCondition {
    const existing = customOf(policy, bizType);
    if (existing) return existing;
    // 建组即给一行：空分组会被后端判非法，让开关一打开就飘红等于设计缺陷。
    // 默认级别取该场景能落地的第一个（可审批场景是「需审批」，机器是「仅提醒」），
    // 写死「需审批」会让无审批通道的场景一点开关就产出一条存不出去的配置
    const created: CustomCondition = {
        bizType,
        severity: defaultRuleSeverity(scenario),
        when: groupOf(createConditionNode(scenario?.fields?.[0])),
    };
    policy.customs = [...(policy.customs ?? []), created];
    return created;
}

/** 自定义条件两段都空时整条移除，避免留下无法求值的空壳 */
export function dropCustomIfEmpty(policy: TriggerPolicy, bizType: string): void {
    const custom = customOf(policy, bizType);
    if (custom && isNodeEmpty(custom.when) && isNodeEmpty(custom.unless)) {
        policy.customs = (policy.customs ?? []).filter((item) => item.bizType !== bizType);
    }
}

export function isNodeEmpty(node?: RuleNode | null): boolean {
    if (!node) return true;
    if (node.kind === 'group') return (node.items ?? []).every(isNodeEmpty);
    if (node.kind === 'segment') return !node.ref;
    return !node.field;
}

export function hasAnyRule(policy: TriggerPolicy): boolean {
    return (policy.checks?.length ?? 0) > 0 || (policy.customs?.length ?? 0) > 0;
}

// ── 完整性与校验 ─────────────────────────────────────────────────

export function isNodeComplete(node: RuleNode | undefined, scenario: PolicyScenario | null): boolean {
    if (!node) return false;
    if (node.kind === 'group') {
        // 刚点「添加条件」还没选字段的占位行不计入约束（那是构建器留下的空行，不是半成品），
        // 但动过的行必须填全。「提交会不会被后端拒」由 validateNode 负责，它会把残留空行也报出来
        const items = (node.items ?? []).filter((item) => !isNodeEmpty(item));
        return items.length > 0 && items.every((item) => isNodeComplete(item, scenario));
    }
    if (node.kind === 'segment') return Boolean(segmentOf(scenario, node.ref));
    const field = fieldOf(scenario, node.field);
    const operator = operatorOf(field, node.op);
    if (!field || !operator) return false;
    switch (operator.valueKind) {
        case 'none':
            return true;
        case 'multi':
            return Array.isArray(node.value) && node.value.length > 0;
        case 'range': {
            const bounds = node.value as unknown[] | undefined;
            return Array.isArray(bounds) && bounds.length === 2 && bounds.every((bound) => bound !== '' && bound !== null && bound !== undefined);
        }
        default:
            return node.value !== '' && node.value !== null && node.value !== undefined;
    }
}

export interface PolicyIssue {
    /** 出错规则定位，与后端 trigger.Error.Source 同一套语义 */
    source: string;
    /** 定位到具体控件，供表单高亮 */
    fieldKey?: string;
    reasonKey: string;
    params?: Record<string, unknown>;
}

/** 保存前校验，与后端 ValidatePolicy 同源：字段/操作符/期望值形态与类型、参数边界、分组非空 */
export function validatePolicy(policy: TriggerPolicy, schema: PolicySchema | null): PolicyIssue[] {
    const issues: PolicyIssue[] = [];

    for (const config of policy.checks ?? []) {
        const scenario = scenarioOf(schema, config.bizType);
        const source = `checks[${config.key}@${config.bizType}]`;
        if (!scenario) {
            issues.push({ source, reasonKey: 'flow.policy.issue.unknownScenario', params: { bizType: config.bizType } });
            continue;
        }
        if (!scenario.checks.some((check) => check.key === config.key)) {
            issues.push({ source, reasonKey: 'flow.policy.issue.unknownCheck' });
        }
        if (!severitiesOf(scenario).includes(config.severity)) {
            issues.push({
                source: `${source}.severity`,
                reasonKey: config.severity === SEVERITY_REQUIRED ? 'flow.policy.issue.severityNoApproval' : 'flow.policy.issue.severityNotConfigurable',
            });
        }
        for (const issue of validateCheckParams(config, scenario)) issues.push({ ...issue, source: `${source}.${issue.source}` });
    }

    const seenBizTypes = new Set<string>();
    for (const custom of policy.customs ?? []) {
        const scenario = scenarioOf(schema, custom.bizType);
        const source = `customs[${custom.bizType}]`;
        if (!scenario) {
            issues.push({ source, reasonKey: 'flow.policy.issue.unknownScenario', params: { bizType: custom.bizType } });
            continue;
        }
        if (seenBizTypes.has(custom.bizType)) issues.push({ source, reasonKey: 'flow.policy.issue.duplicateCustom' });
        seenBizTypes.add(custom.bizType);
        if (!severitiesOf(scenario).includes(custom.severity)) {
            issues.push({
                source: `${source}.severity`,
                reasonKey: custom.severity === SEVERITY_REQUIRED ? 'flow.policy.issue.severityNoApproval' : 'flow.policy.issue.severityNotConfigurable',
            });
        }
        if (isNodeEmpty(custom.when) && isNodeEmpty(custom.unless)) {
            issues.push({ source, reasonKey: 'flow.policy.issue.emptyCustom' });
        }
        for (const [part, node] of [
            ['when', custom.when],
            ['unless', custom.unless],
        ] as const) {
            if (!node) continue;
            issues.push(...validateNode(`${source}.${part}`, node, scenario));
        }
    }
    return issues;
}

function validateCheckParams(config: CheckConfig, scenario: PolicyScenario): PolicyIssue[] {
    const check = scenario.checks.find((item) => item.key === config.key);
    if (!check?.params?.length) return [];
    const issues: PolicyIssue[] = [];
    const provided = config.params ?? {};

    for (const key of Object.keys(provided)) {
        if (!check.params.some((param) => param.key === key)) {
            issues.push({ source: `params.${key}`, reasonKey: 'flow.policy.issue.unknownParam' });
        }
    }
    for (const param of check.params) {
        const value = provided[param.key];
        const empty = value === undefined || value === null || value === '' || (Array.isArray(value) && value.length === 0);
        if (empty) {
            if (param.required) issues.push({ source: `params.${param.key}`, fieldKey: param.key, reasonKey: 'flow.policy.issue.paramRequired' });
            continue;
        }
        // 哪些类型按数值比 min/max 由后端类型注册表决定并随 schema 下发；
        // 前端自己抄一份清单的话，后端新增一个数值型参数就会越界一路绿灯直到保存被拒
        if (param.numeric) {
            const numeric = Number(value);
            if (Number.isNaN(numeric)) {
                issues.push({ source: `params.${param.key}`, fieldKey: param.key, reasonKey: 'flow.policy.issue.paramNotNumber' });
                continue;
            }
            if (param.min !== undefined && numeric < param.min) {
                issues.push({ source: `params.${param.key}`, fieldKey: param.key, reasonKey: 'flow.policy.issue.paramTooSmall', params: { min: param.min } });
            }
            if (param.max !== undefined && numeric > param.max) {
                issues.push({ source: `params.${param.key}`, fieldKey: param.key, reasonKey: 'flow.policy.issue.paramTooLarge', params: { max: param.max } });
            }
        }
    }
    return issues;
}

function validateNode(source: string, node: RuleNode, scenario: PolicyScenario): PolicyIssue[] {
    const issues: PolicyIssue[] = [];
    if (node.kind === 'segment') {
        // 引用必须还能解析：条件组被删掉后这条规则在运行期只会报「无法判定」，
        // 那等于把配置错误的后果推给正在提交操作的人
        if (!segmentOf(scenario, node.ref)) {
            issues.push({ source, fieldKey: node.ref, reasonKey: 'flow.policy.issue.unknownSegment' });
        }
        return issues;
    }
    if (node.kind === 'group') {
        const items = node.items ?? [];
        // 空分组后端直接判非法，这里必须同样报错：不能因为「看着像没填」就跳过，
        // 否则界面全绿、提交被拒，用户无从知道是哪一层出了问题
        if (items.length === 0) issues.push({ source, reasonKey: 'flow.policy.issue.emptyGroup' });
        for (const [index, item] of items.entries()) {
            issues.push(...validateNode(`${source}.items[${index}]`, item, scenario));
        }
        return issues;
    }

    if (!node.field) {
        issues.push({ source, fieldKey: node.field, reasonKey: 'flow.policy.issue.fieldRequired' });
        return issues;
    }
    const field = fieldOf(scenario, node.field);
    if (!field) {
        issues.push({ source, fieldKey: node.field, reasonKey: 'flow.policy.issue.unknownField' });
        return issues;
    }
    const operator = operatorOf(field, node.op);
    if (!operator) {
        issues.push({ source: `${source}.op`, fieldKey: node.field, reasonKey: 'flow.policy.issue.operatorNotAllowed' });
        return issues;
    }
    if (!isNodeComplete(node, scenario)) {
        issues.push({ source: `${source}.value`, fieldKey: node.field, reasonKey: 'flow.policy.issue.valueRequired' });
        // 值都没填时不再判形状：否则一个空值会同时报出「必填」「不是数字」等一串无关提示
        return issues;
    }
    if (operator.valueKind === 'single' && field.type === 'bool' && typeof node.value !== 'boolean') {
        issues.push({ source: `${source}.value`, fieldKey: node.field, reasonKey: 'flow.policy.issue.boolExpected' });
    }
    if (operator.valueKind === 'single' && field.numeric === true && Number.isNaN(Number(node.value))) {
        issues.push({ source: `${source}.value`, fieldKey: node.field, reasonKey: 'flow.policy.issue.numberExpected' });
    }
    // 枚举字段只能取候选值：后端按选项集判非法，这里不拦就会出现「界面全绿、保存被拒」
    if (field.type === 'enum') {
        const candidates = field.options ?? [];
        const values = operator.valueKind === 'multi' ? (Array.isArray(node.value) ? node.value : []) : [node.value];
        for (const value of values) {
            if (!candidates.some((option) => String(option) === String(value))) {
                issues.push({ source: `${source}.value`, fieldKey: node.field, reasonKey: 'flow.policy.issue.enumInvalid', params: { value } });
            }
        }
    }
    // 正则在后端要编译才能用：拖到保存才报「无法编译」会让管理员以为是后端问题
    if (node.op === 'regex') {
        try {
            new RegExp(String(node.value ?? ''));
        } catch {
            issues.push({ source: `${source}.value`, fieldKey: node.field, reasonKey: 'flow.policy.issue.regexInvalid' });
        }
    }
    return issues;
}

/** 校验一棵独立使用的条件树（连线跳转条件、节点完成条件），与后端 ValidateCondition 同源。
 *
 * 未配置（null）是合法的：它表示默认流转。但「有壳子没内容」必须报错——
 * 后端对空分组判非法，前端若放过就会出现界面全绿、提交被拒的分叉
 */
export function validateConditionTree(node: RuleNode | null | undefined, scenario: PolicyScenario | null): PolicyIssue[] {
    if (!node || !scenario) return [];
    return validateNode('condition', node, scenario);
}

// ── 自然语句派生 ─────────────────────────────────────────────────

export type Translate = (key: string, params?: Record<string, unknown>) => string;

/** 把条件树翻成人话片段，如「语句类型 属于 [更新, 删除]」 */
export function describeNode(node: RuleNode | null | undefined, scenario: PolicyScenario | null, translate: Translate): string {
    if (!node || isNodeEmpty(node)) return '';
    if (node.kind === 'segment') {
        // 条件组名称是管理员自拟的展示文本，不是 i18n key，原样展示即可
        const segment = segmentOf(scenario, node.ref);
        return `〔${segment ? segment.name : `${node.ref}？`}〕`;
    }
    if (node.kind === 'group') {
        // 分组只负责用自己的连接词拼子句，不加书名号：
        // 书名号统一由最外层包裹，否则嵌套两层就渲染成「「…」」，多条件时整句读不通
        const parts = (node.items ?? [])
            .map((item) => {
                const text = describeNode(item, scenario, translate);
                // 含多个子句的嵌套分组必须加括号：子组的「且」并入父组的「或」列表而不加标记，
                // 读出来的语义会与真实求值不一致
                if (!text || item.kind !== 'group' || (item.items?.length ?? 0) < 2) return text;
                return `（${text}）`;
            })
            .filter(Boolean);
        if (parts.length === 0) return '';
        // 「均不满足」分组的语义是取反，句面必须把它说出来：只拼子句会把「不满足 X」读成「满足 X」，
        // 规则解读、列表悬停摘要与策略变更历史就会与真实判定相反
        if (node.logic === 'none') return translate('flow.policy.noneGroup', { conditions: parts.join(translate('flow.policy.joinAnd')) });
        if (parts.length === 1) return parts[0];
        return parts.join(translate(node.logic === 'any' ? 'flow.policy.joinOr' : 'flow.policy.joinAnd'));
    }

    const field = fieldOf(scenario, node.field);
    const operator = operatorOf(field, node.op);
    if (!field || !operator) return '';
    return `${translateOr(translate, field.titleKey, field.key)} ${translateOr(translate, operator.labelKey, operator.name)} ${describeValue(node.value, field, translate)}`;
}

/** 文案缺失时回退原始取值：后端新增枚举值/场景不应让界面出现裸 i18n key */
function translateOr(translate: Translate, key: string, fallback: string): string {
    const text = translate(key);
    return text === key ? fallback : text;
}

function describeValue(value: unknown, field: PolicyField, translate: Translate): string {
    const labelOfOption = (raw: unknown) => translateOr(translate, `flow.option.${field.key}.${String(raw)}`, String(raw));
    if (value === undefined || value === null || value === '') return '';
    if (Array.isArray(value)) {
        if (value.length === 2 && field.ops.some((op) => op.name === 'between') && !Array.isArray(value[0])) {
            return `${String(value[0])} ~ ${String(value[1])}`;
        }
        return `[${value.map((item) => labelOfOption(item)).join(', ')}]`;
    }
    if (typeof value === 'boolean') return translate(value ? 'common.yes' : 'common.no');
    if (field.type === 'enum' || field.options?.length) return labelOfOption(value);
    return String(value);
}

/** 检查项配置的一句话描述：参数必须带上，否则「指定语句类型需处置」看不出管的是哪几种语句 */
function checkRuleText(config: CheckConfig, schema: PolicySchema | null, translate: Translate): string {
    const scenario = scenarioOf(schema, config.bizType);
    const check = scenario?.checks.find((item) => item.key === config.key);
    const title = check ? translate(check.titleKey) : config.key;
    const params = Object.entries(config.params ?? {})
        .map(([key, value]) => `${key}=${Array.isArray(value) ? value.join(',') : String(value)}`)
        .join(', ');
    return `${title}${params ? ` (${params})` : ''} → ${translate(SEVERITY_LABEL_KEYS[config.severity])}`;
}

/** 自定义条件的一句话描述。摘要与变更差异共用：两处各写一遍就会出现「列表看到的」与「时间线记下的」不一致 */
function customRuleText(custom: CustomCondition, schema: PolicySchema | null, translate: Translate): string {
    const scenario = scenarioOf(schema, custom.bizType);
    const when = describeNode(custom.when, scenario, translate);
    const unless = describeNode(custom.unless, scenario, translate);
    const parts: string[] = [];
    if (when) parts.push(`${translate('flow.policy.when')} 「${when}」 → ${translate(SEVERITY_LABEL_KEYS[custom.severity])}`);
    if (unless) parts.push(`${translate('flow.policy.unless')} 「${unless}」 → ${translate('flow.policy.exempted')}`);
    return parts.join(' ');
}

/** 整份策略的一句话摘要，用于编辑页实时回显与列表页 tooltip */
export function describePolicy(policy: TriggerPolicy, schema: PolicySchema | null, translate: Translate, bizType?: string): string[] {
    const lines: string[] = [];
    const belongs = (item: string) => !bizType || item === bizType;
    for (const config of policy.checks ?? []) {
        if (!belongs(config.bizType)) continue;
        // 后端只把出现在策略里的检查项视为已配置，所以这里必须跳过未登记的项
        if (!scenarioOf(schema, config.bizType)?.checks.some((item) => item.key === config.key)) continue;
        lines.push(checkRuleText(config, schema, translate));
    }
    for (const custom of policy.customs ?? []) {
        if (!belongs(custom.bizType)) continue;
        const text = customRuleText(custom, schema, translate);
        if (text) lines.push(text);
    }
    if (lines.length === 0) {
        lines.push(`${translate('flow.policy.noRule')} → ${translate(SEVERITY_LABEL_KEYS[policy.defaultSeverity])}`);
    }
    return lines;
}

/** 策略变更差异的一行 */
export interface PolicyDiffLine {
    kind: 'added' | 'removed' | 'changed';
    text: string;
}

/** 把策略摊成「规则标识 -> 规则描述」，描述里带上处置级别，diff 才能同时反映开关与级别变化 */
function ruleMapOf(policy: TriggerPolicy | null | undefined, schema: PolicySchema | null, translate: Translate): Map<string, string> {
    const rules = new Map<string, string>();
    if (!policy) return rules;

    for (const config of policy.checks ?? []) {
        rules.set(`check:${config.bizType}:${config.key}`, checkRuleText(config, schema, translate));
    }

    for (const custom of policy.customs ?? []) {
        const text = customRuleText(custom, schema, translate);
        if (text) rules.set(`custom:${custom.bizType}`, text);
    }

    rules.set('defaultSeverity', `${translate('flow.policy.defaultSeverity')} → ${translate(SEVERITY_LABEL_KEYS[policy.defaultSeverity])}`);
    return rules;
}

/** 逐条对比前后两份策略，只输出真正变化的行。
 *
 * 未变动的规则不进时间线：一份配了十条规则的流程改了一个级别，
 * 时间线若把十条都列出来，改动本身反而被淹没在里面
 */
export function diffPolicy(
    before: TriggerPolicy | null | undefined,
    after: TriggerPolicy | null | undefined,
    schema: PolicySchema | null,
    translate: Translate
): PolicyDiffLine[] {
    const previous = ruleMapOf(before, schema, translate);
    const current = ruleMapOf(after, schema, translate);
    const lines: PolicyDiffLine[] = [];

    for (const [key, text] of current) {
        const old = previous.get(key);
        if (old === undefined) lines.push({ kind: 'added', text });
        else if (old !== text) lines.push({ kind: 'changed', text: `${old} ⇒ ${text}` });
    }
    for (const [key, text] of previous) {
        if (!current.has(key)) lines.push({ kind: 'removed', text });
    }
    return lines;
}

/** 列表页摘要徽标：场景 + 该场景下最严的处置级别。只需策略本身，无需下发 schema */
export function policySummary(policy: TriggerPolicy | null | undefined): { bizType: string; severity: Severity }[] {
    if (!policy) return [];
    const byScenario = new Map<string, Severity>();
    const raise = (bizType: string, severity: Severity) => {
        const current = byScenario.get(bizType);
        if (current === undefined || severity > current) byScenario.set(bizType, severity);
    };
    for (const config of policy.checks ?? []) raise(config.bizType, config.severity);
    for (const custom of policy.customs ?? []) {
        if (!isNodeEmpty(custom.when)) raise(custom.bizType, custom.severity);
    }
    if (byScenario.size === 0) raise('*', policy.defaultSeverity);
    return [...byScenario.entries()].map(([bizType, severity]) => ({ bizType, severity }));
}

/** 试算结论的展示级别 */
export function decisionSeverityLabel(decision: Decision, translate: Translate): string {
    return translate(SEVERITY_LABEL_KEYS[decision.severity]);
}

/** 结论是否要求提工单：与后端 Decision.RequiresApproval 同口径 */
export function decisionRequiresApproval(decision: Decision | null): boolean {
    return decision?.severity === SEVERITY_REQUIRED;
}

export function decisionIsForbidden(decision: Decision | null): boolean {
    return decision?.severity === SEVERITY_FORBIDDEN;
}

/** 试算必填项是否都已填写 */
export function simulateReady(inputs: PolicySimulateInput[] | undefined, raw: Record<string, string>): boolean {
    return (inputs ?? []).every((input) => !input.required || String(raw[input.key] ?? '').trim() !== '');
}

/** 场景名：与编辑器展示同名，避免出现第二套叫法 */
function bizTypeLabel(bizType: string, t: Translate): string {
    const key = `flow.bizTypeName.${bizType}`;
    const label = t(key);
    return label === key ? bizType : label;
}

/**
 * 把问题的机器定位路径翻成人能读懂的位置。
 *
 * PolicyIssue.source 必须保持机器可读（checks[no_where@db_sql_exec_flow].params.min），
 * 它要把问题定位回具体控件；但直接把它显示出来等于将后端字段路径丢给管理员看
 */
export function labelIssueSource(source: string, schema: PolicySchema | null, t: Translate): string {
    if (source === 'defaultSeverity') {
        return t('flow.policy.field.defaultSeverity');
    }
    const head = /^(checks|customs)\[([^\]]+)\](.*)$/.exec(source);
    if (!head) {
        return source;
    }
    const [, kind, inner, rest] = head;
    const [name, bizType = name] = inner.includes('@') ? inner.split('@') : [inner, inner];
    const scenario = scenarioOf(schema, kind === 'checks' ? bizType : name);
    const check = kind === 'checks' ? scenario?.checks.find((item) => item.key === name) : undefined;
    const labels = [bizTypeLabel(kind === 'checks' ? bizType : name, t)];
    labels.push(check ? t(check.titleKey) : kind === 'customs' ? t('flow.policy.field.customRule') : name);
    const tail = rest.replace(/^\./, '');
    if (tail) {
        labels.push(tailLabel(tail, t, check));
    }
    return labels.filter(Boolean).join(' · ');
}

function tailLabel(tail: string, t: Translate, check?: PolicyCheck): string {
    // 按段翻：定位路径可以任意嵌套（when.items[0].value、items[2].op…），
    // 逐段处理才能不管深浅都给成人话，也不会出现「翻了一半仍露出后端字段名」
    const known: Record<string, string> = {
        severity: 'flow.policy.field.severity',
        when: 'flow.policy.field.when',
        unless: 'flow.policy.field.unless',
        op: 'flow.policy.field.op',
        value: 'flow.policy.field.value',
        field: 'flow.policy.field.field',
    };
    const labels: string[] = [];
    const tokens = tail.split(/[.\]]+/).filter(Boolean);
    for (let i = 0; i < tokens.length; i++) {
        const token = tokens[i];
        const item = /^items\[(\d+)$/.exec(token);
        if (item) {
            labels.push(t('flow.policy.field.item', { index: Number(item[1]) + 1 }));
            continue;
        }
        if (token === 'params') {
            const name = tokens[++i];
            const declared = check?.params?.find((param) => param.key === name);
            labels.push(t('flow.policy.field.param', { name: declared ? t(declared.titleKey) : name }));
            continue;
        }
        // 未知段原样保留：宁可露一个字段名，也不要把「是哪一条」的信息弄丢
        labels.push(known[token] ? t(known[token]) : token);
    }
    return labels.join(' · ');
}
