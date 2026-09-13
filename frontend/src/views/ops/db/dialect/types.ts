/**
 * 方言核心类型定义
 *
 * 独立文件以避免循环依赖：shared/defaultRows.ts 需要 RowDefinition，
 * 而 index.ts 需要导入 shared 模块。
 *
 * 本文件与 dbType.ts、registry.ts、shared/* 共同构成方言层的「零依赖内核」：
 * 各方言实现只依赖内核，不依赖 index.ts（聚合出口），故 index.ts 可自由
 * 通过 import.meta.glob 加载全部方言而不形成模块循环。
 *
 * 表编辑上下文（TableEditContext 等）也定义在此：它们是 DbDialect 各 DDL 方法的
 * 入参契约，且完全由本文件的 RowDefinition / IndexDefinition 构成，放在内核里
 * 才能保证方言层对外零依赖。
 */

export interface RowDefinition {
    name: string;
    oldName?: string;
    type: string;
    value: string;
    /** 长度 (模板 v-model.number 编辑后为 number, 初始/新增为 string) */
    length: string | number;
    /** 小数位数 (模板 v-model.number 编辑后为 number, 初始/新增为 string) */
    numScale: string | number;
    notNull: boolean;
    pri: boolean;
    auto_increment: boolean;
    remark: string;
}

export interface IndexDefinition {
    indexName: string;
    columnNames: string[];
    unique: boolean;
    indexType: string;
    indexComment?: string;
}

export interface sqlColumnType {
    udtName: string;
    dataType: string;
    desc: string;
    space: string;
    range?: string;
}

// ==================== 表编辑上下文（DDL 方法入参契约） ====================

/** 表编辑 SQL 生成上下文（方言 DDL 方法的标准入参，替代 Record<string, unknown>） */
export interface TableEditContext {
    /** 当前表名 */
    tableName: string;
    /** 当前表注释 */
    tableComment: string;
    /** 原始表名（重命名场景） */
    oldTableName: string;
    /** 原始表注释 */
    oldTableComment: string;
    /** 数据库名（含 schema，如 "db/schema"） */
    db: string;
    /** 列定义（res=当前，oldFields=原始，用于差异对比） */
    fields: {
        res: RowDefinition[];
        oldFields: RowDefinition[];
    };
    /** 索引定义（res=当前，oldIndexs=原始，columns=可选列下拉） */
    indexs: {
        res: IndexDefinition[];
        oldIndexs: IndexDefinition[];
        columns: { name: string; remark: string }[];
    };
}

/**
 * 表信息（名称/注释）编辑上下文。
 *
 * getModifyTableInfoSql 只依赖名称与注释的新旧值，不需要列/索引明细。窄化契约后，
 * 资源树「重命名表」这类只改名不改结构的场景可直接构造，无需伪造空的 fields/indexs；
 * TableEditContext 在结构上满足本类型，故表编辑器的调用点无需改动。
 */
export type TableInfoEditContext = Pick<TableEditContext, 'db' | 'tableName' | 'oldTableName' | 'tableComment' | 'oldTableComment'>;

/** 列/索引变更差异结果 */
export interface ChangeDiff<T> {
    del: T[];
    add: T[];
    upd: T[];
    changed: boolean;
}

export interface EditorCompletionItem {
    /** 用于显示 */
    label: string;
    /** 用于插入编辑器，可预置一些变量方便使用函数 */
    insertText?: string;
    /** 用于描述 */
    description: string;
}

export interface EditorCompletion {
    /** 关键字 */
    keywords: EditorCompletionItem[];
    /** 操作关键字 */
    operators: EditorCompletionItem[];
    /** 函数,包括内置函数和自定义函数 */
    functions: EditorCompletionItem[];
    /** 内置变量 */
    variables: EditorCompletionItem[];
}

/**
 * SQL 片段模板
 *
 * body 使用 monaco snippet 语法：`${1:placeholder}` 占位、Tab 逐项跳转、`$0` 为最终光标位。
 * 本接口只描述数据形状，不引入 monaco 依赖，故可安全置于方言内核。
 */
export interface SqlSnippetTemplate {
    /** 补全列表中显示的模板名 */
    label: string;
    /** 详情面板展示的模板说明 */
    description: string;
    /** 插入编辑器的 snippet 正文 */
    body: string;
}

// 定义一个数据类型的枚举，包含字符串、数字、日期、时间、日期时间
export enum DataType {
    String = 'string',
    Number = 'number',
    Date = 'date',
    Time = 'time',
    DateTime = 'datetime',
}

/** 列数据类型角标 */
export const ColumnTypeSubscript: Record<string, string> = {
    /** 字符串 */
    string: 'ab',
    /** 数字 */
    number: '12',
    /** 日期 */
    date: 'icon-clock',
    /** 时间 */
    time: 'icon-clock',
    /** 日期时间 */
    datetime: 'icon-clock',
};

