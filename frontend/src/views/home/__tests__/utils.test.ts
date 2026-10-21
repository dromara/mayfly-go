import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// 时间分档是纯逻辑，直接钉住 i18n 输出形态，避免测试依赖真实语言包
vi.mock('@/i18n', () => ({
    i18n: {
        global: {
            t: (key: string, params?: Record<string, unknown>) => (params ? `${key}|${JSON.stringify(params)}` : key),
        },
    },
}));

import { formatFeedTime, greetingKeyOf } from '../utils';

describe('home/utils formatFeedTime', () => {
    beforeEach(() => {
        vi.useFakeTimers();
        vi.setSystemTime(new Date('2026-10-21T15:30:00'));
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    it('空值与非法值统一回退为占位横线', () => {
        expect(formatFeedTime()).toBe('-');
        expect(formatFeedTime('')).toBe('-');
        expect(formatFeedTime('not-a-date')).toBe('-');
    });

    it('1 分钟内显示刚刚，1 小时内带分钟数', () => {
        expect(formatFeedTime('2026-10-21T15:29:40')).toBe('home.justNow');
        expect(formatFeedTime('2026-10-21T15:25:00')).toBe('home.minutesAgo|{"n":5}');
    });

    it('当天超过 1 小时用「今天 HH:mm」，昨天用「昨天 HH:mm」', () => {
        expect(formatFeedTime('2026-10-21T09:05:00')).toBe('home.today 09:05');
        expect(formatFeedTime('2026-10-20T23:10:00')).toBe('home.yesterday 23:10');
    });

    it('跨年时间给出完整日期，不再用相对口径', () => {
        expect(formatFeedTime('2025-12-31T23:59:00')).toBe('2025-12-31');
    });
});

describe('home/utils greetingKeyOf', () => {
    it('按作息分档问候语', () => {
        const at = (hour: number) => greetingKeyOf(new Date(2026, 9, 21, hour, 0, 0));
        expect(at(3)).toBe('home.greetingDawn');
        expect(at(6)).toBe('home.greetingMorning');
        expect(at(11)).toBe('home.greetingNoon');
        expect(at(15)).toBe('home.greetingAfternoon');
        expect(at(19)).toBe('home.greetingEvening');
        expect(at(23)).toBe('home.greetingDawn');
    });
});

