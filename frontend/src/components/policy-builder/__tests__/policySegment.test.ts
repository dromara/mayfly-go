import { describe, expect, it } from 'vitest';

import {
    cloneRuleNode,
    createGroupNode,
    createSegmentNode,
    describeNode,
    describePolicy,
    diffPolicy,
    isNodeComplete,
    isNodeEmpty,
    scenarioOf,
    segmentsOf,
    SEVERITY_DISABLED,
    SEVERITY_FORBIDDEN,
    SEVERITY_REQUIRED,
    SEVERITY_WARNING,
    validateConditionTree,
    type PolicyScenario,
    type PolicySchema,
    type RuleNode,
    type TriggerPolicy,
} from '../policyModel';
import { conditionScenarioOf, triggerScenariosOf, usePolicySchema } from '../usePolicySchema';

// 带参的 key 把参数一并拼进去，以便断言「摘要里到底说了什么」（无参时仍返回 key）
const translate = (key: string, params?: Record<string, unknown>) => (params ? `${key}(${JSON.stringify(params)})` : key);

const numberField: PolicyScenario['fields'][number] = {
    key: 'nrOfCompleted',
    titleKey: 'flow.field.nrOfCompleted',
    group: 'approval',
    type: 'number',
    editorKey: 'number',
    ops: [{ name: 'gte', labelKey: 'flow.op.gte', valueKind: 'single' }],
};

const scenario: PolicyScenario = {
    bizType: 'flow_instance',
    fields: [numberField],
    checks: [],
    segments: [
        { ref: 'dba_members', name: 'DBA 成员' },
        { ref: 'work_hours', name: '工作时段', remark: '09:00-19:00' },
    ],
    conditionPresets: [
        {
            key: 'orSign',
            titleKey: 'flow.orSign',
            descriptionKey: 'flow.orSignTip',
            ruleNode: { kind: 'condition', field: 'nrOfCompleted', op: 'gte', value: 1 },
        },
    ],
};

const schema: PolicySchema = { scenarios: [{ ...scenario, bizType: 'db_sql_exec_flow' }], conditionScenarios: [scenario] };

describe('可复用条件组节点', () => {
    it('引用存在的条件组才算完整，引用不存在的直接判空', () => {
        expect(isNodeEmpty(createSegmentNode('dba_members'))).toBe(false);
        expect(isNodeEmpty(createSegmentNode(''))).toBe(true);
        expect(isNodeComplete(createSegmentNode('work_hours'), scenario)).toBe(true);
        expect(isNodeComplete(createSegmentNode('deleted_group'), scenario)).toBe(false);
    });

    it('描述条件组时展示管理员自拟的名称而不是标识', () => {
        expect(describeNode(createSegmentNode('dba_members'), scenario, translate)).toBe('〔DBA 成员〕');
        // 条件组被删掉后仍要能看出引用了谁，否则规则解读会显示成空白
        expect(describeNode(createSegmentNode('ghost'), scenario, translate)).toContain('ghost');
    });

    it('「均不满足」分组必须在摘要里把取反说出来', () => {
        const negated = createGroupNode('none');
        negated.items = [createSegmentNode('dba_members')];
        const text = describeNode(negated, scenario, translate);
        // 单个子句也不能丢掉分组语义：只返回「〔DBA 成员〕」会被读成「满足该条件组」，与真实判定相反
        expect(text).toContain('flow.policy.noneGroup');
        expect(text).toContain('〔DBA 成员〕');
        expect(text).not.toBe('〔DBA 成员〕');

        const many = createGroupNode('none');
        many.items = [createSegmentNode('dba_members'), createSegmentNode('work_hours')];
        const manyText = describeNode(many, scenario, translate);
        // 多项用「且」连接子句、由分组标签承担取反，而不是逐对拼「且非」（那会把「两者都不」读成「前者成立且后者不成立」）
        expect(manyText).toContain('flow.policy.joinAnd');
        expect(manyText).not.toContain('flow.policy.joinNone');
    });

    it('后端没下发条件组时候选项为空数组，构建器据此隐藏引用入口', () => {
        expect(segmentsOf(null)).toEqual([]);
        expect(segmentsOf({ ...scenario, segments: undefined })).toEqual([]);
    });

    it('条件组参与嵌套分组的完整性判定', () => {
        const tree = createGroupNode('all');
        tree.items = [createSegmentNode('dba_members')];
        expect(isNodeComplete(tree, scenario)).toBe(true);

        const broken = createGroupNode('all');
        broken.items = [createSegmentNode('gone')];
        expect(isNodeComplete(broken, scenario)).toBe(false);
    });
});