/**
 * sql-formatter 的方言标识
 *
 * 只取类型（`typeof import()` 在类型位置不产生运行时依赖），方言层不会因此被链进 sql-formatter；
 * 取值的唯一合法域由 sql-formatter 自己给出，方言写错标识在编译期即报错，
 * 调用方也无需再 `as` 断言（传错值时 sql-formatter 会抛 ConfigError，弹窗直接打不开）。
 */
export type SqlFormatterLanguage = NonNullable<NonNullable<Parameters<typeof import('sql-formatter').format>[1]>['language']>;

/**
 * 数据库基础信息（纯元数据）
 *
 * 刻意不含编辑器联想词：联想词多由 monaco 语言定义派生，若挂在本接口上，
 * 方言模块就必须静态导入那些定义，取个图标的页面也会被拖入其体积。
 * 联想词改由 DbDialect.getEditorCompletions() 异步提供。
 */
export interface DialectInfo {
    /** 数据库类型 label */
    name: string;
    /** 图标 */
    icon: string;
    /** 默认端口 */
    defaultPort: number;
    /** 格式化 sql 的方言（sql-formatter 的方言标识） */
    formatSqlDialect: SqlFormatterLanguage;
    /** 列字段类型 */
    columnTypes: sqlColumnType[];
}

/**
 * 方言能力声明
 *
 * 每个方言通过 getCapabilities() 声明自身支持的特性，
 * 调用方无需通过 DbType 硬编码判断。
 * 参考 Prisma capability flags 设计。
 *
 * 本接口是「方言差异」的唯一事实源：SQL 语法能力、功能开关、表编辑器交互约束、
 * 词法特征（引用符/切割语义）全部在此声明。新增方言只需在自身文件内描述差异，
 * 无需改动任何调用方或其他模块的枚举表。
 *
 * 建议通过 shared/capabilities.ts 的 defineCapabilities() 声明，只写与缺省值不同的项。
 *
 * 收录门槛：仅当「有调用方需要据此分支」时才设为能力位。
 * IDENTITY / 序列 / ON UPDATE / 分页写法等只在方言自身 DDL 生成内部生效的差异，一律由对应方法
 * （getColumnTypes、getPageSql、getPreviewSql 等）命令式表达，不再重复声明为能力位——
 * 同一事实两处并存必然漂移（defaultIndexType 曾与 getDefaultIndex 各写一份而对不上）。
 */
export interface DialectCapabilities {
    // ==================== SQL 语法能力 ====================

    /** 是否支持 schema（如 PostgreSQL/Oracle/MSSQL） */
    supportsSchema: boolean;
    /** 是否支持表注释 */
    supportsTableComment: boolean;
    /** 是否支持列注释 */
    supportsColumnComment: boolean;
    /** 是否支持索引注释 */
    supportsIndexComment: boolean;
    /** 是否支持自增列（clickhouse 无自增概念，表编辑器据此禁用勾选，避免生成非法 DDL） */
    supportsAutoIncrement: boolean;
    /**
     * 默认索引类型（mysql/postgres 系 BTREE、mssql NONCLUSTERED、oracle/dm NORMAL、clickhouse INDEX）。
     * 为 getDefaultIndex() 的唯一事实源，方言内不得另写字面量。
     */
    defaultIndexType: string;

    // ==================== 协议/生态兼容 ====================

    /** 是否 MySQL 协议兼容（mysql/mariadb）：影响表列表是否展示创建时间等 MySQL 专属列 */
    mysqlCompatible: boolean;

    // ==================== 功能开关 ====================

    /** 是否支持可视化表结构编辑（建表/改列/改索引/重命名） */
    supportsTableEdit: boolean;
    /** 是否支持数据导入时的键冲突处理策略（IGNORE/REPLACE） */
    supportsDuplicateStrategy: boolean;

    // ==================== 表编辑器交互约束 ====================

    /**
     * 新建表时能否直接编辑自增列。
     * postgres 系为 false：自增由 serial/identity 列类型表达，非独立列属性
     */
    canEditAutoIncrementOnCreate: boolean;
    /**
     * 编辑已有表时能否修改自增列。
     * postgres 系与 mssql 为 false：identity 属性建表后不可变更
     */
    canEditAutoIncrementOnEdit: boolean;

    // ==================== 连接元信息 ====================

    /**
     * 实例连接方式：
     * - host_port: 填主机与端口（多数方言）
     * - file_path: 填本地文件路径（sqlite 等嵌入式库，无 host/port）
     */
    connectionMode: 'host_port' | 'file_path';
    /**
     * 连接描述符形态：
     * - none: 直接以库名连接（多数方言）
     * - sid_service: 需先选择 SID / Service Name（oracle 系），库名列表为空时回落到实例 SID
     */
    connectDescriptor: 'none' | 'sid_service';

