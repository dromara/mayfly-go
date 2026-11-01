import { commonCustomKeywords, DataType, DuplicateStrategy } from './types';
import type {
    DbDialect,
    DialectCapabilities,
    DialectInfo,
    EditorCompletion,
    EditorCompletionItem,
    IndexDefinition,
    ColumnDefinition,
    SqlColumnType,
    SqlSnippetTemplate,
} from './types';
import { createDefaultColumns, defaultColumnConfigs } from './shared/defaultColumns';
import { appendLimitSql, buildSchemaTable, extractSchema, getDefaultDataType, QuoteEscape, wrapValueDefault } from './shared/utils';
import { defineCapabilities, pgSplitOptions } from './shared/capabilities';
import { limitOffsetPageSnippet } from './shared/snippets';
import { DbType } from './dbType';
import { registerDbDialect } from './registry';
import { SchemaDialectBase } from './shared/schemaDialectBase';
import { diffPrimaryKey } from './shared/primaryKey';
import type { TableEditContext, TableInfoEditContext, ChangeDiff } from './types';

/**
 * 原生 PostgreSQL 的类型清单（官方 datatype 章节的标准类型）。
 *
 * 与 GAUSS_TYPE_LIST 分开是必须的：GaussDB 有 largeserial / nchar / clob 这类自有类型，
 * 而原生 PG 没有；反过来原生 PG 的 json / jsonb / bit / varbit / xml / float4 / float8
 * 在那边也没有。两者共用一个清单时，原生 PG 的下拉里会混入用不了的类型，
 * 常用类型反而选不到。
 */
