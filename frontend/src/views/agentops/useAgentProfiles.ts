/**
 * Agent 装配档目录（档位选项与展示名的单点来源）
 *
 * 为什么需要：后端 `RegisterProfile` 是开放扩展点，而档位清单此前写死在前端静态枚举里——
 * 新增一档必须改前端才选得到。更要紧的是表单能选出「受控执行档 + 定时触发」这种组合：该档
 * 没拿到一次性运行许可时，护栏会拒掉它的**全部**工具调用，于是任务每次运行都「成功但一个工具
 * 也没调过」，产出一份空诊断且没有任何失败信号。后端已在保存期拦截，这里负责提前置灰并说明原因。
 *
 * 分工（避免第二个真源）：
 *   - 「有哪些档、能否无人值守派发」以后端注册表为真源（本文件唯一取数口）；
 *   - 「档叫什么、用什么颜色」是展示关注点，仍由前端枚举持有 i18n 键；未登记的新档回退后端
 *     登记的 name，保证新增档位「立刻可选可看」，中文化按需补一行枚举即可。
 */
import { EnumValue } from '@/common/Enum';
import { OptionsProvider } from '@/components/data-table/SearchForm';
import { i18n } from '@/i18n';
import { agentProfileApi } from './api';
import { ProfileEnum } from './enums';
import type { AgentProfileInfo } from './types';

/** 模块级缓存：多个页面共用一次请求（失败不写缓存，下次进入自动重试） */
let cached: AgentProfileInfo[] | null = null;
let inflight: Promise<AgentProfileInfo[]> | null = null;
/** 失效代际号：失效发生在请求在途时，旧响应不得写回缓存 */
let generation = 0;

/** 拉取档位目录（并发挂载共用一次请求，与 usePromptSkills 同一形态） */
export async function loadAgentProfiles(): Promise<AgentProfileInfo[]> {
    if (cached) {
        return cached;
    }
    if (!inflight) {
        const gen = ++generation;
        inflight = agentProfileApi.list
            .request()
            .then((list) => {
                const profiles = list ?? [];
                if (gen === generation) cached = profiles;
                return profiles;
            })
            .finally(() => {
                inflight = null;
            });
    }
    return inflight;
}

/** 失效缓存（后端档位变更后需要重新拉取时调用）；同时丢弃在途旧结果 */
export function invalidateAgentProfiles(): void {
    generation++;
    cached = null;
    inflight = null;
}

/** 档位展示名：优先前端 i18n，未登记的新档回退后端 name，都没有则显示原 id */
export function profileLabel(id: string): string {
    const registered = EnumValue.getEnumByValue(ProfileEnum, id);
    if (registered) {
        return i18n.global.t(registered.label);
    }
    return cached?.find((p) => p.id === id)?.name ?? id;
}

/** 选项文案：需一次性许可的档位追加说明，让用户明白它为什么不可选 */
function optionLabel(profile: AgentProfileInfo): string {
    const name = profileLabel(profile.id);
    return profile.requireRunGrant ? i18n.global.t('agentops.profileGrantOnly', { name }) : name;
}

/**
 * 表单档位选项：`requireRunGrant` 的档位置灰。
 *
 * 置灰而不是移除：移除会让人以为系统没有这个能力，而它确实存在——只是不能无人值守用
 */
export async function profileFormOptions(): Promise<{ value: string; label: string; disabled?: boolean }[]> {
    const profiles = await loadAgentProfiles();
    return profiles.map((p) => ({ value: p.id, label: optionLabel(p), disabled: p.requireRunGrant }));
}

/**
 * 列表筛选器的档位选项（与表单同源）：新增档位无需改前端即可被筛出。
 *
 * 必须复用 loadAgentProfiles 而不是直接拿 Api 另起一条请求：另起路径会让
 * invalidateAgentProfiles 对筛选器完全失效，同一份数据两条取数口也是双真源。
 *
 * 筛选器不置灰：历史运行可能就是用「当时可选、现在需要许可」的档跑出来的，必须仍可筛
 */
export function profileOptionsApi(): OptionsProvider {
    return {
        params: {},
        immediate: true,
        once: false,
        getOptions: async () => (await loadAgentProfiles()).map((p) => ({ label: profileLabel(p.id), value: p.id })),
    };
}
