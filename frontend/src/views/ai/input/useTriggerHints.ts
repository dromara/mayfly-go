/**
 * 触发器快捷键提示（占位符与 hint 行的单点组装）
 *
 * 此前「从注册表拼 kbd 提示」在 ChatInput 与 PromptEditor 各写一遍（一处拼占位符文本、一处拼
 * kbd 行），改分隔符或展示顺序要同步两处；且全角逗号与省略号写死在代码里，英文界面下占位符
 * 会带着中文标点。收口到本文件后：数据源仍是触发器注册表，文案与分隔符交给 i18n。
 */
import { i18n } from '@/i18n';
import { getTriggerDefs } from './triggers';

/** 单个触发器的提示：触发字符 + 已翻译的动作说明 */
export interface TriggerHint {
    char: string;
    label: string;
}

/** 触发器提示清单（新增触发类型零改动自动回显，`@` 取 chars 而非 i18n 文本以避开转义差异） */
export function triggerHints(): TriggerHint[] {
    const { t } = i18n.global;
    return getTriggerDefs().map((trig) => ({ char: trig.chars[0], label: t(trig.hintI18nKey) }));
}

/**
 * 输入框占位符 = 基础文案 + 提示清单。
 *
 * 分隔符与整句结构由 i18n 文案决定（`ai.chat.inputPlaceholderHints` / `ai.chat.hintSeparator`），
 * 不在代码里拼中文标点
 */
export function placeholderWithHints(baseKey: string): string {
    const { t } = i18n.global;
    const base = t(baseKey);
    const hints = triggerHints();
    if (hints.length === 0) {
        return base;
    }
    const joined = hints.map((h) => `${h.char} ${h.label}`).join(t('ai.chat.hintSeparator'));
    return t('ai.chat.inputPlaceholderHints', { base, hints: joined });
}