export const PG_TYPE_LIST: SqlColumnType[] = [
    // 数值 - 整数
    { udtName: 'smallint', dataType: 'smallint', desc: '2 字节有符号整数，别名 int2', space: '2字节' },
    { udtName: 'integer', dataType: 'integer', desc: '4 字节有符号整数，别名 int / int4', space: '4字节' },
    { udtName: 'bigint', dataType: 'bigint', desc: '8 字节有符号整数，别名 int8', space: '8字节' },
    // 数值 - 任意精度
    { udtName: 'numeric', dataType: 'numeric', desc: '任意精度定点数 NUMERIC(p,s)', space: '变长' },
    { udtName: 'decimal', dataType: 'decimal', desc: '与 numeric 等价', space: '变长' },
    // 数值 - 浮点
    { udtName: 'real', dataType: 'real', desc: '4 字节单精度浮点数，别名 float4', space: '4字节' },
    { udtName: 'double precision', dataType: 'double precision', desc: '8 字节双精度浮点数，别名 float8', space: '8字节' },
    // 数值 - 序列（自增由类型承担，PG 没有独立的 AUTO_INCREMENT 属性）
    { udtName: 'smallserial', dataType: 'smallserial', desc: '自增 smallint，别名 serial2', space: '2字节' },
    { udtName: 'serial', dataType: 'serial', desc: '自增 integer，别名 serial4', space: '4字节' },
    { udtName: 'bigserial', dataType: 'bigserial', desc: '自增 bigint，别名 serial8', space: '8字节' },
    { udtName: 'money', dataType: 'money', desc: '货币金额', space: '8字节' },

    /*
     * 内部类型名（pg_type.typname）：后端元数据对 serial/整数/浮点往往回读成 int2/int4/int8/float4/float8，
     * 清单里缺这些名字时，选定值就成了「下拉里没有的自由文本」，用户重选一次规范名还会把语义换掉。
     * 它们的 DDL 等价于右侧的规范名，desc 里点明别名以便界面显示「规范名，别名: 内部名」。
     */
    { udtName: 'int2', dataType: 'smallint', desc: '内部名 int2，等价 smallint', space: '2字节' },
    { udtName: 'int4', dataType: 'integer', desc: '内部名 int4，等价 integer', space: '4字节' },
    { udtName: 'int8', dataType: 'bigint', desc: '内部名 int8，等价 bigint', space: '8字节' },
    { udtName: 'float4', dataType: 'real', desc: '内部名 float4，等价 real', space: '4字节' },
    { udtName: 'float8', dataType: 'double precision', desc: '内部名 float8，等价 double precision', space: '8字节' },
    { udtName: 'timestamptz', dataType: 'timestamp with time zone', desc: '内部名 timestamptz，等价 timestamp with time zone', space: '8字节' },
    { udtName: 'timetz', dataType: 'time with time zone', desc: '内部名 timetz，等价 time with time zone', space: '12字节' },
    { udtName: 'varbit', dataType: 'bit varying', desc: '内部名 varbit，等价 bit varying', space: '变长' },
    { udtName: 'bpchar', dataType: 'character', desc: '内部名 bpchar，等价 character', space: '变长' },

    // 字符
    { udtName: 'character varying', dataType: 'character varying', desc: '变长字符串，别名 varchar(n)', space: '变长' },
    { udtName: 'varchar', dataType: 'varchar', desc: '变长字符串', space: '变长' },
    { udtName: 'character', dataType: 'character', desc: '定长字符串，不足补空格，别名 char(n)', space: '变长' },
    { udtName: 'char', dataType: 'char', desc: '定长字符串', space: '变长' },
    { udtName: 'text', dataType: 'text', desc: '变长字符串，长度上限约 1GB', space: '变长' },

    // 二进制
    { udtName: 'bytea', dataType: 'bytea', desc: '变长二进制串', space: '变长' },

    // 日期时间
    { udtName: 'date', dataType: 'date', desc: '日期', space: '4字节' },
    { udtName: 'time', dataType: 'time', desc: 'TIME [(p)] 一日内时间，p 为小数秒精度 0~6', space: '8字节' },
    { udtName: 'time with time zone', dataType: 'time with time zone', desc: '带时区的时间', space: '12字节' },
    { udtName: 'timestamp', dataType: 'timestamp', desc: 'TIMESTAMP [(p)] 日期和时间，p 为 0~6', space: '8字节' },
    { udtName: 'timestamp with time zone', dataType: 'timestamp with time zone', desc: '带时区的日期和时间，别名 timestamptz', space: '8字节' },
    { udtName: 'interval', dataType: 'interval', desc: '时间间隔', space: '16字节' },

    // 布尔
    { udtName: 'boolean', dataType: 'boolean', desc: '布尔类型，别名 bool', space: '1字节' },
    { udtName: 'bool', dataType: 'bool', desc: '布尔类型', space: '1字节' },

    // 位串
    { udtName: 'bit', dataType: 'bit', desc: '定长位串 BIT(n)', space: '变长' },
    { udtName: 'bit varying', dataType: 'bit varying', desc: '变长位串，别名 varbit(n)', space: '变长' },

    // 网络地址
    { udtName: 'inet', dataType: 'inet', desc: 'IPv4/IPv6 主机及可选子网', space: '7/19字节' },
    { udtName: 'cidr', dataType: 'cidr', desc: 'IPv4/IPv6 网络', space: '7/19字节' },
    { udtName: 'macaddr', dataType: 'macaddr', desc: 'MAC 地址', space: '6字节' },
    { udtName: 'macaddr8', dataType: 'macaddr8', desc: 'EUI-64 格式 MAC 地址', space: '8字节' },

    // 文档类型
    { udtName: 'json', dataType: 'json', desc: 'JSON 文本（保留原始格式）', space: '变长' },
    { udtName: 'jsonb', dataType: 'jsonb', desc: 'JSON 二进制（可索引，推荐）', space: '变长' },
    { udtName: 'xml', dataType: 'xml', desc: 'XML 文档', space: '变长' },

    // 标识
    { udtName: 'uuid', dataType: 'uuid', desc: '通用唯一标识符（常作主键）', space: '16字节' },

    // 全文检索
    { udtName: 'tsvector', dataType: 'tsvector', desc: '全文检索文档', space: '变长' },
    { udtName: 'tsquery', dataType: 'tsquery', desc: '全文检索查询', space: '变长' },

    // 几何
    { udtName: 'point', dataType: 'point', desc: '平面中的点，如 (x,y)', space: '16字节' },
    { udtName: 'line', dataType: 'line', desc: '无限长直线', space: '24字节' },
    { udtName: 'lseg', dataType: 'lseg', desc: '有限线段，如 ((x1,y1),(x2,y2))', space: '32字节' },
    { udtName: 'box', dataType: 'box', desc: '矩形', space: '32字节' },
    { udtName: 'path', dataType: 'path', desc: '路径（闭合或开放）', space: '16+16n字节' },
    { udtName: 'polygon', dataType: 'polygon', desc: '多边形', space: '40+16n字节' },
    { udtName: 'circle', dataType: 'circle', desc: '圆，如 <(x,y),r>', space: '24字节' },
];

