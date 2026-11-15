import { describe, expect, it, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { reactive } from 'vue';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';

import SqlAiBar from '../SqlAiBar.vue';
import type { DraftIntentId, SqlDraftRequest, SqlDraftResult } from '../types';
import type { DraftState } from '../useSqlDraft';

/**
 * SQL 助手面板的行为契约。
 *
 * 这是「少打字、少一步」的全部落点：键位、零输入自动跑、采纳、取消。它们都只在组件的
 * 事件回调里成立，界面点一遍才知道对错，所以把最关键的几条钉成用例：
 *  1. 从右键菜单点进来的零输入意图必须**自动发起**一次生成（否则用户白点一下）；
 *  2. 自动发起时不能被自己的 cancel 作废掉（序号推进后在途结果会被守卫丢弃，表现就是「点了没反应」）；
 *  3. 描述类意图不许自动跑（否则发出一条没人想要的空描述请求，还要扣一次模型费用）；
 *  4. 等待中必须有可点的取消，且× 与 Esc 都走「收起即中断」同一条路径；
 *  5. 采纳只在可写回时发出，且带的是拼接后的语句。
 */
const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });

// SvgIcon 由 main.ts 全局注册，测试环境没有那一步：不 stub 会在每个用例刷一片解析告警
const globalMount = { plugins: [i18n, ElementPlus], stubs: { SvgIcon: { template: '<i />' } } };

const material = (selection = 'select * from t_user', statement = 'select * from t_user') => ({
    selection,
    statement,
    hasSelection: selection.trim().length > 0,
});

const verifiedResult = {
    intent: 'optimize',
    statements: [{ sql: 'select id from t_user' }],
    verification: 'verified',
    checks: [],
    contextTables: [],
    repairs: 1,
    candidates: 0,
    dialect: 'mysql',
    usage: { promptTokens: 10, completionTokens: 20, totalTokens: 30 },
} as unknown as SqlDraftResult;

function mountBar(overrides: Record<string, unknown> = {}) {
    const state = reactive<DraftState>({ loading: false, result: null, error: '', timedOut: false });
    // 形参写进签名：否则 vi.fn 推不出调用参数类型，payload 断言只能靠断言式 cast
    const generate = vi.fn(async (_payload: SqlDraftRequest) => verifiedResult);
    const cancel = vi.fn();
    const wrapper = mount(SqlAiBar, {
        props: {
            dbId: 1,
            dbName: 'db1',
            materials: material(),
            draft: { state, generate, cancel },
            modelValue: true,
            intent: 'optimize' as DraftIntentId,
            'onUpdate:modelValue': (value: boolean) => wrapper.setProps({ modelValue: value }),
            'onUpdate:intent': (value: DraftIntentId) => wrapper.setProps({ intent: value }),
            runOnOpen: true,
            ...overrides,
        },
        global: globalMount,
    });
    return { wrapper, state, generate, cancel };
}

