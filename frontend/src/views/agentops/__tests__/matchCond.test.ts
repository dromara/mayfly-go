import { describe, expect, it } from 'vitest';
import { parseMatchCond, serializeMatchCond } from '../matchCond';

/**
 * 匹配条件 parse/serialize 的往返契约。
 *
 * 这个关口决定「表单里勾了什么」与「库里存了什么」是否一致：空维度若写成 `"metrics":[]`，
 * 后端 MatchByFields 会把「不限定」判成「白名单为空」→ 触发器永不命中且无任何报错。
 * 历史脏数据（非法 JSON / 扩展键）必须 invalid 暴露，否则一次保存就无痕丢掉用户手写的条件。
 */

describe('parseMatchCond', () => {
    it('空/缺省都是合法的全不限定', () => {
        for (const raw of [undefined, null, '', '   ']) {
            expect(parseMatchCond(raw)).toEqual({ cond: {}, invalid: false });
        }
    });

    it('四维度完整往返', () => {
        const raw = JSON.stringify({ resourceType: 'machine', resourceIds: [5], severities: ['P0'], metrics: ['cpu_usage'] });
        expect(parseMatchCond(raw)).toEqual({ cond: { resourceType: 'machine', resourceIds: [5], severities: ['P0'], metrics: ['cpu_usage'] }, invalid: false });
    });

    it('非法 JSON / 数组 / 标量都判 invalid 而不是给个空对象冒充合法', () => {
        for (const raw of ['{bad json', '[1,2]', '"machine"', '123']) {
            expect(parseMatchCond(raw).invalid, raw).toBe(true);
        }
    });

    it('含契约外的扩展键判 invalid（可视化编辑表达不了它，不能假装解析成功）', () => {
        expect(parseMatchCond('{"resourceType":"machine","customOp":"regex"}').invalid).toBe(true);
    });
});

describe('serializeMatchCond', () => {
    it('空维度不写键：不限定 ≠ 空名单（后者会让触发器永不命中）', () => {
        const json = serializeMatchCond({ resourceType: 'machine', resourceIds: [], severities: undefined, metrics: [] });
        expect(JSON.parse(json)).toEqual({ resourceType: 'machine' });
    });

    it('全部不限定序列化为空串（后端按命中该来源全部事件处理）', () => {
        expect(serializeMatchCond({ resourceType: '', resourceIds: [], severities: [], metrics: [] })).toBe('');
    });

    it('往返保真：parse(serialize(x)) 与原对象等价', () => {
        const cond = { resourceType: 'db', resourceIds: [22, 23], severities: ['P0', 'P1'], metrics: ['slow_query'] };
        expect(parseMatchCond(serializeMatchCond(cond)).cond).toEqual(cond);
    });
});
