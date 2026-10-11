/**
 * TagCodePath 深路径折叠（autoFold）：按容器实际宽度自适应——放得下全段展示，
 * 放不下才把前置段收进「…」（悬浮可见折叠段名），且只收刚够放下的那些，极限处保留末段。
 * 选择框场景（DbSelectTree）依赖它把深路径压单行，同时宽面板不无谓显示「…」。
 *
 * happy-dom 没有排版引擎：量尺装在被测行元素上（clientWidth 固定、scrollWidth 按文本长度折算），
 * 用假 ResizeObserver 手动触发收敛，验证真实收敛算法而不是像素。
 */
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

// 段名按完整 codePath 映射；每段 3 字，配合折算比例可算出「折几段才放得下」的唯一答案
const SEGMENT_NAMES: Record<string, string> = { 'a/': '段名一', 'a/b/': '段名二', 'a/b/c/': '段名三', 'a/b/c/d/': '段名四' };

const listByQuery = vi.fn(async (params: { codes?: string; tagPaths?: string[] }) => {
    if (params.codes) {
        return [{ codePath: params.codes, name: '' }];
    }
    return (params.tagPaths ?? []).map((tp) => ({ codePath: tp, name: SEGMENT_NAMES[tp] ?? '' }));
});

vi.mock('@/views/ops/tag/api', () => ({
    tagApi: { listByQuery: { request: (params: unknown) => listByQuery(params as { codes?: string; tagPaths?: string[] }) } },
}));

import TagCodePath from '../TagCodePath.vue';

const i18n = createI18n({ legacy: false, locale: 'zh', messages: { zh: {} } });

class FakeResizeObserver {
    static callbacks: Array<() => void> = [];

    constructor(cb: () => void) {
        FakeResizeObserver.callbacks.push(cb);
    }

    observe() {}
    unobserve() {}
    disconnect() {}
}

beforeEach(() => {
    FakeResizeObserver.callbacks.length = 0;
    vi.stubGlobal('ResizeObserver', FakeResizeObserver);
});

afterEach(() => {
    vi.unstubAllGlobals();
});

const mountPath = (code: string, autoFold = false) =>
    mount(TagCodePath, {
        props: { code, autoFold },
        global: { plugins: [ElementPlus, i18n], stubs: { SvgIcon: true } },
    });

/** 选择框宿主：还原 DbSelectTree 的 EP 结构——面包屑行挂在 prefix 内，库名由绝对定位 placeholder 承载 */
const mountInSelect = (code: string, label: string) =>
    mount(
        defineComponent({
            render: () =>
                h('div', { class: 'el-select__wrapper' }, [
                    h('div', { class: 'el-select__prefix' }, [h(TagCodePath, { code, autoFold: true })]),
                    h('div', { class: 'el-select__placeholder' }, label),
                ]),
        }),
        { global: { plugins: [ElementPlus, i18n], stubs: { SvgIcon: true } } }
    );

/** 触发一次尺寸收敛（等价于浏览器里 ResizeObserver 回调被唤起） */
const triggerResize = async () => {
    FakeResizeObserver.callbacks.at(-1)?.();
    await flushPromises();
    await flushPromises();
};

/** 组件根是 fragment（template v-if + popover），VTU 的 wrapper.element 是 app 容器；
    量尺要装在真正的折叠行 el-row 上（也是组件用 ResizeObserver 观察的同一个元素） */
const rowOf = (wrapper: VueWrapper) => wrapper.find('.el-row').element;

/** 给行元素装可控量尺：clientWidth 固定，scrollWidth 按当前文本长度折算（0 = 真实默认的宽松环境） */
const stubMetrics = (el: Element, clientWidth: number, pxPerChar: number) => {
    Object.defineProperty(el, 'clientWidth', { configurable: true, get: () => clientWidth });
    Object.defineProperty(el, 'scrollWidth', { configurable: true, get: () => (el.textContent?.length ?? 0) * pxPerChar });
};

/** 库名标签量尺：scrollWidth = 库名文本宽，clientWidth = 容器剩余宽（总宽扣掉行占宽） */
const stubLabelMetrics = (labelEl: Element, totalWidth: number, rowEl: Element, pxPerChar: number) => {
    Object.defineProperty(labelEl, 'scrollWidth', { configurable: true, get: () => (labelEl.textContent?.length ?? 0) * pxPerChar });
    Object.defineProperty(labelEl, 'clientWidth', { configurable: true, get: () => totalWidth - (rowEl.textContent?.length ?? 0) * pxPerChar });
};

