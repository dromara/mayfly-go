import { getSqlScanZone, maskSqlComments, SqlSplitOptions, splitSqlStatements } from '../../component/sqleditor/utils/sqlParser';
import { commonQuotePairs, stripIdentifierQuotes } from './quoter';

/**
 * 光标所在表的上下文信息
 */
export interface TableCtx {
    tableName: string;
    tableAlias: string;
    db: string;
}

/**
 * 光标所在语句信息：语句文本（注释以等长空白掩码，长度与偏移不变）与光标相对语句文本起点的偏移。
 * 注释不参与子句/表名解析，但字面量原文保留（'a;b' 等不影响位置）。
 */
export interface StatementAt {
    text: string;
    cursorOffset: number;
}

/**
 * 提取光标偏移量所在的完整 SQL 语句文本（感知字符串/注释中的分号）。
 * 复用通用解析器 splitSqlStatements，替代旧的简单 lastIndexOf(';') 切分，
 * 避免 `WHERE remark = 'a;b'` 等语句被错误截断。
 *
 * @param fullSql 编辑器完整文本
 * @param offset 光标在全文中的偏移量
 * @param opts 切割方言选项，见 SqlSplitOptions
 * @returns 光标所在语句信息；光标位于语句间隙（分号后未输入）时返回 undefined
 */
export function extractStatementAt(fullSql: string, offset: number, opts: SqlSplitOptions = {}): StatementAt | undefined {
    const statements = splitSqlStatements(fullSql ?? '', ';', opts);
    for (const stmt of statements) {
        // 光标位于语句范围内（含结束分号所在位置）
        if (offset >= stmt.start && offset <= stmt.end) {
            // 偏移以语句文本首字符为基准（前导空白不计入文本），注释掩码为等长空白，下标仍一致
            return { text: maskSqlComments(stmt.text, opts), cursorOffset: offset - stmt.textStart };
        }
    }
    return undefined;
}

/**
 * 解析光标前令牌信息：当前行光标前的文本按空白切分，取最后/次后令牌，
 * 并在 `.` 触发（如 t. 或 db.）时解析出别名或库名。
 *
 * @param lineTextBeforeCursor 当前行光标前文本
 * @returns lastToken/secondToken 均为小写；dotAlias 未命中时为空串
 */
export function parseCursorTokens(lineTextBeforeCursor: string): {
    lastToken: string;
    secondToken: string;
    isDotTrigger: boolean;
    dotAlias: string;
} {
    const tokens = lineTextBeforeCursor.trim().split(/\s+/);
    let lastToken = (tokens[tokens.length - 1] || '').toLowerCase();
    const secondToken = (tokens.length > 2 && tokens[tokens.length - 2].toLowerCase()) || '';

    let dotAlias = '';
    const isDotTrigger = lastToken.indexOf('.') > -1 || secondToken.indexOf('.') > -1;
    if (isDotTrigger) {
        dotAlias = resolveDotAlias(lastToken, secondToken);
        // 兼容带引用符的别名/库名点触发，如 `users`. / "users". / [users].
        dotAlias = stripIdentifierQuotes(dotAlias, commonQuotePairs).toLowerCase();
    }
    return { lastToken, secondToken, isDotTrigger, dotAlias };
}

/**
 * 解析 `.` 触发补全时的别名/库名。
 * 兼容各类书写形态：`t.`、`db.`、`a.creator,a.`（逗号粘连）、`.field`（句首点）等。
 */
function resolveDotAlias(lastToken: string, secondToken: string): string {
    let alias = lastToken.substring(0, lastToken.lastIndexOf('.'));
    if (lastToken.trim().startsWith('.')) {
        alias = secondToken;
    }
    if (!alias && secondToken.indexOf('.') > -1) {
        alias = secondToken.substring(secondToken.indexOf('.') + 1);
    }

    // 如果字符串粘连起了如:'a.creator,a.'，需要重新取出别名
    const aliasArr = lastToken.split(',');
    if (aliasArr.length > 1) {
        const sticky = aliasArr[aliasArr.length - 1];
        alias = sticky.substring(0, sticky.lastIndexOf('.'));
        if (sticky.trim().startsWith('.')) {
            alias = secondToken;
        }
    }
    return alias;
}

/**
 * 表别名占位关键字：表名后的下一个单词命中这些关键字时不视为别名。
 */
