import { describe, expect, it } from 'vitest';

import {
    createConditionNode,
    createGroupNode,
    createPolicy,
    customOf,
    describeNode,
    describePolicy,
    diffPolicy,
    disableCheck,
    dropCustomIfEmpty,
    emptyValueFor,
    enableCheck,
    defaultRuleSeverity,
    ensureCustom,
    fieldOf,
    hasAnyRule,
    isNodeComplete,
    isNodeEmpty,
    paramEditor,
    policySummary,
    SEVERITY_DISABLED,
    SEVERITY_FORBIDDEN,
    SEVERITY_REQUIRED,
    SEVERITY_WARNING,
    validatePolicy,
    VALUE_EDITORS,
    type PolicyField,
    type PolicySchema,
    type RuleNode,
} from '../policyModel';

// 与后端 db_sql_exec_flow 场景的注册内容保持一致（字段/操作符/检查项/参数边界）
const stmtField: PolicyField = {
    key: 'stmtType',
    titleKey: 'flow.field.stmtType',
    group: 'statement',
    type: 'enum',
    editorKey: 'select',
    options: ['select', 'read', 'insert', 'update', 'delete', 'ddl', 'other'],
    ops: [
        { name: 'eq', labelKey: 'flow.op.eq', valueKind: 'single' },
        { name: 'in', labelKey: 'flow.op.in', valueKind: 'multi' },
        { name: 'notIn', labelKey: 'flow.op.notIn', valueKind: 'multi' },
    ],
};

const SCHEMA: PolicySchema = {
    scenarios: [
        {
            bizType: 'db_sql_exec_flow',
            severities: [SEVERITY_WARNING, SEVERITY_REQUIRED, SEVERITY_FORBIDDEN],
            fields: [
                stmtField,
                {
                    key: 'sql',
                    titleKey: 'flow.field.sql',
                    group: 'statement',
                    type: 'string',
                    editorKey: 'input',
                    ops: [
                        { name: 'contains', labelKey: 'flow.op.contains', valueKind: 'single' },
                        { name: 'regex', labelKey: 'flow.op.regex', valueKind: 'single' },
                    ],
                },
                {
                    key: 'tableCount',
                    titleKey: 'flow.field.tableCount',
                    group: 'target',
                    type: 'number',
                    editorKey: 'number',
                    numeric: true,
                    ops: [
                        { name: 'gt', labelKey: 'flow.op.gt', valueKind: 'single' },
                        { name: 'between', labelKey: 'flow.op.between', valueKind: 'range' },
                    ],
                },
                {
                    key: 'dangerous',
                    titleKey: 'flow.field.dangerous',
                    group: 'risk',
                    type: 'bool',
                    editorKey: 'switch',
                    ops: [{ name: 'eq', labelKey: 'flow.op.eq', valueKind: 'single' }],
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
                            // schema 由后端带上控件标识，前端不再按类型猜（见 paramEditor）
                            editorKey: 'select',
                            default: ['insert', 'update', 'delete', 'ddl'],
                            options: ['select', 'update', 'delete', 'ddl'],
                            required: true,
                        },
                    ],
                },
                { key: 'db.dml-without-where', titleKey: 'flow.check.dmlWithoutWhere', default: SEVERITY_REQUIRED },
                {
                    key: 'db.sql-size-exceeds',
                    titleKey: 'flow.check.sqlSizeExceeds',
                    default: SEVERITY_WARNING,
                    params: [
                        {
                            key: 'maxKb',
                            titleKey: 'flow.checkParam.maxKb',
                            type: 'number',
                            editorKey: 'number',
                            // 数值型判定由后端类型注册表随 schema 下发，夹具必须与真实下发一致
                            numeric: true,
                            default: 2048,
                            min: 1,
                            max: 1048576,
                            required: true,
                        },
                    ],
                },
            ],
        },
    ],
};

const BIZ = 'db_sql_exec_flow';
const translate = (key: string, params?: Record<string, unknown>) => {
    if (params) return `${key}:${JSON.stringify(params)}`;
    return key.startsWith('flow.') ? `‹${key}›` : key;
};

