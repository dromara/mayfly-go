/**
 * 方言类型清单的质量守卫。
 *
 * 这些断言锁的是「清单本身能否被 DDL 生成器正确表达」与「清单是否属于该方言」，
 * 不锁具体条目数量——补类型是常态，但补进来的写错会让用户选中后生成必然失败的 DDL：
 *  - 带占位符的字面量（`timestamp(precision) with time zone`）不是类型名，写进 DDL 即非法；
 *  - 别名被写成别的类型（circle 的 dataType 写成 polygon）会让下拉显示与实际值不符；
 *  - 同名条目重复会出现选不出区别的重复项；
 *  - 把其它产品专有的类型混进来（原生 PG 出现 GaussDB 的 largeserial），用户选了就报错。
 */
import { describe, expect, it } from 'vitest';
import { getDbDialect } from '../registry';
import { DbType } from '../dbType';
import type { SqlColumnType } from '../types';

// 触发各实现的注册（registry 里是注册式，未 import 就不会有方言）
import '../mysql_dialect';
import '../mariadb_dialect';
import '../postgres_dialect';
import '../gauss_dialect';
import '../kingbaseES_dialect';
import '../vastbase_dialect';
import '../mssql_dialect';
import '../oracle_dialect';
import '../oracle11_dialect';
import '../dm_dialect';
import '../clickhouse_dialect';
import '../sqlite_dialect';

type DbTypeValue = (typeof DbType)[keyof typeof DbType];

const DIALECTS: Array<[string, DbTypeValue]> = [
    ['mysql', DbType.mysql],
    ['mariadb', DbType.mariadb],
    ['postgres', DbType.postgres],
    ['gauss', DbType.gauss],
    ['kingbase', DbType.kingbaseEs],
    ['vastbase', DbType.vastbase],
    ['mssql', DbType.mssql],
    ['oracle', DbType.oracle],
    ['dm', DbType.dm],
    ['clickhouse', DbType.clickhouse],
    ['sqlite', DbType.sqlite],
];

const typesOf = (dbType: DbTypeValue): SqlColumnType[] => getDbDialect(dbType).getInfo().columnTypes ?? [];
const namesOf = (dbType: DbTypeValue): string[] => typesOf(dbType).map((t) => t.udtName);

describe('方言类型清单质量', () => {
    it.each(DIALECTS)('%s 的类型项非空、无占位符、无重复', (_name, dbType) => {
        const items = typesOf(dbType);
        expect(items.length).toBeGreaterThan(0);

        for (const item of items) {
            // 类型名不能为空，也不能带括号/占位符——那是文档里的参数说明，不是可执行的类型
            expect(item.udtName.trim()).not.toBe('');
            // 占位符形态一定带括号（`timestamp(precision) with time zone`）；
            // `double precision` 这类合法类型名不能误伤，故不按关键字文本判
            expect(item.udtName).not.toMatch(/[(){}<>]/);
            expect(item.udtName.toLowerCase()).not.toContain('datatype specified');
        }

        const names = namesOf(dbType);
        expect(new Set(names).size, `重复的类型项: ${names.filter((n, i) => names.indexOf(n) !== i).join(',')}`).toBe(names.length);
    });

    it('显示名与取值不一致时，显示名必须是该取值的 DDL 语法形态（如 unsigned bigint ↔ bigint unsigned）', () => {
        for (const [, dbType] of DIALECTS) {
            for (const item of typesOf(dbType)) {
                if (item.dataType === item.udtName) continue;
                // 显示名与取值不同是「别名」的正当形态（GaussDB int1↔tinyint、PG int8↔bigint、
                // MySQL unsigned 换序），但必须在 desc 里点明取值本身——否则就是显示名写错了
                // （曾把 circle 的显示名写成 polygon、DM 的 CHAR 写成 VARCHAR，都是无任何说明的错配）
                const norm = (v: string) => v.toLowerCase().replace(/\s+/g, ' ');
                const unsignedSwap = norm(item.dataType) === norm(item.udtName).replace(/^unsigned\s+/, '') + ' unsigned';
                const aliasExplained = (item.desc || '').toLowerCase().includes(item.udtName.toLowerCase());
                expect(unsignedSwap || aliasExplained, `${dbType}: udtName=${item.udtName} 与 dataType=${item.dataType} 不一致，且 desc 未说明该别名`).toBe(true);
            }
        }
    });

    it('原生 postgres 不含 GaussDB 专有类型，GaussDB 保留自有的 largeserial/nchar/clob', () => {
        const pg = namesOf(DbType.postgres).map((n) => n.toLowerCase());
        expect(pg).not.toContain('largeserial');
        expect(pg).not.toContain('nchar');
        expect(pg).not.toContain('clob');

        const gauss = namesOf(DbType.gauss).map((n) => n.toLowerCase());
        expect(gauss).toContain('largeserial');
        expect(gauss).toContain('nchar');
        expect(gauss).toContain('clob');
    });

    it('postgres 覆盖官方常用类型（含 json/jsonb 与带时区时间）', () => {
        const pg = namesOf(DbType.postgres).map((n) => n.toLowerCase());
        for (const t of ['json', 'jsonb', 'uuid', 'bytea', 'bit', 'bit varying', 'interval', 'timestamp with time zone', 'time with time zone']) {
            expect(pg, `postgres 缺类型 ${t}`).toContain(t);
        }
    });

    it('mysql 覆盖官方类型并给出无符号整数选项（回显值必须能命中）', () => {
        const my = namesOf(DbType.mysql);
        for (const t of ['bigint', 'int', 'mediumint', 'smallint', 'tinyint', 'bit', 'bool', 'year', 'tinytext', 'tinyblob', 'json', 'geometry']) {
            expect(my, `mysql 缺类型 ${t}`).toContain(t);
        }
        for (const base of ['tinyint', 'smallint', 'mediumint', 'int', 'bigint']) {
            expect(my, `mysql 缺无符号类型 unsigned ${base}`).toContain(`unsigned ${base}`);
        }
        // 显示用 DDL 语法形态，便于用户和生成的 SQL 对齐
        const unsignedBigint = typesOf(DbType.mysql).find((t) => t.udtName === 'unsigned bigint');
        expect(unsignedBigint?.dataType).toBe('bigint unsigned');
    });

    it('mssql 不含只能用于变量/参数的类型（cursor/table）', () => {
        const ms = namesOf(DbType.mssql).map((n) => n.toLowerCase());
        expect(ms).not.toContain('cursor');
        expect(ms).not.toContain('table');
        expect(ms).toContain('image');
    });
});
