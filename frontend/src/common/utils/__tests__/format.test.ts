import { describe, expect, it } from 'vitest';
import { convertToBytes, formatByteSize, formatDate, formatDocSize, formatDurationMs, formatJson, formatThousands, formatTime, formatTokenCount } from '../format';

describe('formatDate', () => {
    it('格式化日期为默认格式', () => {
        const result = formatDate(new Date('2024-06-15 10:30:00'));
        expect(result).toBe('2024-06-15 10:30:00');
    });

    it('支持自定义格式', () => {
        const result = formatDate(new Date('2024-06-15 10:30:00'), 'YYYY/MM/DD');
        expect(result).toBe('2024/06/15');
    });

    it('空值返回空字符串', () => {
        expect(formatDate(undefined)).toBe('');
        expect(formatDate('')).toBe('');
    });

    it('支持时间戳输入', () => {
        const ts = new Date('2024-01-01 00:00:00').getTime();
        const result = formatDate(ts, 'YYYY-MM-DD');
        expect(result).toBe('2024-01-01');
    });
});

describe('formatByteSize', () => {
    it('0 返回 0B', () => {
        expect(formatByteSize(0)).toBe('0B');
    });

    it('字节级别', () => {
        expect(formatByteSize(512)).toBe('512B');
    });

    it('KB 级别', () => {
        expect(formatByteSize(1024)).toBe('1KB');
        expect(formatByteSize(1536)).toBe('1.5KB');
    });

    it('MB 级别', () => {
        expect(formatByteSize(1024 * 1024)).toBe('1MB');
        expect(formatByteSize(5.25 * 1024 * 1024)).toBe('5.25MB');
    });

    it('GB 级别', () => {
        expect(formatByteSize(1024 * 1024 * 1024)).toBe('1GB');
    });

    it('自定义小数位', () => {
        expect(formatByteSize(1555, 1)).toBe('1.5KB');
    });
});

describe('formatDocSize', () => {
    it('0 返回 0', () => {
        expect(formatDocSize(0)).toBe('0');
    });

    it('使用 1000 进制', () => {
        expect(formatDocSize(1000)).toBe('1K');
        expect(formatDocSize(1000000)).toBe('1M');
    });
});

describe('convertToBytes', () => {
    it('KB 转换', () => {
        expect(convertToBytes('1KB')).toBe(1024);
        expect(convertToBytes('2kb')).toBe(2048);
    });

    it('MB 转换', () => {
        expect(convertToBytes('1MB')).toBe(1024 * 1024);
    });

    it('GB 转换', () => {
        expect(convertToBytes('1GB')).toBe(1024 * 1024 * 1024);
    });

    it('带空格的输入', () => {
        expect(convertToBytes(' 1KB ')).toBe(1024);
    });

    it('无效单位抛出异常', () => {
        expect(() => convertToBytes('1XX')).toThrow('Invalid size unit');
    });
});

describe('formatTime', () => {
    it('秒转人性化时间', () => {
        expect(formatTime(3661)).toBe('1h 1m 1s ');
    });

    it('天级别', () => {
        expect(formatTime(86400)).toBe('1d ');
    });

    it('指定输入单位', () => {
        expect(formatTime(2, 'm')).toBe('2m ');
    });

    it('无效单位', () => {
        expect(formatTime(1, 'x')).toBe('Invalid unit');
    });
});

describe('formatJson', () => {
    it('格式化对象', () => {
        expect(formatJson({ name: 'test' })).toBe('{\n  "name": "test"\n}');
    });

    it('格式化 JSON 字符串', () => {
        expect(formatJson('{"a":1}')).toBe('{\n  "a": 1\n}');
    });

    it('无效 JSON 字符串返回原始值', () => {
        expect(formatJson('not json')).toBe('not json');
    });

    it('空值返回空字符串', () => {
        expect(formatJson(null)).toBe('');
        expect(formatJson(undefined)).toBe('');
    });
});

describe('formatThousands', () => {
    it('千分位分隔', () => {
        expect(formatThousands(91606)).toBe('91,606');
        expect(formatThousands(8234)).toBe('8,234');
    });

    it('0 显示为 0', () => {
        expect(formatThousands(0)).toBe('0');
    });

    it('空值与非法值返回 -', () => {
        expect(formatThousands(null)).toBe('-');
        expect(formatThousands(undefined)).toBe('-');
        expect(formatThousands(Number.NaN)).toBe('-');
    });
});

// token 展示的唯一入口：所有界面不得自己挑 formatThousands
// （原 `formatCountCompact` 已因零消费者删除，其覆盖并入下面的格式断言）
describe('formatTokenCount', () => {
    it('默认只给紧凑值（列表列用，列宽有限）', () => {
        expect(formatTokenCount(1957362)).toBe('1.96M');
        expect(formatTokenCount(29599)).toBe('29.6K');
        expect(formatTokenCount(8234)).toBe('8,234');
    });

    it('精度是 3 位有效数字，不是固定小数位（否则同一屏会混排 20.73K 与 29.6K）', () => {
        expect(formatTokenCount(20734)).toBe('20.7K');
        expect(formatTokenCount(97490)).toBe('97.5K');
        expect(formatTokenCount(2292327)).toBe('2.29M');
        // 百位以上不再带小数，仍然是 3 位有效数字
        expect(formatTokenCount(974900)).toBe('975K');
    });

    it('withExact 时紧凑值后附精确值：一眼看出量级同时能读到准确数做成本对账', () => {
        expect(formatTokenCount(1957362, true)).toBe('1.96M (1,957,362)');
    });

    it('小值不重复输出（紧凑值已等于精确值时不拼成「8,234 (8,234)」）', () => {
        expect(formatTokenCount(8234, true)).toBe('8,234');
    });

    it('空值只回一个 -，不得渲成「- (-)」', () => {
        expect(formatTokenCount(null, true)).toBe('-');
        expect(formatTokenCount(undefined, true)).toBe('-');
    });

    it('0 是合法读数，不能被当空值吞掉', () => {
        expect(formatTokenCount(0, true)).toBe('0');
    });
});

describe('formatDurationMs', () => {
    it('小于 1 秒用 ms', () => {
        expect(formatDurationMs(876)).toBe('876ms');
        expect(formatDurationMs(0)).toBe('0ms');
    });

    it('1 分钟内用秒', () => {
        expect(formatDurationMs(8200)).toBe('8.2s');
        expect(formatDurationMs(59000)).toBe('59s');
    });

    it('跨分显示 Xm Ys', () => {
        expect(formatDurationMs(87902)).toBe('1m 28s');
        expect(formatDurationMs(60000)).toBe('1m 0s');
    });

    it('空值与非法值返回 -', () => {
        expect(formatDurationMs(null)).toBe('-');
        expect(formatDurationMs(undefined)).toBe('-');
    });
});
