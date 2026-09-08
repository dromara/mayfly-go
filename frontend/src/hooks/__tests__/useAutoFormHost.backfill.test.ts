/**
 * 弹层宿主回填契约测试（useAutoFormHost 单一出处，Dialog / Drawer 行为必须一致）
 *
 * 覆盖历史缺陷：回填只监听 visible 的 false→true 跃迁，宿主以 visible=true 直接挂载时
 * （异步组件在点击那刻才加载完成、HMR 重建已打开弹层）永不回填，state.form 保持空对象，
 * 控件拿到 undefined —— element-plus 表现为 "[ElSwitch] model-value must be active-value or inactive-value"。
 *
 * probe 用例（status 缺失）用于证明本文件确实能观测到该警告，避免假阴性。
 */
import { describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';
import AutoFormDialog from '@/components/auto-form/AutoFormDialog.vue';
import AutoFormDrawer from '@/components/auto-form/AutoFormDrawer.vue';
import type { AutoFormItem } from '@/components/auto-form/types';

vi.mock('@/components/monaco/MonacoEditor.vue', () => ({
    default: defineComponent({ name: 'MonacoEditorStub', props: ['modelValue', 'options'], template: '<div class="monaco-stub" />' }),
}));
vi.mock('@/components/svg-icon/index.vue', () => ({
    default: defineComponent({ name: 'SvgIconStub', template: '<i class="svg-icon-stub" />' }),
}));
vi.mock('@/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }));
vi.mock('@/common/request', () => ({ default: { request: vi.fn().mockResolvedValue([]) } }));

const i18n = createI18n({
    legacy: false,
    globalInjection: true,
    locale: 'en',
    messages: { en: { common: { status: 'Status', enable: 'On', disable: 'Off', cancel: 'Cancel', confirm: 'Confirm' } } },
});

/** status 为数字开关（activeValue 1 / inactiveValue 0），挂载时拿到 undefined/false 即告警 */
const items: AutoFormItem[] = [
    { prop: 'name', label: 'name' },
    {
        prop: 'status',
        label: 'common.status',
        type: 'switch',
        props: { inlinePrompt: true, activeText: 'On', inactiveText: 'Off', activeValue: 1, inactiveValue: 0 },
    },
];

const switchWarns = (spy: ReturnType<typeof vi.spyOn>) =>
    spy.mock.calls.map((c) => c.join(' ')).filter((s) => s.includes('active-value or inactive-value'));

/** 以 visible=true 直接挂载宿主（缺陷路径），返回弹层内 el-switch 数量与 model-value 警告 */
const mountHostOpen = async (host: typeof AutoFormDialog | typeof AutoFormDrawer, data: Record<string, unknown> | null) => {
    const warnSpy = vi.spyOn(console, 'warn');
    document.body.innerHTML = '';
    mount(host, {
        attachTo: document.body,
        props: { visible: true, items, data },
        global: { plugins: [ElementPlus, i18n] },
    });
    await flushPromises();
    await new Promise((resolve) => setTimeout(resolve, 50));
    const textInput = document.body.querySelector('input[type="text"]') as HTMLInputElement | null;
    return { switches: document.body.querySelectorAll('.el-switch').length, nameValue: textInput?.value, warns: switchWarns(warnSpy) };
};

describe.each([
    ['AutoFormDrawer', AutoFormDrawer],
    ['AutoFormDialog', AutoFormDialog],
])('%s 宿主回填契约', (_name, host) => {
    it('以 visible=true 直接挂载时编辑数据必须回填', async () => {
        const { switches, nameValue, warns } = await mountHostOpen(host, { name: 'r1', status: 0 });
        expect(switches).toBe(1);
        expect(nameValue).toBe('r1');
        expect(warns).toEqual([]);
    });

    it('以 visible=true 直接挂载的新增态走字段默认值', async () => {
        const { switches, warns } = await mountHostOpen(host, { name: '', status: 1 });
        expect(switches).toBe(1);
        expect(warns).toEqual([]);
    });

    // 探针：若不告警说明本文件没真正观测到 el-switch，上面用例属假阴性
    it('探针：status 缺失时必然产生 model-value 警告', async () => {
        const { switches, warns } = await mountHostOpen(host, { name: 'r1' });
        expect(switches).toBe(1);
        expect(warns.length).toBeGreaterThan(0);
    });
});
