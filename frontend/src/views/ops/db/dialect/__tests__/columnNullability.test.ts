/**
 * 列「长度 / 可为空」变更在各方言生成 SQL 的正确性。
 *
 * 背景（真实故障）：MySQL 走整列 MODIFY 会连带写出可空，pg/oracle/dm 的更新路径却是
 * 「只重写类型」——勾选「可为空」根本不出现在 SQL 里，保存后界面提示成功但库没变。
 * 这里按方言逐一钉住：改长度必须带长度参数，改可空必须在 SQL 里可见。
 */
import { describe, it, expect } from 'vitest';
import { getDbDialect } from '../index';
import type { ChangeDiff, ColumnDefinition, TableEditContext } from '../types';

function field(overrides: Partial<ColumnDefinition> = {}): ColumnDefinition {
    const name = overrides.name ?? 'col';
    return {
        name,
        oldName: overrides.oldName ?? name,
        type: 'varchar',
        value: '',
        length: '',
        numScale: '',
        nullable: false,
        isPrimaryKey: false,
        autoIncrement: false,
        comment: '',
        ...overrides,
    };
}

function ctx(dbType: string, oldFields: ColumnDefinition[], res: ColumnDefinition[]): TableEditContext {
    return {
        tableName: 't_probe',
        tableComment: '',
        oldTableName: 't_probe',
        oldTableComment: '',
        db: dbType === 'postgres' || dbType === 'oracle' || dbType === 'dm' || dbType === 'mssql' ? 'public' : 'testdb',
        fields: { res, oldFields },
        indexes: { res: [], oldIndexes: [], columns: [] },
    };
}

function upd(cols: ColumnDefinition[]): ChangeDiff<ColumnDefinition> {
    return { del: [], add: [], upd: cols, changed: true };
}

describe('各方言：改长度 + 勾选可为空', () => {
    it('PostgreSQL：改类型必须带 USING，否则 text→int 这类会被服务端拒绝', () => {
        const dialect = getDbDialect('postgres');
        const old = field({ type: 'text', nullable: true });
        const now = field({ type: 'int4', nullable: true });
        const sql = dialect.getModifyColumnSql(ctx('postgres', [old], [now]), 't_probe', upd([now]));

        expect(sql).toContain('ALTER COLUMN "col" TYPE int4 USING "col"::int4');
    });

    it('PostgreSQL：长度写进 TYPE，可空单独下发 DROP NOT NULL', () => {
        const dialect = getDbDialect('postgres');
        const old = field({ type: 'varchar', length: 32, nullable: false });
        const now = field({ type: 'varchar', length: 64, nullable: true });
        const sql = dialect.getModifyColumnSql(ctx('postgres', [old], [now]), 't_probe', upd([now]));

        expect(sql).toContain('ALTER COLUMN "col" TYPE varchar(64)');
        expect(sql).toContain('ALTER COLUMN "col" DROP NOT NULL;');
    });

    it('PostgreSQL：只改可空时不下发 TYPE（避免无谓的整列重写）', () => {
        const dialect = getDbDialect('postgres');
        const old = field({ type: 'int4', nullable: false });
        const now = field({ type: 'int4', nullable: true });
        const sql = dialect.getModifyColumnSql(ctx('postgres', [old], [now]), 't_probe', upd([now]));

        expect(sql).toContain('ALTER COLUMN "col" DROP NOT NULL;');
        expect(sql).not.toContain('TYPE');
    });

    it('PostgreSQL：取消可空下发 SET NOT NULL，清空默认值下发 DROP DEFAULT', () => {
        const dialect = getDbDialect('postgres');
        const old = field({ type: 'varchar', length: 32, nullable: true, value: 'abc' });
        const now = field({ type: 'varchar', length: 32, nullable: false, value: '' });
        const sql = dialect.getModifyColumnSql(ctx('postgres', [old], [now]), 't_probe', upd([now]));

        expect(sql).toContain('ALTER COLUMN "col" SET NOT NULL;');
        expect(sql).toContain('ALTER COLUMN "col" DROP DEFAULT;');
    });

    it('Oracle：MODIFY 显式写 NULL，长度带进类型', () => {
        const dialect = getDbDialect('oracle');
        const old = field({ type: 'VARCHAR2', length: 32, nullable: false });
        const now = field({ type: 'VARCHAR2', length: 64, nullable: true });
        const sql = dialect.getModifyColumnSql(ctx('oracle', [old], [now]), 't_probe', upd([now]));

        expect(sql).toContain('MODIFY ( "col" VARCHAR2(64)  NULL );')
        expect(sql).not.toContain('NOT NULL');
    });

    it('ORACLE 11G：同一套可空/长度口径', () => {
        const dialect = getDbDialect('oracle', '11.2');
        const old = field({ type: 'VARCHAR2', length: 32, nullable: true });
        const now = field({ type: 'VARCHAR2', length: 64, nullable: false });
        const sql = dialect.getModifyColumnSql(ctx('oracle', [old], [now]), 't_probe', upd([now]));

        expect(sql).toContain('VARCHAR2(64)');
        expect(sql).toContain('NOT NULL');
        expect(sql).not.toContain('VARCHAR2(64)  NULL');
    });

    it('DM：MODIFY 显式写 NULL，长度带进类型', () => {
        const dialect = getDbDialect('dm');
        const old = field({ type: 'VARCHAR', length: 32, nullable: false });
        const now = field({ type: 'VARCHAR', length: 64, nullable: true });
        const sql = dialect.getModifyColumnSql(ctx('dm', [old], [now]), 't_probe', upd([now]));

        expect(sql).toContain('MODIFY  "col" VARCHAR(64)');
        expect(sql).toContain(' NULL ');
        expect(sql).not.toContain('NOT NULL');
    });

    it('MSSQL：ALTER COLUMN 显式写 NULL / NOT NULL', () => {
        const dialect = getDbDialect('mssql');
        const old = field({ type: 'nvarchar', length: 32, nullable: false });
        const now = field({ type: 'nvarchar', length: 64, nullable: true });
        const sql = dialect.getModifyColumnSql(ctx('mssql', [old], [now]), 't_probe', upd([now]));

        expect(sql).toContain('ALTER COLUMN [col] nvarchar(64) NULL');
    });

    it('SQLite：重建表时带上长度与可空', () => {
        const dialect = getDbDialect('sqlite');
        const old = field({ type: 'varchar', length: 32, nullable: false });
        const now = field({ type: 'varchar', length: 64, nullable: true });
        const sql = dialect.getModifyColumnSql(ctx('sqlite', [old], [now]), 't_probe', upd([now]));

        expect(sql).toContain('"col" varchar(64) NULL');
    });

    it('ClickHouse：可空用 Nullable(T) 表达，绝不写 NOT NULL', () => {
        const dialect = getDbDialect('clickhouse');
        const nullableCol = field({ type: 'String', nullable: true });
        const notNullCol = field({ name: 'col2', type: 'String', nullable: false });
        const sql = dialect.getModifyColumnSql(ctx('clickhouse', [nullableCol], [nullableCol, notNullCol]), 't_probe', upd([nullableCol, notNullCol]));

        expect(sql).toContain('MODIFY COLUMN `col` Nullable(String)');
        expect(sql).toContain('MODIFY COLUMN `col2` String');
        expect(sql).not.toContain('NOT NULL');
    });

    it('MySQL：整列 MODIFY 同时写出长度与可空', () => {
        const dialect = getDbDialect('mysql');
        const old = field({ type: 'varchar', length: 32, nullable: false });
        const now = field({ type: 'varchar', length: 64, nullable: true });
        const sql = dialect.getModifyColumnSql(ctx('mysql', [old], [now]), 't_probe', upd([now]));

        expect(sql).toContain('MODIFY COLUMN `col` varchar(64) NULL');
    });
});

