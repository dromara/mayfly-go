/**
 * 机器模块对话框「能否打开」组件级回归测试。
 *
 * 背景：批量命令/批量文件/健康总览/磁盘分析均通过 v-model:visible 控制 el-dialog。
 * 本测试直接挂载各对话框并置 visible=true，断言 el-dialog 内容真正渲染到 body，
 * 钉住「点击按钮 → visible=true → 对话框出现」这条链路，防止 model 绑定/组件挂载回归。
 *
 * 重依赖（api / auto-form / i18n / hooks / format）用桩替代，只验证开合与标题渲染。
 */
import { describe, expect, it, vi } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';
import { defineComponent, nextTick } from 'vue';
import type { Component } from 'vue';

// 泛化 mock：任意 api 对象、任意方法都返回可解析的 request
vi.mock('@/views/ops/machine/api', () => {
    const anyMethod = () => ({ request: () => Promise.resolve({ list: [], total: 0, data: [] }) });
    const anyApi = () => new Proxy({}, { get: () => anyMethod() });
    const base: Record<string | symbol, unknown> = { uploadFileToStore: () => Promise.resolve('file-key') };
    return new Proxy(base, { get: (t, p) => (p in t ? t[p] : anyApi()) });
});
vi.mock('@/common/request', () => ({ default: { request: vi.fn().mockResolvedValue({}) } }));
vi.mock('@/i18n', () => ({ i18n: { global: { t: (k: string) => k } } }));
vi.mock('@/hooks/useI18n', () => ({
    Msg: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
    useI18nDeleteConfirm: () => () => Promise.resolve(true),
}));
vi.mock('@/common/utils/format', () => ({ formatByteSize: () => '0B', formatDate: () => '2026-01-01' }));
// auto-form 依赖 monaco/store，用桩替代；defineFormItems 原样返回
vi.mock('@/components/auto-form', () => ({
    defineFormItems: (x: unknown) => x,
    AutoFormDrawer: defineComponent({ name: 'AutoFormDrawerStub', props: ['visible', 'data', 'items', 'title'], template: '<div class="afd-stub" />' }),
}));
vi.mock('@/components/svg-icon/index.vue', () => ({ default: defineComponent({ template: '<i class="svg-stub" />' }) }));

const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });

// v-auth 为全局指令，测试中用无操作桩注册，避免因未知指令中断渲染
const authDirective = { mounted: () => {}, updated: () => {} };

const mountDialog = async (comp: Component, props: Record<string, unknown> = {}) => {
    const wrapper = mount(comp, {
        props: { visible: true, ...props },
        global: { plugins: [ElementPlus, i18n], directives: { auth: authDirective } },
        attachTo: document.body,
    });
    await flushPromises();
    await nextTick();
    return wrapper;
};

describe('machine 对话框 visible=true 时应渲染 el-dialog 内容', () => {
    it('批量文件 BatchFileDialog', async () => {
        const C = (await import('../BatchFileDialog.vue')).default;
        await mountDialog(C, { machines: [] });
        expect(document.body.textContent).toContain('machine.batchFile');
    });

    it('批量命令 BatchRunDialog', async () => {
        const C = (await import('../BatchRunDialog.vue')).default;
        await mountDialog(C, { machines: [] });
        expect(document.body.textContent).toContain('machine.batchRunCmd');
    });

    it('健康总览 MachineHealthDialog', async () => {
        const C = (await import('../MachineHealthDialog.vue')).default;
        await mountDialog(C);
        expect(document.body.textContent).toContain('machine.healthOverview');
    });

    it('磁盘分析 DiskAnalyzeDialog', async () => {
        const C = (await import('../DiskAnalyzeDialog.vue')).default;
        await mountDialog(C, { machineId: 1, machineName: 'm1' });
        expect(document.body.textContent).toContain('machine.diskAnalyze');
    });
});
