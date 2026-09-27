/**
 * 表单接管与弹层内容渲染的时序契约（宿主 AutoFormDrawer × 页面 useAutoFormModel）
 *
 * 守护缺陷：抽屉关闭时宿主也会渲染 #footer 内容（el-drawer 面板常驻，显隐由遮罩控制），
 * 该次渲染早于 @opened。页面若在渲染期用 requireForm 派生 UI 状态（向导「下一步」的 disabled 等），
 * 会抛「form is read before @opened」并打断渲染，抽屉再也打不开。
 * 因此契约是：渲染期派生状态读 form 并按未接管兜底，requireForm 只用于交互期（提交/联动）读取。
 */
import { describe, expect, it, vi } from 'vitest';
import { computed, defineComponent, h, nextTick, ref, watch } from 'vue';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';
import AutoFormDrawer from '@/components/auto-form/AutoFormDrawer.vue';
import { useAutoFormModel } from '@/hooks/useAutoFormModel';
import type { AutoFormItem } from '@/components/auto-form/types';

vi.mock('@/components/monaco/MonacoEditor.vue', () => ({
    default: defineComponent({ name: 'MonacoEditorStub', props: ['modelValue', 'options'], template: '<div class="monaco-stub" />' }),
}));
vi.mock('@/components/svg-icon/index.vue', () => ({
    default: defineComponent({ name: 'SvgIconStub', template: '<i class="svg-icon-stub" />' }),
}));
vi.mock('@/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }));
vi.mock('@/common/request', () => ({ default: { request: vi.fn().mockResolvedValue([]) } }));

const i18n = createI18n({ legacy: false, globalInjection: true, locale: 'en', messages: { en: { common: { status: 'Status' } } } });

const items: AutoFormItem[] = [{ prop: 'name', label: 'name' }];

type PageForm = { name?: string };

/** 模拟页面：#footer 里的「下一步」按钮 disabled 由表单是否填齐决定 */
const Page = defineComponent({
    props: { open: { type: Boolean, default: false }, data: { type: Object as () => PageForm | null, default: null } },
    emits: ['hostFormTaken'],
    setup(props, { emit }) {
        const visible = ref(false);
        watch(
            () => props.open,
            (v) => (visible.value = v)
        );

        const { form, onOpened, requireForm } = useAutoFormModel<PageForm>();
        // 渲染期派生状态：未接管即兜底为「未完成」，不得读 requireForm
        const ready = computed(() => !!form.value?.name);

        return () =>
            h(
                AutoFormDrawer,
                {
                    visible: visible.value,
                    'onUpdate:visible': (v: boolean) => (visible.value = v),
                    items,
                    data: props.data,
                    // 宿主保证 confirm 发生在 opened 之后：接管完成那刻起交互期读取必然可用
                    onOpened: (raw: PageForm) => {
                        onOpened(raw);
                        emit('hostFormTaken', requireForm().name);
                    },
                },
                { footer: () => h('button', { class: 'open-order-next', disabled: !ready.value }, 'next') }
            );
    },
});

const nextBtn = () => document.querySelector<HTMLButtonElement>('.open-order-next');

describe('AutoFormDrawer 表单接管与内容渲染时序', () => {
    it('关闭态已渲染 #footer 但尚未接管表单，派生状态兜底而非抛错', async () => {
        const wrapper = mount(Page, { attachTo: document.body, props: { open: false }, global: { plugins: [ElementPlus, i18n] } });
        await flushPromises();

        expect(nextBtn(), '抽屉关闭时宿主也应渲染 #footer').not.toBeNull();
        expect(nextBtn()?.disabled).toBe(true);
        expect(wrapper.emitted('hostFormTaken')).toBeUndefined();
        wrapper.unmount();
    });

    it('打开回填后接管表单，派生状态随宿主表单更新且交互期读取可用', async () => {
        const wrapper = mount(Page, { attachTo: document.body, props: { open: false, data: { name: 'r1' } }, global: { plugins: [ElementPlus, i18n] } });
        await flushPromises();
        expect(nextBtn()?.disabled).toBe(true);

        await wrapper.setProps({ open: true });
        await nextTick();
        await flushPromises();

        expect(wrapper.emitted('hostFormTaken')?.[0]).toEqual(['r1']);
        expect(nextBtn()?.disabled).toBe(false);
        wrapper.unmount();
    });
});

describe('useAutoFormModel 读取边界', () => {
    it('未接管时 requireForm 抛错（交互期时序错误的 fail-fast 语义）', () => {
        const { form, requireForm } = useAutoFormModel<PageForm>();
        expect(form.value).toBeUndefined();
        expect(requireForm).toThrow('form is read before @opened');
    });
});