/**
 * GaussDB（以及以高斯类型体系为准的兼容库）的类型清单。
 *
 * 注释中的「PG兼容模式」说明来自高斯文档：其类型集是 PG 的超集，多出 largeserial / nchar / clob。
 */
export const GAUSS_TYPE_LIST: SqlColumnType[] = [
    // 数值 - 整数型
    { udtName: 'int1', dataType: 'tinyint', desc: '微整数，别名为INT1', space: '1字节', range: '0 ~ +255' },
    { udtName: 'int2', dataType: 'smallint', desc: '小范围整数，别名为INT2。', space: '2字节', range: '-32,768 ~ +32,767' },
    { udtName: 'int4', dataType: 'integer', desc: '常用的整数，别名为INT4。', space: '4字节', range: '-2,147,483,648 ~ +2,147,483,647' },
    { udtName: 'int8', dataType: 'bigint', desc: '大范围的整数，别名为INT8。', space: '8字节', range: '很大' },

    // 数值 - 任意精度型
    {
        udtName: 'numeric',
        dataType: 'numeric',
        desc: '精度(总位数)取值范围为[1,1000]，标度(小数位数)取值范围为[0,精度]。',
        space: '每四位（十进制位）占用两个字节，然后在整个数据上加上八个字节的额外开销',
        range: '未指定精度的情况下，小数点前最大131,072位，小数点后最大16,383位',
    },
    // 数值 - 任意精度型
    { udtName: 'decimal', dataType: 'decimal', desc: '等同于number类型', space: '等同于number类型' },

    // 数值 - 序列整型
    { udtName: 'smallserial', dataType: 'smallserial', desc: '二字节序列整型。', space: '2字节', range: '-32,768 ~ +32,767' },
    { udtName: 'serial', dataType: 'serial', desc: '四字节序列整型。', space: '4字节', range: '-2,147,483,648 ~ +2,147,483,647' },
    { udtName: 'bigserial', dataType: 'bigserial', desc: '八字节序列整型', space: '8字节', range: '-9,223,372,036,854,775,808 ~ +9,223,372,036,854,775,807' },
    {
        udtName: 'largeserial',
        dataType: 'largeserial',
        desc: '默认插入十六字节序列整型，实际数值类型和numeric相同',
        space: '变长类型，每四位（十进制位）占用两个字节，然后在整个数据上加上八个字节的额外开销。',
        range: '小数点前最大131,072位，小数点后最大16,383位。',
    },

    // 数值 - 浮点类型（不常用 就不列出来了）

    // 货币类型
    { udtName: 'money', dataType: 'money', desc: '货币金额', space: '8字节', range: '-92233720368547758.08 ~ +92233720368547758.07' },

    // 布尔类型
    { udtName: 'bool', dataType: 'bool', desc: '布尔类型', space: '1字节', range: 'true：真 , false：假 , null：未知（unknown）' },

    // 字符类型
    { udtName: 'char', dataType: 'char', desc: '定长字符串，不足补空格。n是指字节长度，如不带精度n，默认精度为1。', space: '最大为10MB' },
    { udtName: 'character', dataType: 'character', desc: '定长字符串，不足补空格。n是指字节长度，如不带精度n，默认精度为1。', space: '最大为10MB' },
    { udtName: 'nchar', dataType: 'nchar', desc: '定长字符串，不足补空格。n是指字节长度，如不带精度n，默认精度为1。', space: '最大为10MB' },
    { udtName: 'varchar', dataType: 'varchar', desc: '变长字符串。PG兼容模式下，n是字符长度。其他兼容模式下，n是指字节长度。', space: '最大为10MB。' },
    { udtName: 'text', dataType: 'text', desc: '变长字符串。', space: '最大稍微小于1GB-1。' },
    { udtName: 'clob', dataType: 'clob', desc: '文本大对象。是TEXT类型的别名。', space: '最大稍微小于32TB-1。' },

    //特殊字符类型  用的很少，先屏蔽了
    // { udtName: 'name', dataType: 'name', desc: '用于对象名的内部类型。', space: '64字节。' },
    // { udtName: '"char"', dataType: '"char"', desc: '单字节内部类型。', space: '1字节。' },

    // 二进制类型
    { udtName: 'bytea', dataType: 'bytea', desc: '变长的二进制字符串', space: '4字节加上实际的二进制字符串。最大为1GB减去8203字节（即1073733621字节）。' },

    // UUID 类型（常作主键：下拉里没这一项时只能手输，与后端类型引擎识别 uuid 的能力不对齐）
    { udtName: 'uuid', dataType: 'uuid', desc: '通用唯一标识符（UUID）', space: '16字节' },

    // 日期/时间类型
    { udtName: 'date', dataType: 'date', desc: '日期', space: '4字节' },
    { udtName: 'time', dataType: 'time', desc: 'TIME [(p)] 只用于一日内时间,p表示小数点后的精度，取值范围为0~6。', space: '8-12字节' },
    { udtName: 'timestamp', dataType: 'timestamp', desc: 'TIMESTAMP[(p)]日期和时间,p表示小数点后的精度，取值范围为0~6', space: '8字节' },
    // 带时区的时间戳用的少，先屏蔽了
    //{ udtName: 'TIMESTAMPTZ', dataType: 'TIMESTAMP WITH TIME ZONE', desc: '带时区的时间戳', space: '8字节' },
    {
        udtName: 'interval',
        dataType: 'interval',
        desc: '时间间隔', // 可以跟参数：YEAR，MONTH，DAY，HOUR，MINUTE，SECOND，DAY TO HOUR，DAY TO MINUTE，DAY TO SECOND，HOUR TO MINUTE，HOUR TO SECOND，MINUTE TO SECOND
        space: '精度取值范围为0~6，且参数为SECOND，DAY TO SECOND，HOUR TO SECOND或MINUTE TO SECOND时，参数p才有效',
    },
    // 几何类型
    { udtName: 'point', dataType: 'point', desc: '平面中的点， 如:(x,y)', space: '16字节' },
    { udtName: 'lseg', dataType: 'lseg', desc: '（有限）线段， 如:((x1,y1),(x2,y2))', space: '32字节' },
    { udtName: 'box', dataType: 'box', desc: '矩形， 如:((x1,y1),(x2,y2))', space: '32字节' },
    // 闭合路径与开放路径是同一个类型的两种书写形态，不能列两项（下拉会出现同名重复项）
    { udtName: 'path', dataType: 'path', desc: '路径（闭合 ((x1,y1),...) 或开放 [(x1,y1),...]）', space: '16+16n字节' },
    { udtName: 'polygon', dataType: 'polygon', desc: '多边形（与闭合路径相似）， 如:((x1,y1),...)', space: '40+16n字节' },
    { udtName: 'circle', dataType: 'circle', desc: '圆,如:<(x,y),r> （圆心和半径）', space: '24 字节' },

    // 网络地址类型
    { udtName: 'cidr', dataType: 'cidr', desc: 'IPv4网络', space: '7字节' },
    { udtName: 'inet', dataType: 'inet', desc: 'IPv4主机和网络', space: '7字节' },
    { udtName: 'macaddr', dataType: 'macaddr', desc: 'MAC地址', space: '6字节' },
];