describe('单棵条件树的校验', () => {
    it('未配置是合法的（默认流转），有壳子没内容才报错', () => {
        expect(validateConditionTree(null, scenario)).toEqual([]);
        expect(validateConditionTree(undefined, scenario)).toEqual([]);
        // 空壳子与后端同口径判非法：放过就会出现界面全绿、提交被拒
        expect(validateConditionTree(createGroupNode('all'), scenario).map((issue) => issue.reasonKey)).toEqual(['flow.policy.issue.emptyGroup']);
    });

    it('空分组与被删的条件组都在保存前挡下', () => {
        const emptyGroup: RuleNode = { kind: 'group', logic: 'all', items: [] };
        expect(validateConditionTree(emptyGroup, scenario).map((issue) => issue.reasonKey)).toContain('flow.policy.issue.emptyGroup');

        const dangling = createGroupNode('all');
        dangling.items = [createSegmentNode('gone')];
        expect(validateConditionTree(dangling, scenario).map((issue) => issue.reasonKey)).toContain('flow.policy.issue.unknownSegment');
    });

    it('字典还没下发时不报错：此时界面显示加载中，而不是满屏红', () => {
        expect(validateConditionTree(createGroupNode('all'), null)).toEqual([]);
    });
});

describe('策略变更差异', () => {
    const policy = (checks: TriggerPolicy['checks'], severity = SEVERITY_REQUIRED): TriggerPolicy => ({
        version: 1,
        defaultSeverity: severity,
        checks,
    });

    it('新增、删除、改级别各自成一行，未变动的规则不进时间线', () => {
        const before = policy([{ key: 'db.dml-without-where', bizType: 'db_sql_exec_flow', severity: SEVERITY_REQUIRED }]);
        const moved = policy([{ key: 'db.dml-without-where', bizType: 'db_sql_exec_flow', severity: SEVERITY_WARNING }]);

        // 只改级别：不该出现「删除 + 新增」两行，那读起来像是换了规则
        const changed = diffPolicy(before, moved, schema, translate);
        expect(changed).toHaveLength(1);
        expect(changed[0].kind).toBe('changed');

        const removed = diffPolicy(before, policy([]), schema, translate);
        expect(removed.map((line) => line.kind)).toEqual(['removed']);

        const added = diffPolicy(policy([]), before, schema, translate);
        expect(added.some((line) => line.kind === 'added')).toBe(true);

        expect(diffPolicy(before, before, schema, translate)).toEqual([]);
    });

    it('兜底级别的变化同样要出现在时间线里', () => {
        const lines = diffPolicy(policy([], SEVERITY_REQUIRED), policy([], SEVERITY_DISABLED), schema, translate);
        expect(lines).toHaveLength(1);
        expect(lines[0].kind).toBe('changed');
        expect(lines[0].text).toContain('flow.policy.defaultSeverity');
    });

    it('删除流程时 after 为空，时间线读作「整套策略被撤掉」', () => {
        const before = policy([{ key: 'redis.dangerous-cmd', bizType: 'db_sql_exec_flow', severity: SEVERITY_REQUIRED }]);
        const lines = diffPolicy(before, null, schema, translate);
        expect(lines.every((line) => line.kind === 'removed')).toBe(true);
        expect(lines.length).toBeGreaterThan(0);
    });
});

