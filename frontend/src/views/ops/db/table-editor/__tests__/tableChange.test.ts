/**
 * 表编辑「变更求值 + SQL 组装」的矩阵用例。
 *
 * 这些断言直接对应界面上每种编辑动作最终落到库里的那条 SQL，覆盖：
 * 无变更、改列注释/长度/可空/默认值、改名、增删列、改主键、改索引、改表名与表注释、
 * 以及「改名必须排在列变更之前」的顺序约束。
 */
import { describe, it, expect } from 'vitest';
import { getDbDialect } from '../../dialect/index';
import { buildTableEditSql, evaluateTableEdit, filterChangedData } from '../tableChange';
import type { ColumnDefinition, IndexDefinition, TableEditContext } from '../../dialect/types';

function col(overrides: Partial<ColumnDefinition> = {}): ColumnDefinition {
    const name = overrides.name ?? 'col';
    return {
        name,
        oldName: overrides.oldName ?? name,
        type: 'varchar',
        value: '',
        length: 32,
        numScale: '',
        nullable: false,
        isPrimaryKey: false,
        autoIncrement: false,
        comment: '',
        ...overrides,
    };
}

function index(overrides: Partial<IndexDefinition> = {}): IndexDefinition {
    return { indexName: 'idx_a', columnNames: ['a'], unique: false, indexType: 'BTREE', indexComment: '', ...overrides };
}

/** 造一份「编辑前 = 编辑后」的上下文，调用方再改 res 制造变更 */
function context(res: ColumnDefinition[], indexes: IndexDefinition[] = []): TableEditContext {
    return {
        tableName: 't_probe',
        tableComment: '',
        oldTableName: 't_probe',
        oldTableComment: '',
        db: 'e2e',
        fields: { res, oldFields: res.map((item) => structuredClone(item)) },
        indexes: { res: indexes, oldIndexes: indexes.map((item) => structuredClone(item)), columns: [] },
    };
}

const mysql = getDbDialect('mysql');

describe('filterChangedData', () => {
    it('无变更（数字与字符串形态相同）不产生任何 diff', () => {
        const old = [col({ length: 32 })];
        const now = [col({ length: '32' })];
        const diff = filterChangedData(old, now, 'name');
        expect(diff.changed).toBe(false);
        expect(diff.upd).toEqual([]);
    });

    it('改名只产生一条 upd，不会被拆成删一列+加一列', () => {
        const old = [col({ name: 'a' })];
        const now = [col({ name: 'b', oldName: 'a' })];
        const diff = filterChangedData(old, now, 'name');
        expect(diff.add).toEqual([]);
        expect(diff.del).toEqual([]);
        expect(diff.upd.map((item) => item.name)).toEqual(['b']);
    });

    it('新增与删除各自归位', () => {
        const diff = filterChangedData([col({ name: 'a' })], [col({ name: 'b' })], 'name');
        expect(diff.add.map((item) => item.name)).toEqual(['b']);
        expect(diff.del.map((item) => item.name)).toEqual(['a']);
    });

    it('把列改名成另一已存在列的名字时也只产生一条 upd', () => {
        // 重名本身非法，但求值不能因此把同一改名项输出两次：那会拼出两条相同的 CHANGE COLUMN，
        // 第二条必然报 Unknown column，把「列名重复」这个真实原因盖掉
        const old = [col({ name: 'id' }), col({ name: 'a' }), col({ name: 'b' })];
        const now = [col({ name: 'id' }), col({ name: 'a' }), col({ name: 'a', oldName: 'b' })];
        const diff = filterChangedData(old, now, 'name');
        expect(diff.upd).toHaveLength(1);
        expect(diff.upd[0]).toMatchObject({ name: 'a', oldName: 'b' });
    });
});