    // ==================== 词法特征 ====================

    /**
     * 方言接受的标识符引用符，首项为主引用符，其余为兼容形式。
     * 用于补全时判断光标处是否已被包裹，避免 `` `id` `` 二次包裹为 ``` ``id`` ```
     */
    quotePairs: QuotePair[];
    /**
     * SQL 切割与光标区域判定的词法语义。
     * 缺省 {} 表示按标准 SQL 处理，详见 SqlSplitOptions
     */
    sqlSplitOptions: SqlSplitOptions;
}

/**
 * 标识符引用符对，如 mysql 的 ` 与 mssql 的 [ ]
 */
export interface QuotePair {
    open: string;
    close: string;
}

/**
 * 复合语句块（BEGIN..END）感知档位：
 * - none: 不感知，块内分号照常切割（sqlite/clickhouse 等无过程语句的方言）
 * - sql:  仅感知 BEGIN..END 与 CASE..END（T-SQL 的 IF/WHILE 无 END 闭合，PG 函数体均在引号内，
 *         仅需覆盖 BEGIN ATOMIC..END 与 BEGIN TRY..END TRY，故不纳入过程关键字）
 * - procedural: 额外感知 IF..END IF / LOOP..END LOOP / WHILE..END WHILE（MySQL 存储程序、Oracle/DM 系 PL-SQL）
 */
export type SqlBlockMode = 'none' | 'sql' | 'procedural';

/**
 * SQL 切割方言语义（与后端 dbm/sqlparser 的切割语义对齐，并按各家真实语法补齐）。
 * 默认值保持历史行为（mysql 反斜杠转义、其余能力关闭），各方言在能力声明中给出开启项。
 *
 * 归属 dialect 层的原因：这是方言的词法特征，由方言自身声明后，切割状态机（sqlParser）
 * 只消费传入的选项、不感知任何具体方言，新增方言零修改切割器（开闭原则）。
 */
export interface SqlSplitOptions {
    /** 字符串内反斜杠是否为转义符（mysql 系/clickhouse true；标准 SQL/PG/Oracle 等 false） */
    backslashEscape?: boolean;
    /** 反引号是否为标识符引用符（mysql/clickhouse/sqlite true），`` `a;b` `` 内分号不切分 */
    backtickQuote?: boolean;
    /** 是否支持 # 行注释（mysql true） */
    hashComment?: boolean;
    /** 是否支持 PG dollar-quoted 字符串（$$..$$ / $tag$..$tag$，postgres 系 true），其内分号不切分 */
    dollarQuote?: boolean;
    /** 是否支持 E'...' 转义字符串（postgres 系 true：普通串内 \ 为普通字符，仅 E 串内为转义符） */
    escapeStringPrefix?: boolean;
    /** 块注释是否支持嵌套（postgres 系 true，其余方言遇到嵌套 /* 按普通字符处理） */
    nestedBlockComment?: boolean;
    /** 是否支持 [标识符]（mssql 主引用符、sqlite 兼容 MySQL 语法），[a;b] 内分号不切分，]] 为转义 ] */
    bracketQuote?: boolean;
    /** 是否支持 Oracle 系 q'[...]' 替代引用字面量（oracle/dm true），其内分号与单引号均不切割 */
    altQuoteLiteral?: boolean;
    /** 双引号是否为标识符引用符（标准 SQL/PG/Oracle/mssql/sqlite true；mysql 系/clickhouse false，其为字符串字面量）。
     * 不影响切割（两种语义下内部分号均不切），仅影响光标区域判定：标识符内仍给代码提示 */
    doubleQuoteAsIdentifier?: boolean;
    /** 双横线行注释是否要求后随空白（mysql 系 true：`1--2` 为减法运算而非注释，不得吞掉后续语句） */
    lineCommentNeedsWhitespace?: boolean;
    /** 是否支持 mysql 的可执行注释（形如 `!` 紧跟开块注释，或带四位版本号门控；mysql 系 true）：
     * 其内容会被服务端当作 SQL 执行，故不参与补全的注释掩码 */
    executableComment?: boolean;
    /** 复合语句块感知档位，见 SqlBlockMode */
    blockMode?: SqlBlockMode;
}

export const commonCustomKeywords = ['GROUP BY', 'ORDER BY', 'LEFT JOIN', 'RIGHT JOIN', 'INNER JOIN', 'SELECT * FROM'];

/** 数据导入时的键冲突处理策略 */
export enum DuplicateStrategy {
    NONE = -1,
    IGNORE = 1,
    REPLACE = 2,
}

