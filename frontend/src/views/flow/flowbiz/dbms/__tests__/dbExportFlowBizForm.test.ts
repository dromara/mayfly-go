/**
 * 导出工单的**提单**表单。
 *
 * 这组用例守的是两个「静态检查全绿但功能不可用」的坑（都是本轮真机踩出来的）：
 * 1. 不主动上报资源 → 宿主解析不出流程定义，抽屉停在「不存在审批节点」、确定按钮永远禁用；
 * 2. 不暴露 validateBizForm → 宿主调用时是 TypeError，被 catch 成「请正确填写信息」，
 *    字段全填好也提交不了。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import ElementPlus from 'element-plus';
import DbExportFlowBizForm from '../DbExportFlowBizForm.vue';
import { TagResourceTypeEnum } from '@/common/commonEnum';

const dbsRequest = vi.fn();

vi.mock('@/views/ops/db/api', () => ({
    dbApi: { dbs: { request: (...a: unknown[]) => dbsRequest(...a) } },
}));

const i18n = createI18n({
    legacy: false,
    globalInjection: true,
    locale: 'zh-cn',
    missingWarn: false,
    fallbackWarn: false,
    // 只给导出类型这三条：dumpTypeLabel 走 te() 判定，字典为空会退回 '-'
    messages: {
        'zh-cn': {
            flow: {
                exportFormTip: '申请导出的范围',
                exportDb: '数据库',
                exportContent: '导出内容',
                exportTableCount: '表数量',
                exportTables: '表清单',
                exportDumpType1: '仅结构',
                exportDumpType2: '仅数据',
                exportDumpType3: '结构与数据',
            },
        },
    },
});

function mountForm(bizForm: Record<string, unknown>) {
    return mount(DbExportFlowBizForm, {
        props: { bizForm },
        global: { plugins: [i18n, ElementPlus] },
    });
}

beforeEach(() => {
    dbsRequest.mockReset().mockResolvedValue({ list: [{ code: 'T6txf4tTxU' }] });
});

describe('导出工单提单表单', () => {
    it('挂载后按 dbId 反查资源编码并上报（否则宿主管不出审批节点）', async () => {
        const wrapper = mountForm({ dbId: 1, dbName: 'mayfly_go', tables: ['t_a'], dumpType: '3' });
        await flushPromises();
        await flushPromises();

        expect(dbsRequest).toHaveBeenCalledWith({ id: 1 });
        const emitted = wrapper.emitted('changeResourceCode');
        expect(emitted).toBeTruthy();
        expect(emitted?.[0]).toEqual([TagResourceTypeEnum.Db.value, 'T6txf4tTxU']);
    });

    it('暴露 validateBizForm / resetBizForm（缺失会让提交报“请正确填写信息”）', () => {
        const wrapper = mountForm({ dbId: 1, dbName: 'mayfly_go', tables: ['t_a'], dumpType: '3' });
        // 必须查 defineExpose 的结果：VTU 的 wrapper.vm 在 dev 下会代理到 setupState，
        // 直接读 vm.validateBizForm 即使没 expose 也能拿到，断言会变假绿
        const exposed = (wrapper.vm.$ as unknown as { exposed: Record<string, unknown> }).exposed;
        expect(typeof exposed.validateBizForm).toBe('function');
        expect(typeof exposed.resetBizForm).toBe('function');
    });

    it('回显申请范围（审批人要知道自己批的是什么）', async () => {
        const wrapper = mountForm({ dbId: 1, dbName: 'mayfly_go', tables: ['t_a', 't_b'], dumpType: '2' });
        await flushPromises();
        const text = wrapper.text();
        expect(text).toContain('mayfly_go');
        expect(text).toContain('t_a');
        expect(text).toContain('t_b');
        expect(text).toContain('仅数据');
    });

    it('反查不到资源编码时不瞎上报', async () => {
        dbsRequest.mockRejectedValue(new Error('boom'));
        const wrapper = mountForm({ dbId: 1, dbName: 'mayfly_go', tables: ['t_a'], dumpType: '3' });
        await flushPromises();
        await flushPromises();
        expect(wrapper.emitted('changeResourceCode')).toBeFalsy();
    });
});