const aliasExclusiveKeywords = new Set([
    'where', 'on', 'set', 'values', 'as', 'group', 'order', 'left', 'right', 'inner', 'outer', 'full', 'cross',
    'union', 'having', 'limit', 'offset', 'select', 'into', 'from', 'join', 'and', 'or', 'using', 'returning',
    'default', 'when', 'then', 'else', 'end', 'not', 'in', 'is', 'like', 'between',
]);

/** FROM/JOIN/UPDATE/INTO 子句的表名（排除括号，兼容 `INSERT INTO t(a,b)` 无空格形态） */
const TABLE_CLAUSE_REGEX = /(?:from|join|update|into)\s+([^\s(),;]+)/gi;

/** 表名后的别名形态：裸标识符或引用符包裹（`x` "x" [x]），支持可选 AS */
const TABLE_ALIAS_REGEX = /^\s+(?:as\s+)?(`[^`]*`|"[^"]*"|\[[^\]]*\]|[a-zA-Z_$][\w$]*)/i;

/** 去除表名/别名首尾的引用符（反引号、双引号、方括号） */
const stripQuoteAndBracket = (name: string) => name.replace(/[`"\[\]]/g, '');

/**
 * 解析语句中全部表引用（含出现位置），供第一个表/别名匹配/就近表复用。
 * 直接在原始文本上扫描（\s+ 已覆盖换行），保证位置与光标偏移一致。
 */
function parseTableRefs(sql: string, defaultDb: string): { table: TableCtx; index: number; end: number }[] {
    const refs: { table: TableCtx; index: number; end: number }[] = [];
    TABLE_CLAUSE_REGEX.lastIndex = 0;
    let matches;
    while ((matches = TABLE_CLAUSE_REGEX.exec(sql)) !== null) {
        let tableName = stripQuoteAndBracket(matches[1]);
        let db = defaultDb;
        if (tableName.indexOf('.') >= 0) {
            const info = tableName.split('.');
            db = info[0];
            if (defaultDb.indexOf('/') > 0) {
                db = defaultDb.substring(0, defaultDb.indexOf('/') + 1) + db;
            }
            tableName = info[1];
        }

        // 表名后的下一个单词为别名，占位关键字（WHERE/SET/VALUES…）不视为别名
        const tableEnd = matches.index + matches[0].length;
        const aliasMatch = sql.substring(tableEnd).match(TABLE_ALIAS_REGEX);
        let tableAlias = tableName;
        if (aliasMatch) {
            const alias = stripQuoteAndBracket(aliasMatch[1]);
            if (alias && !aliasExclusiveKeywords.has(alias.toLowerCase())) {
                tableAlias = alias;
            }
        }
        refs.push({ table: { tableName, tableAlias, db }, index: matches.index, end: tableEnd });
    }
    return refs;
}

/**
 * 从 SQL 语句中解析表上下文（表名、别名、库名）。
 * 支持 FROM/JOIN/UPDATE/INTO 子句（含 INSERT INTO/DELETE FROM）、
 * `db.table` 形式及标识符引用符包裹的名称。
 *
 * @param sql 完整 SQL 语句文本
 * @param alias 指定匹配的表别名；为空时返回语句中的第一个表
 * @param defaultDb 默认库名
 */
export function resolveTableContext(sql: string, alias: string = '', defaultDb: string): TableCtx | undefined {
    const refs = parseTableRefs(sql, defaultDb);
    const tables = refs.map((r) => r.table);

    if (alias) {
        // 如果指定了别名参数，则返回对应的表名（忽略大小写，兼容 FROM Users U + U. 等写法）
        const lowerAlias = alias.toLowerCase();
        return tables.find((t) => t.tableAlias?.toLowerCase() === lowerAlias || t.tableName?.toLowerCase() === lowerAlias);
    }
    // 如果未指定别名参数，则返回第一个表名
    return tables.length > 0 ? tables[0] : undefined;
}

/**
 * 解析光标偏移之前最近的表子句对应的表上下文（JOIN 多表场景按光标就近取表）。
 * 光标之前无任何表子句时退回语句第一个表（兜底行为与旧版一致）。
 *
 * @param sql 光标所在的完整 SQL 语句文本
 * @param cursorOffset 光标相对语句起点的偏移
 * @param defaultDb 默认库名
 */
