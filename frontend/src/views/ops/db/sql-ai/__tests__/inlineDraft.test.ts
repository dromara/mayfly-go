import { describe, expect, it } from 'vitest';
import { LineThrottle, commentTextAtLine, ghostInsertText } from '../inlineDraft';

describe('注释行补全触发判定', () => {
    it('光标在注释行尾时返回注释正文', () => {
        const line = '-- 统计每个实例的库数量';
        expect(commentTextAtLine(line, line.length + 1)).toBe('统计每个实例的库数量');
    });

    it('行尾有空格也算末尾', () => {
        const line = '-- 统计每个实例的库数量   ';
        expect(commentTextAtLine(line, line.length + 1)).toBe('统计每个实例的库数量');
    });

    it('光标不在行尾时不触发（避免把后半行当成注释内容）', () => {
        const line = '-- 统计每个实例的库数量';
        expect(commentTextAtLine(line, 5)).toBe('');
    });

    it('过短的注释不触发（不值得花一次模型调用）', () => {
        expect(commentTextAtLine('-- 查', 5)).toBe('');
    });

    it('非注释行不触发', () => {
        expect(commentTextAtLine('select 1 from t', 16)).toBe('');
        expect(commentTextAtLine('/* 块注释 */', 12)).toBe('');
    });
});

describe('行级节流表', () => {
    it('窗口期内同一行只放行一次，过窗后再放行', () => {
        const throttle = new LineThrottle(1000, 10);
        expect(throttle.allow('a', 0)).toBe(true);
        // 判定与记账是同一次调用：紧接着的第二次必须被拦住，否则等于没有节流
        expect(throttle.allow('a', 999)).toBe(false);
        expect(throttle.allow('a', 1000)).toBe(true);
    });

    it('不同行互不影响', () => {
        const throttle = new LineThrottle(1000, 10);
        expect(throttle.allow('a', 0)).toBe(true);
        expect(throttle.allow('b', 0)).toBe(true);
    });

    it('forget 后立刻可重试（请求失败不必白等一个窗口）', () => {
        const throttle = new LineThrottle(1000, 10);
        expect(throttle.allow('a', 0)).toBe(true);
        throttle.forget('a');
        expect(throttle.allow('a', 1)).toBe(true);
    });

    it('条目数有上界：装满后不会无界增长', () => {
        const throttle = new LineThrottle(1000, 4);
        for (let i = 0; i < 50; i++) {
            // 每行都不同且都在同一时刻，模拟「长脚本里逐行触发」——正是无界增长的场景
            throttle.allow(`line-${i}`, 0);
        }
        // 不变量是「不超过容量上界」，而不是「一定小于上界」：淘汰发生在写入之前，
        // 写满到上界再触发下一次淘汰是正常状态
        expect(throttle.size).toBeLessThanOrEqual(4);
        expect(throttle.size).toBeGreaterThan(0);
    });

    it('过期条目在装满时被优先清掉，未过期的新条目保留', () => {
        const throttle = new LineThrottle(1000, 4);
        throttle.allow('old-1', 0);
        throttle.allow('old-2', 0);
        throttle.allow('fresh-1', 900);
        throttle.allow('fresh-2', 950);
        // 第 5 条把表装满：old-* 已过期，应被清掉而不是挤掉刚记的新条目
        expect(throttle.allow('next', 1100)).toBe(true);
        expect(throttle.allow('fresh-1', 1200)).toBe(false);
        expect(throttle.allow('old-1', 1200)).toBe(true);
    });
});
