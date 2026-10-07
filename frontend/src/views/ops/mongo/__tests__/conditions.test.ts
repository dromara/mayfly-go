/**
 * 条件区纯计算测试。
 *
 * 守护的是那条硬约束：chip 上显示的摘要必须与即将发出的条件同源，
 * 且「只清掉排序」这类局部修改不能把其他条件一起改掉。
 */
import { describe, expect, it } from 'vitest';

import type { MongoDoc } from '../types';
import {
    activeChips,
    applyEquality,
    applySort,
    chipValue,
    clearedText,
    compactValue,
    hasCondition,
    idInFilter,
    isEmptyCondition,
    parseSortDoc,
    projectionSummary,
    sortSummary,
} from '../docview/conditions';

describe('isEmptyCondition / compactValue', () => {
    it('空串、空白与 {} 都算没条件', () => {
        expect(isEmptyCondition('')).toBe(true);
        expect(isEmptyCondition('   \n ')).toBe(true);
        expect(isEmptyCondition('{ }')).toBe(true);
        expect(isEmptyCondition('{"a":1}')).toBe(false);
    });

    it('摘要压成单行并按长度截断', () => {
        expect(compactValue('{\n  "a": 1\n}')).toBe('{"a":1}');
        const long = JSON.stringify({ description: 'x'.repeat(80) });
        const short = compactValue(long, 20);
        expect(short.length).toBe(21); // 20 字符 + 省略号
        expect(short.endsWith('…')).toBe(true);
    });
});

describe('sortSummary', () => {
    it('数字方向翻成箭头，键序即优先级所以不重排', () => {
        expect(sortSummary('{"qty":-1,"createdAt":1}')).toBe('qty ↓, createdAt ↑');
    });

    it('文本索引方向原样透出而不是假装升降序', () => {
        expect(sortSummary('{"loc":"2dsphere","title":"text"}')).toBe('loc: 2dsphere, title: text');
    });

    it('文本非法时退回压缩原文，摘要不能显示一个与请求不同的条件', () => {
        expect(sortSummary('{ bad json')).toBe('{badjson');
    });

    it('无条件时给空串', () => {
        expect(sortSummary('')).toBe('');
    });
});

describe('projectionSummary', () => {
    it('纯包含与纯排除分别用 +/- 前缀', () => {
        expect(projectionSummary('{"amount":1,"buyer":1}')).toBe('+amount, buyer');
        expect(projectionSummary('{"_id":0}')).toBe('-_id');
    });

    it('包含与排除混用时退回原文（那种写法本身就有歧义）', () => {
        expect(projectionSummary('{"a":1,"b":0}')).toBe('{"a":1,"b":0}');
    });
});

describe('activeChips', () => {
    const texts = { filterText: '{"status":"paid"}', sortText: '{"createdAt":-1}', projectionText: '' };

    it('只列出真正生效的条件，顺序按先筛后序再取列', () => {
        expect(activeChips(texts).map((chip) => chip.name)).toEqual(['filter', 'sort']);
    });

    it('filter 回到 {} 即不再算生效', () => {
        expect(activeChips({ ...texts, filterText: '{}' }).map((chip) => chip.name)).toEqual(['sort']);
    });

    it('三项都空时没有任何条件', () => {
        expect(hasCondition({ filterText: ' {} ', sortText: '', projectionText: '  ' })).toBe(false);
    });
});

describe('chipValue', () => {
    it('每个条件名都用自己的摘要规则', () => {
        expect(chipValue('sort', '{"a":1}')).toBe('a ↑');
        expect(chipValue('projection', '{"a":1}')).toBe('+a');
        expect(chipValue('filter', '{"a":1}')).toBe('{"a":1}');
    });
});

describe('applySort', () => {
    it('升 → 降 → 取消，取消后回空串（回默认排序而不是留个 {}）', () => {
        let text = applySort('', 'qty');
        expect(text).toBe('{"qty":1}');
        text = applySort(text, 'qty');
        expect(text).toBe('{"qty":-1}');
        expect(applySort(text, 'qty')).toBe('');
    });

    it('换排序字段时保留原有优先级键', () => {
        expect(applySort('{"a":1}', 'b')).toBe('{"a":1,"b":1}');
    });

    it('文本非法时从空排序重新开始，不把坏文本带进请求', () => {
        expect(applySort('{oops', 'b')).toBe('{"b":1}');
    });
});

describe('parseSortDoc / clearedText / applyEquality', () => {
    it('非法排序文本解成空文档', () => {
        expect(parseSortDoc('[1,2]')).toEqual({});
        expect(parseSortDoc('{"a":-1}')).toEqual({ a: -1 });
    });

    it('清除 filter 回到 {}，清除另两项回到空串', () => {
        expect(clearedText('filter')).toBe('{}');
        expect(clearedText('sort')).toBe('');
        expect(clearedText('projection')).toBe('');
    });

    it('按值过滤只追加一项，不动已有条件', () => {
        expect(applyEquality('{"status":"paid"}', 'buyer.level', 2)).toBe('{"status":"paid","buyer.level":2}');
    });

    it('原条件非法时从该字段重新开始', () => {
        expect(applyEquality('nope', 'a', 1)).toBe('{"a":1}');
    });
});

describe('idInFilter', () => {
    const doc = (id: unknown): MongoDoc => ({ idToken: 't', idKind: 'objectId', hash: 'h', mode: 'plain', doc: { _id: id } });

    it('主键连同类型包装一起进 $in，ObjectId 不会与同形字符串混淆', () => {
        const filter = idInFilter([doc({ $oid: '507f1f77bcf86cd799439011' }), doc('507f1f77bcf86cd799439011')]);
        expect(filter).toEqual({ _id: { $in: [{ $oid: '507f1f77bcf86cd799439011' }, '507f1f77bcf86cd799439011'] } });
    });

    it('一条主键都没有时返回 null，调用方据此拒绝按选中导出', () => {
        expect(idInFilter([{ idToken: '', idKind: 'other', hash: 'h', mode: 'plain', doc: { name: 'x' } }])).toBeNull();
    });
});
