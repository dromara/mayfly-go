import { describe, expect, it } from 'vitest';
import { labelIssueSource, type PolicySchema } from '../policyModel';

/**
 * 策略问题定位必须说人话。
 *
 * PolicyIssue.source 是机器定位路径（checks[no_where@db_sql_exec_flow].params.min），
 * 它要把问题定位回具体控件，但不能原样显示给管理员——那等于把后端字段路径丢进界面
 */
const schema = {
    scenarios: [
        {
            bizType: 'db_sql_exec_flow',
            titleKey: 'x',
            checks: [
                {
                    key: 'no_where',
                    titleKey: 'flow.check.noWhere',
                    default: 2,
                    params: [{ key: 'max_bytes', titleKey: 'flow.check.maxBytes', type: 'number', editorKey: 'e' }],
                },
            ],
            fields: [],
        },
    ],
} as unknown as PolicySchema;

// 只做前缀翻译的假 t：能区分「翻过了」与「原样丢出来」
const t = (key: string, params?: Record<string, unknown>) => {
    const map: Record<string, string> = {
        'flow.bizTypeName.db_sql_exec_flow': 'DBMS-执行SQL',
        'flow.check.noWhere': '缺少 WHERE 条件',
        'flow.check.maxBytes': 'SQL 体积上限',
        'flow.policy.field.customRule': '自定义条件',
        'flow.policy.field.severity': '处置级别',
        'flow.policy.field.defaultSeverity': '未配置规则时',
        'flow.policy.field.param': `参数 ${params?.name ?? ''}`.trim(),
        'flow.policy.field.item': `第 ${params?.index} 项`,
        'flow.policy.field.value': '期望值',
        'flow.policy.field.when': '触发条件',
        'flow.policy.field.unless': '豁免条件',
        'flow.policy.field.op': '运算符',
    };
    return map[key] ?? key;
};

describe('策略问题定位的展示', () => {
    it('检查项路径翻成「场景 · 检查项 · 参数名」', () => {
        expect(labelIssueSource('checks[no_where@db_sql_exec_flow].params.max_bytes', schema, t)).toBe('DBMS-执行SQL · 缺少 WHERE 条件 · 参数 SQL 体积上限');
    });

    it('自定义条件路径不再暴露 customs[bizType]', () => {
        const label = labelIssueSource('customs[db_sql_exec_flow].severity', schema, t);
        expect(label).toBe('DBMS-执行SQL · 自定义条件 · 处置级别');
        expect(label).not.toContain('customs[');
    });

    it('条件树里的第 N 项要能定位到具体那一条', () => {
        expect(labelIssueSource('customs[db_sql_exec_flow].items[2].value', schema, t)).toContain('第 3 项');
        expect(labelIssueSource('customs[db_sql_exec_flow].items[2].value', schema, t)).toContain('期望值');
    });

    it('多层嵌套也要逐段翻（上一版只翻了首段，when.items[0].value 仍露字段名）', () => {
        const label = labelIssueSource('customs[db_sql_exec_flow].when.items[0].value', schema, t);
        expect(label).toBe('DBMS-执行SQL · 自定义条件 · 触发条件 · 第 1 项 · 期望值');
        expect(label).not.toContain('items[');
        expect(label).not.toContain('.value');
        expect(labelIssueSource('customs[db_sql_exec_flow].unless.items[1].op', schema, t)).toContain('豁免条件 · 第 2 项 · 运算符');
    });

    it('兜底级别与未知路径各有归处', () => {
        expect(labelIssueSource('defaultSeverity', schema, t)).toBe('未配置规则时');
        // 未知场景不硬编一个假名字，保留原值好排查
        expect(labelIssueSource('customs[not_registered]', schema, t)).toContain('not_registered');
    });

    it('场景名查不到时退回 bizType，不能显示成空白', () => {
        const label = labelIssueSource('checks[x@not_registered]', schema, t);
        expect(label).toContain('not_registered');
        expect(label.trim().length).toBeGreaterThan(0);
    });
});
