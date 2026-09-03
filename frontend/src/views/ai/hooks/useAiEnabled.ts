/**
 * useAiEnabled - AI 能力可用性探测
 *
 * 依据系统配置 AiModelConfig（Permission: all，后端 /sys/configs/value 对
 * 公开配置免鉴权返回）判断 AI 模型是否已配置；model/apiKey/baseUrl 三要素
 * 齐全才视为可用。模块级缓存：全站只探测一次，所有消费方共享同一结果，
 * 管理员中途配置需刷新页面生效（与 GlobalNotificationFab 的轻量定位一致）。
 */
import { ref } from 'vue';
import { getAiModelConfig } from '@/common/sysconfig';

/** true: 已配置可用；false: 未配置或探测失败（探测失败静默降级，不打断页面） */
const aiEnabled = ref(false);
let probed = false;

async function probe() {
    try {
        const conf = await getAiModelConfig();
        aiEnabled.value = !!(conf && conf.model && conf.apiKey && conf.baseUrl);
    } catch (e) {
        // 请求失败（未登录/网络异常）视为不可用，console.warn 仅记录异常对象不输出配置内容
        console.warn('[ai] probe AiModelConfig failed:', e);
        aiEnabled.value = false;
    }
}

export function useAiEnabled() {
    if (!probed) {
        probed = true;
        probe();
    }
    return { aiEnabled };
}