const replaceFunctions: EditorCompletionItem[] = [];

let pgDialectInfo: DialectInfo;
let pgCompletions: EditorCompletion;

export class PostgresqlDialect extends SchemaDialectBase {
    /**
     * 列类型表达式（类型名 + 参数）。
     *
     * 带时区的时间类型必须在基础类型后插精度：`timestamp(3) with time zone`。
     * 直接按「类型名 + (n)」拼会得到 `timestamp with time zone(3)`——PG 语法错误。
     */
    private columnTypeExpression(cl: ColumnDefinition): string {
        const type = (cl.type || '').trim();
        const length = this.getTypeLengthSql(cl);
        const withTimeZone = /^(timestamp|time)\s+with\s+time\s+zone$/i.exec(type);
        if (withTimeZone && length) {
            return `${withTimeZone[1]}${length} with time zone`;
        }
        return `${type}${length}`;
    }

    /** 覆盖基类：类型片段改由 columnTypeExpression 生成（带时区类型需要特殊插值） */
    genColumnBasicSql(cl: ColumnDefinition): string {
        const defVal = this.getDefaultValueSql(cl);
        const parts = [
            this.quoteIdentifier(this.resolveColumnName(cl)),
            this.columnTypeExpression(cl),
            cl.nullable ? '' : 'NOT NULL',
            defVal,
        ];
        return parts.filter(Boolean).join(' ');
    }

