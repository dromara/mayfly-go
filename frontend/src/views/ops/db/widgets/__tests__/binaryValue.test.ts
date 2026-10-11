import { describe, expect, it } from 'vitest';
import { bytesToBase64, hexToBytes, isBinaryColumnType, utf8Preview } from '../binaryValue';

/**
 * 二进制列值呈现的守卫
 *
 * 存在理由：单元格里的二进制值是后端回读出的 hex 串，换算错一位就等于给用户看假内容；
 * 而「哪些列类型算二进制」必须是登记式判据——放宽到按前缀猜，会把一个正常文本列
 * 换成只读预览，误伤比缺功能更糟。
 */

describe('isBinaryColumnType', () => {
    it('各方言的二进制/大对象类型都认得，含长度参数与大小写差异', () => {
        ['blob', 'TINYBLOB', 'mediumblob', 'LONGBLOB', 'binary(4)', 'varbinary(20)', 'bytea', 'RAW(16)', 'long raw', 'IMAGE', 'bfile'].forEach((type) => {
            expect(isBinaryColumnType(type)).toBe(true);
        });
    });

    it('pg 数组形态只认 bytea 数组，int 数组不是二进制', () => {
        expect(isBinaryColumnType('_bytea')).toBe(true);
        expect(isBinaryColumnType('bytea[]')).toBe(true);
        expect(isBinaryColumnType('_int8')).toBe(false);
        expect(isBinaryColumnType('int[]')).toBe(false);
    });

    it('文本/数值/JSON 类不算二进制，clickhouse 的 FixedString 也不放宽', () => {
        ['text', 'varchar(64)', 'jsonb', 'int8', 'numeric', 'String', 'FixedString(16)', 'uuid'].forEach((type) => {
            expect(isBinaryColumnType(type)).toBe(false);
        });
    });

    it('空值不抛，按非二进制处理', () => {
        expect(isBinaryColumnType(undefined)).toBe(false);
        expect(isBinaryColumnType('')).toBe(false);
    });
});

describe('hexToBytes / bytesToBase64', () => {
    it('hex 还原为字节', () => {
        expect(Array.from(hexToBytes('616263') ?? [])).toEqual([97, 98, 99]);
        expect(Array.from(hexToBytes('0001FF') ?? [])).toEqual([0, 1, 255]);
    });

    it('非 hex 形态返回 null，而不是硬解出一串无关字节', () => {
        expect(hexToBytes('')).toBeNull();
        expect(hexToBytes('abc')).toBeNull(); // 奇数长度
        expect(hexToBytes('zz01')).toBeNull();
        expect(hexToBytes('hello!')).toBeNull();
    });

    it('base64 与标准实现一致，且大内容分块不炸栈', () => {
        expect(bytesToBase64(new Uint8Array([97, 98, 99]))).toBe('YWJj');
        const big = new Uint8Array(200_000).fill(65);
        // 每 3 字节一组补到 4 字符（末尾不足也占整组），不是按总长比例取整
        expect(bytesToBase64(big).length).toBe(4 * Math.ceil(big.length / 3));
    });
});

describe('utf8Preview', () => {
    it('可打印文本占比为 1，不标警告', () => {
        const preview = utf8Preview(new Uint8Array([104, 105, 32, 10])); // "hi \n"
        expect(preview.text).toBe('hi \n');
        expect(preview.printableRatio).toBe(1);
        expect(preview.truncated).toBe(false);
    });

    it('非法字节被替换，占比随之下降（所以只能当参考而非原文）', () => {
        const preview = utf8Preview(new Uint8Array([0xff, 0xfe, 0x41]));
        expect(preview.text).toContain('\uFFFD');
        expect(preview.printableRatio).toBeLessThan(1);
    });

    it('超长内容标记被截断，不假装看到全貌', () => {
        const preview = utf8Preview(new Uint8Array(5000).fill(97));
        expect(preview.truncated).toBe(true);
        expect(preview.text.length).toBe(4096);
    });
});
