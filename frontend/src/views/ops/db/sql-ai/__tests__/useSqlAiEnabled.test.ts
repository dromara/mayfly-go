import { describe, expect, it, vi } from 'vitest';
import { ref } from 'vue';

// 模拟全站共享的 AI 探测结果（真实实现只探测一次并共享同一个 ref）
const aiEnabledRef = ref(false);
const ensureAiProbed = vi.fn();
vi.mock('@/views/ai/hooks/useAiEnabled', () => ({
    ensureAiProbed,
    useAiEnabled: () => ({ aiEnabled: aiEnabledRef }),
}));

const { useSqlAiEnabled } = await import('../useSqlAiEnabled');

describe('SQL 侧 AI 入口可用性', () => {
    it('未配置模型时不可用（入口不渲染）', () => {
        aiEnabledRef.value = false;
        const { aiEnabled } = useSqlAiEnabled();
        expect(aiEnabled.value).toBe(false);
        expect(ensureAiProbed).toHaveBeenCalled();
    });

    it('配置启用后随共享探测结果变化（探测晚到也能出现入口）', () => {
        aiEnabledRef.value = false;
        const { aiEnabled } = useSqlAiEnabled();
        expect(aiEnabled.value).toBe(false);

        // 登录后补探完成：共享 ref 变 true，入口必须跟着出现（不是 setup 时缓存死值）
        aiEnabledRef.value = true;
        expect(aiEnabled.value).toBe(true);
    });
});