    getCapabilities(): DialectCapabilities {
        return defineCapabilities({
            // pg 没有 DROP PRIMARY KEY：撤销主键靠 t-keys 下发的约束名做 DROP CONSTRAINT
            // 自增由 serial/identity 列类型表达而非独立列属性，故表编辑器不提供自增勾选
            canEditAutoIncrementOnCreate: false,
            canEditAutoIncrementOnEdit: false,
            // libpq 原生 sslmode 四档，证书由后端落临时文件下发
            tlsModes: ['require', 'verify-ca', 'verify-full'],
            sqlSplitOptions: pgSplitOptions,
        });
    }

    getInfo(): DialectInfo {
        if (pgDialectInfo) {
            return pgDialectInfo;
        }

        pgDialectInfo = {
            name: 'PostgreSQL',
            icon: 'icon db/postgres',
            defaultPort: 5432,
            formatSqlDialect: 'postgresql',
            columnTypes: PG_TYPE_LIST.sort((a, b) => a.udtName.localeCompare(b.udtName)),
        };
        return pgDialectInfo;
    }

    /** 编辑器联想词由 monaco 语言定义派生，按需加载并缓存（详见 types.ts 的 DialectInfo 注释） */
    async getEditorCompletions(): Promise<EditorCompletion> {
        if (pgCompletions) {
            return pgCompletions;
        }

        const { language: pgsqlLanguage } = await import('monaco-editor/languages/definitions/pgsql/pgsql.js');
        let { keywords, operators, builtinVariables, builtinFunctions } = pgsqlLanguage;
        let replaceFunctionNames = replaceFunctions.map((a) => a.label);
        let functions = builtinFunctions
            .filter((a: string) => replaceFunctionNames.indexOf(a) < 0)
            .map((a: string): EditorCompletionItem => ({ label: a, insertText: `${a}()`, description: 'func' }))
            .concat(replaceFunctions);
        let excludeKeywords = new Set(builtinFunctions.concat(replaceFunctionNames).concat(operators));

        pgCompletions = {
            keywords: keywords
                .filter((a: string) => !excludeKeywords.has(a)) // 移除已存在的operator、function
                .map((a: string): EditorCompletionItem => ({ label: a, description: 'keyword' }))
                .concat(commonCustomKeywords.map((a): EditorCompletionItem => ({ label: a, description: 'keyword' }))),
            operators: operators.map((a: string): EditorCompletionItem => ({ label: a, description: 'operator' })),
            functions,
            variables: builtinVariables.map((a: string): EditorCompletionItem => ({ label: a, description: 'var' })),
        };
        return pgCompletions;
    }

    getDefaultSelectSql(db: string, table: string, condition: string, orderBy: string, pageNum: number, limit: number) {
        return `SELECT * FROM ${this.quoteIdentifier(table)} ${condition ? 'WHERE ' + condition : ''} ${orderBy ? orderBy : ''} ${this.getPageSql(
            pageNum,
            limit
        )};`;
    }