describe('深路径折叠（autoFold 宽度自适应）', () => {
    it('未开启 autoFold 时不折叠：其它使用场景行为不变', async () => {
        const wrapper = mountPath('a/b/');
        await flushPromises();

        // 即使容器溢出、内容变化，没有 autoFold 就永远不折
        stubMetrics(rowOf(wrapper), 50, 20);
        await wrapper.setProps({ code: 'a/b/c/d/' });
        await flushPromises();

        expect(wrapper.find('.tag-path-fold-marker').exists()).toBe(false);
        for (const name of ['段名一', '段名二', '段名三', '段名四']) {
            expect(wrapper.text()).toContain(name);
        }
    });

    it('autoFold 开启但宽度足够：全段展示，不出现「…」', async () => {
        const wrapper = mountPath('a/b/c/d/', true);
        await flushPromises();

        // 200px 装得下 12 字（120px）：重新收敛的结果仍是不折
        stubMetrics(rowOf(wrapper), 200, 10);
        await triggerResize();

        expect(wrapper.find('.tag-path-fold-marker').exists()).toBe(false);
        for (const name of ['段名一', '段名二', '段名三', '段名四']) {
            expect(wrapper.text()).toContain(name);
        }
    });

    it('溢出时折到刚好放得下：只收起放不下的前置段', async () => {
        const wrapper = mountPath('a/b/c/d/', true);
        await flushPromises();

        // 150px：全量 240px 装不下；收 2 段后（… + 2 段 = 140px）放得下，收 1 段（200px）仍放不下
        stubMetrics(rowOf(wrapper), 150, 20);
        await triggerResize();

        expect(wrapper.find('.tag-path-fold-marker').exists()).toBe(true);
        expect(wrapper.text()).toContain('段名三');
        expect(wrapper.text()).toContain('段名四');
        expect(wrapper.text()).not.toContain('段名一');
        expect(wrapper.text()).not.toContain('段名二');
    });

    it('持续溢出时至少保留末段（极限折叠），溢出交给外层省略号', async () => {
        const wrapper = mountPath('a/b/c/d/', true);
        await flushPromises();

        // 50px 连「… + 末段」（80px）都装不下：折到只剩末段为止
        stubMetrics(rowOf(wrapper), 50, 20);
        await triggerResize();

        expect(wrapper.find('.tag-path-fold-marker').exists()).toBe(true);
        expect(wrapper.text()).toContain('段名四');
        expect(wrapper.text()).not.toContain('段名一');
        expect(wrapper.text()).not.toContain('段名二');
        expect(wrapper.text()).not.toContain('段名三');
    });

    it('选择框宿主：行放得下但库名被挤压时继续折叠，折到库名刚好放得下（回展会饿回去）', async () => {
        const wrapper = mountInSelect('a/b/c/d/', 'mayfly_pg');
        await flushPromises();

        const rowEl = rowOf(wrapper);
        const labelEl = wrapper.find('.el-select__placeholder').element;
        // 总宽 200px、10px/字：全展开行占 120px，库名只剩 80px 而需 90px；折 1 段后库名得 100px 刚好放下
        stubMetrics(rowEl, 200, 10);
        stubLabelMetrics(labelEl, 200, rowEl, 10);
        await triggerResize();

        expect(wrapper.find('.tag-path-fold-marker').exists()).toBe(true);
        expect(wrapper.text()).not.toContain('段名一');
        expect(wrapper.text()).toContain('段名二');
        expect(wrapper.text()).toContain('段名三');
        expect(wrapper.text()).toContain('段名四');
    });

    it('选择框宿主：折叠不足以救回库名时折到极限并保留末段', async () => {
        const wrapper = mountInSelect('a/b/c/d/', 'an_extremely_long_database_name');
        await flushPromises();

        const rowEl = rowOf(wrapper);
        const labelEl = wrapper.find('.el-select__placeholder').element;
        // 行永远放得下（120px ≤ 200px）；库名固定要 300px、只有 30px：折到只剩末段也救不回
        stubMetrics(rowEl, 200, 10);
        Object.defineProperty(labelEl, 'scrollWidth', { configurable: true, get: () => 300 });
        Object.defineProperty(labelEl, 'clientWidth', { configurable: true, get: () => 30 });
        await triggerResize();

        expect(wrapper.find('.tag-path-fold-marker').exists()).toBe(true);
        expect(wrapper.text()).toContain('段名四');
        expect(wrapper.text()).not.toContain('段名一');
        expect(wrapper.text()).not.toContain('段名二');
        expect(wrapper.text()).not.toContain('段名三');
    });

    it('选择框宿主：无值占位文案（is-transparent）被截断不驱动折叠', async () => {
        const wrapper = mountInSelect('a/b/c/d/', '请选择数据库');
        await flushPromises();

        const rowEl = rowOf(wrapper);
        const labelEl = wrapper.find('.el-select__placeholder').element;
        labelEl.classList.add('is-transparent');
        // 行完全放得下；未选中时的占位文案虽被截断，但不代表内容饥饿
        stubMetrics(rowEl, 200, 10);
        Object.defineProperty(labelEl, 'scrollWidth', { configurable: true, get: () => 100 });
        Object.defineProperty(labelEl, 'clientWidth', { configurable: true, get: () => 30 });
        await triggerResize();

        expect(wrapper.find('.tag-path-fold-marker').exists()).toBe(false);
    });

    it('选择框宿主：行与库名都放得下时不折叠', async () => {
        const wrapper = mountInSelect('a/b/c/d/', '库一');
        await flushPromises();

        const rowEl = rowOf(wrapper);
        // 全展开行占 120px，库名得 80px 且只需 20px：全段展示
        stubMetrics(rowEl, 200, 10);
        stubLabelMetrics(wrapper.find('.el-select__placeholder').element, 200, rowEl, 10);
        await triggerResize();

        expect(wrapper.find('.tag-path-fold-marker').exists()).toBe(false);
        for (const name of ['段名一', '段名二', '段名三', '段名四']) {
            expect(wrapper.text()).toContain(name);
        }
    });
});