import { beforeEach, describe, expect, it, vi } from 'vitest';

// 模块级缓存（probed / aiEnabled）是每个用例重新导入模块才隔离的，故统一动态 import
const loadHook = async () => {
    vi.resetModules();
    return await import('../useAiEnabled');
};

const mockGetToken = vi.fn();
const mockGetStatus = vi.fn();

vi.mock('@/common/utils/storage', () => ({ getToken: () => mockGetToken() }));
vi.mock('../../api', () => ({ aiApi: { getStatus: { request: () => mockGetStatus() } } }));

describe('useAiEnabled', () => {
    beforeEach(() => {
        mockGetToken.mockReset();
        mockGetStatus.mockReset();
        vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    });

    it('未登录不发请求，也不占用探测标志：登录后首个挂载点仍会探测', async () => {
        mockGetToken.mockReturnValue('');
        const { useAiEnabled } = await loadHook();

        const first = useAiEnabled();
        expect(first.aiEnabled.value).toBe(false);
        expect(mockGetStatus).not.toHaveBeenCalled();

        // 登录后再次挂载（组件重新 setup）
        mockGetToken.mockReturnValue('token-1');
        mockGetStatus.mockResolvedValue({ enabled: true });
        const second = useAiEnabled();
        await vi.waitFor(() => expect(second.aiEnabled.value).toBe(true));
        expect(mockGetStatus).toHaveBeenCalledTimes(1);
    });

    it('已登录且模型已配置时置为可用', async () => {
        mockGetToken.mockReturnValue('token-1');
        mockGetStatus.mockResolvedValue({ enabled: true, model: 'qwen3.8-flash' });
        const { useAiEnabled } = await loadHook();

        const { aiEnabled } = useAiEnabled();
        await vi.waitFor(() => expect(aiEnabled.value).toBe(true));
    });

    it('探测失败静默降级为不可用', async () => {
        mockGetToken.mockReturnValue('token-1');
        mockGetStatus.mockRejectedValue(new Error('permission denied'));
        const { useAiEnabled } = await loadHook();

        const { aiEnabled } = useAiEnabled();
        expect(aiEnabled.value).toBe(false);
        // 异步 reject 已内部捕获，不应抛出未处理异常
        await vi.waitFor(() => expect(mockGetStatus).toHaveBeenCalledTimes(1));
    });

    it('ensureAiProbed 支持登录后补探，且已探测后不重复请求', async () => {
        mockGetToken.mockReturnValue('');
        const { ensureAiProbed, useAiEnabled } = await loadHook();

        ensureAiProbed();
        expect(mockGetStatus).not.toHaveBeenCalled();

        mockGetToken.mockReturnValue('token-1');
        mockGetStatus.mockResolvedValue({ enabled: true });
        const { aiEnabled } = useAiEnabled();
        await vi.waitFor(() => expect(aiEnabled.value).toBe(true));

        ensureAiProbed();
        expect(mockGetStatus).toHaveBeenCalledTimes(1);
    });

    it('探测成功后不再重复请求（全站只探一次）', async () => {
        mockGetToken.mockReturnValue('token-1');
        mockGetStatus.mockResolvedValue({ enabled: true });
        const { useAiEnabled } = await loadHook();

        useAiEnabled();
        await vi.waitFor(() => expect(mockGetStatus).toHaveBeenCalledTimes(1));
        useAiEnabled();
        expect(mockGetStatus).toHaveBeenCalledTimes(1);
    });
});