describe('策略结构与勾选状态', () => {
    it('新建策略带版本号，兜底级别为需审批', () => {
        const policy = createPolicy();
        expect(policy.version).toBe(1);
        expect(policy.defaultSeverity).toBe(SEVERITY_REQUIRED);
        expect(hasAnyRule(policy)).toBe(false);
    });

    it('勾选状态以配置是否存在为唯一真源', () => {
        const policy = createPolicy();
        enableCheck(policy, BIZ, SCHEMA.scenarios[0].checks[1]);
        expect(policy.checks).toHaveLength(1);
        expect(validatePolicy(policy, SCHEMA)).toEqual([]);

        disableCheck(policy, BIZ, 'db.dml-without-where');
        expect(policy.checks).toHaveLength(0);
        expect(validatePolicy(policy, SCHEMA)).toEqual([]);
    });

    it('勾选时按参数 schema 填入默认值，重复勾选不产生第二份配置', () => {
        const policy = createPolicy();
        const check = SCHEMA.scenarios[0].checks[0];
        enableCheck(policy, BIZ, check);
        enableCheck(policy, BIZ, check);
        expect(policy.checks).toHaveLength(1);
        expect(policy.checks?.[0].params).toEqual({ stmtTypes: ['insert', 'update', 'delete', 'ddl'] });
        expect(policy.checks?.[0].bizType).toBe(BIZ);
    });

    it('自定义条件两段皆空时整条移除，不留无法求值的空壳', () => {
        const policy = createPolicy();
        ensureCustom(policy, BIZ);
        expect(customOf(policy, BIZ)).not.toBeNull();

        dropCustomIfEmpty(policy, BIZ);
        expect(customOf(policy, BIZ)).toBeNull();

        const custom = ensureCustom(policy, BIZ);
        custom.when = createConditionNode(stmtField);
        dropCustomIfEmpty(policy, BIZ);
        expect(customOf(policy, BIZ)).not.toBeNull();
    });
});

describe('条件完整性判定', () => {
    it('按操作符期望值形态判定', () => {
        const scenario = SCHEMA.scenarios[0];
        expect(isNodeComplete({ kind: 'condition', field: 'stmtType', op: 'eq', value: 'update' }, scenario)).toBe(true);
        expect(isNodeComplete({ kind: 'condition', field: 'stmtType', op: 'eq', value: '' }, scenario)).toBe(false);
        expect(isNodeComplete({ kind: 'condition', field: 'stmtType', op: 'in', value: [] }, scenario)).toBe(false);
        expect(isNodeComplete({ kind: 'condition', field: 'tableCount', op: 'between', value: [1, ''] }, scenario)).toBe(false);
        expect(isNodeComplete({ kind: 'condition', field: 'tableCount', op: 'between', value: [1, 5] }, scenario)).toBe(true);
        expect(isNodeComplete({ kind: 'condition', field: 'stmtType', op: 'ghost', value: 'x' }, scenario)).toBe(false);
    });

    it('分组要求至少一个非空子项，已动手填写的子项必须完整', () => {
        const scenario = SCHEMA.scenarios[0];
        const good: RuleNode = { kind: 'condition', field: 'stmtType', op: 'eq', value: 'update' };
        // 刚点「添加条件」还没选字段的占位行不计入约束
        const untouched: RuleNode = { kind: 'condition', field: '', op: '', value: '' };
        // 选了字段却没填完的行属于半成品，必须让整组判为不完整，否则界面与提交内容会不一致
        const halfFilled: RuleNode = { kind: 'condition', field: 'stmtType', op: '', value: '' };
        expect(isNodeComplete(createGroupNode('all'), scenario)).toBe(false);
        expect(isNodeComplete(group(good), scenario)).toBe(true);
        expect(isNodeComplete(group(good, untouched), scenario)).toBe(true);
        expect(isNodeComplete(group(good, halfFilled), scenario)).toBe(false);
        expect(isNodeComplete(group(untouched), scenario)).toBe(false);
    });

    it('空分组与只含空条件的分组都算空', () => {
        expect(isNodeEmpty(createGroupNode('all'))).toBe(true);
        expect(isNodeEmpty(group(createGroupNode('any')))).toBe(true);
        expect(isNodeEmpty(undefined)).toBe(true);
    });
});

