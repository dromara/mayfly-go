/**
 * 模板系统 i18n 契约测试
 *
 * 守护 P2-10 的重构目标：内置模板「直接持 i18n key」，不再有「中文串 → key」反查表。
 * 反查表以中文字面量为键，数据里的中文一改就静默失配、退回显示原始中文，
 * 且天生覆盖不全（字典/审计表列、全部索引注释都曾无处登记），非中文语言下中英混排。
 */
import { describe, it, expect, vi, afterEach } from 'vitest';

// 用真实词条构造可控 i18n 单例；工厂内动态 import 以规避 vi.mock 提升导致的 TDZ
vi.mock('@/i18n', async () => {
    const { createI18n } = await import('vue-i18n');
    const zhDb = (await import('@/i18n/zh-cn/db')).default;
    const enDb = (await import('@/i18n/en/db')).default;
    const i18n = createI18n({
        legacy: false,
        globalInjection: true,
        locale: 'zh-cn',
        fallbackLocale: 'zh-cn',
        messages: { 'zh-cn': { db: zhDb.db }, en: { db: enDb.db } },
    });
    return { i18n };
});

import { i18n } from '@/i18n';
import { templateService } from '../templateService';
import { builtinTemplateData } from '../templateData';
import * as templateData from '../templateData';

const setLocale = (locale: 'zh-cn' | 'en') => {
    (i18n.global.locale as unknown as { value: string }).value = locale;
};

afterEach(() => setLocale('zh-cn'));

describe('内置模板 i18n：直接持 key', () => {
    it('可翻译字段一律持 db.tpl* key，不含任何中文字面量', () => {
        const cjk = /[\u4e00-\u9fa5]/;
        for (const tpl of builtinTemplateData) {
            expect(tpl.name, `${tpl.id}.name 应持 key`).toMatch(/^db\.tpl/);
            expect(tpl.description, `${tpl.id}.description 应持 key`).toMatch(/^db\.tpl/);
            for (const tag of tpl.tags) {
                expect(tag, `${tpl.id}.tags 应持 key`).toMatch(/^db\.tpl/);
            }
            for (const col of tpl.columns) {
                expect(col.comment, `${tpl.id}.${col.name}.comment 应持 key`).toMatch(/^db\.tpl/);
                expect(cjk.test(col.comment), `${tpl.id}.${col.name}.comment 不应含中文`).toBe(false);
            }
            for (const idx of tpl.indexes ?? []) {
                expect(idx.comment, `${tpl.id}.${idx.name}.comment 应持 key`).toMatch(/^db\.tpl/);
            }
        }
    });

    it('已删除中文串反查表 COMMENT_I18N 与平行映射表 TEMPLATE_I18N', () => {
        const ns = templateData as unknown as Record<string, unknown>;
        expect(ns.COMMENT_I18N, 'COMMENT_I18N 反查表应已删除').toBeUndefined();
        expect(ns.TEMPLATE_I18N, 'TEMPLATE_I18N 平行映射表应已删除').toBeUndefined();
    });

    it('按当前语言翻译元数据、列注释与索引注释（覆盖此前从不翻译的字典列/索引）', () => {
        setLocale('zh-cn');
        const zh = templateService.getBuiltinTemplates().find((t) => t.id === 'dictionary_table')!;
        expect(zh.name).toBe('字典表');
        expect(zh.columns.find((c) => c.name === 'dict_type')!.comment).toBe('字典类型');
        expect(zh.indexes!.find((i) => i.name === 'idx_dict_type')!.comment).toBe('字典类型索引');

        setLocale('en');
        const en = templateService.getBuiltinTemplates().find((t) => t.id === 'dictionary_table')!;
        expect(en.name).toBe('Dictionary Table');
        expect(en.columns.find((c) => c.name === 'dict_type')!.comment).toBe('Dict Type');
        expect(en.indexes!.find((i) => i.name === 'idx_dict_type')!.comment).toBe('Dict type index');
    });

    it('applyTemplate 落地的列备注是译文而非 i18n key', () => {
        setLocale('en');
        const tpl = templateService.getBuiltinTemplates().find((t) => t.id === 'user_table')!;
        const table = templateService.applyTemplate(tpl, { tableName: 't_user' });
        const idCol = table.columns.find((c) => c.name === 'id')!;
        expect(idCol.remark).toBe('Primary Key ID');
        expect(idCol.remark).not.toContain('db.tpl');
    });
});
