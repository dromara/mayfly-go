import { describe, it, expect } from 'vitest';

import { getDbDialect, getDbDialectMap, getDialectCapabilities, DbType } from '../index';
import { limitCommaPageSnippet, limitOffsetPageSnippet, offsetFetchPageSnippet, rownumPageSnippet } from '../shared/snippets';
import type { DbDialect } from '../types';

/**
 * 方言注册表契约测试
 *
 * 这里刻意**不 mock monaco 语言定义**：dialect 层已与 monaco 解耦（联想词改为
 * getEditorCompletions() 内按需 await import），若哪天有人把静态 import 加回方言文件，
 * 本文件会因为要把整份 monaco 拉进测试环境而显著变慢或直接失败，从而暴露回归。
 */

/** 全部已注册方言：基础方言取自注册表，版本方言（oracle11）按 dbType+version 取 */
function allDialects(): { label: string; dialect: DbDialect }[] {
    const list = [...getDbDialectMap().entries()].map(([type, dialect]) => ({ label: type, dialect }));
    list.push({ label: `${DbType.oracle}11`, dialect: getDbDialect(DbType.oracle, '11') });
    return list;
}

describe('方言注册表', () => {
    it('所有 dbType 均已注册方言，且注册表可枚举', () => {
        const map = getDbDialectMap();
        expect(map.size).toBeGreaterThan(10);
        for (const [type, dialect] of map) {
            expect(dialect, `${type} 注册的方言实例为空`).toBeTruthy();
            expect(getDbDialect(type), type).toBe(dialect);
        }
    });

    it('未识别的 dbType 回退到 MySQL 语法，不返回 undefined', () => {
        expect(getDbDialect('some-unknown-db')).toBe(getDbDialect(DbType.mysql));
        expect(getDbDialect('')).toBe(getDbDialect(DbType.mysql));
    });
});