export function resolveNearestTableContext(sql: string, cursorOffset: number, defaultDb: string): TableCtx | undefined {
    const refs = parseTableRefs(sql, defaultDb);
    if (refs.length === 0) {
        return undefined;
    }
    // 取光标之前最后出现的表子句，无命中（光标在首个 FROM/JOIN 之前）时退回第一个表
    let nearest = refs[0];
    for (const ref of refs) {
        if (ref.index < cursorOffset) {
            nearest = ref;
        } else {
            break;
        }
    }
    return nearest.table;
}

/**
 * 解析语句作用域内的所有表（按出现顺序，含别名/库名），
 * 用于 SELECT/WHERE/ON 等列上下文提示全作用域字段（对齐 DataGrip 多表场景行为）。
 */
export function resolveScopeTableContexts(sql: string, defaultDb: string): TableCtx[] {
    return parseTableRefs(sql, defaultDb).map((r) => r.table);
}

/** 光标所处子句类型：table（表名期望位置）/ column（字段/表达式位置）/ free（无法判定，全量兜底） */
export type CursorClause = 'table' | 'column' | 'free';

/** 表名期望位置的关键字（其后应提示表名/库名而非字段） */
const tableExpectedKeywords = new Set(['from', 'join', 'into', 'update']);

/** 子句关键字扫描（按 word 边界，覆盖常见写法；group/order 取首词即可） */
const CLAUSE_KEYWORD_REGEX = /\b(select|insert|update|delete|from|join|on|where|set|group|order|having|limit|offset|into|values|union)\b/gi;

/**
 * 判定光标所处子句类型（对齐国际产品的上下文感知建议）：
 * - FROM/JOIN/INTO/UPDATE 等表名期望位置 → table（仅提示表/库）
 * - SELECT/WHERE/ON/SET/ORDER BY/GROUP BY/HAVING 及括号内列清单 → column（提示作用域内字段）
 * - 语句为空或无关键字 → free（全量兜底）
 *
 * @param statement 光标所在语句文本（注释已等长掩码）
 * @param cursorOffset 光标相对语句文本起点的偏移
 * @param lineEndsWithParen 当前行光标前是否紧邻 '('（如 INSERT INTO t ( 的列清单位置）
 */
export function resolveCursorClause(statement: string, cursorOffset: number, lineEndsWithParen: boolean): CursorClause {
    if (!statement) {
        return 'free';
    }
    // 左括号后（列清单/IN 子查询参数位）视为字段上下文
    if (lineEndsWithParen) {
        return 'column';
    }
    const head = statement.substring(0, Math.max(cursorOffset, 0));
    CLAUSE_KEYWORD_REGEX.lastIndex = 0;
    let last = '';
    let matches;
    while ((matches = CLAUSE_KEYWORD_REGEX.exec(head)) !== null) {
        last = matches[1].toLowerCase();
    }
    if (!last) {
        return 'free';
    }
    if (tableExpectedKeywords.has(last)) {
        // INSERT INTO t (col：表名后的未闭合括号内属于列清单上下文
        if (last === 'into') {
            const openIdx = head.lastIndexOf('(');
            if (openIdx > head.lastIndexOf(')')) {
                return 'column';
            }
        }
        return 'table';
    }
    return 'column';
}

/** 光标所处区域 */
export type CursorZone = 'code' | 'string' | 'comment';

/**
 * 判断光标偏移量所处区域：代码、字符串字面量或注释。
 * 字符串/注释内不提供代码提示，避免噪音建议。
 * 区域判定直接复用切割扫描器（getSqlScanZone）的状态机，保证与 splitSqlStatements
 * 语义单一来源（含反引号/双引号/方括号标识符、PG dollar-quote 与 E 串、Oracle q-quote、
 * mysql # 注释与 `--` 空白规则等），否则会出现“补全判定的语句”与“实际执行的语句”不一致。
 * 标识符引用符内（`` `x` ``/"x"/[x]）归为 code：反引号内输入仍需提示，
 * 而区域判定只负责阻断其内部的 -- /* 被误判为注释。
 *
 * @param fullSql 编辑器完整文本
 * @param offset 光标在全文中的偏移量
 * @param opts 切割方言选项，见 SqlSplitOptions
 */
export function resolveCursorZone(fullSql: string, offset: number, opts: SqlSplitOptions = {}): CursorZone {
    const zone = getSqlScanZone(fullSql, offset, opts);
    return zone === 'string' ? 'string' : zone === 'comment' ? 'comment' : 'code';
}