    getPageSql(pageNum: number, limit: number) {
        return ` OFFSET ${(pageNum - 1) * limit} LIMIT ${limit};`;
    }

    getPreviewSql(sql: string, limit = 1): string {
        return appendLimitSql(sql, limit);
    }

    getPageSnippet(): SqlSnippetTemplate {
        return limitOffsetPageSnippet;
    }

    getDefaultColumns(): ColumnDefinition[] {
        return createDefaultColumns(defaultColumnConfigs.postgres);
    }

    // ==================== 默认值格式化钩子（覆写基类的配置方法） ====================

    /** PG 默认值需加引号的类型名（基类 matchType 大小写不敏感，故小写即可） */
    protected getDefaultValueQuotedTypeNames(): string[] {
        return ['char', 'time', 'date', 'text'];
    }

    /** PG 时间/日期列的默认值中不加引号的函数名（基类统一转小写比较） */
    protected getUnquotedDefaultFunctionNames(): string[] {
        return ['pg_systimestamp()', 'current_timestamp'];
    }

    // ==================== PG 特有的 DDL 方法（基类无法覆盖的差异） ====================

    getCreateTableSql(data: TableEditContext): string {
        let createSql = '';
        let tableCommentSql = '';
        let columCommentSql = '';

        // 创建表结构
        let pks = [] as string[];
        let fields: string[] = [];
        data.fields.res.forEach((item: ColumnDefinition) => {
            item.name && fields.push(this.genColumnBasicSql(item));
            if (item.isPrimaryKey) {
                pks.push(this.quoteIdentifier(item.name));
            }
            // 列注释
            if (item.comment) {
                columCommentSql += `COMMENT ON COLUMN ${this.quoteIdentifier(data.tableName)}.${this.quoteIdentifier(item.name)} IS '${QuoteEscape(item.comment)}';`;
            }
        });
        // 建表
        const pkClause = pks.length > 0 ? `,\n  PRIMARY KEY (${pks.join(',')})` : '';
        createSql = `CREATE TABLE ${this.quoteIdentifier(data.tableName)} (\n  ${fields.join(',\n  ')}${pkClause}\n);`;
        // 表注释
        if (data.tableComment) {
            tableCommentSql = `COMMENT ON TABLE ${this.quoteIdentifier(data.tableName)} IS '${QuoteEscape(data.tableComment)}';`;
        }

        return createSql + tableCommentSql + columCommentSql;
    }

    getCreateIndexSql(tableData: TableEditContext): string {
        let dbTable = buildSchemaTable(this.quoteIdentifier, tableData.db, tableData.tableName);
        let schema = extractSchema(tableData.db);
        let sql: string[] = [];
        tableData.indexes.res.forEach((a: IndexDefinition) => {
            let colArr = a.columnNames.map((c: string) => this.quoteIdentifier(c));
            sql.push(`CREATE ${a.unique ? 'UNIQUE ' : ''}INDEX ${this.quoteIdentifier(a.indexName)} ON ${dbTable} (${colArr.join(',')})`);
            if (a.indexComment) {
                const idxRef = schema ? `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(a.indexName)}` : this.quoteIdentifier(a.indexName);
                sql.push(`COMMENT ON INDEX ${idxRef} IS '${QuoteEscape(a.indexComment)}'`);
            }
        });
        return sql.join(';');
    }

    getDropTableSql(db: string, table: string): string {
        // 必须带 schema 限定，否则会落到连接 search_path 的首个 schema 而删错表
        return `DROP TABLE ${buildSchemaTable(this.quoteIdentifier, db, table)}`;
    }

    /**
     * 类型/长度/精度是否发生变化。
     *
     * pg 的 ALTER COLUMN TYPE 会重写整列、代价不低，故只在确有必要时下发；
     * 只改可空或只改默认值的编辑，不该顺带触发一次类型重写。
     */
    private typeChanged(old: ColumnDefinition, now: ColumnDefinition): boolean {
        return (
            old.type !== now.type || String(old.length ?? '') !== String(now.length ?? '') || String(old.numScale ?? '') !== String(now.numScale ?? '')
        );
    }

