/**
 * SQL 侧 AI 入口的可见性判据。
 *
 * 语义：只有系统里**配好了可用模型**才出现 AI 相关入口（工具栏按钮、注释补全开关、右键菜单）。
 * 探测只做一次并全站共享（复用 AI 助手同一份 `/ai/status` 探测，避免两处各发一次请求、
 * 也避免"助手入口出现了但草稿入口没出现"这类不一致）。
 *
 * 未配置时入口直接不渲染，而不是渲染后点了报错：用户没有配置模型时，界面上不该出现
 * 一个注定失败的功能。
 */
import { computed } from 'vue';
import { ensureAiProbed, useAiEnabled } from '@/views/ai/hooks/useAiEnabled';

export function useSqlAiEnabled() {
    // 已探测过则不会再发请求；未登录时该调用是 no-op（登录后由首个挂载点补探）
    ensureAiProbed();
    const { aiEnabled } = useAiEnabled();
    return { aiEnabled: computed(() => aiEnabled.value === true) };
}
