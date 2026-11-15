import { describe, expect, it } from 'vitest';
import { DRAFT_INTENTS, canRunWithoutPrompt, defaultDraftIntent, findDraftIntent, listDraftIntents } from '../draftIntents';
import type { DraftEditorContext } from '../draftIntents';

// i18n 字典与运行时同源：意图标题/提示写的是 key，缺 key 时界面会原样吐出 key
const zhModules = import.meta.glob<{ default: object }>('@/i18n/zh-cn/*.ts', { eager: true });
const enModules = import.meta.glob<{ default: object }>('@/i18n/en/*.ts', { eager: true });

function merge(modules: Record<string, { default: object }>): Record<string, unknown> {
    return Object.values(modules).reduce<Record<string, unknown>>((acc, mod) => Object.assign(acc, mod.default), {});
}

function hasKey(dict: Record<string, unknown>, key: string): boolean {
    return key.split('.').reduce<unknown>((node, part) => (node as Record<string, unknown>)?.[part], dict) !== undefined;
}

const dictionaries = { zh: merge(zhModules), en: merge(enModules) };

const material = (ctx: Partial<DraftEditorContext>): DraftEditorContext => ({ hasSelection: false, statement: '', ...ctx });

describe('SQL 草稿意图注册表', () => {
    it('意图 id 集合与后端注册集合一致', () => {
        // 后端 sqldraft/intent.go 注册的四个意图；任一侧新增都必须同步，否则入口会 404
        expect([...DRAFT_INTENTS.map((intent) => intent.id)].sort()).toEqual(['draft', 'explain', 'fix', 'optimize']);
    });

    it('没有材料时只展示「按描述生成」', () => {
        expect(listDraftIntents(material({})).map((intent) => intent.id)).toEqual(['draft']);
    });

    it('有光标语句时展示全部四个意图，并按 order 排序', () => {
        expect(listDraftIntents(material({ statement: 'select * from t_user' })).map((intent) => intent.id)).toEqual(['draft', 'fix', 'optimize', 'explain']);
    });

    it('每个意图的标题与提示在两种语言字典里都存在', () => {
        for (const intent of DRAFT_INTENTS) {
            for (const key of [intent.titleKey, intent.hintKey]) {
                expect(hasKey(dictionaries.zh, key), `zh 缺少 ${key}`).toBe(true);
                expect(hasKey(dictionaries.en, key), `en 缺少 ${key}`).toBe(true);
            }
        }
        expect(findDraftIntent('fix')?.requiresSelection).toBe(true);
    });
});

describe('意图与零输入判定', () => {
    it('有材料时默认「改写这段」，没有材料时默认「按描述生成」', () => {
        expect(defaultDraftIntent(material({}))).toBe('draft');
        expect(defaultDraftIntent(material({ hasSelection: true }))).toBe('fix');
        expect(defaultDraftIntent(material({ statement: 'select 1' }))).toBe('fix');
        // 光标语句只有空白等于没有语句：不能据此进入「改写」，否则会撞上一条必然失败的请求
        expect(defaultDraftIntent(material({ statement: '   ' }))).toBe('draft');
    });

    it('零输入判定同时要求「允许空描述」与「材料已就位」', () => {
        expect(canRunWithoutPrompt(findDraftIntent('optimize'), material({ hasSelection: true }))).toBe(true);
        expect(canRunWithoutPrompt(findDraftIntent('optimize'), material({ statement: 'select 1' }))).toBe(true);
        // 优化/解释必须有 SQL 材料：只看 promptOptional 会放行一条必然被后端拒掉的空请求
        expect(canRunWithoutPrompt(findDraftIntent('optimize'), material({}))).toBe(false);
        expect(canRunWithoutPrompt(findDraftIntent('draft'), material({ hasSelection: true }))).toBe(false);
        expect(canRunWithoutPrompt(undefined, material({ hasSelection: true }))).toBe(false);
    });
});