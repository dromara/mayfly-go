/**
 * cron 模型用例。
 *
 * 重点钉住与后端调度器（robfig/cron v3 + SecondOptional）的一致性：
 * 表达式形态、边界值与名称别名、star 位（决定日/周取交集还是并集）、以及运行时间推算的实际命中结果。
 */
import { describe, expect, it } from 'vitest';
import { formatDate } from '@/common/utils/format';
import {
    CRON_FIELDS,
    CRON_FIELD_KEYS,
    CRON_PRESETS,
    buildExpression,
    defaultSpec,
    inspectCronExpression,
    isRestricted,
    nextRunTimes,
    parseCronExpression,
    PREVIEW_COUNT,
    ruleValues,
    withRule,
    type CronSpec,
} from '../cronSpec';

/** 表达式 → 规则集 → 表达式，断言中间无损耗 */
function toSpec(expression: string): CronSpec {
    const parsed = parseCronExpression(expression);
    expect(parsed.error, `表达式 ${expression} 应解析成功`).toBeNull();
    return parsed.spec;
}

function parseError(expression: string) {
    return parseCronExpression(expression).error;
}

describe('表达式解析与序列化', () => {
    it('紧凑形态按原样往返，不改变表达式', () => {
        const expressions = [
            '* * * * * ?',
            '* * * * * *',
            '0 * * * * ?',
            '0 0/5 * * * ?',
            '30 0 9 * * ?',
            '0 15,45 * * * ?',
            '0 0 8-22 * * ?',
            '0 0 0 1 * ?',
            '0 0 0 1-31 * 1',
            '0 0 0 ? * 1-5',
            '0 0 0 ? * 0',
            '0 0 0 1,15 1,7 *',
            '0/10 0 3 * * 6',
        ];
        for (const expression of expressions) {
            expect(buildExpression(toSpec(expression)), expression).toBe(expression);
        }
    });

    it('5 段表达式按后端 SecondOptional 默认补秒为 0', () => {
        // 「0 0 * * *」是分/时为 0，补秒后仍是每天零点
        expect(buildExpression(toSpec('0 0 * * *'))).toBe('0 0 0 * * *');
        expect(buildExpression(toSpec('30 9 * * 1-5'))).toBe('0 30 9 * * 1-5');
    });

    it('月/周名称别名归一为数字取值，与后端 parseIntOrName 一致', () => {
        expect(toSpec('0 0 0 ? * mon-fri').week).toEqual({ kind: 'cycle', from: 1, to: 5 });
        expect(toSpec('0 0 0 ? * SUN').week).toEqual({ kind: 'list', values: [0] });
        expect(toSpec('0 0 0 1 JAN-jun *').month).toEqual({ kind: 'cycle', from: 1, to: 6 });
        expect(toSpec('0 0 0 1 jan,mar *').month).toEqual({ kind: 'list', values: [1, 3] });
        // 日字段没有名称表，写名字属于笔误
        expect(parseError('0 0 0 mon * ?')).toMatchObject({ key: 'components.crontab.errSegment', field: 'day' });
    });

    it('混合列表展开为等价的离散取值', () => {
        expect(toSpec('0 1-5,10 * * * ?').min).toEqual({ kind: 'list', values: [1, 2, 3, 4, 5, 10] });
        expect(toSpec('0 0 0 1 * 1,3,5').week).toEqual({ kind: 'list', values: [1, 3, 5] });
    });

    it('非法写法逐条报错，而不是静默改语义', () => {
        expect(parseError('* * * *')).toMatchObject({ key: 'components.crontab.errFieldCount' });
        expect(parseError('* * * * * * *')).toMatchObject({ key: 'components.crontab.errFieldCount' });
        expect(parseError('0 0 0 *')).toMatchObject({ key: 'components.crontab.errFieldCount' });
        // 日为 1-31、月为 1-12、周为 0-6，后端不做 7→0 归一
        expect(parseError('0 0 0 0 * ?')).toMatchObject({ key: 'components.crontab.errValueRange', field: 'day' });
        expect(parseError('0 0 0 * 0 ?')).toMatchObject({ key: 'components.crontab.errValueRange', field: 'month' });
        expect(parseError('0 0 0 ? * 7')).toMatchObject({ key: 'components.crontab.errValueRange', field: 'week' });
        expect(parseError('0 0 24 * * ?')).toMatchObject({ key: 'components.crontab.errValueRange', field: 'hour' });
        // 后端不支持区间回绕
        expect(parseError('0 0 22-2 * * ?')).toMatchObject({ key: 'components.crontab.errRange', field: 'hour' });
        expect(parseError('0 0 0 * * ? 2026')).toMatchObject({ key: 'components.crontab.errFieldCount' });
        // 后端不支持的 Quartz 专有写法
        expect(parseError('0 0 0 L * ?')).toMatchObject({ key: 'components.crontab.errSegment', field: 'day' });
        expect(parseError('0 0 0 ? * 1#2')).toMatchObject({ key: 'components.crontab.errSegment', field: 'week' });
        // 空列表项：后端会静默丢弃，面板判错
        expect(parseError('0 0 0 ,1 * ?')).toMatchObject({ key: 'components.crontab.errEmptySegment', field: 'day' });
        expect(parseError('0 0 0 1, * ?')).toMatchObject({ key: 'components.crontab.errEmptySegment', field: 'day' });
        expect(parseError('0 0/0 * * * ?')).toMatchObject({ key: 'components.crontab.errStep', field: 'min' });
        expect(parseError('0 0/x * * * ?')).toMatchObject({ key: 'components.crontab.errStep', field: 'min' });
        expect(parseError('0 0 0 * * 1/2/3')).toMatchObject({ key: 'components.crontab.errSegment', field: 'week' });
    });

    it('解析失败时回落默认规则集，面板仍可继续配置', () => {
        const parsed = parseCronExpression('0 0 0 32 * ?');
        expect(parsed.kind).toBe('panel');
        expect(parsed.spec).toEqual(defaultSpec());
    });
});

