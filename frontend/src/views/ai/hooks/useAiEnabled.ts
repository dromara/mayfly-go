/**
 * useAiEnabled - AI 能力可用性探测
 *
 * 走 GET /ai/status（只回 enabled/model，不回配置原文）：模型配置行的值里含
 * apiKey，浏览器端不需要也不应该拿到密钥。
 *
 * 该接口需要登录态，未登录时必然 501 —— 因此无 token 时不发请求，也不占用
 * 「已探测」标志：登录页不白跑一次请求，登录后首个挂载点再探测。
 * 模块级缓存：全站只成功探测一次，管理员中途配置需刷新页面生效。
 */
import { ref } from 'vue';
import { getToken } from '@/common/utils/storage';
import { aiApi } from '../api';

/** true: 已配置可用；false: 未配置或探测失败（探测失败静默降级，不打断页面） */
const aiEnabled = ref(false);
let probed = false;

async function probe() {
    probed = true;
    try {
        const status = await aiApi.getStatus.request();
        aiEnabled.value = !!status?.enabled;
    } catch (e) {
        // 请求失败（未登录/网络异常）视为不可用，console.warn 仅记录异常对象不输出配置内容
        console.warn('[ai] probe AI status failed:', e);
        aiEnabled.value = false;
    }
}

/** 按需探测：已探测过或未登录时直接返回 */
export function ensureAiProbed() {
    if (!probed && getToken()) {
        probe();
    }
}

export function useAiEnabled() {
    ensureAiProbed();
    return { aiEnabled };
}
