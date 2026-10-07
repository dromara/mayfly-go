import { describe, expect, it, vi } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { i18n } from '@/i18n';

import type { PolicySchema } from '@/components/policy-builder';
import { procdefApi } from '@/views/flow/api';
import ProcdefPolicyHistory from '../ProcdefPolicyHistory.vue';

vi.mock('@/views/flow/api', async (importOriginal) => {
    const actual = await importOriginal<typeof import('@/views/flow/api')>();
    return {
        ...actual,
        procdefApi: { ...actual.procdefApi, policyHistory: { request: vi.fn() } },
    };
});

/**
 * 变更历史区的空态容错。
 *
 * 没有过策略变更的流程定义，接口 data 返回 null，运行时并不是声明里的数组；
 * 少一次兜底就是每次打开流程定义编辑抽屉往控制台丢一条未捕获 rejection
 */
describe('ProcdefPolicyHistory 无历史记录', () => {
    it('接口返回 null 时不渲染历史区，也不产生未捕获 rejection', async () => {
        const rejections: unknown[] = [];
        const onRejection = (reason: unknown) => rejections.push(reason);
        process.on('unhandledRejection', onRejection);
        vi.mocked(procdefApi.policyHistory.request).mockResolvedValue(undefined as never);

        try {
            const wrapper = mount(ProcdefPolicyHistory, {
                props: { procdefId: 1, schema: { scenarios: [] } as unknown as PolicySchema },
                global: { plugins: [i18n, ElementPlus] },
            });
            await flushPromises();
            await new Promise((resolve) => setTimeout(resolve, 0));

            expect(wrapper.find('.policy-history').exists(), '没有历史记录时不该渲染整块区域').toBe(false);
            expect(rejections, 'load() 对 undefined 直接取 map').toHaveLength(0);
        } finally {
            process.off('unhandledRejection', onRejection);
        }
    });
});