describe('star 位保真（决定日/周是交集还是并集）', () => {
    it('`*`、`?` 与 `*`/1 归一为未限定', () => {
        expect(toSpec('* * * * * ?').week).toEqual({ kind: 'none' });
        expect(toSpec('0 */1 * * * ?').min).toEqual({ kind: 'all' });
        // 带步长的 `?` 写法归一为 `*`，star 语义相同，仅字面形态被规范化
        expect(toSpec('0 0 0 ?/1 * ?').day).toEqual({ kind: 'all' });
        expect(buildExpression(toSpec('0 0 0 ?/1 * ?'))).toBe('0 0 0 * * ?');
    });

    it('覆盖全量不等于未限定：带区间或步长写法不补 star 位', () => {
        const fullCycle = toSpec('0 0 0 1-31 * ?');
        expect(fullCycle.day).toEqual({ kind: 'cycle', from: 1, to: 31 });
        expect(isRestricted(fullCycle.day)).toBe(true);

        const fullStep = toSpec('0 0-59/1 * * * ?');
        expect(fullStep.min).toEqual({ kind: 'list', values: Array.from({ length: 60 }, (_, i) => i) });

        const fullList = toSpec('0 0 0 1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31 * ?');
        expect(fullList.day).toEqual({ kind: 'list', values: Array.from({ length: 31 }, (_, i) => i + 1) });
    });

    it('非日/周字段的 ? 按后端语义等同 *', () => {
        expect(toSpec('? 0 0 * * ?').second).toEqual({ kind: 'all' });
    });

    it('日/周都限定时预览按并集命中，与后端 dayMatches 一致', () => {
        // day=1-31（无 star）+ week=周一 → 或：每天都命中；若误判成 star 则只剩周一
        const spec = toSpec('0 0 0 1-31 * 1');
        const times = nextRunTimes(spec, 2, new Date(2026, 8, 24, 10, 0, 0)).times.map((time) => formatDate(time));
        expect(times).toEqual(['2026-09-25 00:00:00', '2026-09-26 00:00:00']);
    });

    it('一侧未限定时预览按交集命中', () => {
        const spec = toSpec('0 0 0 * * 1');
        const times = nextRunTimes(spec, 2, new Date(2026, 8, 24, 10, 0, 0)).times.map((time) => formatDate(time));
        expect(times).toEqual(['2026-09-28 00:00:00', '2026-10-05 00:00:00']);
    });
});

describe('描述符', () => {
    it('只透传不编辑，且只认后端支持的白名单', () => {
        expect(parseCronExpression('@every 5m')).toMatchObject({ kind: 'descriptor', error: null });
        expect(inspectCronExpression('@daily').valid).toBe(true);
        expect(inspectCronExpression('@every 1h30m').valid).toBe(true);
        expect(inspectCronExpression('@every 0.5s').valid).toBe(true);
        expect(inspectCronExpression('@every 5 minutes').valid).toBe(false);
        expect(inspectCronExpression('@bogus').valid).toBe(false);
        // 零时长后端会接受但会把调度打成忙循环，一并判错
        expect(inspectCronExpression('@every 0s').valid).toBe(false);
    });
});

describe('输入行状态', () => {
    it('空值不报错，段数不足视为还在敲', () => {
        expect(inspectCronExpression('')).toMatchObject({ incomplete: false, valid: true, nextRun: null });
        expect(inspectCronExpression('0 0 0 L')).toMatchObject({ incomplete: true, valid: true, nextRun: null });
    });

    it('段数齐了才判定合法性', () => {
        expect(inspectCronExpression('0 0 0 L * ?')).toMatchObject({ incomplete: false, valid: false });
        expect(inspectCronExpression('0 0 0 32 * ?')).toMatchObject({ valid: false });
        expect(inspectCronExpression('0 0 0 * * ?')).toMatchObject({ valid: true });
    });

    it('合法表达式给出下一次运行时间，且严格晚于当前时刻', () => {
        const from = new Date(2026, 8, 24, 10, 0, 0);
        expect(formatDate(inspectCronExpression('0 0 9 ? * 1-5', from).nextRun!)).toBe('2026-09-25 09:00:00');
        expect(formatDate(inspectCronExpression('* * * * * *', from).nextRun!)).toBe('2026-09-24 10:00:01');
    });
});

