/**
 * usePromptSkills - 技能引用列表的唯一获取点（`/` 触发菜单数据源）
 *
 * 为什么单独一层：聊天输入框与表单里的提示词编辑器都要这份列表。各调一次接口会出现两个
 * 请求点与两份失败处理；模块级缓存 + 同刻去重后，接口签名变化只需改这里。
 *
 * 失败不写缓存（下次进入可重试），并静默降级为空列表——引用菜单空掉不该阻断输入本身。
 */
import { ref } from 'vue';
import { aiApi } from '../api';
import type { SkillItem } from './types';

let cached: SkillItem[] | null = null;
let inflight: Promise<SkillItem[]> | null = null;
/** 失效代际号：失效发生在请求在途时，旧响应不得写回缓存 */
let generation = 0;

const fetchSkills = (): Promise<SkillItem[]> => {
    if (cached) return Promise.resolve(cached);
    if (!inflight) {
        const gen = ++generation;
        inflight = aiApi.listSkills
            .request()
            .then((list) => {
                const skills = list || [];
                // 期间被 invalidate 过（技能刚改完）：丢掉这份旧结果，不写回缓存
                if (gen === generation) cached = skills;
                return skills;
            })
            .catch((e) => {
                console.error('[ai] load skills failed:', e);
                return [] as SkillItem[];
            })
            .finally(() => {
                inflight = null;
            });
    }
    return inflight;
};

export function usePromptSkills() {
    const skills = ref<SkillItem[]>(cached ?? []);

    /** 加载技能列表（已缓存则同步命中）；调用方在挂载时触发一次即可 */
    const load = async () => {
        skills.value = await fetchSkills();
    };

    return { skills, load };
}

/**
 * 失效技能缓存（技能新增/改名/删除后调用）。
 *
 * 必须是模块级函数而不是实例方法：缓存跨组件实例共享，而写入发生在插件管理页、
 * 消费发生在输入框，两者没有父子关系。不失效的表现是「刚编辑完技能，本会话内的 `/` 菜单
 * 永远读到旧列表」，刷新页面又正常，极难归因
 */
export function invalidatePromptSkills(): void {
    // 同时推代际号并丢弃在途 Promise：只置 cached=null 的话，一个在途旧请求的回调
    // 仍会把旧列表写回缓存，失效等于没生效
    generation++;
    cached = null;
    inflight = null;
}
