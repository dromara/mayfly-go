/**
 * 值字面量转义：各方言把值拼进 SQL 时的引号/反斜杠处理。
 *
 * 这两条曾经都是真缺陷：单引号不转义会让 `O'Brien` 拼出提前闭合的字面量
 * （整个语句被判为未闭合区域而拒绝执行）；MySQL 反斜杠不加倍会把 `a\b`
 * 静默写成退格。
 */
import { describe, expect, it } from 'vitest';
import { getDbDialect } from '../index';

const mysql = getDbDialect('mysql');
const pg = getDbDialect('postgres');
const mssql = getDbDialect('mssql');
const oracle = getDbDialect('oracle');
const clickhouse = getDbDialect('clickhouse');

describe('值字面量转义', () => {
    it('MySQL：单引号双写', () => {
        expect(mysql.wrapValue('varchar', "O'Brien")).toBe(`'O''Brien'`);
    });

    it('MySQL：反斜杠加倍（否则被当转义符吞掉后一个字符）', () => {
        expect(mysql.wrapValue('varchar', 'a\\b')).toBe(`'a\\\\b'`);
    });

    it('PostgreSQL：单引号双写，反斜杠保持原样（标准字符串不解释转义）', () => {
        expect(pg.wrapValue('varchar', "O'Brien")).toBe(`'O''Brien'`);
        expect(pg.wrapValue('varchar', 'a\\b')).toBe(`'a\\b'`);
    });

    it('SQL Server：N 前缀 + 单引号双写，反斜杠保持原样', () => {
        expect(mssql.wrapValue('varchar', "can't")).toBe(`N'can''t'`);
        expect(mssql.wrapValue('varchar', 'a\\b')).toBe(`N'a\\b'`);
    });

    it('Oracle：单引号双写，反斜杠保持原样', () => {
        expect(oracle.wrapValue('varchar', "can't")).toBe(`'can''t'`);
        expect(oracle.wrapValue('varchar', 'a\\b')).toBe(`'a\\b'`);
    });

    it('ClickHouse：单引号双写且反斜杠加倍', () => {
        expect(clickhouse.wrapValue('varchar', "can't")).toBe(`'can''t'`);
        expect(clickhouse.wrapValue('varchar', 'a\\b')).toBe(`'a\\\\b'`);
    });

    it('数值与 NULL 不受影响', () => {
        expect(mysql.wrapValue('int', 42)).toBe(42);
        expect(mysql.wrapValue('varchar', null)).toBe('NULL');
    });
});