describe('主键变更（跨方言）', () => {
    it('MySQL：勾选主键生成 DROP + ADD PRIMARY KEY，与列 MODIFY 同批', () => {
        const dialect = getDbDialect('mysql');
        const idCol = field({ name: 'id', isPrimaryKey: true });
        const nameCol = field({ name: 'name' });
        const newName = { ...nameCol, isPrimaryKey: true };
        const sql = dialect.getModifyColumnSql(ctx('mysql', [idCol, nameCol], [idCol, newName]), 't_probe', upd([newName]));

        expect(sql).toContain('DROP PRIMARY KEY');
        expect(sql).toContain('ADD PRIMARY KEY (`id`,`name`)');
    });

    it('Oracle：复合主键要带上全部成员，并用 ADD CONSTRAINT 语法', () => {
        const dialect = getDbDialect('oracle');
        const a = field({ name: 'a', type: 'VARCHAR2', isPrimaryKey: true });
        const b = field({ name: 'b', type: 'VARCHAR2' });
        const newB = { ...b, isPrimaryKey: true };
        const sql = dialect.getModifyColumnSql(ctx('oracle', [a, b], [a, newB]), 't_probe', upd([]));

        expect(sql).toContain('DROP PRIMARY KEY');
        expect(sql).toContain('ADD CONSTRAINT "PK_t_probe" PRIMARY KEY ("a","b")');
        expect(sql).not.toContain('ADD PRIMARY KEY (');
    });

    it('DM：复合主键同样带全部成员', () => {
        const dialect = getDbDialect('dm');
        const a = field({ name: 'a', type: 'VARCHAR', isPrimaryKey: true });
        const b = field({ name: 'b', type: 'VARCHAR' });
        const newB = { ...b, isPrimaryKey: true };
        const sql = dialect.getModifyColumnSql(ctx('dm', [a, b], [a, newB]), 't_probe', upd([]));

        expect(sql).toContain('DROP PRIMARY KEY');
        expect(sql).toContain('ADD PRIMARY KEY ("a","b")');
    });

    it('只改列名不触发主键重建（成员没变，只是名字变了）', () => {
        const dialect = getDbDialect('mysql');
        const oldCol = field({ name: 'id', isPrimaryKey: true });
        const renamed = { ...oldCol, name: 'id2', oldName: 'id' };
        const sql = dialect.getModifyColumnSql(ctx('mysql', [oldCol], [renamed]), 't_probe', upd([renamed]));

        expect(sql).toContain('CHANGE COLUMN `id` `id2`');
        expect(sql).not.toContain('PRIMARY KEY');
    });
});