describe('schema 的两类场景分栏', () => {
    it('触发策略编辑器只遍历 scenarios，条件编辑器按 bizType 取 conditionScenarios', () => {
        expect(triggerScenariosOf(schema).map((item: PolicyScenario) => item.bizType)).toEqual(['db_sql_exec_flow']);
        expect(conditionScenarioOf(schema, 'flow_instance')?.bizType).toBe('flow_instance');
        expect(conditionScenarioOf(schema, 'ghost')).toBeNull();
    });

    it('scenarioOf 是条件编辑器共用的取数入口，必须能取到「仅条件」场景', () => {
        // 曾经它只查 scenarios：条件组页面按 flow_instance 取到 null，
        // 字段字典为空、一行条件都加不出来，界面永远停在「加载中」
        expect(scenarioOf(schema, 'flow_instance')?.bizType).toBe('flow_instance');
        expect(scenarioOf(schema, 'flow_instance')?.fields.map((item) => item.key)).toEqual(['nrOfCompleted']);
        expect(scenarioOf(schema, 'db_sql_exec_flow')?.bizType).toBe('db_sql_exec_flow');
        expect(scenarioOf(schema, 'ghost')).toBeNull();
        expect(scenarioOf(null, 'flow_instance')).toBeNull();
    });

    it('后端未下发 conditionScenarios 时不炸：新字段是可选的', () => {
        const legacy: PolicySchema = { scenarios: [] };
        expect(triggerScenariosOf(legacy)).toEqual([]);
        expect(conditionScenarioOf(legacy, 'flow_instance')).toBeNull();
    });
});

describe('policy-schema 缓存', () => {
    it('同一会话内只请求一次；条件组增删后 reload 立即重取并让宿主读到新字典', async () => {
        let calls = 0;
        const loader = async (): Promise<PolicySchema> => {
            calls += 1;
            return { scenarios: [], conditionScenarios: [] };
        };

        const first = usePolicySchema(loader);
        await first.load();
        await first.load();
        expect(calls).toBe(1);

        // 只清缓存不算修好：宿主组件已经挂载，onMounted 不会再跑，界面会永远停在加载中
        const second = usePolicySchema(loader);
        await second.reload();
        expect(calls).toBe(2);
        expect(second.policySchema.value).not.toBeNull();
    });
});

describe('按场景出规则解读', () => {
    it('列表浮层要能分清每条规则属于哪个场景', () => {
        const twoScenarios: TriggerPolicy = {
            version: 1,
            defaultSeverity: SEVERITY_REQUIRED,
            checks: [
                { key: 'a-check', bizType: segTestScenario.bizType, severity: SEVERITY_REQUIRED },
                { key: 'b-check', bizType: 'redis_run_cmd_flow', severity: SEVERITY_FORBIDDEN },
            ],
        };
        const schema: PolicySchema = {
            scenarios: [
                { ...segTestScenario, checks: [{ key: 'a-check', titleKey: 'flow.check.a', default: SEVERITY_REQUIRED }] },
                { ...segTestScenario, bizType: 'redis_run_cmd_flow', checks: [{ key: 'b-check', titleKey: 'flow.check.b', default: SEVERITY_REQUIRED }] },
            ],
        };

        const all = describePolicy(twoScenarios, schema, (key) => key);
        expect(all).toHaveLength(2);
        // 不带 bizType 时行为不变：编辑页的场景卡片已经给出归属，不需要重复前缀
        expect(describePolicy(twoScenarios, schema, (key) => key, segTestScenario.bizType)).toEqual(['flow.check.a → flow.severity.required']);
    });
});

const segTestScenario: PolicyScenario = { bizType: 'db_sql_exec_flow', fields: [], checks: [] };

describe('条件树的深拷贝', () => {
    const nested: RuleNode = {
        kind: 'group',
        logic: 'all',
        items: [{ kind: 'condition', field: 'nrOfCompleted', op: 'gte', value: 1 }],
    };

    it('编辑副本不会改动原树', () => {
        // 编辑抽屉若直接持有列表行里的树对象，加一行条件就等于改了列表；
        // 保存失败后取消，列表上会留下一个从未落库的条件组，看起来像已经保存成功
        const clone = cloneRuleNode(nested);
        clone!.items!.push({ kind: 'segment', ref: 'dba_members' });

        expect(nested.items?.length).toBe(1);
        expect(clone?.items?.length).toBe(2);
    });

    it('空值原样返回 null，不产出可编辑的空对象', () => {
        expect(cloneRuleNode(null)).toBeNull();
        expect(cloneRuleNode(undefined)).toBeNull();
    });
});