describe('getInfo()：纯元数据，不依赖 monaco', () => {
    it.each(allDialects().map((d) => [d.label, d.dialect] as const))('%s 同步返回完整元信息', (_label, dialect) => {
        const info = dialect.getInfo();
        expect(info.name).toBeTruthy();
        expect(info.icon).toMatch(/^icon db\//);
        expect(typeof info.defaultPort).toBe('number');
        expect(info.formatSqlDialect).toBeTruthy();
        expect(info.columnTypes.length).toBeGreaterThan(0);
    });

    it.each(allDialects().map((d) => [d.label, d.dialect] as const))('%s 的 getInfo 不再携带 editorCompletions', (_label, dialect) => {
        // 联想词已迁至异步的 getEditorCompletions()，留在 getInfo 上会把 monaco 拽回静态依赖图
        expect('editorCompletions' in dialect.getInfo()).toBe(false);
    });

    it('元信息按方言缓存：多次调用返回同一对象', () => {
        for (const { label, dialect } of allDialects()) {
            expect(dialect.getInfo(), `${label} 的 getInfo 未缓存`).toBe(dialect.getInfo());
        }
    });
});

describe('getEditorCompletions()：异步按需加载', () => {
    it.each(allDialects().map((d) => [d.label, d.dialect] as const))('%s 返回四类联想词', async (_label, dialect) => {
        const completions = await dialect.getEditorCompletions();
        expect(Array.isArray(completions.keywords)).toBe(true);
        expect(Array.isArray(completions.operators)).toBe(true);
        expect(Array.isArray(completions.functions)).toBe(true);
        expect(Array.isArray(completions.variables)).toBe(true);
        expect(completions.keywords.length, '关键字联想为空，语言定义可能未正确加载').toBeGreaterThan(0);
        // 每项建议至少要有 label，否则 monaco 渲染补全列表时会拿到 undefined
        expect(completions.keywords.every((k) => !!k.label)).toBe(true);
    });

    it('MySQL 的联想词确实来自 monaco 语言定义并叠加了自定义函数', async () => {
        const { keywords, functions } = await getDbDialect(DbType.mysql).getEditorCompletions();
        expect(keywords.length).toBeGreaterThan(100);
        // 自定义重写函数（带参数提示）应覆盖同名内置函数，而不是与其并存
        const concat = functions.filter((f) => f.label === 'CONCAT');
        expect(concat).toHaveLength(1);
        expect(concat[0].insertText).toBe('CONCAT(str1,str2,...)');
    });

    it('结果按方言缓存：二次调用返回同一对象，不重复构建', async () => {
        for (const { label, dialect } of allDialects()) {
            const first = await dialect.getEditorCompletions();
            expect(await dialect.getEditorCompletions(), `${label} 的联想词未缓存`).toBe(first);
        }
    });

    it('派生方言复用基类联想词（MariaDB 与 MySQL 一致）', async () => {
        const mysql = await getDbDialect(DbType.mysql).getEditorCompletions();
        const mariadb = await getDbDialect(DbType.mariadb).getEditorCompletions();
        expect(mariadb).toBe(mysql);
    });
});

describe('getDialectCapabilities()：实例级缓存', () => {
    it('同一方言多次读取命中缓存，返回同一对象', () => {
        const dialect = getDbDialect(DbType.postgresql);
        expect(getDialectCapabilities(dialect)).toBe(getDialectCapabilities(dialect));
    });

    it('缓存内容与方言自身声明一致（缓存不改变语义）', () => {
        for (const { label, dialect } of allDialects()) {
            // 方言的 getCapabilities() 每次合并缺省值新建对象，这正是需要缓存的原因
            expect(dialect.getCapabilities(), label).not.toBe(getDialectCapabilities(dialect));
            expect(getDialectCapabilities(dialect), label).toEqual(dialect.getCapabilities());
        }
    });

    it.each(allDialects().map((d) => [d.label, d.dialect] as const))('%s 的能力位均已合并缺省值', (_label, dialect) => {
        const cap = getDialectCapabilities(dialect);
        // 新增方言若漏用 defineCapabilities()，这些字段会是 undefined，导致调用方分支静默走错
        expect(typeof cap.supportsSchema, 'supportsSchema').toBe('boolean');
        expect(typeof cap.supportsTableComment, 'supportsTableComment').toBe('boolean');
        expect(typeof cap.supportsAutoIncrement, 'supportsAutoIncrement').toBe('boolean');
        expect(typeof cap.canEditAutoIncrementOnCreate, 'canEditAutoIncrementOnCreate').toBe('boolean');
        expect(typeof cap.canEditAutoIncrementOnEdit, 'canEditAutoIncrementOnEdit').toBe('boolean');
        expect(cap.defaultIndexType).toBeTruthy();
        expect(cap.quotePairs.length).toBeGreaterThan(0);
        expect(cap.sqlSplitOptions).toBeTruthy();
    });

    it('能力位与实际行为自洽：不支持自增的方言不可编辑自增列', () => {
        // 表编辑器据这两个能力位决定是否禁用「自增」勾选框，两者矛盾会让 UI 与生成的 DDL 打架
        for (const { label, dialect } of allDialects()) {
            const cap = getDialectCapabilities(dialect);
            if (!cap.supportsAutoIncrement) {
                expect(cap.canEditAutoIncrementOnCreate, `${label} 不支持自增却允许建表时编辑`).toBe(false);
                expect(cap.canEditAutoIncrementOnEdit, `${label} 不支持自增却允许编辑时修改`).toBe(false);
            }
        }
    });
});

describe('getPageSnippet()：分页写法由方言自描述', () => {
    it.each(allDialects().map((d) => [d.label, d.dialect] as const))('%s 返回结构完整的分页模板', (_label, dialect) => {
        const snippet = dialect.getPageSnippet();
        // 补全层直接把它渲染进 monaco 建议列表，任一字段为空都会显示成 undefined
        expect(snippet.label).toBeTruthy();
        expect(snippet.description).toBeTruthy();
        expect(snippet.body).toBeTruthy();
        // snippet 正文必须带占位符与终止光标位，否则 Tab 跳转与光标落点失效
        expect(snippet.body, 'body 缺少 ${n:...} 占位符').toContain('${');
        expect(snippet.body, 'body 缺少 $0 终止光标位').toContain('$0');
    });

    it('模板复用方言层预设，而非各处散落的一次性字面量', () => {
        // 预设是模块级常量、方言直接引用而非拷贝，故可用引用相等断言「每个方言都显式挑选过预设」。
        // 新增方言若自行拼字面量，会被本用例拦下并要求沉淀到 shared/snippets.ts 供同族复用。
        const presets = [limitCommaPageSnippet, limitOffsetPageSnippet, offsetFetchPageSnippet, rownumPageSnippet];
        for (const { label, dialect } of allDialects()) {
            expect(presets, `${label} 的分页模板未复用方言层预设`).toContain(dialect.getPageSnippet());
        }
    });

    it('派生方言继承基类分页模板，无需重复声明', () => {
        const mysql = getDbDialect(DbType.mysql).getPageSnippet();
        const postgres = getDbDialect(DbType.postgresql).getPageSnippet();
        const oracle = getDbDialect(DbType.oracle).getPageSnippet();

        expect(getDbDialect(DbType.mariadb).getPageSnippet(), 'mariadb 应继承 mysql').toBe(mysql);
        expect(getDbDialect(DbType.gauss).getPageSnippet(), 'gauss 应继承 postgres').toBe(postgres);
        expect(getDbDialect(DbType.kingbaseEs).getPageSnippet(), 'kingbaseEs 应继承 postgres').toBe(postgres);
        expect(getDbDialect(DbType.vastbase).getPageSnippet(), 'vastbase 应继承 postgres').toBe(postgres);
        // oracle11 为版本特化方言，按 dbType + version 取
        expect(getDbDialect(DbType.oracle, '11').getPageSnippet(), 'oracle11 应继承 oracle').toBe(oracle);
    });

    it('四个语法族的模板互不相同（避免整族退化为同一写法）', () => {
        // LIMIT 逗号形（mysql 系）/ LIMIT-OFFSET 形（pg 系）/ OFFSET-FETCH 形（mssql）/ ROWNUM 形（oracle 系）
        const used = new Set(allDialects().map(({ dialect }) => dialect.getPageSnippet()));
        expect(used.size).toBe(4);
        expect(getDbDialect(DbType.mysql).getPageSnippet()).toBe(limitCommaPageSnippet);
        expect(getDbDialect(DbType.postgresql).getPageSnippet()).toBe(limitOffsetPageSnippet);
        expect(getDbDialect(DbType.mssql).getPageSnippet()).toBe(offsetFetchPageSnippet);
        expect(getDbDialect(DbType.oracle).getPageSnippet()).toBe(rownumPageSnippet);
    });
});