describe('buildTableEditSql（MySQL）', () => {
    it('没有任何变更时返回空串（界面据此提示「无更改」）', () => {
        expect(buildTableEditSql(mysql, context([col()]), true)).toBe('');
    });

    it('只改注释：只发一条 MODIFY COLUMN', () => {
        const data = context([col({ name: 'name', comment: '名称' })]);
        data.fields.res[0].comment = '名称2';
        const sql = buildTableEditSql(mysql, data, true);
        expect(sql).toContain('MODIFY COLUMN `name` varchar(32) NOT NULL COMMENT \'名称2\'');
        expect(sql.match(/MODIFY COLUMN/g)).toHaveLength(1);
    });

    it('改长度与可空一并体现在同一条 MODIFY 里', () => {
        const data = context([col({ name: 'name' })]);
        data.fields.res[0].length = 64;
        data.fields.res[0].nullable = true;
        expect(buildTableEditSql(mysql, data, true)).toContain('MODIFY COLUMN `name` varchar(64) NULL');
    });

    it('表达式默认值不加引号，ON UPDATE 原样带回', () => {
        const data = context([col({ name: 'token', value: 'uuid()', valueIsExpression: true })]);
        data.fields.res[0].comment = '令牌';
        expect(buildTableEditSql(mysql, data, true)).toContain("DEFAULT (uuid()) COMMENT '令牌'");

        const data2 = context([col({ name: 'updated_at', type: 'datetime', length: '', value: 'CURRENT_TIMESTAMP', onUpdate: 'ON UPDATE CURRENT_TIMESTAMP' })]);
        data2.fields.res[0].nullable = true;
        const sql = buildTableEditSql(mysql, data2, true);
        expect(sql).toContain('DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP');
    });

    it('默认值/注释里的单引号必须转义，不能拼出会截断的 SQL', () => {
        const data = context([col({ name: 'name' })]);
        data.fields.res[0].value = "a'b";
        data.fields.res[0].comment = "备注'x";
        const sql = buildTableEditSql(mysql, data, true);

        expect(sql).toContain("DEFAULT 'a''b'");
        expect(sql).toContain("COMMENT '备注''x'");
    });

    it('勾选主键：同一批里 DROP 旧主键再 ADD 新主键', () => {
        const data = context([col({ name: 'id', isPrimaryKey: true }), col({ name: 'name' })]);
        data.fields.res[1].isPrimaryKey = true;
        const sql = buildTableEditSql(mysql, data, true);
        expect(sql).toContain('DROP PRIMARY KEY');
        expect(sql).toContain('ADD PRIMARY KEY (`id`,`name`)');
    });

    // 真实故障：name 先被改成可空，再勾主键 → 旧实现只发 ADD PRIMARY KEY，MySQL 报 1171
    // （All parts of a PRIMARY KEY must be NOT NULL）
    it('勾选主键时该列必须同时改成 NOT NULL', () => {
        const data = context([col({ name: 'id', isPrimaryKey: true }), col({ name: 'name', nullable: true })]);
        data.fields.res[1].isPrimaryKey = true;
        data.fields.res[1].nullable = true; // 表单上仍是可空
        const sql = buildTableEditSql(mysql, data, true);

        expect(sql).toContain('MODIFY COLUMN `name` varchar(32) NOT NULL');
        expect(sql).toContain('ADD PRIMARY KEY (`id`,`name`)');
        expect(sql).not.toContain('varchar(32) NULL');
    });

    it('主键列本来可空（元数据异常）也不会生成可空的主键', () => {
        const data = context([col({ name: 'id', isPrimaryKey: true, nullable: true })]);
        data.fields.res[0].comment = '主键';
        expect(buildTableEditSql(mysql, data, true)).toContain('MODIFY COLUMN `id` varchar(32) NOT NULL');
    });

    it('取消主键：只发 DROP PRIMARY KEY，不再 ADD', () => {
        const data = context([col({ name: 'id', isPrimaryKey: true })]);
        data.fields.res[0].isPrimaryKey = false;
        const sql = buildTableEditSql(mysql, data, true);
        expect(sql).toContain('DROP PRIMARY KEY');
        expect(sql).not.toContain('ADD PRIMARY KEY');
    });

    it('只改列名：CHANGE COLUMN 带上原列名与完整定义', () => {
        const data = context([col({ name: 'name' })]);
        data.fields.res[0].name = 'name2';
        const sql = buildTableEditSql(mysql, data, true);
        expect(sql).toContain('CHANGE COLUMN `name` `name2` varchar(32) NOT NULL');
        expect(sql).not.toContain('DROP COLUMN');
    });

    it('增删列各自成句', () => {
        const data = context([col({ name: 'a' }), col({ name: 'b' })]);
        data.fields.res[0].isPrimaryKey = false;
        data.fields.res = [data.fields.res[0], col({ name: 'c', length: 10, nullable: true })];
        const sql = buildTableEditSql(mysql, data, true);
        expect(sql).toContain('DROP COLUMN `b`');
        expect(sql).toContain('ADD COLUMN `c` varchar(10) NULL');
    });

    it('索引换列：先 DROP 旧索引再 ADD 新索引，且与列语句一样带库限定', () => {
        const data = context([col({ name: 'a' }), col({ name: 'b' })], [index({ indexName: 'idx_a', columnNames: ['a'] })]);
        data.indexes.res[0].columnNames = ['b'];
        const sql = buildTableEditSql(mysql, data, true);
        expect(sql).toContain('DROP INDEX `idx_a`');
        expect(sql).toContain('ADD INDEX `idx_a`(`b`)');
        // 裸表名会落到连接默认库：索引 DDL 必须与列 DDL 同一套 `库`.`表` 命名
        expect(sql).toContain('ALTER TABLE `e2e`.`t_probe`');
    });

    it('改表名：RENAME 必须排在列变更之前（列语句引用的是新表名）', () => {
        const data = context([col({ name: 'name' })]);
        data.tableName = 't_probe_renamed';
        data.fields.res[0].comment = '新注释';
        const sql = buildTableEditSql(mysql, data, true);
        const renameAt = sql.indexOf('RENAME TO `t_probe_renamed`');
        const modifyAt = sql.indexOf('MODIFY COLUMN');
        expect(renameAt).toBeGreaterThan(-1);
        expect(modifyAt).toBeGreaterThan(renameAt);
        expect(sql).toContain('MODIFY COLUMN `name` varchar(32) NOT NULL');
    });

    it('改表注释：只发表注释语句', () => {
        const data = context([col({ name: 'name' })]);
        data.tableComment = '新表注释';
        const sql = buildTableEditSql(mysql, data, true);
        expect(sql).toContain("COMMENT '新表注释'");
        expect(sql).not.toContain('MODIFY COLUMN');
    });

    it('新建表：建表 + 建索引，语句不以空语句结尾', () => {
        const data = context([col({ name: 'id', isPrimaryKey: true, autoIncrement: true })], [index()]);
        const sql = buildTableEditSql(mysql, data, false);
        expect(sql).toContain('CREATE TABLE `t_probe`');
        expect(sql).toContain('ADD INDEX `idx_a`(`a`)');
        expect(sql).not.toContain(';;');
        expect(sql.endsWith(';')).toBe(true);
    });
});

describe('evaluateTableEdit', () => {
    it('表注释/表名变化被识别', () => {
        const data = context([col()]);
        expect(evaluateTableEdit(data).tableInfoChanged).toBe(false);
        data.tableComment = 'x';
        expect(evaluateTableEdit(data).tableInfoChanged).toBe(true);
    });
});
