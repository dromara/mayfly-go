/**
 * 事件触发源目录（触发源选项的单点来源）
 *
 * 为什么需要：`trigger.Registry` 是后端开放扩展点——新增一种来源只需实现契约并 Install 一行。
 * 但表单的「触发源」下拉此前拿前端静态枚举装配，于是新来源**根本无法被创建**（表单里没有这个
 * 选项），已存在的行还会在列表里渲染成红色「-」，看起来像数据异常。注册表的可扩展性因此是假的。
 *
 * 分工（避免第二个真源）：
 *   - 「有哪些来源」以后端注册表为真源（本文件唯一取数口）；
 *   - 「来源叫什么」是展示关注点，仍由前端枚举持有 i18n 键；未登记的新来源回退显示原始类型键
 *     （alert / webhook / 自定义名），保证「立刻可选可看」，中文化按需补一行枚举即可。
 *     后端刻意只回类型键不回中文名：后端写死中文会让英文界面永远看到中文。
 */
import { EnumValue } from '@/common/Enum';
import { OptionsProvider } from '@/components/data-table/SearchForm';
import { i18n } from '@/i18n';
import { agentTriggerSourceApi } from './api';
import { TriggerSourceEnum } from './enums';

/** 模块级缓存：多个页面共用一次请求（失败不写缓存，下次进入自动重试） */
let cached: string[] | null = null;
let inflight: Promise<string[]> | null = null;
/** 失效代际号：失效发生在请求在途时，旧响应不得写回缓存 */
let generation = 0;

/** 拉取已注册触发源的类型标识清单 */
export async function loadTriggerSources(): Promise<string[]> {
    if (cached) {
        return cached;
    }
    if (!inflight) {
        const gen = ++generation;
        inflight = agentTriggerSourceApi.list
            .request()
            .then((list) => {
                const sources = list ?? [];
                if (gen === generation) cached = sources;
                return sources;
            })
            .finally(() => {
                inflight = null;
            });
    }
    return inflight;
}

/** 失效缓存（后端来源变更后需要重新拉取时调用）；同时丢弃在途旧结果 */
export function invalidateTriggerSources(): void {
    generation++;
    cached = null;
    inflight = null;
}

/** 来源展示名：已登记类型走前端 i18n，未登记的新来源回退原始类型键 */
export function triggerSourceLabel(type: string): string {
    const registered = EnumValue.getEnumByValue(TriggerSourceEnum, type);
    return registered ? i18n.global.t(registered.label) : type;
}

/** 表单来源选项：以注册表为准，新增来源无需改前端即可被选中 */
export async function triggerSourceFormOptions(): Promise<{ value: string; label: string }[]> {
    return (await loadTriggerSources()).map((type) => ({ value: type, label: triggerSourceLabel(type) }));
}

/**
 * 列表筛选器的来源选项（与表单同源）：复用同一取数口，避免第二条请求让失效逻辑失效。
 */
export function triggerSourceOptionsApi(): OptionsProvider {
    return {
        params: {},
        immediate: true,
        once: false,
        getOptions: async () => (await loadTriggerSources()).map((type) => ({ label: triggerSourceLabel(type), value: type })),
    };
}
