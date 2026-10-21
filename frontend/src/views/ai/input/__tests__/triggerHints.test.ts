/**
 * 输入框提示的组装契约（M6）
 *
 * 此前「从注册表拼 kbd 提示」在 ChatInput 与 PromptEditor 各写一遍，且全角逗号与省略号写死在
 * 代码里——英文界面会带出中文标点。收口到 useTriggerHints 后，这里锁两件事：
 * 清单必须与注册表一一对应（新增触发类型零改动回显），标点必须来自 i18n 而不是代码。
 */
import { afterEach, describe, expect, it } from 'vitest';
import { i18n } from '@/i18n';
// 副作用导入：注册 skill / resource 触发器（不注册则清单为空，下面的断言会假通过）
import '../triggers';
import { getTriggerDefs } from '../triggers';
import { placeholderWithHints, triggerHints } from '../useTriggerHints';

const originalLocale = i18n.global.locale.value;

afterEach(() => {
    i18n.global.locale.value = originalLocale;
});

describe('触发器提示组装', () => {
    it('提示清单与注册表一一对应', () => {
        const defs = getTriggerDefs();
        expect(defs.length).toBeGreaterThan(0);

        const hints = triggerHints();
        expect(hints.map((h) => h.char)).toEqual(defs.map((d) => d.chars[0]));
        for (const hint of hints) {
            expect(hint.label).toBeTruthy();
            // vue-i18n 取不到文案时会原样返回 key：说明 hintI18nKey 拼错或两侧文案缺失
            expect(hint.label).not.toMatch(/^ai\./);
        }
    });

    it('占位符包含每个触发器的「字符 + 说明」', () => {
        const text = placeholderWithHints('ai.chat.inputPlaceholder');
        for (const hint of triggerHints()) {
            expect(text).toContain(`${hint.char} ${hint.label}`);
        }
    });

    it('标点跟随语言切换，不在代码里写死中文标点', () => {
        i18n.global.locale.value = 'en';
        const en = placeholderWithHints('ai.chat.inputPlaceholder');
        expect(en).toBeTruthy();
        // 若分隔符或省略号硬编码在代码里，切到英文后这些全角符号会一起泄漏进文案
        expect(en).not.toMatch(/[，、；…]/);
        expect(en).toContain(i18n.global.t('ai.chat.hintSeparator'));
    });

    it('基础文案始终作为前缀出现（提示拼接不得吃掉原文案）', () => {
        for (const locale of ['zh-cn', 'en']) {
            i18n.global.locale.value = locale;
            const base = i18n.global.t('ai.chat.inputPlaceholder');
            expect(placeholderWithHints('ai.chat.inputPlaceholder').startsWith(base)).toBe(true);
        }
    });
});
