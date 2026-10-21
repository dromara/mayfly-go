/**
 * 消息模板选择器的检索契约回归。
 *
 * 钉住两条不变式：
 * 1. 初始回填按模板主键精确定位（否则下拉里没有选中项，只能显示裸值而不是模板名称）
 * 2. 用户手输的关键字一律按编码模糊搜——**与 valueField 无关**
 *
 * 第 2 条是真机踩过的坑：此前按 valueField 分流，使得按主键绑定的选择器（流程定义的通知模板）
 * 把用户输入的文字当 id 传给后端，后端 cast 成 0 后忽略该条件，搜索静默退化成「返回全部首页」。
 * 界面上输入框照常有反应、下拉照常出候选，只是永远搜不出想要的东西——没有任何报错可供察觉。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';
import MsgTmplSelect from '../MsgTmplSelect.vue';

const listRequest = vi.fn();

vi.mock('@/views/msg/api', () => ({
    tmplApi: { list: { request: (...args: unknown[]) => listRequest(...args) } },
}));
vi.mock('@/components/auth/auth', () => ({ hasPerm: () => true, hasPerms: () => true }));
vi.mock('@/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }));

const i18n = createI18n({
    legacy: false,
    globalInjection: true,
    locale: 'zh-cn',
    messages: {
        'zh-cn': {
            msg: { selectTmplPlaceholder: '请选择消息模板', tmplNoReadPerm: '无消息模板读取权限' },
        },
    },
});

/** 挂载选择器并返回「触发一次远端检索」的入口（即传给 el-select 的 remote-method） */
async function mountSelect(modelValue: string | number | null, valueField?: 'id' | 'code') {
    const wrapper = mount(MsgTmplSelect, {
        props: valueField ? { modelValue, valueField } : { modelValue },
        global: { plugins: [ElementPlus, i18n] },
    });
    // onMounted 里的首帧请求先落地，后续断言只看用户输入触发的那一次
    await wrapper.vm.$nextTick();
    await new Promise((r) => setTimeout(r, 0));
    listRequest.mockClear();

    const remoteMethod = wrapper.findComponent({ name: 'ElSelect' }).props('remoteMethod') as (kw: string) => void;
    expect(typeof remoteMethod).toBe('function');
    return { wrapper, search: (kw: string) => remoteMethod(kw) };
}

beforeEach(() => {
    listRequest.mockReset();
    listRequest.mockResolvedValue({ list: [] });
});

describe('消息模板选择器检索', () => {
    it('按主键绑定时：手输关键字按 code 搜而不是当 id 传', async () => {
        listRequest.mockResolvedValue({ list: [{ id: 3, code: 'agentopsDiagnosisResult', name: '运维诊断结果通知' }] });
        const { search } = await mountSelect(3);

        search('agentops');
        expect(listRequest).toHaveBeenCalledWith({ code: 'agentops' });
        expect(listRequest).not.toHaveBeenCalledWith({ id: 'agentops' });
    });

    it('按编码绑定时：手输关键字同样按 code 搜（行为不因 valueField 分叉）', async () => {
        const { search } = await mountSelect('zWaUq5gb', 'code');

        search('zWaUq5gb');
        expect(listRequest).toHaveBeenCalledWith({ code: 'zWaUq5gb' });
    });

    it('展开下拉（空关键字）取首页列表，不带任何检索条件', async () => {
        const { search } = await mountSelect(null);

        search('');
        expect(listRequest).toHaveBeenCalledWith({ code: '' });
    });
});

describe('消息模板选择器初始回填', () => {
    it('已绑定模板主键时按 id 拉候选，保证下拉能显示模板名称', async () => {
        listRequest.mockResolvedValue({ list: [] });
        mount(MsgTmplSelect, { props: { modelValue: 7 }, global: { plugins: [ElementPlus, i18n] } });
        await new Promise((r) => setTimeout(r, 0));

        expect(listRequest).toHaveBeenCalledWith({ id: 7 });
    });

    it('已绑定模板编码时按 code 拉候选', async () => {
        listRequest.mockResolvedValue({ list: [] });
        mount(MsgTmplSelect, {
            props: { modelValue: '7u2MRCaB', valueField: 'code' },
            global: { plugins: [ElementPlus, i18n] },
        });
        await new Promise((r) => setTimeout(r, 0));

        expect(listRequest).toHaveBeenCalledWith({ code: '7u2MRCaB' });
    });
});