function group(...items: RuleNode[]): RuleNode {
    return { kind: 'group', logic: 'all', items };
}

describe('保存前校验与后端同源', () => {
    const reasonKeys = (policy: Parameters<typeof validatePolicy>[0]) => validatePolicy(policy, SCHEMA).map((issue) => issue.reasonKey);

    it('完整策略通过校验', () => {
        const policy = createPolicy(SEVERITY_DISABLED);
        enableCheck(policy, BIZ, SCHEMA.scenarios[0].checks[0]);
        const custom = ensureCustom(policy, BIZ);
        custom.when = group({ kind: 'condition', field: 'stmtType', op: 'in', value: ['update', 'delete'] });
        custom.unless = { kind: 'condition', field: 'dangerous', op: 'eq', value: false };
        expect(validatePolicy(policy, SCHEMA)).toEqual([]);
    });

    it('检查项未注册 / 级别为不处置 / 参数越界都被拦下', () => {
        const policy = createPolicy();
        policy.checks = [
            { key: 'ghost.check', bizType: BIZ, severity: SEVERITY_REQUIRED },
            { key: 'db.dml-without-where', bizType: BIZ, severity: SEVERITY_DISABLED },
            { key: 'db.sql-size-exceeds', bizType: BIZ, severity: SEVERITY_WARNING, params: { maxKb: 0 } },
            { key: 'db.sql-size-exceeds', bizType: BIZ, severity: SEVERITY_WARNING, params: { ghost: 1 } },
        ];
        const keys = reasonKeys(policy);
        expect(keys).toContain('flow.policy.issue.unknownCheck');
        expect(keys).toContain('flow.policy.issue.severityNotConfigurable');
        expect(keys).toContain('flow.policy.issue.paramTooSmall');
        expect(keys).toContain('flow.policy.issue.unknownParam');
    });

    it('必填参数缺失被拦下', () => {
        const policy = createPolicy();
        policy.checks = [{ key: 'db.dml-requires-approval', bizType: BIZ, severity: SEVERITY_REQUIRED }];
        expect(reasonKeys(policy)).toContain('flow.policy.issue.paramRequired');
    });

    it('场景未注册与重复自定义条件被拦下', () => {
        const policy = createPolicy();
        policy.checks = [{ key: 'db.dml-without-where', bizType: 'mongo_flow', severity: SEVERITY_REQUIRED }];
        expect(validatePolicy(policy, SCHEMA)[0].reasonKey).toBe('flow.policy.issue.unknownScenario');
        expect(validatePolicy(policy, SCHEMA)[0].source).toContain('mongo_flow');

        const duplicated = createPolicy();
        ensureCustom(duplicated, BIZ);
        ensureCustom(duplicated, BIZ);
        duplicated.customs = [...(duplicated.customs ?? []), { bizType: BIZ, severity: SEVERITY_REQUIRED }];
        expect(reasonKeys(duplicated)).toContain('flow.policy.issue.duplicateCustom');
    });

    it('条件引用未知字段或不支持的操作符被拦下', () => {
        const policy = createPolicy();
        const custom = ensureCustom(policy, BIZ);
        custom.when = group({ kind: 'condition', field: 'ghostField', op: 'eq', value: 'x' });
        expect(reasonKeys(policy)).toContain('flow.policy.issue.unknownField');

        custom.when = group({ kind: 'condition', field: 'stmtType', op: 'contains', value: 'x' });
        expect(reasonKeys(policy)).toContain('flow.policy.issue.operatorNotAllowed');

        custom.when = group({ kind: 'condition', field: 'dangerous', op: 'eq', value: 'true' });
        expect(reasonKeys(policy)).toContain('flow.policy.issue.boolExpected');

        custom.when = group({ kind: 'condition', field: 'tableCount', op: 'gt', value: 'many' });
        expect(reasonKeys(policy)).toContain('flow.policy.issue.numberExpected');
    });

    it('空条件组与空自定义条件被拦下', () => {
        const policy = createPolicy();
        const custom = ensureCustom(policy, BIZ);
        custom.when = createGroupNode('all');
        custom.unless = undefined;
        const keys = reasonKeys(policy);
        expect(keys).toContain('flow.policy.issue.emptyCustom');
        expect(keys).toContain('flow.policy.issue.emptyGroup');
    });

    it('schema 缺失时不静默放过，逐条给出定位', () => {
        const policy = createPolicy();
        enableCheck(policy, BIZ, SCHEMA.scenarios[0].checks[0]);
        const issues = validatePolicy(policy, null);
        expect(issues.length).toBeGreaterThan(0);
        expect(issues[0].reasonKey).toBe('flow.policy.issue.unknownScenario');
    });
});

