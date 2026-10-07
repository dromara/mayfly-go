/**
 * TagTreeCheck 的 tagType 必须响应变化。
 *
 * 「可治理资源类型」现在由 policy-schema 异步下发，而子组件的 onMounted 早于父组件拿到数据：
 * 只在挂载时请求一次，会出现「首次打开抽屉缺机器节点、重开才对」的竞态，
 * 表现像数据问题而不是代码问题，极难被发现
 */
import { describe, expect, it, vi } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';

const getTagTrees = vi.fn(async (_params: unknown): Promise<unknown[]> => []);

vi.mock('@/views/ops/tag/api', () => ({
    tagApi: { getTagTrees: { request: (params: unknown) => getTagTrees(params) } },
}));

import TagTreeCheck from '../TagTreeCheck.vue';

const i18n = createI18n({ legacy: false, locale: 'zh', messages: { zh: {} } });

const mountTree = (tagType: (number | string)[]) =>
    mount(TagTreeCheck, {
        props: { modelValue: [], tagType },
        global: { plugins: [ElementPlus, i18n] },
    });

describe('生效资源树的资源类型筛选', () => {
    it('挂载时按当前 tagType 请求一次', async () => {
        getTagTrees.mockClear();
        mountTree([3]);
        await flushPromises();

        expect(getTagTrees).toHaveBeenCalledTimes(1);
        expect(getTagTrees).toHaveBeenLastCalledWith({ type: '3' });
    });

    it('tagType 变化（schema 到货）后重新请求', async () => {
        getTagTrees.mockClear();
        const wrapper = mountTree([3]);
        await flushPromises();

        // 父组件的 schema 请求在子组件挂载之后才完成，此时才补上机器类型
        await wrapper.setProps({ tagType: [1, 3] });
        await flushPromises();

        expect(getTagTrees).toHaveBeenCalledTimes(2);
        expect(getTagTrees).toHaveBeenLastCalledWith({ type: '1,3' });
    });

    it('tagType 内容没变时不重复请求', async () => {
        getTagTrees.mockClear();
        const wrapper = mountTree([3]);
        await flushPromises();

        // 同一个数组换了引用但内容一致：不应再打一次请求
        await wrapper.setProps({ tagType: [3] });
        await flushPromises();

        expect(getTagTrees).toHaveBeenCalledTimes(1);
    });
});