    getModifyColumnSql(tableData: TableEditContext, tableName: string, changeData: ChangeDiff<ColumnDefinition>): string {
        let dbTable = buildSchemaTable(this.quoteIdentifier, tableData.db, tableName);

        let modifySql = '';
        let dropSql = '';
        let renameSql = '';
        let commentSql = '';

        if (changeData.add.length > 0) {
            changeData.add.forEach((a) => {
                modifySql += `ALTER TABLE ${dbTable} ADD ${this.genColumnBasicSql(a)};`;
                if (a.comment) {
                    commentSql += `COMMENT ON COLUMN ${dbTable}.${this.quoteIdentifier(a.name)} IS '${QuoteEscape(a.comment)}';`;
                }
            });
        }

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                let cmtSql = `COMMENT ON COLUMN ${dbTable}.${this.quoteIdentifier(a.name)} IS '${QuoteEscape(a.comment)}';`;
                if (a.comment && a.oldName === a.name) {
                    commentSql += cmtSql;
                }
                // 修改了字段名
                if (a.oldName !== a.name) {
                    renameSql += `ALTER TABLE ${dbTable} RENAME COLUMN ${this.quoteIdentifier(a.oldName!)} TO ${this.quoteIdentifier(a.name)};`;
                    if (a.comment) {
                        commentSql += cmtSql;
                    }
                }
                // 如果有原名以原名为准
                let name = a.oldName && a.name !== a.oldName ? a.oldName : a.name;
                const old = tableData.fields.oldFields.find((f) => f.name === (a.oldName || a.name));
                // pg 的 NOT NULL 是列属性、TYPE 子句不会连带改写它，故三类变更必须分别下发，
                // 否则「只改可空/只改默认值」的编辑不会出现在 SQL 里，保存后界面显示成功但库没变
                if (!old || this.typeChanged(old, a)) {
                    // pg 的跨类型改写不会自动插入转换：text→int 这类会直接报「cannot be cast automatically」，
                    // 必须显式给 USING（服务端自己给出的提示就是 `USING col::目标类型`）
                    const typeSql = this.columnTypeExpression(a);
                    const colRef = this.quoteIdentifier(name);
                    modifySql += `ALTER TABLE ${dbTable} ALTER COLUMN ${colRef} TYPE ${typeSql} USING ${colRef}::${typeSql};`;
                }
                if (old && old.nullable !== a.nullable) {
                    modifySql += `ALTER TABLE ${dbTable} ALTER COLUMN ${this.quoteIdentifier(name)} ${a.nullable ? 'DROP' : 'SET'} NOT NULL;`;
                }
                if (a.value) {
                    modifySql += `ALTER TABLE ${dbTable} ALTER COLUMN ${this.quoteIdentifier(name)} SET${this.getDefaultValueSql(a)};`;
                } else if (old && old.value) {
                    modifySql += `ALTER TABLE ${dbTable} ALTER COLUMN ${this.quoteIdentifier(name)} DROP DEFAULT;`;
                }
            });
        }

        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                dropSql += `ALTER TABLE ${dbTable} DROP COLUMN ${this.quoteIdentifier(a.name)};`;
            });
        }

        // 主键是表级约束：pg 没有 DROP PRIMARY KEY，必须按约束名 DROP CONSTRAINT（名字来自元数据 t-keys）。
        // 命名缺失时退回 pg 的默认命名 `<表>_pkey`，但重建时优先沿用原名，避免同名约束被工具/脚本引用后失效
        const primaryKey = diffPrimaryKey(tableData.fields.oldFields, tableData.fields.res);
        const hasOldPrimaryKey = tableData.fields.oldFields.some((field) => field.isPrimaryKey);
        const constraintName = tableData.primaryKeyName || `${tableName}_pkey`;
        let pkSql = '';
        if (primaryKey.changed && hasOldPrimaryKey) {
            pkSql += `ALTER TABLE ${dbTable} DROP CONSTRAINT ${this.quoteIdentifier(constraintName)};`;
        }
        if (primaryKey.changed && primaryKey.columns.length > 0) {
            const cols = primaryKey.columns.map((name) => this.quoteIdentifier(name)).join(',');
            pkSql += hasOldPrimaryKey
                ? `ALTER TABLE ${dbTable} ADD CONSTRAINT ${this.quoteIdentifier(constraintName)} PRIMARY KEY (${cols});`
                : `ALTER TABLE ${dbTable} ADD PRIMARY KEY (${cols});`;
        }

        return modifySql + dropSql + renameSql + commentSql + pkSql;
    }

    getModifyIndexSql(tableData: TableEditContext, tableName: string, changeData: ChangeDiff<IndexDefinition>): string {
        let dbTable = buildSchemaTable(this.quoteIdentifier, tableData.db, tableName);
        let schema = extractSchema(tableData.db);

        let dropIndexNames: string[] = [];
        let addIndexes: IndexDefinition[] = [];

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                dropIndexNames.push(a.indexName);
                addIndexes.push(a);
            });
        }

        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                dropIndexNames.push(a.indexName);
            });
        }

        if (changeData.add.length > 0) {
            changeData.add.forEach((a) => {
                addIndexes.push(a);
            });
        }

        if (dropIndexNames.length > 0 || addIndexes.length > 0) {
            let sql: string[] = [];
            if (dropIndexNames.length > 0) {
                dropIndexNames.forEach((a) => {
                    const idxRef = schema ? `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(a)}` : this.quoteIdentifier(a);
                    sql.push(`DROP INDEX ${idxRef}`);
                });
            }

            if (addIndexes.length > 0) {
                addIndexes.forEach((a) => {
                    let colArr = a.columnNames.map((c: string) => this.quoteIdentifier(c));
                    sql.push(`CREATE ${a.unique ? 'UNIQUE ' : ''}INDEX ${this.quoteIdentifier(a.indexName)} ON ${dbTable} (${colArr.join(',')})`);
                    if (a.indexComment) {
                        const idxRef = schema ? `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(a.indexName)}` : this.quoteIdentifier(a.indexName);
                        sql.push(`COMMENT ON INDEX ${idxRef} IS '${QuoteEscape(a.indexComment)}'`);
                    }
                });
            }
            return sql.join(';');
        }
        return '';
    }

    getModifyTableInfoSql(tableData: TableInfoEditContext): string {
        let schema = extractSchema(tableData.db);

        let sql = '';
        if (tableData.tableComment != tableData.oldTableComment) {
            let dbTable = `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(tableData.oldTableName)}`;
            sql = `COMMENT ON TABLE ${dbTable} IS '${QuoteEscape(tableData.tableComment)}';`;
        }
        if (tableData.tableName != tableData.oldTableName) {
            let dbTable = `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(tableData.oldTableName)}`;
            sql += `ALTER TABLE ${dbTable} RENAME TO ${this.quoteIdentifier(tableData.tableName)};`;
        }
        return sql;
    }

    getDataType(columnType: string): DataType {
        return getDefaultDataType(columnType) as DataType;
    }

    wrapValue(columnType: string, value: unknown): string | number {
        return wrapValueDefault(columnType, value) as string | number;
    }

    getBatchInsertPreviewSql(tableName: string, fieldArr: string[], duplicateStrategy: DuplicateStrategy): string {
        // 构建占位符字符串 "($1, $2, $3 ...)"
        let placeholder = fieldArr.map((_, i) => `$${i + 1}`).join(',');
        let suffix = '';
        if (duplicateStrategy === DuplicateStrategy.IGNORE) {
            suffix = ' ON CONFLICT DO NOTHING';
        } else if (duplicateStrategy === DuplicateStrategy.REPLACE) {
            suffix = ' ON CONFLICT ON CONSTRAINT {your_constraint_name1} DO UPDATE SET ' + fieldArr.map((a) => `${a}=excluded.${a}`).join(',');
        }

        return `INSERT INTO ${tableName} (${fieldArr.join(',')}) VALUES (${placeholder}) \n ${suffix};`;
    }
}

registerDbDialect(DbType.postgres, new PostgresqlDialect());
