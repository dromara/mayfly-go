import { describe, expect, it } from 'vitest';
import { getDefaultDataType } from '../utils';

/**
 * 列类型 → 基本数据类型 归一的守卫
 *
 * 钉的是角标显示错误的根因链：结果列/表列的类型字符串必须正确判成 number，否则数字列被画成
 * `ab`（字符串）角标。覆盖三类真实输入：后端归一名（uint64/int8…）、带长度与修饰的 DDL 类型
 * （bigint(20) unsigned / decimal(10,2)）、驱动返回的大写 SQL 类型名（BIGINT/VARCHAR）。
 * 并锁死误判防护：含 int 子串但非数值的类型（point/interval）不得被判 number。
 */
describe('getDefaultDataType', () => {
    it('数值：后端归一名、Go 风格整型/浮点、带长度修饰的 DDL、大写 SQL 名', () => {
        for (const t of ['uint64', 'uint8', 'uint16', 'uint32', 'int8', 'int16', 'int32', 'int64', 'float32', 'float64']) {
            expect(getDefaultDataType(t), t).toBe('number');
        }
        for (const t of ['bigint(20) unsigned', 'decimal(10,2)', 'int(11)', 'tinyint', 'smallint', 'mediumint', 'double precision']) {
            expect(getDefaultDataType(t), t).toBe('number');
        }
        // 驱动 DatabaseTypeName 常为大写
        for (const t of ['BIGINT', 'DECIMAL', 'NUMERIC', 'FLOAT', 'REAL']) {
            expect(getDefaultDataType(t), t).toBe('number');
        }
    });

    it('字符串：varchar/text/string 一律 string', () => {
        for (const t of ['varchar(255)', 'VARCHAR', 'text', 'string', 'char(8)', 'json']) {
            expect(getDefaultDataType(t), t).toBe('string');
        }
    });

    it('日期时间族按 SQL 语义归类', () => {
        expect(getDefaultDataType('datetime')).toBe('datetime');
        expect(getDefaultDataType('DATETIME')).toBe('datetime');
        expect(getDefaultDataType('timestamp without time zone')).toBe('datetime');
        expect(getDefaultDataType('date')).toBe('date');
        expect(getDefaultDataType('time')).toBe('time');
    });

    it('误判防护：含 int 子串但非数值的类型不得被判 number', () => {
        // point 含 "int"、interval 含 "int"，但都不是整词，必须落回 string
        expect(getDefaultDataType('point')).toBe('string');
        expect(getDefaultDataType('interval')).toBe('string');
    });
});
