/**
 * 改列类型时「残留长度」的处理（编辑表单换类型，长度输入框会留着旧值）。
 *
 * 真实风险：把 varchar(64) 改成 json/date/enum 后，若仍按旧长度写出 `json(64)`，
 * MySQL 直接语法错误（实测 1064）；`datetime(320)` 报 1426；`int(300)` 报显示宽度越界。
 * 这里按方言逐一钉住：只有真正接受参数的类型才写括号，时间类型还要在精度区间内。
 */
import { describe, it, expect } from 'vitest';
import { getDbDialect } from '../index';
import type { ColumnDefinition, TableEditContext } from '../types';

function field(type: string, length: string | number = '', numScale: string | number = ''): ColumnDefinition {
    return {
        name: 'c',
        oldName: 'c',
        type,
        value: '',
        length,
        numScale,
        nullable: false,
        isPrimaryKey: false,
        autoIncrement: false,
        comment: '',
    };
}

function ctx(dbType: string, col: ColumnDefinition): TableEditContext {
    return {
        tableName: 't_probe',
        tableComment: '',
        oldTableName: 't_probe',
        oldTableComment: '',
        db: dbType === 'postgres' || dbType === 'mssql' ? 'public' : 'testdb',
        fields: { res: [col], oldFields: [col] },
        indexes: { res: [], oldIndexes: [], columns: [] },
    };
}

/** 取建表语句里该列的定义片段（去掉列名与前后空白） */
function columnDef(dbType: string, col: ColumnDefinition): string {
    const sql = getDbDialect(dbType).getCreateTableSql(ctx(dbType, col));
    const line = sql
        .split('\n')
        .map((l) => l.trim())
        .find((l) => /^[`"[]?c[`"\]]?\s/.test(l));
    if (!line) {
        throw new Error(`未找到列定义：${sql}`);
    }
    return line.replace(/,?$/, '').trim();
}

describe('MySQL：换类型后残留长度的处理', () => {
    it('不接受参数的类型一律不写括号', () => {
        for (const type of ['json', 'date', 'text', 'blob', 'longtext', 'enum', 'set']) {
            expect(columnDef('mysql', field(type, 64)), type).toBe(`\`c\` ${type} NOT NULL`);
        }
    });

    it('时间类型只接受 0~6 的小数秒精度', () => {
        expect(columnDef('mysql', field('datetime', 3))).toBe('`c` datetime(3) NOT NULL');
        expect(columnDef('mysql', field('datetime', 32))).toBe('`c` datetime NOT NULL');
        expect(columnDef('mysql', field('timestamp', 6))).toBe('`c` timestamp(6) NOT NULL');
    });

    it('整数只接受 1~255 的显示宽度', () => {
        expect(columnDef('mysql', field('int', 64))).toBe('`c` int(64) NOT NULL');
        expect(columnDef('mysql', field('int', 300))).toBe('`c` int NOT NULL');
        expect(columnDef('mysql', field('unsigned bigint', 20))).toBe('`c` bigint(20) unsigned NOT NULL');
        expect(columnDef('mysql', field('unsigned bigint', 300))).toBe('`c` bigint unsigned NOT NULL');
    });

    it('double 必须带小数位，float 可以只给精度', () => {
        expect(columnDef('mysql', field('double', 22))).toBe('`c` double NOT NULL');
        expect(columnDef('mysql', field('double', 22, 2))).toBe('`c` double(22,2) NOT NULL');
        expect(columnDef('mysql', field('float', 12))).toBe('`c` float(12) NOT NULL');
    });

    it('text/blob/json 不写字面量默认值（换类型残留旧默认值会 1101）', () => {
        for (const type of ['text', 'longtext', 'blob', 'json']) {
            const sql = getDbDialect('mysql').getCreateTableSql(ctx('mysql', { ...field(type, 64), value: 'abc' }));
            expect(sql, type).not.toContain('DEFAULT');
        }
        // varchar 仍照常带默认值
        expect(getDbDialect('mysql').getCreateTableSql(ctx('mysql', { ...field('varchar', 64), value: 'abc' }))).toContain("DEFAULT 'abc'");
    });

    it('字符与定点数照常带参数', () => {
        expect(columnDef('mysql', field('varchar', 64))).toBe('`c` varchar(64) NOT NULL');
        expect(columnDef('mysql', field('decimal', 10, 3))).toBe('`c` decimal(10,3) NOT NULL');
        expect(columnDef('mysql', field('decimal', 10))).toBe('`c` decimal(10) NOT NULL');
    });
});

describe('PostgreSQL：换类型后残留长度的处理', () => {
    it('text/int4 不写括号，timestamp 只在 0~6 时写', () => {
        expect(columnDef('postgres', field('text', 32))).toBe('"c" text NOT NULL');
        expect(columnDef('postgres', field('int4', 64))).toBe('"c" int4 NOT NULL');
        expect(columnDef('postgres', field('timestamp', 3))).toBe('"c" timestamp(3) NOT NULL');
        expect(columnDef('postgres', field('timestamp', 32))).toBe('"c" timestamp NOT NULL');
        expect(columnDef('postgres', field('varchar', 64))).toBe('"c" varchar(64) NOT NULL');
        expect(columnDef('postgres', field('numeric', 10, 2))).toBe('"c" numeric(10, 2) NOT NULL');
    });
});

describe('MSSQL：换类型后残留长度的处理', () => {
    it('int 不写括号，nvarchar 写，datetime2 只在 0~7 时写', () => {
        expect(columnDef('mssql', field('int', 64))).toBe('[c] int NOT NULL');
        expect(columnDef('mssql', field('nvarchar', 64))).toBe('[c] nvarchar(64) NOT NULL');
        expect(columnDef('mssql', field('datetime2', 3))).toBe('[c] datetime2(3) NOT NULL');
        expect(columnDef('mssql', field('datetime2', 32))).toBe('[c] datetime2 NOT NULL');
        expect(columnDef('mssql', field('decimal', 10, 2))).toBe('[c] decimal(10,2) NOT NULL');
    });
});

describe('ClickHouse：换类型后残留长度的处理', () => {
    it('只有 FixedString/Decimal/DateTime64 接受参数', () => {
        expect(columnDef('clickhouse', field('String', 64))).toBe('`c` String');
        expect(columnDef('clickhouse', field('Int64', 64))).toBe('`c` Int64');
        expect(columnDef('clickhouse', field('FixedString', 16))).toBe('`c` FixedString(16)');
        expect(columnDef('clickhouse', field('Decimal', 10, 2))).toBe('`c` Decimal(10,2)');
        expect(columnDef('clickhouse', field('DateTime64', 3))).toBe('`c` DateTime64(3)');
    });
});
