/**
 * DDL 预检结果框。
 *
 * 钉住四件事：
 * 1. 打开即请求，且把「不可逆」与「需注意」分开展示——两者混在一起会让人忽略真风险；
 * 2. 文案按 code 取（后端只给 code + args），大表行数要能插入；
 * 3. 没有发现时给「未发现结构性风险」的明确结论，而不是空白；
 * 4. 请求失败要把原因显示出来，不能静默空白（否则用户以为「没问题」）。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import ElementPlus from 'element-plus';
import DdlPrecheckDialog from '../DdlPrecheckDialog.vue';

const precheckRequest = vi.fn();

vi.mock('../../api', () => ({
    dbApi: { dbDdlPrecheck: { request: (...a: unknown[]) => precheckRequest(...a) } },
}));

const i18n = createI18n({
    legacy: false,
    globalInjection: true,
    locale: 'zh-cn',
    missingWarn: false,
    fallbackWarn: false,
    messages: {
        'zh-cn': {
            common: { close: '关闭', refresh: '刷新', yes: '是' },
            db: {
                ddlPrecheck: 'DDL 预检',
                ddlPrecheckClean: '未发现结构性风险',
                ddlPrecheckSummary: '预检发现：{danger} 项不可逆、{warn} 项需注意',
                ddlPrecheckTables: '涉及表规模',
                ddlNoPk: '无主键',
                ddlRiskLevel_danger: '不可逆',
                ddlRiskLevel_warn: '需注意',
                ddlRiskLevel_info: '提示',
                ddlRisk_drop_table: 'DROP TABLE 会连同数据一起删除',
                ddlRisk_large_table: '该表约 {rows} 行，按大表预案执行',
                ddlRisk_no_primary_key: '该表无主键',
                ddlRisk_table_unknown: '未读到该表的结构信息',
                tableName: '表名',
                ddlTableRows: '行数',
                ddlTableSize: '数据大小',
                ddlPrimaryKey: '主键',
                noSqlToRunMsg: '没有可执行的SQL',
            },
        },
    },
});

function mountDialog(sql = 'ALTER TABLE t_a DROP COLUMN c') {
    return mount(DdlPrecheckDialog, {
        props: { modelValue: false, dbId: 1, dbName: 'mayfly_go', sql },
        // el-dialog 默认 teleport 到 body：桩掉 teleport 才能在 wrapper 里断言内容
        global: { plugins: [i18n, ElementPlus], stubs: { teleport: true } },
    });
}

beforeEach(() => {
    precheckRequest.mockReset();
});

describe('DDL 预检结果框', () => {
    it('打开即请求，并区分不可逆与需注意', async () => {
        precheckRequest.mockResolvedValue({
            statements: [{ sql: 'ALTER TABLE t_a DROP COLUMN c', kind: 'alter_table', tables: ['t_a'] }],
            tables: [{ name: 't_a', rows: 2000000, bytes: 2048, hasPk: false }],
            findings: [
                { level: 'danger', code: 'drop_column', table: 't_a' },
                { level: 'warn', code: 'large_table', table: 't_a', args: { rows: 2000000 } },
                { level: 'warn', code: 'no_primary_key', table: 't_a' },
            ],
        });
        const wrapper = mountDialog();
        await wrapper.setProps({ modelValue: true });
        await flushPromises();

        expect(precheckRequest).toHaveBeenCalledWith({ id: 1, db: 'mayfly_go', sql: 'ALTER TABLE t_a DROP COLUMN c' });
        const text = wrapper.text();
        // 汇总里的 1/2 与逐条文字都要在；大表行数插值成功
        expect(text).toContain('预检发现：1 项不可逆、2 项需注意');
        expect(text).toContain('不可逆');
        expect(text).toContain('该表约 2000000 行');
        expect(text).toContain('该表无主键');
        // 表规模要能看见，不能只说「大」
        expect(text).toContain('2.0 KB');
        expect(text).toContain('无主键');
    });

    it('code 没有对应文案时原样显示，不吞掉行', async () => {
        precheckRequest.mockResolvedValue({
            statements: [],
            tables: [],
            findings: [{ level: 'warn', code: 'future_rule', table: 't_a' }],
        });
        const wrapper = mountDialog();
        await wrapper.setProps({ modelValue: true });
        await flushPromises();
        expect(wrapper.text()).toContain('db.ddlRisk_future_rule');
    });

    it('没有发现时给明确结论', async () => {
        precheckRequest.mockResolvedValue({ statements: [], tables: [], findings: [] });
        const wrapper = mountDialog();
        await wrapper.setProps({ modelValue: true });
        await flushPromises();
        expect(wrapper.text()).toContain('未发现结构性风险');
    });

    it('请求失败要显示原因，不能静默空白', async () => {
        precheckRequest.mockRejectedValue(new Error('预检不完整：读取表规模失败'));
        const wrapper = mountDialog();
        await wrapper.setProps({ modelValue: true });
        await flushPromises();
        expect(wrapper.text()).toContain('预检不完整');
        expect(wrapper.find('.el-alert').exists()).toBe(true);
    });

    it('空脚本不发请求，直接提示', async () => {
        const wrapper = mountDialog('   ');
        await wrapper.setProps({ modelValue: true });
        await flushPromises();
        expect(precheckRequest).not.toHaveBeenCalled();
        expect(wrapper.text()).toContain('没有可执行的SQL');
    });
});