describe('方言能力位：主键是否可编辑', () => {
    it('关系型方言全部可编辑（pg/mssql 靠 t-keys 下发的约束名做 DROP CONSTRAINT）', () => {
        for (const dbType of ['mysql', 'oracle', 'dm', 'sqlite', 'postgres', 'mssql']) {
            expect(getDbDialect(dbType).getCapabilities().supportsPrimaryKeyEdit, dbType).toBe(true);
        }
    });

    it('ClickHouse 禁用（主键是建表 ORDER BY 表达式，不是可勾选的列属性）', () => {
        expect(getDbDialect('clickhouse').getCapabilities().supportsPrimaryKeyEdit).toBe(false);
    });
});

describe('pg / mssql 主键撤销与重建（依赖约束名）', () => {
    function pkCtx(dbType: string, oldFields: ColumnDefinition[], res: ColumnDefinition[], primaryKeyName?: string): TableEditContext {
        const ctx = {
            tableName: 't_probe',
            tableComment: '',
            oldTableName: 't_probe',
            oldTableComment: '',
            db: dbType === 'mssql' ? 'public' : 'e2e/public',
            fields: { res, oldFields },
            indexes: { res: [], oldIndexes: [], columns: [] },
        } as TableEditContext;
        if (primaryKeyName) {
            ctx.primaryKeyName = primaryKeyName;
        }
        return ctx;
    }

    it('PostgreSQL：按约束名 DROP CONSTRAINT，重建沿用原名', () => {
        const dialect = getDbDialect('postgres');
        const idCol = field({ name: 'id', isPrimaryKey: true });
        const nameCol = field({ name: 'name' });
        const newName = { ...nameCol, isPrimaryKey: true };
        const sql = dialect.getModifyColumnSql(pkCtx('postgres', [idCol, nameCol], [idCol, newName], 't_probe_pkey'), 't_probe', upd([]));

        expect(sql).toContain('ALTER TABLE "public"."t_probe" DROP CONSTRAINT "t_probe_pkey"');
        expect(sql).toContain('ALTER TABLE "public"."t_probe" ADD CONSTRAINT "t_probe_pkey" PRIMARY KEY ("id","name")');
    });

    it('PostgreSQL：没有旧主键时只 ADD（不猜约束名）', () => {
        const dialect = getDbDialect('postgres');
        const nameCol = field({ name: 'name' });
        const newName = { ...nameCol, isPrimaryKey: true };
        const sql = dialect.getModifyColumnSql(pkCtx('postgres', [nameCol], [newName]), 't_probe', upd([]));

        expect(sql).not.toContain('DROP CONSTRAINT');
        expect(sql).toContain('ADD PRIMARY KEY ("name")');
    });

    it('PostgreSQL：取消主键只 DROP，不再 ADD', () => {
        const dialect = getDbDialect('postgres');
        const idCol = field({ name: 'id', isPrimaryKey: true });
        const notPk = { ...idCol, isPrimaryKey: false };
        const sql = dialect.getModifyColumnSql(pkCtx('postgres', [idCol], [notPk], 't_probe_pkey'), 't_probe', upd([]));

        expect(sql).toContain('DROP CONSTRAINT "t_probe_pkey"');
        expect(sql).not.toContain('ADD CONSTRAINT');
    });

    it('MSSQL：按约束名 DROP CONSTRAINT，再以同名 ADD CONSTRAINT', () => {
        const dialect = getDbDialect('mssql');
        const idCol = field({ name: 'id', isPrimaryKey: true });
        const nameCol = field({ name: 'name', type: 'nvarchar', length: 32 });
        const newName = { ...nameCol, isPrimaryKey: true };
        const sql = dialect.getModifyColumnSql(pkCtx('mssql', [idCol, nameCol], [idCol, newName], 'PK_t_probe'), 't_probe', upd([]));

        expect(sql).toContain('ALTER TABLE [public].[t_probe] DROP CONSTRAINT [PK_t_probe]');
        expect(sql).toContain('ALTER TABLE [public].[t_probe] ADD CONSTRAINT [PK_t_probe] PRIMARY KEY ([id],[name])');
    });
});

describe('默认值表达式（跨方言）', () => {
    it('PG：字面量加引号、表达式不加引号', () => {
        const dialect = getDbDialect('postgres');
        const literal = field({ name: 'note', type: 'text', value: 'abc' });
        const expr = field({ name: 'uid', type: 'uuid', value: 'gen_random_uuid()', valueIsExpression: true });
        const sql = dialect.getModifyColumnSql(ctx('postgres', [literal, expr], [literal, expr]), 't_probe', upd([literal, expr]));

        expect(sql).toContain("ALTER COLUMN \"note\" SET DEFAULT 'abc'");
        expect(sql).toContain('ALTER COLUMN "uid" SET DEFAULT gen_random_uuid()');
        expect(sql).not.toContain("DEFAULT 'gen_random_uuid()'");
    });
});
