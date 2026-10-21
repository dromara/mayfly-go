/**
 * 告警三个「按标签匹配」表单的提示文案契约。
 *
 * 通知策略、升级策略、静默三个表单都有同一个「匹配标签」输入，且三者的标签留空都表示
 * 「匹配该资源类型下的全部告警」（见 alert/application/label_match.go 的 matchRuleLabelsMap）。
 * 这个 fail-open 默认值必须在各自的提示里说出来，否则配置人会以为「不填就是不生效」。
 *
 * 真机踩过的形态：升级策略表单复用了静默的提示键，界面上写着「只有标签匹配的告警才会被静默」——
 * 说的是另一个功能的后果；而真正要紧的「留空则匹配全部」被埋在后半句，配置人漏看之后
 * 留下了一条没有标签绑定的启用态升级策略，于是每一条机器告警未恢复满 5 分钟就被推到外部群。
 */
import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

const alertDir = join(__dirname, '..');

const zhModules = import.meta.glob<{ default: Record<string, unknown> }>('@/i18n/zh-cn/alert.ts', { eager: true });
const enModules = import.meta.glob<{ default: Record<string, unknown> }>('@/i18n/en/alert.ts', { eager: true });

/** 取出 alert 命名空间下的文案字典（zh / en 各一份） */
function alertDict(modules: Record<string, { default: Record<string, unknown> }>): Record<string, string> {
    const mod = Object.values(modules)[0].default as { alert: Record<string, string> };
    return mod.alert;
}

const zh = alertDict(zhModules);
const en = alertDict(enModules);

/** 三个表单各自「匹配标签」字段用的提示键 */
const forms = [
    { name: '通知策略', file: 'notify-policy/AlertNotifyPolicyList.vue', key: 'notifyPolicyTips' },
    { name: '升级策略', file: 'escalation/AlertEscalationList.vue', key: 'escalationMatchLabelsTips' },
    { name: '静默', file: 'silence/AlertSilenceList.vue', key: 'silenceMatchLabelsTips' },
] as const;

/** 读出表单源码里「匹配标签」那一栏实际引用的提示键 */
function referencedTipKey(source: string): string | undefined {
    // 匹配标签栏的 tooltip 紧跟在 $t('alert.matchLabels') 之后，取其后第一个 Tips 键
    const at = source.indexOf("$t('alert.matchLabels')");
    expect(at, '表单里找不到「匹配标签」栏，用例前提已失效').toBeGreaterThan(-1);
    const m = /\$t\('alert\.([A-Za-z0-9_]*Tips)'\)/.exec(source.slice(at));
    return m?.[1];
}

describe('告警标签匹配提示文案', () => {
    it('三个表单各引用自己的提示键，不再共用一个', () => {
        const used = forms.map((f) => {
            const source = readFileSync(join(alertDir, f.file), 'utf-8');
            const key = referencedTipKey(source);
            expect(key, `${f.name} 表单（${f.file}）的匹配标签栏没有引用任何 *Tips 键`).toBeTruthy();
            expect(key, `${f.name} 表单引用的提示键与用例登记的不一致`).toBe(f.key);
            return key as string;
        });

        expect(new Set(used).size, `三个表单仍在共用提示键：${used.join(', ')}`).toBe(used.length);
    });

    it('每条提示都存在中英两份，且都说出「留空即匹配全部」这个后果', () => {
        for (const f of forms) {
            for (const [lang, dict] of [
                ['zh', zh],
                ['en', en],
            ] as const) {
                const text = dict[f.key];
                expect(text, `alert.${f.key} 缺少 ${lang} 文案`).toBeTruthy();
                // fail-open 默认值必须显式说出来：留空不是「不生效」，而是「全都命中」
                expect(text, `alert.${f.key} 的 ${lang} 文案没有说明留空的后果`)
                    .toMatch(/留空|Leave .*empty|Leave empty/i);
                expect(text, `alert.${f.key} 的 ${lang} 文案没有说明留空会匹配全部`)
                    .toMatch(/所有|全部|all|every/i);
            }
        }
    });

    it('文案不得串台：升级策略的提示里不能出现「静默」，静默的提示里不能出现「升级」', () => {
        expect(zh.escalationMatchLabelsTips).not.toMatch(/静默/);
        expect(en.escalationMatchLabelsTips).not.toMatch(/silenc/i);
        expect(zh.silenceMatchLabelsTips).not.toMatch(/升级/);
        expect(en.silenceMatchLabelsTips).not.toMatch(/escalat/i);
        // 升级策略的文案必须真的在讲升级
        expect(zh.escalationMatchLabelsTips).toMatch(/升级/);
        expect(en.escalationMatchLabelsTips).toMatch(/escalat/i);
    });
});
