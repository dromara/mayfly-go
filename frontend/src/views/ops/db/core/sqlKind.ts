/**
 * SQL 语句种类判定（与 DbInst 解耦的纯函数，供执行链路与缓存失效共用同一把尺子）。
 *
 * 两个判定都以「语句首个整词关键字」为准，判型前先剥离前导注释（stripLeadingComments）：
 * - isQueryStatement：只读查询 vs 写入，决定结果 tab 布局与「执行备注确认」；
 * - isDdlSql：是否结构变更，决定执行成功后要不要失效补全元数据缓存。
 * 二者语义不同（DML 非查询但不改结构），各自独立演进。
 */

import { getSqlScanZone, type SqlSplitOptions } from '../sql-editor/utils/sqlParser';

/**
 * 改变库结构的语句关键字：建/改/删表、清空、重命名，以及各方言的注释语句（comment on）
 * 与授权语句（grant/revoke）——注释与权限同样会改变补全要展示的内容（表/列注释来源）。
 */
const STRUCTURE_KEYWORDS = ['create', 'alter', 'drop', 'truncate', 'rename', 'comment', 'grant', 'revoke'];

/**
 * 只读语句首词基线：命中即按「查询」处理（独立结果 tab、不弹执行备注确认）。
 *
 * 与后端 sqlparser.stmtTypeFromKeyword 的只读集合严格对齐——select/with/table 归 StmtTypeSelect，
 * show/explain/describe/desc 归 StmtTypeOther，两者都由后端走查询通道返回结果集；余下（含
 * insert/update/delete/DDL 与未识别动词）后端一律 doExec 按写入执行。前端「查询」必须与之同集合，
 * 否则要么把写入误当查询而漏掉确认（危险），要么把只读误当写入而错乱结果展示。
 *
 * 未列入者一律按写入处理（fail-closed）：REPLACE/MERGE/CALL/SET 等不在此列，宁可多一次确认，
 * 也不放过破坏性写入。方言特有的只读语句（如 sqlite 的 pragma）由 DialectCapabilities
 * .queryStatementKeywords 追加，不在此基线内硬编（开闭原则）。
 */
const COMMON_QUERY_STATEMENT_KEYWORDS = ['select', 'with', 'table', 'show', 'explain', 'describe', 'desc'];

/**
 * 剥离语句前导的空白与注释（`--`、`#` 行注释与块注释），返回首个真实语句起始处。
 *
 * 语句切割（无论是后端还是编辑器本地）会保留注释原文，而「按前缀字符判类型」在注释前会命中
 * 注释里的单词（如 `-- 说明\nDELETE FROM t`），故必须先跳过注释区域。迭代而非递归：
 * 粘贴的 dump 脚本可能带成百上千行注释头，递归会按注释行数加深调用栈。
 *
 * 供 isDdlSql 与 DbInst.isQuerySql 共用：两者都靠「语句首词」判类型，都必须先越过前导注释，
 * 否则带 `--` 注释的 DDL/DML 会被 isQuerySql 误判为查询而各自独立成 tab。返回值仅用于判型，
 * 绝不回写执行文本——实际下发的仍是含注释的原文，剥离不会截断 SQL。
 */
export function stripLeadingComments(sql: string): string {
    let rest = sql.trimStart();
    for (;;) {
        if (rest.startsWith('--') || rest.startsWith('#')) {
            const end = rest.indexOf('\n');
            if (end === -1) {
                return '';
            }
            rest = rest.slice(end + 1).trimStart();
            continue;
        }
        if (rest.startsWith('/*')) {
            const end = rest.indexOf('*/');
            if (end === -1) {
                return '';
            }
            rest = rest.slice(end + 2).trimStart();
            continue;
        }
        return rest;
    }
}

/**
 * 从「已剥离前导注释」的文本取首个整词关键字（小写）：以字母/下划线开头，续接 [a-z0-9_$]。
 *
 * 整词而非前缀：`create_time`/`table_name` 这类以关键字开头的标识符不会被误判为语句动词。
 * 非字母/下划线开头（如以 ( 起始的括号查询、纯符号）返回空串，交由调用方按语义处理。
 */
function firstWord(strippedSql: string): string {
    const match = /^[a-z_][a-z0-9_$]*/i.exec(strippedSql);
    return match ? match[0].toLowerCase() : '';
}

