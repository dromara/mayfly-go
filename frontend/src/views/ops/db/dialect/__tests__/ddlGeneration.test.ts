import { describe, it, expect, beforeAll, vi } from 'vitest';

// Mock problematic dependencies before any imports
vi.mock('@/views/ops/tag/api', () => ({
    tagApi: {
        listByQuery: () => Promise.resolve([]),
        getTagTrees: () => Promise.resolve([]),
        saveTagTree: () => Promise.resolve(),
    },
}));

vi.mock('@/views/ops/component/TagCodePath.vue', () => ({
    default: {},
}));

vi.mock('@/components/system-message/machine/MachineFileUploadProgress.vue', () => ({
    default: {},
}));

vi.mock('../../db', () => ({
    DbInst: {
        isNumber: (type: string) => /int|decimal|numeric|float|double/i.test(type),
    },
}));

import type { TableEditContext, ChangeDiff } from '@/views/ops/db/dialect';
import type { RowDefinition, IndexDefinition } from '../index';
import { getDbDialect } from '../index';
import { DbType } from '../dbType';

// 辅助函数：创建测试用的 TableEditContext
function createTestContext(overrides: Partial<TableEditContext> = {}): TableEditContext {
    return {
        tableName: 'test_table',
        tableComment: '测试表',
        oldTableName: 'test_table',
        oldTableComment: '测试表',
        db: 'test_db/test_schema',
        fields: {
            res: [],
            oldFields: [],
        },
        indexs: {
            res: [],
            oldIndexs: [],
            columns: [],
        },
        ...overrides,
    };
}

// 辅助函数：创建测试用的字段定义
function createTestField(overrides: Partial<RowDefinition> = {}): RowDefinition {
    const name = overrides.name || 'id';
    return {
        name: name,
        oldName: overrides.oldName || name, // oldName 默认等于 name
        type: 'bigint',
        length: overrides.length !== undefined ? overrides.length : '', // 只在明确指定时才有长度
        numScale: '',
        value: '',
        notNull: true,
        pri: false, // 默认不是主键
        auto_increment: false, // 默认不自增
        remark: '',
        ...overrides,
    };
}

// 辅助函数：创建测试用的索引定义
function createTestIndex(overrides: Partial<IndexDefinition> = {}): IndexDefinition {
    return {
        indexName: 'idx_test',
        columnNames: ['name'],
        unique: false,
        indexType: 'BTREE',
        indexComment: '',
        ...overrides,
    };
}