describe('SQL 助手面板 SqlAiBar', () => {
    it('零输入意图 + runOnOpen：打开即发起一次生成，且不被自己的 cancel 作废', async () => {
        const { wrapper, generate, cancel } = mountBar();
        await wrapper.vm.$nextTick();
        await new Promise((resolve) => setTimeout(resolve, 0));

        expect(generate).toHaveBeenCalledTimes(1);
        const payload = generate.mock.calls[0][0];
        expect(payload.intent).toBe('optimize');
        expect(payload.prompt).toBe('');
        // 材料带上：用户不需要再说一个字
        expect(payload.selection).toBe('select * from t_user');
        expect(payload.anchorStatement).toBe('select * from t_user');
        // 自动发起这一轮绝不能先 cancel：序号推进后，本次在途结果会被守卫当成过期而丢弃
        expect(cancel).not.toHaveBeenCalled();
    });

    it('描述类意图（按描述生成）不自动跑：等用户写完再发', async () => {
        const { wrapper, generate } = mountBar({ intent: 'draft', runOnOpen: true });
        await wrapper.vm.$nextTick();
        await new Promise((resolve) => setTimeout(resolve, 0));
        expect(generate).not.toHaveBeenCalled();
    });

    it('结果可写回时点采纳抛出拼接后的语句（补分号，避免粘回编辑器被当成一条）', async () => {
        const { wrapper, state } = mountBar({ runOnOpen: false });
        state.result = verifiedResult;
        await wrapper.vm.$nextTick();

        const adopt = wrapper.findAll('button').find((btn) => btn.text() === 'db.draft.barAdopt');
        expect(adopt, '可写回时必须有采纳按钮').toBeTruthy();
        await adopt?.trigger('click');
        expect(wrapper.emitted('apply')?.[0]).toEqual(['select id from t_user;']);
    });

    it('结论为 invalid 时不给采纳入口', async () => {
        const { wrapper, state } = mountBar({ runOnOpen: false });
        state.result = { ...verifiedResult, verification: 'invalid', statements: [{ sql: 'drop table t' }] };
        await wrapper.vm.$nextTick();
        expect(wrapper.findAll('button').some((btn) => btn.text() === 'db.draft.barAdopt')).toBe(false);
    });

    it('等待中必须有可点的「取消」，且它真去中断在途请求', async () => {
        const { wrapper, state, cancel } = mountBar({ runOnOpen: false });
        state.loading = true;
        await wrapper.vm.$nextTick();

        const cancelBtn = wrapper.findAll('button').find((btn) => btn.text() === 'common.cancel');
        expect(cancelBtn, '转圈时必须给得出入口').toBeTruthy();
        await cancelBtn?.trigger('click');
        expect(cancel).toHaveBeenCalledTimes(1);
    });

    it('× 关闭浮层，并由「收起即放弃」统一中断在途请求', async () => {
        const { wrapper, state, cancel } = mountBar({ runOnOpen: false });
        state.result = verifiedResult;
        await wrapper.vm.$nextTick();

        const close = wrapper.find('.bar-close');
        expect(close.exists(), '鼠标用户需要一个看得见的关闭入口').toBe(true);
        await close.trigger('click');
        await wrapper.vm.$nextTick();
        expect(wrapper.props('modelValue')).toBe(false);
        // 作废逻辑只写一处（watch(visible)），× 与 Esc 两个入口行为必然一致
        expect(cancel).toHaveBeenCalled();
    });

    it('Tab 采纳；无结果时不吞键位', async () => {
        const { wrapper, state } = mountBar({ runOnOpen: false });
        state.result = verifiedResult;
        await wrapper.vm.$nextTick();

        await wrapper.find('textarea').trigger('keydown', { key: 'Tab' });
        expect(wrapper.emitted('apply')).toHaveLength(1);

        state.result = null;
        await wrapper.vm.$nextTick();
        await wrapper.find('textarea').trigger('keydown', { key: 'Tab' });
        // 没有可采纳的结果时不再抛 apply（键位归还给焦点流转）
        expect(wrapper.emitted('apply')).toHaveLength(1);
    });

    it('回车发起生成，并带上上一版结果作为细化输入', async () => {
        const { wrapper, state, generate } = mountBar({ runOnOpen: false });
        state.result = verifiedResult;
        await wrapper.vm.$nextTick();

        await wrapper.find('textarea').setValue('再按金额倒序');
        await wrapper.find('textarea').trigger('keydown', { key: 'Enter' });
        await new Promise((resolve) => setTimeout(resolve, 0));

        expect(generate).toHaveBeenCalledTimes(1);
        const payload = generate.mock.calls[0][0];
        expect(payload.prompt).toBe('再按金额倒序');
        // 连续回车默认「继续细化」而不是推倒重做
        expect(payload.refineOf).toBe('select id from t_user;');
    });

    it('切换意图作废上一版结论（换问题不能沿用旧答案）', async () => {
        const { wrapper, state, cancel } = mountBar({ runOnOpen: false });
        state.result = verifiedResult;
        await wrapper.vm.$nextTick();

        const chips = wrapper.findAll('.intent-chip');
        // 有材料时四个意图全部可用，按 order 排序：第一个是「按描述生成」
        expect(chips).toHaveLength(4);
        await chips[0].trigger('click');
        expect(wrapper.emitted('update:intent')?.[0]).toEqual(['draft']);
        expect(cancel).toHaveBeenCalledTimes(1);
    });
});
