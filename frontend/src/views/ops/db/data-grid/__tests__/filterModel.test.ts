import { describe, expect, it } from 'vitest';

import { DbType, getDbDialect } from '@/views/ops/db/dialect';
import type { TableColumnDef } from '@/views/ops/db/types';
import {
    columnKind,
    createFilterCondition,
    createFilterGroup,
    datePickerType,
    isConditionComplete,
    isConditionUsable,
    operatorsForKind,
    serializeFilterGroup,
} from '../filterModel';

// 可视化过滤条件模型契约：列类型归化、按类型操作符集合、WHERE 序列化。
// 用真实 mysql 方言验证引号包裹与值包装，避免 mock 与实现同错。
const dialect = getDbDialect(DbType.mysql);
const q = (name: string) => dialect.quoteIdentifier(name);

const columns: TableColumnDef[] = [
    { columnName: 'creator', columnType: 'varchar(64)' },
    { columnName: 'id', columnType: 'int' },
    { columnName: 'create_time', columnType: 'datetime' },
];

const cond = (patch: Partial<ReturnType<typeof createFilterCondition>>) => ({ ...createFilterCondition(), ...patch });

describe('filterModel 列类型归化与操作符集合', () => {
    it('方言类型归化为 string / number / date 三维度', () => {
        expect(columnKind(columns[0], dialect)).toBe('string');
        expect(columnKind(columns[1], dialect)).toBe('number');
        expect(columnKind(columns[2], dialect)).toBe('date');
    });

    it('操作符集合按列类型收敛', () => {
        const stringOps = operatorsForKind('string').map((x) => x.value);
        expect(stringOps).toContain('like');
        expect(stringOps).not.toContain('>');
        const numberOps = operatorsForKind('number').map((x) => x.value);
        expect(numberOps).toContain('between');
        expect(numberOps).not.toContain('like');
    });

    it('带长度/unsigned 修饰的展示类型归化为数值', () => {
        const uidCol: TableColumnDef = { columnName: 'uid', columnType: 'unsigned bigint(20)' };
        expect(columnKind(uidCol, dialect)).toBe('number');
    });
});

describe('serializeFilterGroup WHERE 序列化', () => {
    it('单值条件按类型包装值、列名经方言引号包裹', () => {
        const group = createFilterGroup();
        group.conditions.push(cond({ columnName: 'creator', operator: '=', value: 'admin' }));
        expect(serializeFilterGroup(group, columns, dialect)).toBe(`${q('creator')} = 'admin'`);
    });

    it('包含/不包含展开为双侧通配 LIKE', () => {
        const group = createFilterGroup();
        group.conditions.push(cond({ columnName: 'creator', operator: 'like', value: 'adm' }));
        expect(serializeFilterGroup(group, columns, dialect)).toBe(`${q('creator')} LIKE '%adm%'`);
    });

    it('between / in / isNull 形态', () => {
        const group = createFilterGroup();
        group.conditions.push(cond({ columnName: 'id', operator: 'between', value: '1', value2: '10' }));
        group.conditions.push(cond({ columnName: 'creator', operator: 'in', inValues: ['a', 'b'] }));
        group.conditions.push(cond({ columnName: 'create_time', operator: 'isNull' }));
        expect(serializeFilterGroup(group, columns, dialect)).toBe(
            `${q('id')} BETWEEN 1 AND 10 AND ${q('creator')} IN ('a', 'b') AND ${q('create_time')} IS NULL`
        );
    });

    it('不完整条件（缺值/缺列）静默跳过，全空序列化为空串', () => {
        const group = createFilterGroup();
        group.conditions.push(cond({ columnName: 'creator', operator: '=', value: '' }));
        group.conditions.push(cond({ columnName: 'no_such_column', operator: '=', value: 'x' }));
        expect(serializeFilterGroup(group, columns, dialect)).toBe('');
    });

    it('OR 组合逻辑按扁平逻辑连接', () => {
        const group = createFilterGroup();
        group.logic = 'OR';
        group.conditions.push(cond({ columnName: 'id', operator: '=', value: '1' }), cond({ columnName: 'id', operator: '=', value: '2' }));
        expect(serializeFilterGroup(group, columns, dialect)).toBe(`${q('id')} = 1 OR ${q('id')} = 2`);
    });

    it('unsigned 数值列的值不加引号', () => {
        const uidCol: TableColumnDef = { columnName: 'uid', columnType: 'unsigned bigint(20)' };
        const group = createFilterGroup();
        group.conditions.push(cond({ columnName: 'uid', operator: '=', value: '7' }));
        group.conditions.push(cond({ columnName: 'uid', operator: 'between', value: '1', value2: '100' }));
        expect(serializeFilterGroup(group, [uidCol], dialect)).toBe(`${q('uid')} = 7 AND ${q('uid')} BETWEEN 1 AND 100`);
    });
});

describe('值合法性与 SQL 转义', () => {
    it('数值列拒绝非数字文本，字符串列不受限', () => {
        expect(isConditionComplete(cond({ columnName: 'id', operator: '=', value: 'abc' }), 'number')).toBe(false);
        expect(isConditionComplete(cond({ columnName: 'id', operator: '=', value: ' 12.5 ' }), 'number')).toBe(true);
        expect(isConditionComplete(cond({ columnName: 'id', operator: '=', value: 'abc' }), 'string')).toBe(true);
    });

    it('含撇号值序列化时双写单引号转义（含 LIKE / IN 形态）', () => {
        const group = createFilterGroup();
        group.conditions.push(cond({ columnName: 'creator', operator: '=', value: "O'Brien" }));
        group.conditions.push(cond({ columnName: 'creator', operator: 'like', value: "a'b" }));
        group.conditions.push(cond({ columnName: 'creator', operator: 'in', inValues: ["x'y", 'z'] }));
        expect(serializeFilterGroup(group, columns, dialect)).toBe(
            `${q('creator')} = 'O''Brien' AND ${q('creator')} LIKE '%a''b%' AND ${q('creator')} IN ('x''y', 'z')`
        );
    });

    it('time 列归化为 time picker 形态，值按字符串包装', () => {
        const timeCol: TableColumnDef = { columnName: 'start_at', columnType: 'time' };
        expect(datePickerType(timeCol, dialect)).toBe('time');
        const group = createFilterGroup();
        group.conditions.push(cond({ columnName: 'start_at', operator: '=', value: '12:00:00' }));
        expect(serializeFilterGroup(group, [timeCol], dialect)).toBe(`${q('start_at')} = '12:00:00'`);
    });

    it('isConditionUsable 联合列存在性与类型校验', () => {
        expect(isConditionUsable(cond({ columnName: 'id', operator: '=', value: 'abc' }), columns, dialect)).toBe(false);
        expect(isConditionUsable(cond({ columnName: 'nope', operator: '=', value: '1' }), columns, dialect)).toBe(false);
        expect(isConditionUsable(cond({ columnName: 'id', operator: '=', value: '1' }), columns, dialect)).toBe(true);
    });
});