/**
 * 方言契约
 *
 * 参考 Hibernate Dialect / Prisma Connector 设计，按职责分域：
 * - 元信息域：getInfo、getEditorCompletions、getDefaultRows、getDefaultIndex、getCapabilities
 * - DDL 生成域：getCreateTableSql、getCreateIndexSql、getDropTableSql、getModifyColumnSql、getModifyIndexSql、getModifyTableInfoSql
 * - 查询域：getDefaultSelectSql、getPageSql、getPreviewSql、getPageSnippet
 * - 数据类型域：getDataType、wrapValue、getBatchInsertPreviewSql
 * - 标识符域：quoteIdentifier
 *
 * 新增方言 = 新建一个实现本接口的文件并声明能力，无需改动任何调用方。
 */
export interface DbDialect {
    /** 获取数据库元信息（名称、图标、端口、列类型列表） */
    getInfo(): DialectInfo;

    /**
     * 获取编辑器联想词（关键字、操作符、函数、变量）
     *
     * 异步的原因见 DialectInfo 注释：让 monaco 语言定义只在真正触发编辑器补全时按需加载，
     * 不进方言模块的静态依赖图。实现方应自行缓存结果，避免每次补全都重新构建。
     */
    getEditorCompletions(): Promise<EditorCompletion>;

    /** 获取方言能力声明（是否支持 schema/注释/自增、引用符、切割语义等） */
    getCapabilities(): DialectCapabilities;

    /** 获取默认查询 SQL */
    getDefaultSelectSql(db: string, table: string, condition: string, orderBy: string, pageNum: number, limit: number): string;

    /** 获取分页 SQL 片段 */
    getPageSql(pageNum: number, limit: number): string;

    /**
     * 把任意查询语句包装为仅取前 limit 行的预览 SQL（数据预览/同步预览等场景）。
     * 各家限行语法差异大（mssql 需用 TOP 包裹子查询、oracle 用 ROWNUM 追加条件），
     * 由方言自描述后调用方不再按 DbType 分支。
     */
    getPreviewSql(sql: string, limit?: number): string;

    /**
     * 获取分页查询的片段模板（SQL 补全的 SELECT PAGE 建议）。
     *
     * 与 getPageSql / getPreviewSql 同属「限行写法」这一方言事实。此前补全层另维护一份
     * dbType → 模板 的平行映射表，新增方言漏登记时不报错、只是静默退化为通用模板，
     * 现由方言自描述后该映射表已删除。
     *
     * 建议直接返回 shared/snippets.ts 的预设之一；确有特殊写法时再自行构造。
     */
    getPageSnippet(): SqlSnippetTemplate;

    /** 获取默认审计字段（新建表时预填充） */
    getDefaultRows(): RowDefinition[];

    /** 获取默认索引结构 */
    getDefaultIndex(): IndexDefinition;

    /** 引用标识符（包裹表名/字段名避免关键字冲突） */
    quoteIdentifier(name: string): string;

    /** 生成创建表 SQL */
    getCreateTableSql(tableData: TableEditContext): string;

    /** 生成创建索引 SQL */
    getCreateIndexSql(tableData: TableEditContext): string;

    /**
     * 生成删表 SQL。
     *
     * schema 型方言（postgres/oracle/dm/mssql 及其兼容方言）必须带 schema 限定，
     * 否则会落到连接默认 schema（如 pg 的 search_path）而删错表；
     * 非 schema 型方言（mysql/sqlite/clickhouse）只引用表名。
     *
     * 归属方言层的原因：本语句此前在资源树与表管理面板各拼一份，两者对 schema 的
     * 处理已经漂移（后者漏掉限定），收敛为方言方法后调用方不再各自构造 DDL。
     *
     * @param db 数据库名（schema 型方言形如 "db/schema"，由方言自行 extractSchema）
     * @param table 表名
     */
    getDropTableSql(db: string, table: string): string;

    /** 生成修改列 SQL */
    getModifyColumnSql(tableData: TableEditContext, tableName: string, changeData: ChangeDiff<RowDefinition>): string;

    /** 生成修改索引 SQL */
    getModifyIndexSql(tableData: TableEditContext, tableName: string, changeData: ChangeDiff<IndexDefinition>): string;

    /**
     * 生成修改表信息 SQL（重命名/注释）。
     * 只需名称与注释的新旧值，故契约窄化为 TableInfoEditContext，
     * 使「仅重命名」等场景可直接构造，无需伪造列/索引明细。
     */
    getModifyTableInfoSql(tableData: TableInfoEditContext): string;

    /** 通过数据库字段类型返回基本数据类型 */
    getDataType(columnType: string): DataType;

    /** 包装值（处理引号、N 前缀、日期函数等方言差异） */
    wrapValue(columnType: string, value: unknown): string | number | boolean | null;

    /** 生成批量插入预览 SQL */
    getBatchInsertPreviewSql(tableName: string, columns: string[], duplicateStrategy: DuplicateStrategy): string;
}
