import { describe, it, expect } from 'vitest';
import { buildColumnDefinitions, declaredColumnArgs, normalizeDefaultValue, normalizeOnUpdate } from '../columnDefinition';
import type { ColumnMetadata } from '../../types';

/** 造一条元数据（字段与后端 dbi.Column 的 JSON 同名同义） */
function meta(overrides: Partial<ColumnMetadata>): ColumnMetadata {
    return {
        columnName: 'col',
        dataType: 'varchar',
        ...overrides,
    };
}

describe('declaredColumnArgs', () => {
    it('字符类型回填声明的长度', () => {
        // mayfly_go.t_ai_memory.creator 实测：varchar(32)
        expect(declaredColumnArgs(meta({ dataType: 'varchar', charMaxLength: 32 }))).toEqual({ length: 32, numScale: '' });
        expect(declaredColumnArgs(meta({ dataType: 'nvarchar', charMaxLength: 100 }))).toEqual({ length: 100, numScale: '' });
        expect(declaredColumnArgs(meta({ dataType: 'VARCHAR2', charMaxLength: 100 }))).toEqual({ length: 100, numScale: '' });
        // 二进制类型的长度同样必须保留，否则 MODIFY 出 varbinary 无参数
        expect(declaredColumnArgs(meta({ dataType: 'varbinary', charMaxLength: 16 }))).toEqual({ length: 16, numScale: '' });
    });

    it('定点数回填精度与小数位', () => {
        expect(declaredColumnArgs(meta({ dataType: 'decimal', numPrecision: 10, numScale: 2 }))).toEqual({ length: 10, numScale: 2 });
        expect(declaredColumnArgs(meta({ dataType: 'numeric', numPrecision: 12, numScale: 0 }))).toEqual({ length: 12, numScale: '' });
        expect(declaredColumnArgs(meta({ dataType: 'NUMBER', numPrecision: 8 }))).toEqual({ length: 8, numScale: '' });
        // 无符号 decimal 的 dataType 同样带 unsigned 前缀，长度口径不受影响
        expect(declaredColumnArgs(meta({ dataType: 'unsigned decimal', numPrecision: 10, numScale: 2 }))).toEqual({ length: 10, numScale: 2 });
    });

    it('时间类型回填小数秒精度', () => {
        expect(declaredColumnArgs(meta({ dataType: 'datetime', numPrecision: 3 }))).toEqual({ length: 3, numScale: '' });
        expect(declaredColumnArgs(meta({ dataType: 'timestamp', numPrecision: 6 }))).toEqual({ length: 6, numScale: '' });
        // 无小数秒的 datetime 元数据为 0，不回填（避免写出 datetime(0)）
        expect(declaredColumnArgs(meta({ dataType: 'datetime', numPrecision: 0 }))).toEqual({ length: '', numScale: '' });
    });

    it('整数/浮点不回填：NUMERIC_PRECISION 对它们是位宽/显示宽度，写进 DDL 会失真', () => {
        // MySQL 8.0.46 实测：bigint unsigned → NUMERIC_PRECISION=20，double → 22，tinyint → 3
        expect(declaredColumnArgs(meta({ dataType: 'unsigned bigint', numPrecision: 20 }))).toEqual({ length: '', numScale: '' });
        expect(declaredColumnArgs(meta({ dataType: 'double', numPrecision: 22 }))).toEqual({ length: '', numScale: '' });
        expect(declaredColumnArgs(meta({ dataType: 'tinyint', numPrecision: 3 }))).toEqual({ length: '', numScale: '' });
        expect(declaredColumnArgs(meta({ dataType: 'int4', numPrecision: 32 }))).toEqual({ length: '', numScale: '' });
    });

    it('无参数类型留空', () => {
        expect(declaredColumnArgs(meta({ dataType: 'text' }))).toEqual({ length: '', numScale: '' });
        expect(declaredColumnArgs(meta({ dataType: 'date' }))).toEqual({ length: '', numScale: '' });
        expect(declaredColumnArgs(meta({ dataType: 'enum', charMaxLength: 8 }))).toEqual({ length: '', numScale: '' });
    });
});