describe('自然语句派生', () => {
    it('条件行渲染为 字段 + 操作符 + 取值', () => {
        const scenario = SCHEMA.scenarios[0];
        expect(describeNode({ kind: 'condition', field: 'stmtType', op: 'in', value: ['update', 'delete'] }, scenario, translate)).toBe(
            '‹flow.field.stmtType› ‹flow.op.in› [‹flow.option.stmtType.update›, ‹flow.option.stmtType.delete›]'
        );
    });

    it('枚举取值缺文案时回退原始值，不显示裸 key', () => {
        const scenario = SCHEMA.scenarios[0];
        const rawFallback = (key: string) => (key === 'flow.option.stmtType.update' ? key : `‹${key}›`);
        const text = describeNode({ kind: 'condition', field: 'stmtType', op: 'eq', value: 'update' }, scenario, rawFallback);
        expect(text).toBe('‹flow.field.stmtType› ‹flow.op.eq› update');
    });

    it('分组按逻辑词串联，单子项不加分组词', () => {
        const scenario = SCHEMA.scenarios[0];
        const leaf: RuleNode = { kind: 'condition', field: 'stmtType', op: 'eq', value: 'ddl' };
        expect(describeNode(group(leaf), scenario, translate)).not.toContain('‹flow.policy.joinAnd›');
        expect(describeNode(group(leaf, leaf), scenario, translate)).toContain('‹flow.policy.joinAnd›');
        expect(describeNode({ kind: 'group', logic: 'any', items: [leaf, leaf] }, scenario, translate)).toContain('‹flow.policy.joinOr›');
    });

    it('嵌套子组含多个子句时加括号，避免子组的「且」被读成父组「或」的一项', () => {
        const scenario = SCHEMA.scenarios[0];
        const leaf = (value: string): RuleNode => ({ kind: 'condition', field: 'stmtType', op: 'eq', value });
        const inner: RuleNode = { kind: 'group', logic: 'all', items: [leaf('update'), leaf('delete')] };
        const outer: RuleNode = { kind: 'group', logic: 'any', items: [leaf('ddl'), inner] };
        const text = describeNode(outer, scenario, translate);
        expect(text).toContain('（');
        expect(text).toContain('）');
        // 单子句子组语义无歧义，不加多余括号
        const singleChild: RuleNode = { kind: 'group', logic: 'all', items: [leaf('ddl')] };
        const flat = describeNode({ kind: 'group', logic: 'any', items: [leaf('update'), singleChild] }, scenario, translate);
        expect(flat).not.toContain('（');
    });

    it('整份策略摘要含检查项与自定义条件，无规则时回显兜底级别', () => {
        const policy = createPolicy();
        expect(describePolicy(policy, SCHEMA, translate)).toEqual(['‹flow.policy.noRule› → ‹flow.severity.required›']);

        enableCheck(policy, BIZ, SCHEMA.scenarios[0].checks[1]);
        const custom = ensureCustom(policy, BIZ);
        custom.unless = { kind: 'condition', field: 'dangerous', op: 'eq', value: true };
        const lines = describePolicy(policy, SCHEMA, translate);
        expect(lines).toContain('‹flow.check.dmlWithoutWhere› → ‹flow.severity.required›');
        expect(lines.some((line) => line.includes('‹flow.policy.unless›'))).toBe(true);
    });
});