describe('日与周的互斥', () => {
    it('限定其一即把另一项置为不指定', () => {
        const byDay = withRule(defaultSpec(), 'day', { kind: 'list', values: [1, 15] });
        expect(byDay.week).toEqual({ kind: 'none' });
        expect(buildExpression(byDay)).toBe('* * * 1,15 * ?');

        const byWeek = withRule(byDay, 'week', { kind: 'cycle', from: 1, to: 5 });
        expect(byWeek.day).toEqual({ kind: 'none' });
        expect(buildExpression(byWeek)).toBe('* * * ? * 1-5');
    });

    it('放开为全部时不牵连另一项', () => {
        const byDay = withRule(defaultSpec(), 'day', { kind: 'list', values: [3] });
        const relaxed = withRule(byDay, 'day', { kind: 'all' });
        expect(relaxed.day).toEqual({ kind: 'all' });
        expect(relaxed.week).toEqual({ kind: 'none' });
    });
});

describe('取值集合', () => {
    it('步长自起始值起按间隔递增至字段上界', () => {
        expect(ruleValues(CRON_FIELDS.min, { kind: 'step', start: 5, every: 20 })).toEqual([5, 25, 45]);
        expect(ruleValues(CRON_FIELDS.week, { kind: 'none' })).toEqual([0, 1, 2, 3, 4, 5, 6]);
        expect(ruleValues(CRON_FIELDS.day, { kind: 'cycle', from: 1, to: 3 })).toEqual([1, 2, 3]);
    });

    it('列表规则返回副本，调用方排序不会污染真源', () => {
        const source = [5, 1];
        ruleValues(CRON_FIELDS.day, { kind: 'list', values: source }).sort((a, b) => a - b);
        expect(source).toEqual([5, 1]);
    });
});

describe('运行时间推算', () => {
    /** 2026-09-24 是星期四 */
    const from = () => new Date(2026, 8, 24, 10, 0, 0);

    const nextOf = (expression: string, count = PREVIEW_COUNT) => nextRunTimes(toSpec(expression), count, from()).times.map((time) => formatDate(time));

    it('秒级表达式逐秒命中，且只取 from 之后的时刻', () => {
        expect(nextOf('* * * * * ?', 3)).toEqual(['2026-09-24 10:00:01', '2026-09-24 10:00:02', '2026-09-24 10:00:03']);
    });

    it('步长按字段上界收敛', () => {
        expect(nextOf('0 0/5 * * * ?', 3)).toEqual(['2026-09-24 10:05:00', '2026-09-24 10:10:00', '2026-09-24 10:15:00']);
    });

    it('不指定日期时由星期决定', () => {
        expect(nextOf('0 0 9 ? * 1-5', 5)).toEqual([
            '2026-09-25 09:00:00',
            '2026-09-28 09:00:00',
            '2026-09-29 09:00:00',
            '2026-09-30 09:00:00',
            '2026-10-01 09:00:00',
        ]);
    });

    it('0 号星期为周日', () => {
        expect(nextOf('0 0 0 ? * 0', 2)).toEqual(['2026-09-27 00:00:00', '2026-10-04 00:00:00']);
    });

    it('日与周都限定时取并集', () => {
        expect(nextOf('0 0 0 1 * 1', 5)).toEqual([
            '2026-09-28 00:00:00',
            '2026-10-01 00:00:00',
            '2026-10-05 00:00:00',
            '2026-10-12 00:00:00',
            '2026-10-19 00:00:00',
        ]);
    });

    it('闰日表达式跨到下一个闰年，检索窗口足以凑齐请求条数', () => {
        expect(nextOf('0 0 0 29 2 ?', 2)).toEqual(['2028-02-29 00:00:00', '2032-02-29 00:00:00']);
        const preview = nextRunTimes(toSpec('0 0 0 29 2 ?'), PREVIEW_COUNT, from());
        expect(preview.times).toHaveLength(PREVIEW_COUNT);
        expect(preview.exhausted).toBe(false);
    });

    it('无法命中的表达式返回空结果并标记窗口耗尽', () => {
        const preview = nextRunTimes(toSpec('0 0 0 30 2 ?'), PREVIEW_COUNT, from());
        expect(preview.times).toEqual([]);
        expect(preview.exhausted).toBe(true);
    });
});

describe('预设', () => {
    it('每条预设都能解析并原样序列化', () => {
        for (const preset of CRON_PRESETS) {
            const parsed = parseCronExpression(preset.expr);
            expect(parsed.error, preset.expr).toBeNull();
            expect(buildExpression(parsed.spec), preset.expr).toBe(preset.expr);
        }
    });

    it('预设产出均落在后端支持的 6 段语法内', () => {
        for (const preset of CRON_PRESETS) {
            expect(preset.expr.split(' ')).toHaveLength(CRON_FIELD_KEYS.length);
            expect(preset.expr).not.toMatch(/[LW#]/);
        }
    });
});