describe('buildColumnDefinitions', () => {
    it('把元数据整表转成表单列定义（长度不再依赖 initColumns 的事后写入）', () => {
        const defs = buildColumnDefinitions([
            meta({ columnName: 'id', dataType: 'unsigned bigint', numPrecision: 20, nullable: false, isPrimaryKey: true, autoIncrement: true }),
            meta({ columnName: 'creator', dataType: 'varchar', charMaxLength: 32, columnComment: '创建人' }),
            meta({ columnName: 'content', dataType: 'text', columnComment: '记忆内容' }),
            meta({ columnName: 'tags', dataType: 'varchar', charMaxLength: 512, nullable: true, columnDefault: 'NULL' }),
        ]);

        expect(defs.map((d) => [d.name, d.type, d.length, d.numScale, d.nullable])).toEqual([
            ['id', 'unsigned bigint', '', '', false],
            ['creator', 'varchar', 32, '', false],
            ['content', 'text', '', '', false],
            ['tags', 'varchar', 512, '', true],
        ]);
        // oldName 必须与 name 一致：差异对比按 oldName 识别「改名」，缺失会被判成删列+加列
        expect(defs.map((d) => d.oldName)).toEqual(['id', 'creator', 'content', 'tags']);
        expect(defs[1].comment).toBe('创建人');
    });

    it('默认值去掉字面量引号与 pg 的类型转换后缀', () => {
        const [pgDefault, emptyDefault] = buildColumnDefinitions([
            meta({ columnName: 'a', dataType: 'varchar', charMaxLength: 10, columnDefault: "'x'::character varying" }),
            meta({ columnName: 'b', dataType: 'date' }),
        ]);
        expect(pgDefault.value).toBe('x');
        expect(emptyDefault.value).toBe('');
    });
});

describe('normalizeDefaultValue', () => {
    it('表达式默认值原样保留（不能去引号也不能加引号）', () => {
        expect(normalizeDefaultValue('uuid()', true)).toBe('uuid()');
        expect(normalizeDefaultValue("timezone('utc'::text, now())", true)).toBe("timezone('utc'::text, now())");
    });

    it('pg 字面量：去 cast 后缀与引号，还原成裸值', () => {
        expect(normalizeDefaultValue("'abc'", false)).toBe('abc');
        expect(normalizeDefaultValue("'abc'::character varying", false)).toBe('abc');
        expect(normalizeDefaultValue("'abc'::text", false)).toBe('abc');
        expect(normalizeDefaultValue("'x'::character varying(10)", false)).toBe('x');
        // 内部单引号双写还原
        expect(normalizeDefaultValue("'a''b'", false)).toBe("a'b");
    });

    it('数字/布尔/函数关键字的裸值原样保留；空值归一为空串', () => {
        expect(normalizeDefaultValue('0.00', false)).toBe('0.00');
        expect(normalizeDefaultValue('true', false)).toBe('true');
        expect(normalizeDefaultValue('CURRENT_TIMESTAMP', false)).toBe('CURRENT_TIMESTAMP');
        expect(normalizeDefaultValue('', false)).toBe('');
    });
});

describe('normalizeOnUpdate', () => {
    it('关键字大写化，表达式原文保留', () => {
        expect(normalizeOnUpdate('on update CURRENT_TIMESTAMP')).toBe('ON UPDATE CURRENT_TIMESTAMP');
        expect(normalizeOnUpdate('ON UPDATE CURRENT_TIMESTAMP(3)')).toBe('ON UPDATE CURRENT_TIMESTAMP(3)');
        expect(normalizeOnUpdate('')).toBeUndefined();
        expect(normalizeOnUpdate(undefined)).toBeUndefined();
    });
});