describe('列表页摘要', () => {
    it('按场景取最严级别，无规则时回落到兜底级别', () => {
        const policy = createPolicy(SEVERITY_WARNING);
        enableCheck(policy, BIZ, SCHEMA.scenarios[0].checks[0]);
        expect(policySummary(policy)).toEqual([{ bizType: BIZ, severity: SEVERITY_REQUIRED }]);

        policy.checks?.push({ key: 'db.dml-without-where', bizType: 'redis_run_cmd_flow', severity: SEVERITY_FORBIDDEN });
        const summary = policySummary(policy);
        expect(summary.find((item) => item.bizType === 'redis_run_cmd_flow')?.severity).toBe(SEVERITY_FORBIDDEN);
        expect(summary.find((item) => item.bizType === BIZ)?.severity).toBe(SEVERITY_REQUIRED);
    });

    it('摘要与变更历史对同一条规则给同一句话，且带上参数', () => {
        // 「指定语句类型需处置」不写参数就看不出管的是插入还是 DDL；
        // 更要紧的是摘要与时间线必须同一套措辞，否则管理员在两处看到两句话会以为是两条规则
        const check = SCHEMA.scenarios[0].checks[0];
        const policy = createPolicy(SEVERITY_REQUIRED);
        policy.checks = [{ key: check.key, bizType: BIZ, severity: SEVERITY_REQUIRED, params: { stmtTypes: ['insert', 'update'] } }];

        const summary = describePolicy(policy, SCHEMA, translate);
        expect(summary).toHaveLength(1);
        expect(summary[0]).toContain('stmtTypes=insert,update');

        const added = diffPolicy(createPolicy(SEVERITY_DISABLED), policy, SCHEMA, translate);
        expect(added.map((line) => line.text)).toContain(summary[0]);
    });

    it('只配置了豁免条件不构成处置', () => {
        const policy = createPolicy(SEVERITY_DISABLED);
        const custom = ensureCustom(policy, BIZ);
        custom.when = undefined;
        custom.unless = { kind: 'condition', field: 'dangerous', op: 'eq', value: true };
        expect(policySummary(policy)).toEqual([{ bizType: '*', severity: SEVERITY_DISABLED }]);
    });
});

describe('值编辑器分派与形态归一', () => {
    it('分派表只认编辑器标识，不再自己存一份「类型名 -> 控件」', () => {
        // 类型到标识的换算由后端注册表负责；前端若再按类型名建键，新增字段类型时两边必然分叉
        expect(Object.keys(VALUE_EDITORS).sort()).toEqual(['input', 'number', 'select', 'switch', 'tags']);
        for (const typeName of ['enum', 'bool', 'string', 'list<string>', 'list<number>', 'number-list']) {
            expect(VALUE_EDITORS).not.toHaveProperty(typeName);
        }
    });

    it('检查项参数的控件取后端下发的 editorKey', () => {
        const param = { key: 'maxKb', titleKey: 'flow.checkParam.maxKb', type: 'number', editorKey: 'number', default: 1 };
        expect(paramEditor(param).field.editorKey).toBe('number');
        // 老 schema（没带 editorKey）也不会渲染崩掉：落到文本输入
        expect(paramEditor({ key: 'k', titleKey: 't', type: 'bytes' }).field.editorKey).toBe('input');
    });

    it('切换操作符时按期望值形态给出干净的初值', () => {
        expect(emptyValueFor('none')).toBeUndefined();
        expect(emptyValueFor('multi')).toEqual([]);
        expect(emptyValueFor('range')).toEqual(['', '']);
        expect(emptyValueFor('single')).toBe('');
    });

    it('新建条件取字段首个操作符并配套取值形态', () => {
        const node = createConditionNode(stmtField);
        expect(node.op).toBe('eq');
        expect(node.value).toBe('');
        expect(createConditionNode(undefined).field).toBe('');
    });

    it('数组型默认值判定为多值参数', () => {
        const [, multi] = [null, paramEditor(SCHEMA.scenarios[0].checks[0].params![0])];
        expect(multi.operator.valueKind).toBe('multi');
        const single = paramEditor(SCHEMA.scenarios[0].checks[2].params![0]);
        expect(single.operator.valueKind).toBe('single');
        expect(single.field.editorKey).toBe('number');
    });

    it('字段查找容错', () => {
        expect(fieldOf(SCHEMA.scenarios[0], 'stmtType')?.key).toBe('stmtType');
        expect(fieldOf(null, 'stmtType')).toBeNull();
        expect(fieldOf(SCHEMA.scenarios[0], undefined)).toBeNull();
    });

    it('级别常量顺序即约束强度', () => {
        expect(SEVERITY_DISABLED).toBeLessThan(SEVERITY_WARNING);
        expect(SEVERITY_WARNING).toBeLessThan(SEVERITY_REQUIRED);
        expect(SEVERITY_REQUIRED).toBeLessThan(SEVERITY_FORBIDDEN);
    });
});

