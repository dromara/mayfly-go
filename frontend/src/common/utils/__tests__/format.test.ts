import { describe, expect, it } from 'vitest';
import { convertToBytes, formatByteSize, formatDate, formatDocSize, formatJson, formatTime } from '../format';

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
