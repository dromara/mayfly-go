/**
 * MonacoEditorBox 命令式弹窗行为测试
 *
 * 守护三条容易被改坏的语义：
 * 1. 弹窗外壳同步渲染：本函数被 api 模块等底层文件静态引用，外壳若改成动态导入，「点击 → 弹窗出现」
 *    就变成毫无反馈的空白，而且容器要等一个微任务才存在、下面的幂等判断会挡不住连点；
 * 2. 容器占位 + 幂等 + 关闭后清理，保证重复点击不会堆出多个弹窗；
 * 3. json 校验与压缩在回传给 confirmFn 之前完成，非法内容不得覆盖原值。
 *
 * 编辑器本体（MonacoEditor.vue）在弹窗内是异步组件，不在本文件的静态图上，故此处无需关心其加载反馈。
 */
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import MonacoEditorBox from '../MonacoEditorBox';

const boxState = vi.hoisted(() => ({ attrs: null as Record<string, unknown> | null }));

// 弹窗内部依赖 monaco 与 worker，用桩组件替代；同时捕获 Box 传入的回调以便断言
vi.mock('../MonacoEditorDialog.vue', () => ({
    default: defineComponent({
        name: 'MonacoEditorDialogStub',
        inheritAttrs: false,
        setup(_props, { attrs }) {
            boxState.attrs = attrs as Record<string, unknown>;
            return () => h('div', { class: 'monaco-dialog-stub' });
        },
    }),
}));
vi.mock('@/hooks/useI18n', () => ({ Msg: { error: vi.fn() } }));

const DIALOG_ID = 'monaco-editor-dialog-id';
const DRAWER_ID = 'monaco-editor-drawer-id';

const stubs = () => document.querySelectorAll(`#${DIALOG_ID} .monaco-dialog-stub, #${DRAWER_ID} .monaco-dialog-stub`);

afterEach(() => {
    for (const el of document.body.querySelectorAll(`#${DIALOG_ID}, #${DRAWER_ID}`)) el.remove();
    boxState.attrs = null;
});

describe('MonacoEditorBox', () => {
    it('调用后同一 tick 内弹窗即渲染完成（无异步加载空窗）', () => {
        MonacoEditorBox({ content: '{}', title: 't', language: 'json' });

        const container = document.getElementById(DIALOG_ID);
        expect(container).not.toBeNull();
        expect(container!.querySelector('.monaco-dialog-stub')).not.toBeNull();
        expect(stubs().length).toBe(1);
    });

    it('弹窗未关闭时重复调用不会再开一个容器', () => {
        MonacoEditorBox({ content: '1', title: 't', language: 'text' });
        MonacoEditorBox({ content: '2', title: 't', language: 'text' });

        expect(document.querySelectorAll(`#${DIALOG_ID}`).length).toBe(1);
        expect(stubs().length).toBe(1);
    });

    it('dialog 与 drawer 各自独立，互不阻塞', () => {
        MonacoEditorBox({ content: 'a', title: 't', language: 'text' });
        MonacoEditorBox({ content: 'b', title: 't', language: 'text', useDrawer: true });

        expect(document.querySelectorAll(`#${DIALOG_ID}`).length).toBe(1);
        expect(document.querySelectorAll(`#${DRAWER_ID}`).length).toBe(1);
        expect(stubs().length).toBe(2);
    });

    it('关闭时卸载弹窗、移除容器并回调 closeFn', () => {
        const closeFn = vi.fn();
        MonacoEditorBox({ content: 'x', title: 't', language: 'text', closeFn });

        (boxState.attrs!.onClose as () => void)();

        expect(document.getElementById(DIALOG_ID)).toBeNull();
        expect(closeFn).toHaveBeenCalledTimes(1);
        // 容器已清理，故可以重新打开
        MonacoEditorBox({ content: 'y', title: 't', language: 'text' });
        expect(stubs().length).toBe(1);
    });

    it('json 语言下内容非法则不回调 confirmFn', async () => {
        const confirmFn = vi.fn();
        const { Msg } = await import('@/hooks/useI18n');
        MonacoEditorBox({ content: '{bad', title: 't', language: 'json', confirmFn });

        (boxState.attrs!.onConfirm as () => void)();

        expect(confirmFn).not.toHaveBeenCalled();
        expect(Msg.error).toHaveBeenCalledWith('common.invalidJson');
    });

    it('json 语言下确认时回传压缩后的文本', () => {
        const confirmFn = vi.fn();
        MonacoEditorBox({ content: '{ "a": 1 }', title: 't', language: 'json', confirmFn });

        (boxState.attrs!.onConfirm as () => void)();

        expect(confirmFn).toHaveBeenCalledWith('{"a":1}');
    });

    it('编辑器内容变更回写到 props，确认时以最新值为准', () => {
        const confirmFn = vi.fn();
        MonacoEditorBox({ content: 'before', title: 't', language: 'text', confirmFn });

        (boxState.attrs!['onUpdate:modelValue'] as (value: string) => void)('after');
        (boxState.attrs!.onConfirm as () => void)();

        expect(confirmFn).toHaveBeenCalledWith('after');
    });

    it('不把编辑内容写回调用方传入的配置对象', () => {
        const options = { content: 'origin', title: 't', language: 'text' };
        MonacoEditorBox(options);

        (boxState.attrs!['onUpdate:modelValue'] as (value: string) => void)('edited');
        (boxState.attrs!.onClose as () => void)();

        // 复用时必须还是初始值，否则第二次打开会带出上一次的内容与默认值
        expect(options).toEqual({ content: 'origin', title: 't', language: 'text' });
    });

    it('弹窗渲染抛错时不留僵尸容器（否则后续点击会被幂等判断永久挡掉）', async () => {
        vi.resetModules();
        vi.doMock('../MonacoEditorDialog.vue', () => ({
            default: defineComponent({
                name: 'MonacoEditorDialogBroken',
                setup() {
                    throw new Error('broken dialog');
                },
            }),
        }));
        const { default: freshBox } = await import('../MonacoEditorBox');
        const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined);

        const run = () => freshBox({ content: 'z', title: 't', language: 'text' });
        // Vue 是否把组件错误重抛出 render() 由本用例锁定：不重抛则 Box 的清理分支需要改写
        expect(run).toThrow('broken dialog');
        expect(document.getElementById(DIALOG_ID)).toBeNull();

        errorSpy.mockRestore();
        vi.doUnmock('../MonacoEditorDialog.vue');
    });

    it('弹窗挂载后仍是空容器时撤销占位（生产下组件错误不抛出，只留空壳）', async () => {
        vi.resetModules();
        vi.doMock('../MonacoEditorDialog.vue', () => ({
            default: defineComponent({
                name: 'MonacoEditorDialogEmpty',
                setup() {
                    return () => null;
                },
            }),
        }));
        const { default: freshBox } = await import('../MonacoEditorBox');

        freshBox({ content: 'e', title: 't', language: 'text' });

        // 空壳不能被当成「弹窗已打开」，否则此后每次点击都被幂等判断静默挡掉
        expect(document.getElementById(DIALOG_ID)).toBeNull();
        vi.doUnmock('../MonacoEditorDialog.vue');
    });
});
