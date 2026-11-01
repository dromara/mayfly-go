/**
 * 类型表达式的插值规则守卫：类型名 + 参数不是一个简单的字符串拼接。
 *
 * 真实库对照（2026-10-31）：
 *  - PG `CREATE TABLE t (c timestamp(3) with time zone)` ✅；`c timestamp with time zone(3)` ❌ syntax error
 *  - MySQL `mediumint unsigned` / `bit(8)` / `year` / `tinyblob` / `tinytext` / `bool` / `json` / `geometry` ✅
 *  - MySQL 无符号必须换序成 `bigint(20) unsigned`，写成 `unsigned bigint` 是 1064
 */
import { describe, expect, it } from 'vitest';
import { getDbDialect } from '../registry';
import { DbType } from '../dbType';
import type { ColumnDefinition, DbDialect } from '../types';

import '../mysql_dialect';
import '../postgres_dialect';
import '../mssql_dialect';
import '../oracle_dialect';
import '../dm_dialect';
import '../sqlite_dialect';

/**
 * 列级 DDL 片段不在 DbDialect 的公开契约里（属 SchemaDialectBase 的实现细节）。
 * 这里做运行时收窄再取用：方法不存在就立刻失败，而不是用断言把编译器嘴堵上。
 */
type ColumnSqlCapable = { genColumnBasicSql(cl: ColumnDefinition): string };
const asColumnSql = (dialect: DbDialect): ColumnSqlCapable => {
    const candidate = dialect as Partial<ColumnSqlCapable>;
    if (typeof candidate.genColumnBasicSql !== 'function') {
        throw new Error(`${dialect.getInfo().name} 未提供 genColumnBasicSql`);
    }
    return candidate as ColumnSqlCapable;
};

const col = (partial: Partial<ColumnDefinition>): ColumnDefinition =>
    ({
        name: 'c',
        oldName: 'c',
        type: 'varchar',
        length: '',
        numScale: '',
        nullable: true,
        isPrimaryKey: false,
        autoIncrement: false,
        comment: '',
        value: '',
        ...partial,
    }) as ColumnDefinition;

describe('类型表达式插值', () => {
    const pg = asColumnSql(getDbDialect(DbType.postgres));

    it('pg 带时区类型把精度插在基础类型后', () => {
        expect(pg.genColumnBasicSql(col({ type: 'timestamp with time zone', length: 3 }))).toBe('"c" timestamp(3) with time zone');
        expect(pg.genColumnBasicSql(col({ type: 'time with time zone', length: 2 }))).toBe('"c" time(2) with time zone');
        // 不带精度时保持原样，不能凭空补 ()
        expect(pg.genColumnBasicSql(col({ type: 'timestamp with time zone' }))).toBe('"c" timestamp with time zone');
    });

    it('pg 多词类型的参数写在类型名后面（character varying / bit varying）', () => {
        expect(pg.genColumnBasicSql(col({ type: 'character varying', length: 32 }))).toBe('"c" character varying(32)');
        expect(pg.genColumnBasicSql(col({ type: 'bit varying', length: 8 }))).toBe('"c" bit varying(8)');
        // double precision 不接受精度参数，不能写 double precision(53)
        expect(pg.genColumnBasicSql(col({ type: 'double precision', length: 53 }))).toBe('"c" double precision');
    });

    const mysql = asColumnSql(getDbDialect(DbType.mysql));

    it('mysql 无符号整数换序为 DDL 语法形态，且长度插在基础类型后', () => {
        // MySQL 必须显式写可空性（省略即丢），故末尾带 NULL
        expect(mysql.genColumnBasicSql(col({ type: 'unsigned bigint', length: 20 }))).toBe('`c` bigint(20) unsigned NULL');
        expect(mysql.genColumnBasicSql(col({ type: 'unsigned mediumint' }))).toBe('`c` mediumint unsigned NULL');
    });

    it('mysql 新增类型不再被长度规则误加参数', () => {
        // bit 接受长度
        expect(mysql.genColumnBasicSql(col({ type: 'bit', length: 8 }))).toBe('`c` bit(8) NULL');
        // year 只接受 4
        expect(mysql.genColumnBasicSql(col({ type: 'year', length: 4 }))).toBe('`c` year(4) NULL');
        // text/blob 家族、json、几何、bool 都不写参数
        for (const type of ['tinytext', 'tinyblob', 'json', 'geometry', 'bool']) {
            expect(mysql.genColumnBasicSql(col({ type, length: 10 })), `${type} 不应带长度`).toBe(`\`c\` ${type} NULL`);
        }
    });
});