describe('DDL Generation Tests', () => {
    describe('MySQL Dialect', () => {
        const dbType = DbType.mysql;

        it('should generate CREATE TABLE with proper quoting', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({
                db: 'test_db',
                fields: {
                    res: [
                        createTestField({ name: 'id', type: 'bigint', length: '20', pri: true, auto_increment: true, remark: '主键ID' }),
                        createTestField({ name: 'name', type: 'varchar', length: '100', notNull: true, remark: '姓名' }),
                    ],
                    oldFields: [],
                },
            });

            const sql = dialect.getCreateTableSql(ctx);

            expect(sql).toContain('CREATE TABLE `test_table`');
            expect(sql).toContain('`id` bigint(20)');
            expect(sql).toContain('NOT NULL');
            expect(sql).toContain('AUTO_INCREMENT');
            expect(sql).toContain('`name` varchar(100)');
            expect(sql).toContain("COMMENT '姓名'");
            expect(sql).toContain('PRIMARY KEY (`id`)');
            expect(sql).toContain("COMMENT='测试表'");
        });

        it('should generate CREATE INDEX with proper quoting', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({
                db: 'test_db',
                indexs: {
                    res: [createTestIndex({ indexName: 'idx_name', columnNames: ['name', 'age'], unique: false })],
                    oldIndexs: [],
                    columns: [],
                },
            });

            const sql = dialect.getCreateIndexSql(ctx);

            expect(sql).toContain('ALTER TABLE `test_table`');
            expect(sql).toContain('ADD INDEX `idx_name`(`name`,`age`)');
        });

        it('should generate ALTER TABLE ADD COLUMN', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({ db: 'test_db' });
            const changeData: ChangeDiff<RowDefinition> = {
                del: [],
                add: [createTestField({ name: 'email', type: 'varchar', length: '255' })],
                upd: [],
                changed: true,
            };

            const sql = dialect.getModifyColumnSql(ctx, 'test_table', changeData);

            expect(sql).toContain('ALTER TABLE `test_db`.`test_table`');
            expect(sql).toContain('ADD COLUMN `email` varchar(255)');
        });

        it('should generate ALTER TABLE DROP COLUMN', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({ db: 'test_db' });
            const changeData: ChangeDiff<RowDefinition> = {
                del: [createTestField({ name: 'old_field' })],
                add: [],
                upd: [],
                changed: true,
            };

            const sql = dialect.getModifyColumnSql(ctx, 'test_table', changeData);

            expect(sql).toContain('DROP COLUMN `old_field`');
        });

        it('should generate CHANGE COLUMN for rename', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({ db: 'test_db' });
            const changeData: ChangeDiff<RowDefinition> = {
                del: [],
                add: [],
                upd: [createTestField({ name: 'new_name', oldName: 'old_name', type: 'varchar', length: '100' })],
                changed: true,
            };

            const sql = dialect.getModifyColumnSql(ctx, 'test_table', changeData);

            expect(sql).toContain('CHANGE COLUMN `old_name` `new_name` varchar(100)');
        });

        it('should generate RENAME TABLE', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({
                db: 'test_db',
                tableName: 'new_table',
                oldTableName: 'old_table',
            });

            const sql = dialect.getModifyTableInfoSql(ctx);

            expect(sql).toContain('ALTER TABLE `test_db`.`old_table`');
            expect(sql).toContain('RENAME TO `new_table`');
        });
    });

    describe('PostgreSQL Dialect', () => {
        const dbType = DbType.postgresql;

        it('should generate CREATE TABLE with double quotes', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({
                fields: {
                    res: [
                        createTestField({ name: 'id', type: 'bigserial', pri: true, remark: '主键ID' }),
                        createTestField({ name: 'name', type: 'varchar', length: '100', notNull: true, remark: '姓名' }),
                    ],
                    oldFields: [],
                },
            });

            const sql = dialect.getCreateTableSql(ctx);

            expect(sql).toContain('CREATE TABLE "test_table"');
            expect(sql).toContain('"id" bigserial');
            expect(sql).toContain('"name" varchar(100) NOT NULL');
            expect(sql).toContain('PRIMARY KEY ("id")');
            expect(sql).toContain('COMMENT ON TABLE "test_table"');
        });

        it('should generate CREATE INDEX with schema', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({
                db: 'test_db/test_schema',
                indexs: {
                    res: [createTestIndex({ indexName: 'idx_name', columnNames: ['name'] })],
                    oldIndexs: [],
                    columns: [],
                },
            });

            const sql = dialect.getCreateIndexSql(ctx);

            expect(sql).toContain('CREATE INDEX "idx_name"');
            expect(sql).toContain('"test_schema"."test_table"');
            expect(sql).toContain('("name")');
        });

        it('should generate RENAME COLUMN', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({ db: 'test_db/test_schema' });
            const changeData: ChangeDiff<RowDefinition> = {
                del: [],
                add: [],
                upd: [createTestField({ name: 'new_name', oldName: 'old_name', type: 'varchar' })],
                changed: true,
            };

            const sql = dialect.getModifyColumnSql(ctx, 'test_table', changeData);

            expect(sql).toContain('RENAME COLUMN "old_name" TO "new_name"');
        });
    });

    describe('SQLite Dialect', () => {
        const dbType = DbType.sqlite;

        it('should generate CREATE TABLE without schema', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({
                fields: {
                    res: [
                        createTestField({ name: 'id', type: 'integer', pri: true, auto_increment: true, remark: '主键ID' }),
                        createTestField({ name: 'name', type: 'text', notNull: true, remark: '姓名' }),
                    ],
                    oldFields: [],
                },
            });

            const sql = dialect.getCreateTableSql(ctx);

            expect(sql).toContain('CREATE TABLE "test_table"');
            expect(sql).toContain('"id" integer');
            expect(sql).toContain('PRIMARY KEY');
            expect(sql).toContain('AUTOINCREMENT');
            expect(sql).toContain('"name" text');
            expect(sql).toContain('NOT NULL');
        });

        it('should generate CREATE INDEX', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({
                indexs: {
                    res: [createTestIndex({ indexName: 'idx_name', columnNames: ['name'], unique: true })],
                    oldIndexs: [],
                    columns: [],
                },
            });

            const sql = dialect.getCreateIndexSql(ctx);

            expect(sql).toContain('CREATE UNIQUE INDEX "idx_name"');
            expect(sql).toContain('ON "test_table"');
        });

        it('should generate RENAME TABLE', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({
                tableName: 'new_table',
                oldTableName: 'old_table',
            });

            const sql = dialect.getModifyTableInfoSql(ctx);

            expect(sql).toContain('ALTER TABLE "old_table" RENAME TO "new_table"');
        });
    });

    describe('MSSQL Dialect', () => {
        const dbType = DbType.mssql;

        it('should generate CREATE TABLE with brackets', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({
                db: 'test_db/test_schema',
                fields: {
                    res: [
                        createTestField({ name: 'id', type: 'bigint', pri: true, auto_increment: true, remark: '主键ID' }),
                        createTestField({ name: 'name', type: 'nvarchar', length: '100', notNull: true, remark: '姓名' }),
                    ],
                    oldFields: [],
                },
            });

            const sql = dialect.getCreateTableSql(ctx);

            expect(sql).toContain('CREATE TABLE [test_schema].[test_table]');
            expect(sql).toContain('[id] bigint');
            expect(sql).toContain('IDENTITY(1,1)');
            expect(sql).toContain('[name] nvarchar(100) NOT NULL');
            expect(sql).toContain('PRIMARY KEY CLUSTERED ([id])');
            expect(sql).toContain('sp_addextendedproperty');
        });

        it('should not include IDENTITY/DEFAULT in ALTER COLUMN', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({ db: 'test_db/test_schema' });
            const changeData: ChangeDiff<RowDefinition> = {
                del: [],
                add: [],
                upd: [
                    createTestField({
                        name: 'name',
                        type: 'nvarchar',
                        length: '200',
                        auto_increment: true,
                        value: 'GETDATE()',
                    }),
                ],
                changed: true,
            };

            const sql = dialect.getModifyColumnSql(ctx, 'test_table', changeData);

            expect(sql).toContain('ALTER COLUMN [name] nvarchar(200)');
            expect(sql).not.toContain('IDENTITY');
            expect(sql).not.toContain('DEFAULT');
        });
    });

    describe('Oracle Dialect', () => {
        const dbType = DbType.oracle;

        it('should generate CREATE TABLE with double quotes', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({
                db: 'test_db/test_schema',
                fields: {
                    res: [
                        createTestField({ name: 'ID', type: 'NUMBER', pri: true, auto_increment: true, remark: '主键ID' }),
                        createTestField({ name: 'NAME', type: 'VARCHAR2', length: '100', notNull: true, remark: '姓名' }),
                    ],
                    oldFields: [],
                },
            });

            const sql = dialect.getCreateTableSql(ctx);

            expect(sql).toContain('CREATE TABLE "test_schema"."test_table"');
            expect(sql).toContain('"ID"');
            expect(sql).toContain('IDENTITY');
            expect(sql).toContain('"NAME" VARCHAR2(100)');
            expect(sql).toContain('NOT NULL');
            expect(sql).toContain('PRIMARY KEY');
            expect(sql).toContain('COMMENT ON TABLE');
        });

        it('should generate CREATE INDEX with proper quoting', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({
                db: 'test_db/test_schema',
                indexs: {
                    res: [createTestIndex({ indexName: 'IDX_NAME', columnNames: ['NAME', 'AGE'] })],
                    oldIndexs: [],
                    columns: [],
                },
            });

            const sql = dialect.getCreateIndexSql(ctx);

            expect(sql).toContain('CREATE INDEX "IDX_NAME"');
            expect(sql).toContain('"test_schema"."test_table"');
            expect(sql).toContain('("NAME","AGE")');
        });

        it('should generate RENAME COLUMN', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({ db: 'test_db/test_schema' });
            const changeData: ChangeDiff<RowDefinition> = {
                del: [],
                add: [],
                upd: [createTestField({ name: 'NEW_NAME', oldName: 'OLD_NAME', type: 'VARCHAR2' })],
                changed: true,
            };

            const sql = dialect.getModifyColumnSql(ctx, 'test_table', changeData);

            expect(sql).toContain('RENAME COLUMN "OLD_NAME" TO "NEW_NAME"');
        });
    });

    describe('ClickHouse Dialect', () => {
        const dbType = DbType.clickhouse;

        it('should generate CREATE TABLE with backticks', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({
                fields: {
                    res: [
                        createTestField({ name: 'id', type: 'UInt64', pri: true, notNull: true, remark: '主键ID' }),
                        createTestField({ name: 'name', type: 'String', notNull: true, remark: '姓名' }),
                    ],
                    oldFields: [],
                },
            });

            const sql = dialect.getCreateTableSql(ctx);

            expect(sql).toContain('CREATE TABLE `test_table`');
            expect(sql).toContain('`id` UInt64 NOT NULL');
            expect(sql).toContain('`name` String NOT NULL');
            expect(sql).toContain('ENGINE = MergeTree()');
            expect(sql).toContain('ORDER BY (`id`)');
            expect(sql).toContain("COMMENT '测试表'");
        });

        it('should generate ALTER TABLE ADD COLUMN', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext();
            const changeData: ChangeDiff<RowDefinition> = {
                del: [],
                add: [createTestField({ name: 'email', type: 'String', notNull: true })],
                upd: [],
                changed: true,
            };

            const sql = dialect.getModifyColumnSql(ctx, 'test_table', changeData);

            expect(sql).toContain('ALTER TABLE `test_table`');
            expect(sql).toContain('ADD COLUMN `email` String NOT NULL');
        });
    });

    describe('DM Dialect', () => {
        const dbType = DbType.dm;

        it('should generate CREATE TABLE with schema and double quotes', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({
                db: 'test_db/test_schema',
                fields: {
                    res: [
                        createTestField({ name: 'id', type: 'BIGINT', pri: true, auto_increment: true, remark: '主键ID' }),
                        createTestField({ name: 'name', type: 'VARCHAR', length: '100', notNull: true, remark: '姓名' }),
                    ],
                    oldFields: [],
                },
            });

            const sql = dialect.getCreateTableSql(ctx);

            expect(sql).toContain('CREATE TABLE "test_schema"."test_table"');
            expect(sql).toContain('"id" BIGINT');
            expect(sql).toContain('IDENTITY');
            expect(sql).toContain('"name" VARCHAR(100)');
            expect(sql).toContain('NOT NULL');
            expect(sql).toContain('PRIMARY KEY');
            expect(sql).toContain('COMMENT ON TABLE');
        });

        it('should generate CREATE INDEX with schema', () => {
            const dialect = getDbDialect(dbType);
            const ctx = createTestContext({
                db: 'test_db/test_schema',
                indexs: {
                    res: [createTestIndex({ indexName: 'IDX_NAME', columnNames: ['NAME'] })],
                    oldIndexs: [],
                    columns: [],
                },
            });

            const sql = dialect.getCreateIndexSql(ctx);

            expect(sql).toContain('CREATE INDEX "IDX_NAME"');
            expect(sql).toContain('"test_schema"."test_table"');
            expect(sql).toContain('("NAME")');
        });
    });
});