/** 取语句首个整词关键字（先剥离前导空白与注释）。 */
function leadingKeyword(sql: string): string {
    return firstWord(stripLeadingComments(sql));
}

/**
 * 是否为结构变更语句。
 *
 * 消费方是「执行成功后要不要失效本地元数据缓存」：结构变了却不失效，SQL 补全会继续给旧表清单/旧列。
 * 与 isQueryStatement 的「查询」不是一回事——DML 也非查询但不改结构，故两个判定各自演进。
 *
 * 只看语句首词（整词匹配），因此认不出被 `EXEC`/存储过程包住的 DDL（如 mssql 的
 * `EXECUTE sp_addextendedproperty`）——这类方言特殊写法由其调用方（表编辑弹框）无条件失效兜底。
 */
export function isDdlSql(sql: string): boolean {
    return STRUCTURE_KEYWORDS.includes(leadingKeyword(sql));
}

/** isQueryStatement 的可选入参 */
export interface QueryStatementOptions {
    /** 方言追加的只读首词（DialectCapabilities.queryStatementKeywords，如 sqlite 的 pragma） */
    extraReadKeywords?: readonly string[];
    /** 该方言的 SQL 切割语义，用于区分「字符串/注释/标识符内的同名词」与「代码区的真实关键字」 */
    splitOptions?: SqlSplitOptions;
}

/**
 * 语句体内是否存在「落在代码区」的某关键字：命中位置若在字符串/注释/标识符内则不算。
 *
 * 复用切割器的区域判定（getSqlScanZone），不另写词法，避免与切割语义漂移；
 * 用于识别「首词是只读、但体内藏写入」的语句（见 isQueryStatement）。
 */
function hasCodeKeyword(sql: string, pattern: RegExp, splitOptions?: SqlSplitOptions): boolean {
    const re = new RegExp(pattern.source, 'gi');
    let match: RegExpExecArray | null;
    while ((match = re.exec(sql)) !== null) {
        if (getSqlScanZone(sql, match.index, splitOptions ?? {}) === 'code') {
            return true;
        }
    }
    return false;
}

/**
 * 首词虽为只读、但体内出现某词即实为写入的形态：readKeyword → 体内触发词（整词，仅代码区命中才算）。
 * - with：数据修改型 CTE（Postgres `WITH d AS (DELETE ...) SELECT`）。不含 replace——`REPLACE()` 是
 *   mysql 常见字符串函数，`WITH x AS (SELECT replace(...))` 这类只读会被误伤；
 * - select：`SELECT ... INTO 新表`（mssql/pg 建表写入）；
 * - explain：`EXPLAIN ANALYZE` 会真正执行底层语句（含写）。
 * 新增只读首词的隐藏写入形态只需在此加一行，不改判定分支（开闭原则）。
 */
const READ_KEYWORD_HIDDEN_WRITE: Record<string, RegExp> = {
    with: /\b(insert|update|delete|merge)\b/,
    select: /\binto\b/,
    explain: /\b(analyze|analyse)\b/,
};

/**
 * 是否为只读查询语句（决定结果 tab 布局与要不要弹执行备注确认）。
 *
 * 判型前先剥离前导注释（切割器保留注释原文，`-- 说明\nALTER ...` 的首词是 alter 而非注释），
 * 再取整词首词比对只读集合。以 ( 起始的括号查询/UNION 后端按 SelectStmt 解析，故也归只读。
 * 未命中只读集合一律按写入（fail-closed）。首词只读但体内藏写入者，见 READ_KEYWORD_HIDDEN_WRITE 降级。
 *
 * @param sql 单条语句（可含前导注释；仅用于判型，绝不回写执行文本）
 */
export function isQueryStatement(sql: string, opts: QueryStatementOptions = {}): boolean {
    const stripped = stripLeadingComments(sql);
    if (stripped.startsWith('(')) {
        return true;
    }
    const keyword = firstWord(stripped);
    const readByKeyword = COMMON_QUERY_STATEMENT_KEYWORDS.includes(keyword) || (opts.extraReadKeywords ?? []).includes(keyword);
    if (!readByKeyword) {
        return false;
    }
    // 首词只读，但体内出现该形态的写入词（仅代码区命中，字符串/注释里的同名词不算）则降级为写入
    const hiddenWrite = READ_KEYWORD_HIDDEN_WRITE[keyword];
    if (hiddenWrite && hasCodeKeyword(sql, hiddenWrite, opts.splitOptions)) {
        return false;
    }
    return true;
}
