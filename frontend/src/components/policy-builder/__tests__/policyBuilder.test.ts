/**
 * 触发策略构建器交互回归。
 *
 * 钉住四条不变式：
 * 1. 组件直接读写宿主传入的策略草稿，不保存第二份条件状态（界面回显即提交内容）
 * 2. 检查项勾选状态 = 配置是否存在于策略里，取消勾选即整条移除
 * 3. 换字段/换操作符时按期望值形态重置取值，不残留与新形态不匹配的结构
 * 4. 元素清单完全来自后端 schema，未注册场景下编辑器只读而不是渲染空壳
 */
import { describe, expect, it, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { reactive } from 'vue';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';

import ProcdefPolicyEditor from '@/views/flow/components/ProcdefPolicyEditor.vue';
import SimulateDbSelect from '@/views/flow/components/SimulateDbSelect.vue';
import DbSelectTree from '@/views/ops/db/widgets/DbSelectTree.vue';
import PolicyCheckList from '../PolicyCheckList.vue';
import PolicyConditionNode from '../PolicyConditionNode.vue';
// 库选择树的回显链路会按 id 查库资产：测试环境无后端，给空列表让它安静落地
vi.mock('@/views/ops/db/api', () => ({
    dbApi: { dbs: { request: vi.fn().mockResolvedValue({ list: [] }) } },
}));
import zhFlow from '@/i18n/zh-cn/flow';
import zhCommon from '@/i18n/zh-cn/common';
import {
    createPolicy,
    SEVERITY_DISABLED,
    SEVERITY_FORBIDDEN,
    validatePolicy,
    SEVERITY_REQUIRED,
    SEVERITY_WARNING,
    type PolicySchema,
    type RuleNode,
    type TriggerPolicy,
} from '../policyModel';

vi.mock('@/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }));

const i18n = createI18n({ legacy: false, globalInjection: true, locale: 'zh-cn', messages: { 'zh-cn': { ...zhCommon, ...zhFlow } } });

const BIZ = 'db_sql_exec_flow';

const SCHEMA: PolicySchema = {
    scenarios: [
        {
            bizType: BIZ,
            severities: [SEVERITY_WARNING, SEVERITY_REQUIRED, SEVERITY_FORBIDDEN],
            fields: [
                {
                    key: 'stmtType',
                    titleKey: 'flow.field.stmtType',
                    group: 'statement',
                    type: 'enum',
                    editorKey: 'select',
                    options: ['select', 'update', 'delete', 'ddl'],
                    ops: [
                        { name: 'eq', labelKey: 'flow.op.eq', valueKind: 'single' },
                        { name: 'in', labelKey: 'flow.op.in', valueKind: 'multi' },
                    ],
                },
                {
                    key: 'tableCount',
                    titleKey: 'flow.field.tableCount',
                    group: 'target',
                    type: 'number',
                    editorKey: 'number',
                    numeric: true,
                    options: [],
                    ops: [
                        { name: 'gt', labelKey: 'flow.op.gt', valueKind: 'single' },
                        { name: 'between', labelKey: 'flow.op.between', valueKind: 'range' },
                    ],
                },
            ],
            checks: [
                {
                    key: 'db.dml-requires-approval',
                    titleKey: 'flow.check.dmlRequiresApproval',
                    default: SEVERITY_REQUIRED,
                    params: [
                        {
                            key: 'stmtTypes',
                            titleKey: 'flow.checkParam.stmtTypes',
                            type: 'enum',
                            default: ['update'],
                            options: ['update', 'delete'],
                            required: true,
                        },
                    ],
                },
                { key: 'db.dml-without-where', titleKey: 'flow.check.dmlWithoutWhere', default: SEVERITY_WARNING },
                {
                    key: 'db.sql-size-exceeds',
                    titleKey: 'flow.check.sqlSizeExceeds',
                    default: SEVERITY_WARNING,
                    params: [
                        { key: 'maxKb', titleKey: 'flow.checkParam.maxKb', type: 'number', numeric: true, default: 2048, min: 1, max: 1048576, required: true },
                    ],
                },
            ],
            presets: [
                { key: 'standard', titleKey: 'flow.policy.presetStandard', checkKeys: ['db.dml-requires-approval'] },
                { key: 'strict', titleKey: 'flow.policy.presetStrict', checkKeys: ['db.dml-requires-approval', 'db.dml-without-where'] },
            ],
            simulateFields: [{ key: 'sql', titleKey: 'flow.simulate.sql', type: 'string', placeholderKey: 'flow.simulate.sqlPlaceholder', required: true }],
        },
    ],
};

// 宿主表单里的策略对象是响应式的（AutoForm 的 form），测试同样以 reactive 承载，
// 否则「就地改草稿后界面是否跟进」这条真实链路测不到
function mountEditor(policy: TriggerPolicy = reactive(createPolicy()), schema: PolicySchema | null = SCHEMA, disabled = false) {
    return mount(ProcdefPolicyEditor, {
        props: { policy, schema, disabled },
        global: { plugins: [i18n, ElementPlus] },
    });
}

// 切到「自定义条件」页签：点击页签条走真实交互，不向 EP 组件抛未声明的事件
async function openCustomTab(wrapper: ReturnType<typeof mountEditor>) {
    const tab = wrapper.findAll('.el-tabs__item').find((node) => node.text() === '自定义条件');
    expect(tab, '自定义条件页签应渲染').toBeTruthy();
    await tab!.trigger('click');
    await wrapper.vm.$nextTick();
}

describe('检查项页签', () => {
    it('勾选写入配置并带出参数默认值，取消勾选整条移除', async () => {
        const policy = createPolicy();
        const wrapper = mountEditor(policy);

        const switches = wrapper.findAllComponents({ name: 'ElSwitch' });
        await switches[0].vm.$emit('update:modelValue', true);
        await wrapper.vm.$nextTick();

        expect(policy.checks).toHaveLength(1);
        expect(policy.checks?.[0]).toMatchObject({ key: 'db.dml-requires-approval', bizType: BIZ, severity: SEVERITY_REQUIRED });
        expect(policy.checks?.[0].params).toEqual({ stmtTypes: ['update'] });

        const after = wrapper.findAllComponents({ name: 'ElSwitch' });
        await after[0].vm.$emit('update:modelValue', false);
        await wrapper.vm.$nextTick();
        expect(policy.checks).toHaveLength(0);
    });

    it('预置包由 schema 下发，套用后该场景只保留模板内的检查项', async () => {
        const policy = createPolicy();
        const wrapper = mountEditor(policy);

        const strict = wrapper.findAllComponents({ name: 'ElButton' }).find((btn: any) => btn.text().includes('严格'));
        expect(strict, '预置模板按钮应渲染').toBeTruthy();
        await strict!.trigger('click');
        await wrapper.vm.$nextTick();

        expect(policy.checks?.map((item) => item.key)).toEqual(['db.dml-requires-approval', 'db.dml-without-where']);

        const standard = wrapper.findAllComponents({ name: 'ElButton' }).find((btn) => btn.text().includes('标准'));
        await standard!.trigger('click');
        await wrapper.vm.$nextTick();
        expect(policy.checks?.map((item) => item.key)).toEqual(['db.dml-requires-approval']);
    });

    it('规则级别下拉不提供「不处置」，取消勾选才是关闭语义', () => {
        const wrapper = mountEditor();
        const labels = wrapper.findAllComponents({ name: 'ElOption' }).map((option) => String(option.props('label')));
        expect(labels).toContain('需审批');
        expect(labels).not.toContain('不处置');
    });
});

describe('多场景隔离', () => {
    it('点某张场景卡片的预置包不得清掉其它场景的手工配置', async () => {
        const twoScenarios: PolicySchema = {
            scenarios: [
                SCHEMA.scenarios[0],
                {
                    bizType: 'redis_run_cmd_flow',
                    severities: [SEVERITY_WARNING, SEVERITY_FORBIDDEN],
                    fields: [
                        {
                            key: 'cmd',
                            titleKey: 'flow.field.cmd',
                            group: 'command',
                            type: 'string',
                            editorKey: 'input',
                            ops: [{ name: 'eq', labelKey: 'flow.op.eq', valueKind: 'single' }],
                        },
                    ],
                    checks: [{ key: 'redis.dangerous-cmd', titleKey: 'flow.check.dangerousCmd', default: SEVERITY_REQUIRED }],
                    presets: [{ key: 'standard', titleKey: 'flow.policy.presetStandard', checkKeys: ['redis.dangerous-cmd'] }],
                },
            ],
        };
        const policy = createPolicy();
        const wrapper = mountEditor(policy, twoScenarios);
        // 检查项与自定义条件页签各场景一张卡（2×2），试算页签只为声明了上下文字段的场景建卡（1 张）：
        // 不可试算的场景不再留一张空卡片，改为折叠行下方的一句计数
        expect(wrapper.findAll('.scenario-card')).toHaveLength(5);

        // 三个页签常驻挂载，ElSwitch 全局序号跨页签，必须按检查项清单实例定位
        const checkLists = wrapper.findAllComponents(PolicyCheckList);
        expect(checkLists).toHaveLength(2);
        await checkLists[1].findAllComponents({ name: 'ElSwitch' })[0].vm.$emit('update:modelValue', true);
        await wrapper.vm.$nextTick();
        expect(policy.checks?.map((item) => item.key)).toEqual(['redis.dangerous-cmd']);

        // 点 DBMS 场景卡片（检查项页签第一张）上的预置包：只应影响该场景
        const dbmsCard = wrapper.findAll('.scenario-card')[0];
        const strict = dbmsCard.findAllComponents({ name: 'ElButton' }).find((btn: any) => btn.text().includes('严格'));
        expect(strict, 'DBMS 场景卡片应带预置包入口').toBeTruthy();
        await strict!.trigger('click');
        await wrapper.vm.$nextTick();

        const keys = policy.checks?.map((item) => `${item.bizType}/${item.key}`) ?? [];
        expect(keys).toContain('redis_run_cmd_flow/redis.dangerous-cmd');
        expect(keys).toContain(`${BIZ}/db.dml-requires-approval`);
        expect(keys.filter((key) => key.startsWith('redis_run_cmd_flow'))).toHaveLength(1);
    });
});

describe('自定义条件页签', () => {
    it('开关写入按场景的自定义条件，两段都空时整条移除', async () => {
        const policy = createPolicy();
        const wrapper = mountEditor(policy);
        await openCustomTab(wrapper);

        const customSwitch = wrapper.findAllComponents({ name: 'ElSwitch' }).at(-1);
        await customSwitch!.vm.$emit('update:modelValue', true);
        await wrapper.vm.$nextTick();
        expect(policy.customs?.[0]).toMatchObject({ bizType: BIZ, severity: SEVERITY_REQUIRED });

        const removeSwitch = wrapper.findAllComponents({ name: 'ElSwitch' }).at(-1);
        await removeSwitch!.vm.$emit('update:modelValue', false);
        await wrapper.vm.$nextTick();
        expect(policy.customs).toHaveLength(0);
    });

    it('开启开关即带一行条件，点添加落在草稿节点上且组件不另存副本', async () => {
        const policy = createPolicy();
        const wrapper = mountEditor(policy);
        await openCustomTab(wrapper);
        const customSwitch = wrapper.findAllComponents({ name: 'ElSwitch' }).at(-1);
        await customSwitch!.vm.$emit('update:modelValue', true);
        await wrapper.vm.$nextTick();

        // 建组即给一行：空分组会被后端判非法，开关一打开就飘红属于设计缺陷
        const when = policy.customs?.[0].when as RuleNode;
        expect(when.items).toHaveLength(1);
        expect(when.items?.[0]).toMatchObject({ kind: 'condition', field: 'stmtType', op: 'eq', value: '' });

        // 按文案定位而非 DOM 序号：分组里有条件行时，行尾删除按钮会排在添加入口之前
        const addCondition = wrapper.findAllComponents({ name: 'ElButton' }).find((btn) => btn.text().includes('添加条件') && !btn.text().includes('条件组'));
        await addCondition!.trigger('click');
        await wrapper.vm.$nextTick();
        expect(when.items).toHaveLength(2);
    });

    it('嵌套分组可删除：只给添加入口会让用户被空分组卡死', async () => {
        const root = reactive<RuleNode>({ kind: 'group', logic: 'all', items: [] });
        const wrapper = mount(PolicyConditionNode, {
            props: { node: root, scenario: SCHEMA.scenarios[0] },
            global: { plugins: [i18n, ElementPlus] },
        });
        const addGroup = wrapper.findAllComponents({ name: 'ElButton' }).find((btn) => btn.text().includes('添加条件组'));
        await addGroup!.trigger('click');
        await wrapper.vm.$nextTick();
        expect(root.items?.[0]).toMatchObject({ kind: 'group', items: [] });

        const nested = wrapper.findAll('.policy-condition-tree')[1];
        const remove = wrapper.findAllComponents({ name: 'ElButton' }).find((btn) => btn.text().includes('删除本组'));
        expect(remove, '子分组必须有删除入口').toBeTruthy();
        expect(nested.exists()).toBe(true);
        await remove!.trigger('click');
        await wrapper.vm.$nextTick();
        expect(root.items).toHaveLength(0);
    });

    it('换操作符按期望值形态重置取值，避免残留被后端判为非法', async () => {
        const node = reactive<RuleNode>({ kind: 'condition', field: 'stmtType', op: 'eq', value: 'update' });
        const wrapper = mount(PolicyConditionNode, {
            props: { node, scenario: SCHEMA.scenarios[0] },
            global: { plugins: [i18n, ElementPlus] },
        });

        const selects = wrapper.findAllComponents({ name: 'ElSelect' });
        await selects[1].vm.$emit('update:modelValue', 'in');
        await wrapper.vm.$nextTick();

        expect(node.op).toBe('in');
        expect(node.value).toEqual([]);
    });

    it('嵌套分组递归渲染并就地增删子项', async () => {
        const root = reactive<RuleNode>({ kind: 'group', logic: 'all', items: [] });
        const wrapper = mount(PolicyConditionNode, {
            props: { node: root, scenario: SCHEMA.scenarios[0] },
            global: { plugins: [i18n, ElementPlus] },
        });

        const addGroup = wrapper.findAllComponents({ name: 'ElButton' }).find((btn) => btn.text().includes('添加条件组'));
        await addGroup!.trigger('click');
        await wrapper.vm.$nextTick();
        expect(root.items?.[0]).toMatchObject({ kind: 'group', logic: 'all', items: [] });

        // 子分组自身也渲染出了完整的条件树节点（递归生效），并带着自己的两个添加入口
        expect(wrapper.findAll('.policy-condition-tree')).toHaveLength(2);
        expect(wrapper.findAll('.depth-1')).toHaveLength(1);
        expect(wrapper.findAll('.group-actions')).toHaveLength(2);
    });

    it('字段下拉按后端分组归类展示', async () => {
        const node = reactive<RuleNode>({ kind: 'condition', field: 'stmtType', op: 'eq', value: 'update' });
        const wrapper = mount(PolicyConditionNode, {
            props: { node, scenario: SCHEMA.scenarios[0] },
            global: { plugins: [i18n, ElementPlus] },
        });
        const groups = wrapper.findAllComponents({ name: 'ElOptionGroup' });
        expect(groups.map((group) => group.props('label'))).toEqual(['语句', '目标对象']);
    });
});

describe('试算页签', () => {
    it('必填输入未填齐前不给运行，填齐后带上当前草稿策略去求值', async () => {
        const wrapper = mountEditor();
        const tab = wrapper.findAll('.el-tabs__item').find((node) => node.text() === '试算');
        await tab!.trigger('click');
        await wrapper.vm.$nextTick();

        const run = wrapper.findAllComponents({ name: 'ElButton' }).find((btn) => btn.text().includes('运行试算'));
        expect(run, '试算按钮应渲染').toBeTruthy();
        expect(run!.props('disabled')).toBe(true);

        const input = wrapper.findComponent({ name: 'ElInput' });
        await input.vm.$emit('update:modelValue', 'UPDATE orders SET status = 2');
        await wrapper.vm.$nextTick();
        expect(
            wrapper
                .findAllComponents({ name: 'ElButton' })
                .find((btn) => btn.text().includes('运行试算'))!
                .props('disabled')
        ).toBe(false);
    });

    it('结论区不得泄漏未翻译的 i18n key，且解析取值逐项展示', async () => {
        const policy = reactive(createPolicy());
        const wrapper = mountEditor(policy);
        const tab = wrapper.findAll('.el-tabs__item').find((node) => node.text() === '试算');
        await tab!.trigger('click');
        await wrapper.vm.$nextTick();

        // 直接注入一份结果，验证渲染模板而不是网络链路
        (wrapper.vm as unknown as { simulations: Record<string, unknown> }).simulations = {
            [BIZ]: {
                decision: {
                    severity: SEVERITY_WARNING,
                    matched: true,
                    findings: [{ source: 'db.sql-size-exceeds', severity: SEVERITY_WARNING, title: 'flow.check.sqlSizeExceeds' }],
                },
                fields: { stmtType: 'update', sqlLength: 42, sqlHasWhere: true },
            },
        };
        await wrapper.vm.$nextTick();

        const result = wrapper.find('.simulate-result');
        expect(result.exists(), '结论区应渲染').toBe(true);
        // 泄漏原始 key 意味着模板引用了 i18n 里不存在的键
        expect(result.text()).not.toContain('flow.policy.');
        expect(result.text()).not.toContain('flow.check.db');
        const chips = wrapper.findAll('.field-item');
        expect(chips).toHaveLength(3);
        expect(chips.map((chip) => chip.text().trim())).toEqual(['stmtType = update', 'sqlLength = 42', 'sqlHasWhere = true']);
        // 裸 JSON 串不应再出现在界面上
        expect(result.text()).not.toContain('{"');
    });

    it('库引用用资源树选择而不是 id 输入框，选中后库名按 nameKey 自动带出', async () => {
        const schemaWithDbRef: PolicySchema = {
            scenarios: [
                {
                    ...SCHEMA.scenarios[0],
                    simulateFields: [
                        { key: 'dbId', titleKey: 'flow.simulate.dbId', type: 'number', editorKey: 'db-select', nameKey: 'db', required: true },
                        { key: 'db', titleKey: 'flow.simulate.db', type: 'string' },
                        { key: 'sql', titleKey: 'flow.simulate.sql', type: 'string', placeholderKey: 'flow.simulate.sqlPlaceholder', required: true },
                    ],
                },
            ],
        };
        const wrapper = mountEditor(createPolicy(), schemaWithDbRef);
        const tab = wrapper.findAll('.el-tabs__item').find((node) => node.text() === '试算');
        await tab!.trigger('click');
        await wrapper.vm.$nextTick();

        const fields = wrapper.findAll('.simulate-field');
        // 库 id 是内部主键，运维只知道资源叫什么名字：不能退回文本框
        expect(fields[0].findComponent({ name: 'ElInput' }).exists()).toBe(false);
        expect(wrapper.findComponent(SimulateDbSelect).exists(), '库引用应渲染资源树选择控件').toBe(true);

        // 走真实事件链：资源树选中库节点回传数字 id 与节点参数，包装层把 id 换成字符串写入 raw，
        // 库名按场景声明的 nameKey 自动带进对应输入项
        const dbTree = wrapper.findComponent(SimulateDbSelect).findComponent(DbSelectTree);
        await dbTree.vm.$emit('update:dbId', 42);
        await dbTree.vm.$emit('selectDb', { id: 42, db: 'orders' });
        await wrapper.vm.$nextTick();

        const nameInput = fields[1].findComponent({ name: 'ElInput' });
        expect(nameInput.props('modelValue')).toBe('orders');

        // raw.dbId 没被正确写成字符串时 simulateReady 不会放行：按钮可点即回填有效的间接断言
        const sqlInput = fields[2].findComponent({ name: 'ElInput' });
        await sqlInput.vm.$emit('update:modelValue', 'UPDATE orders SET status = 2');
        await wrapper.vm.$nextTick();
        const run = wrapper.findAllComponents({ name: 'ElButton' }).find((btn) => btn.text().includes('运行试算'));
        expect(run!.props('disabled')).toBe(false);
    });

    it('场景未声明试算输入时不建空卡片，只给一句计数说明', async () => {
        const withoutSimulate: PolicySchema = {
            scenarios: [{ ...SCHEMA.scenarios[0], simulateFields: undefined }],
        };
        const wrapper = mountEditor(createPolicy(), withoutSimulate);
        const tab = wrapper.findAll('.el-tabs__item').find((node) => node.text() === '试算');
        await tab!.trigger('click');
        await wrapper.vm.$nextTick();
        // 该场景不该再占一张空卡片，也不该出现任何试算输入框
        expect(wrapper.findAll('.scenario-card')).toHaveLength(2);
        expect(wrapper.findAll('.simulate-field')).toHaveLength(0);
        expect(wrapper.find('.simulate-unsupported').text()).toContain('另有 1 个场景');
    });
});

describe('回显与降级', () => {
    // 折叠是这次改动新增的界面逻辑，两条一起守：默认收起的是「没配规则的场景」，
    // 而预置包这类起步入口不能被折进隐藏区
    it('未配置规则的场景默认折叠，但预设入口留在折叠行上', async () => {
        const twoSchema: PolicySchema = {
            scenarios: [SCHEMA.scenarios[0], { ...SCHEMA.scenarios[0], bizType: 'redis_run_cmd_flow', presets: [], simulateFields: undefined }],
        };
        const wrapper = mountEditor(createPolicy(), twoSchema);
        const lists = wrapper.findAllComponents(PolicyCheckList);
        expect(lists).toHaveLength(2);

        // 空策略下两个场景都没配检查项 → 明细一律收起（v-show 折叠表现为 display: none）
        for (const list of lists) expect(list.attributes('style') ?? '').toContain('display: none');

        const firstCard = wrapper.findAll('.scenario-card')[0];
        const buttons = () => firstCard.findAllComponents({ name: 'ElButton' });
        expect(
            buttons().find((btn: any) => btn.text().includes('标准')),
            '折叠行把预置包入口藏掉了'
        ).toBeTruthy();

        // 展开一个场景不应连带展开另一个
        await buttons()
            .find((btn: any) => btn.text().includes('展开检查项'))!
            .trigger('click');
        await wrapper.vm.$nextTick();
        expect(lists[0].attributes('style') ?? '').not.toContain('display: none');
        expect(lists[1].attributes('style') ?? '').toContain('display: none');

        // 再收起应回到折叠态
        const firstCardNow = wrapper.findAll('.scenario-card')[0];
        await firstCardNow
            .findAllComponents({ name: 'ElButton' })
            .find((btn: any) => btn.text().includes('收起检查项'))!
            .trigger('click');
        await wrapper.vm.$nextTick();
        expect(lists[0].attributes('style') ?? '').toContain('display: none');
    });

    it('已配置规则的场景默认就是展开的', async () => {
        const policy = reactive(createPolicy());
        policy.checks = [{ key: 'db.dml-without-where', bizType: BIZ, severity: SEVERITY_WARNING }];
        const wrapper = mountEditor(policy);
        expect(wrapper.findAllComponents(PolicyCheckList)[0].attributes('style') ?? '').not.toContain('display: none');
    });

    it('期望值是否按数值比较只认后端下发的 numeric，不再由 type 推', () => {
        // 两个字段类型相反：bytes 是 string 但声明按数值比较，plain 是 number 却没声明 numeric。
        // 判据只能是后端下发的 numeric，否则「类型 -> 是否数值」就在前端又存了一份
        const schema: PolicySchema = {
            scenarios: [
                {
                    bizType: BIZ,
                    fields: [
                        {
                            key: 'bytes',
                            titleKey: 'flow.field.tableCount',
                            group: 'target',
                            type: 'string',
                            editorKey: 'input',
                            numeric: true,
                            ops: [{ name: 'gt', labelKey: 'flow.op.gt', valueKind: 'single' }],
                        },
                        {
                            key: 'plain',
                            titleKey: 'flow.field.tableCount',
                            group: 'target',
                            type: 'number',
                            editorKey: 'number',
                            ops: [{ name: 'gt', labelKey: 'flow.op.gt', valueKind: 'single' }],
                        },
                    ],
                    checks: [],
                },
            ],
        };

        const numericField = createPolicy();
        numericField.customs = [{ bizType: BIZ, severity: SEVERITY_REQUIRED, when: { kind: 'condition', field: 'bytes', op: 'gt', value: 'abc' } } as never];
        expect(validatePolicy(numericField, schema).map((issue) => issue.reasonKey)).toContain('flow.policy.issue.numberExpected');

        const typedField = createPolicy();
        typedField.customs = [{ bizType: BIZ, severity: SEVERITY_REQUIRED, when: { kind: 'condition', field: 'plain', op: 'gt', value: 'abc' } } as never];
        expect(validatePolicy(typedField, schema).map((issue) => issue.reasonKey)).not.toContain('flow.policy.issue.numberExpected');
    });

    it('兜底级别在该场景落不了地时，配置期就把实际生效级别说出来', async () => {
        const clampedSchema: PolicySchema = {
            scenarios: [
                {
                    bizType: BIZ,
                    severities: [SEVERITY_WARNING, SEVERITY_FORBIDDEN],
                    failClosedSeverity: SEVERITY_FORBIDDEN,
                    fields: [],
                    checks: [],
                },
            ],
        };
        // 兜底配了「需审批」，但该场景没有审批通道：运行期引擎会收敛成「禁止执行」
        const wrapper = mountEditor(createPolicy(), clampedSchema);
        await wrapper.vm.$nextTick();
        expect(wrapper.text()).toContain('该场景没有审批通道');
        expect(wrapper.text()).toContain('禁止执行');

        // 兜底本身是「不处置」时不该制造这条噪音
        const relaxed = createPolicy();
        relaxed.defaultSeverity = SEVERITY_DISABLED;
        expect(mountEditor(relaxed, clampedSchema).text()).not.toContain('该场景没有审批通道');
    });

    it('规则解读实时派生自草稿，不额外发请求', async () => {
        const policy = reactive(createPolicy());
        const wrapper = mountEditor(policy);
        expect(wrapper.find('.preview-list').text()).toContain('未配置任何规则');

        policy.checks = [{ key: 'db.dml-without-where', bizType: BIZ, severity: SEVERITY_FORBIDDEN }];
        await wrapper.vm.$nextTick();
        expect(wrapper.find('.preview-list').text()).toContain('更新/删除缺少 WHERE 条件');
        expect(wrapper.find('.preview-list').text()).toContain('禁止执行');
    });

    it('校验问题按定位逐条列出，供保存前自查', async () => {
        const policy = reactive(createPolicy());
        policy.checks = [{ key: 'db.sql-size-exceeds', bizType: BIZ, severity: SEVERITY_WARNING, params: { maxKb: 0 } }];
        const wrapper = mountEditor(policy);
        const issues = wrapper.find('.policy-issues').text();
        // 定位要说人话：管理员要知道是哪条规则、哪个参数，而不是去读后端字段路径
        expect(issues).toContain('SQL 体积超过上限');
        expect(issues).toContain('参数不得小于 1');
        expect(issues).not.toContain('checks[');
    });

    it('schema 为空时不渲染任何场景卡片，避免留下无法配置的壳', () => {
        const wrapper = mountEditor(reactive(createPolicy()), null);
        expect(wrapper.find('.policy-check-list').exists()).toBe(false);
        expect(wrapper.findAll('.scenario-card')).toHaveLength(0);
        // 兜底级别仍是有效旋钮，与场景无关，因此保留可用
        expect(wrapper.find('.policy-default').exists()).toBe(true);
    });
});
