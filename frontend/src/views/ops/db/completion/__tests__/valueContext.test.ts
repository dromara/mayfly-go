import { describe, expect, it } from 'vitest';
import { matchesValuePartial, resolveValueTarget, valueInsertText } from '../valueContext';

/**
 * 值补全上下文判定的守卫
 *
 * 判错的代价不对称：漏判只是少一批候选，误判则会在列名位置把列名/函数候选整批挤掉，
 * 用户会觉得「补全坏了」。因此除了正向形态，这里重点钉住「不该接管」的形态。
 */

describe('resolveValueTarget 值位置识别', () => {
    it('识别等值比较后的值位置，取出列名与已输片段', () => {
        expect(resolveValueTarget('SELECT * FROM t WHERE status = ')).toMatchObject({ column: 'status', qualifier: '', partial: '', quoted: false });
        expect(resolveValueTarget("SELECT * FROM t WHERE status = 'ac")).toMatchObject({ column: 'status', partial: 'ac', quoted: true });
        expect(resolveValueTarget('SELECT * FROM t WHERE id = 1')).toMatchObject({ column: 'id', partial: '1' });
    });

    it('识别别名限定、IN 与 LIKE、引用符包裹的列名', () => {
        expect(resolveValueTarget('SELECT * FROM users u WHERE u.name = ')).toMatchObject({ qualifier: 'u', column: 'name' });
        expect(resolveValueTarget('SELECT * FROM t WHERE status IN (')).toMatchObject({ column: 'status', partial: '' });
        expect(resolveValueTarget("SELECT * FROM t WHERE name LIKE 'a")).toMatchObject({ column: 'name', partial: 'a', quoted: true });
        expect(resolveValueTarget('SELECT * FROM t WHERE `order` = ')).toMatchObject({ column: 'order' });
    });

    it('非值位置一律不接管', () => {
        expect(resolveValueTarget('SELECT * FROM t WHERE ')).toBeNull();
        expect(resolveValueTarget('SELECT id, name FROM t')).toBeNull();
        expect(resolveValueTarget('SELECT * FROM t WHERE a = 1 AND b')).toBeNull();
        expect(resolveValueTarget('')).toBeNull();
    });

    it('多条件时只认光标前那一段，不拿前一个条件的值当片段', () => {
        // 片段无引号时遇到空格即止：否则会把 '1 AND b' 当成 a 的片段，给出一堆错列候选
        expect(resolveValueTarget('SELECT * FROM t WHERE a = 1 AND b = 2')).toMatchObject({ column: 'b', partial: '2', quoted: false });
        // 开引号内可以含空格
        expect(resolveValueTarget("SELECT * FROM t WHERE city = 'new york")).toMatchObject({ column: 'city', partial: 'new york', quoted: true });
    });

    it('只认光标前文本，光标后已有内容不影响判定', () => {
        expect(resolveValueTarget("SELECT * FROM t WHERE status = 'pe' AND x = 1")?.column).toBe('x');
    });
});

describe('候选过滤与插入文本', () => {
    it('按已输片段做大小写不敏感的前缀过滤', () => {
        expect(matchesValuePartial('Active', 'ac')).toBe(true);
        expect(matchesValuePartial('closed', 'ac')).toBe(false);
        expect(matchesValuePartial('anything', '')).toBe(true);
    });

    it('数字裸写、文本加引号、已有开引号时只给裸值', () => {
        expect(valueInsertText('42', false)).toBe('42');
        expect(valueInsertText('active', false)).toBe("'active'");
        expect(valueInsertText('active', true)).toBe('active');
    });

    it('值内的单引号双写，不把语句截断', () => {
        expect(valueInsertText("O'Brien", false)).toBe("'O''Brien'");
    });
});