describe('新建规则的默认级别与参数边界都只认后端声明', () => {
    const scenario = () => SCHEMA.scenarios[0];

    it('默认级别优先「需审批」，与后端数组顺序无关', () => {
        // 「取第一个元素」会把后端下发顺序变成隐式契约：顺序一变，默认级别就悄悄变了
        expect(defaultRuleSeverity({ ...scenario(), severities: [SEVERITY_WARNING, SEVERITY_REQUIRED, SEVERITY_FORBIDDEN] })).toBe(SEVERITY_REQUIRED);
        expect(defaultRuleSeverity({ ...scenario(), severities: [SEVERITY_FORBIDDEN, SEVERITY_REQUIRED, SEVERITY_WARNING] })).toBe(SEVERITY_REQUIRED);
    });

    it('没有审批通道的场景不会默认出「需审批」', () => {
        // 机器命令这类场景默认成需审批，等于一点开关就产出一条保存不出去的配置
        expect(defaultRuleSeverity({ ...scenario(), severities: [SEVERITY_WARNING, SEVERITY_FORBIDDEN] })).toBe(SEVERITY_WARNING);
        // 级别没下发时也不替场景发明能力：取最轻的可执行级别，由管理员自己决定要不要改
        expect(defaultRuleSeverity({ ...scenario(), severities: undefined })).toBe(SEVERITY_WARNING);
    });

    it('参数是否按数值判边界只看后端下发的 numeric，不看类型名', () => {
        // 前端一旦按 type 名维护一份「哪些类型算数值」的清单，后端新增数值型参数就会越界一路绿灯直到保存被拒
        const outOfRange = () => {
            const policy = createPolicy();
            policy.checks = [{ key: 'db.sql-size-exceeds', bizType: BIZ, severity: SEVERITY_WARNING, params: { maxKb: 0 } }];
            return validatePolicy(policy, SCHEMA).map((issue) => issue.reasonKey);
        };
        expect(outOfRange()).toContain('flow.policy.issue.paramTooSmall');

        const schemaNoFlag: PolicySchema = structuredClone(SCHEMA);
        const param = schemaNoFlag.scenarios[0].checks.find((item) => item.key === 'db.sql-size-exceeds')!.params!.find((item) => item.key === 'maxKb')!;
        delete param.numeric;
        const policy = createPolicy();
        policy.checks = [{ key: 'db.sql-size-exceeds', bizType: BIZ, severity: SEVERITY_WARNING, params: { maxKb: 0 } }];
        expect(validatePolicy(policy, schemaNoFlag).map((issue) => issue.reasonKey)).not.toContain('flow.policy.issue.paramTooSmall');
    });
});
