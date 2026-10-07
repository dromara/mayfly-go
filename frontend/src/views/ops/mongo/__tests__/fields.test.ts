/**
 * 文档呈现派生的测试。
 *
 * 重点是 queryPath：它决定「按值过滤」写出去的条件的 Mongo 语义。展示路径 `tags[0]` 是合法
 * JSON 键名，作为 Mongo 条件却永远匹配不到任何文档——那种「查得到却查不出」的错比报错更难发现。
 */
import { describe, expect, it } from 'vitest';

import type { MongoDoc } from '../types';
import { docKey, flattenFields, idKindLabel, idText, type DocField } from '../docview/fields';

function doc(source: Record<string, unknown>, idToken = ''): MongoDoc {
    return { idToken, idKind: 'objectId', hash: 'h', mode: 'plain', doc: source };
}

function fieldOf(fields: DocField[], path: string): DocField {
    const found = fields.find((field) => field.path === path);
    if (!found) {
        throw new Error(`field ${path} not found: ${fields.map((f) => f.path).join(', ')}`);
    }
    return found;
}

describe('flattenFields', () => {
    it('嵌套文档用点号路径，展示与查询一致', () => {
        const fields = flattenFields({ _id: '1', buyer: { name: 'momo', addr: { city: 'hz' } } });
        expect(fields.map((field) => field.path)).toEqual(['_id', 'buyer.name', 'buyer.addr.city']);
        expect(fields.map((field) => field.queryPath)).toEqual(['_id', 'buyer.name', 'buyer.addr.city']);
    });

    it('数组子项展示带下标，查询路径指回数组本身', () => {
        const fields = flattenFields({ tags: ['vip', 'urgent'] });
        expect(fields.map((field) => field.path)).toEqual(['tags[0]', 'tags[1]']);
        expect(fields.map((field) => field.queryPath)).toEqual(['tags', 'tags']);
    });

    it('嵌套数组同样按 Mongo 语义给路径：items.tags 而非 items[0].tags[0]', () => {
        const fields = flattenFields({ items: [{ tags: ['a'] }] });
        expect(fields.map((field) => field.path)).toEqual(['items[0].tags[0]']);
        expect(fields.map((field) => field.queryPath)).toEqual(['items.tags']);
    });

    it('空文档与空数组留一行且不可过滤', () => {
        const fields = flattenFields({ emptyDoc: {}, emptyArr: [] });
        expect(fieldOf(fields, 'emptyDoc').kind).toBe('Document');
        expect(fieldOf(fields, 'emptyDoc').display).toBe('{}');
        expect(fieldOf(fields, 'emptyDoc').queryPath).toBe('');
        expect(fieldOf(fields, 'emptyArr').kind).toBe('Array');
        expect(fieldOf(fields, 'emptyArr').display).toBe('[]');
        expect(fieldOf(fields, 'emptyArr').queryPath).toBe('');
    });

    it('类型包装是叶子，不再下钻，并标出 typed', () => {
        const fields = flattenFields({ at: { $date: { $numberLong: '1760000000000' } } });
        const date = fieldOf(fields, 'at');
        expect(date.kind).toBe('Date');
        expect(date.typed).toBe(true);
        expect(date.queryPath).toBe('at');
        // 可读值是本地时间串，只断言它已展开成日期而不是嵌套键列表
        expect(date.display).toMatch(/^202\d-\d{2}-\d{2} /);
    });

    it('null 是合法值而不是缺字段', () => {
        const fields = flattenFields({ note: null });
        expect(fieldOf(fields, 'note').display).toBe('null');
        expect(fieldOf(fields, 'note').queryPath).toBe('note');
    });
});

describe('docKey / idKindLabel / idText', () => {
    it('有令牌时用令牌做行 key，翻页后仍稳定', () => {
        expect(docKey(doc({ a: 1 }, 'tok-1'))).toBe('tok-1');
        expect(docKey(doc({ a: 1 }, 'tok-1'))).toBe(docKey(doc({ a: 2 }, 'tok-1')));
    });

    it('无令牌时退回内容特征，并且不承诺唯一', () => {
        expect(docKey(doc({ a: 1 }, ''), 3)).toContain('pos-3');
    });

    it('主键类型显示名与后端 idKind 同源，未知类型原样透出', () => {
        expect(idKindLabel('objectId')).toBe('ObjectId');
        expect(idKindLabel('string')).toBe('String');
        expect(idKindLabel('whateverNextVersionAdds')).toBe('whateverNextVersionAdds');
    });

    it('同形主键的显示文本可以一样，但类型标签不同', () => {
        const hexLike = doc({ _id: '507f1f77bcf86cd799439011' });
        const realOid = doc({ _id: { $oid: '507f1f77bcf86cd799439011' } });
        expect(idText(hexLike)).toBe(idText(realOid));
        expect(idKindLabel('objectId')).not.toBe(idKindLabel('string'));
    });
});
