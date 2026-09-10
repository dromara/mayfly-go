import { editor, Position } from 'monaco-editor';
import { DbType } from '../../../dialect/dbType';

export interface SqlStatement {
    /** 语句文本（注释与字面量均保留原文，仅去除首尾空白） */
    text: string;
    /** 语句起始偏移（含前导空白，即上一条分隔符之后） */
    start: number;
    /** 语句结束偏移（分号所在位置，无分号时为文本末尾） */
    end: number;
    /** text 首字符在全文中的偏移（start 之后的首个非空白字符）：光标偏移换算须以此为准 */
    textStart: number;
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
 * 默认值保持历史行为（mysql 反斜杠转义、其余能力关闭），各方言的开启项见 getSqlSplitOptions。
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

/** 默认切割选项（不传 options 时的兼容行为：仅 mysql 反斜杠转义语义） */
export const defaultSplitOptions: Required<SqlSplitOptions> = {
    backslashEscape: true,
    backtickQuote: false,
    hashComment: false,
    dollarQuote: false,
    escapeStringPrefix: false,
    nestedBlockComment: false,
    bracketQuote: false,
    altQuoteLiteral: false,
    doubleQuoteAsIdentifier: false,
    lineCommentNeedsWhitespace: false,
    executableComment: false,
    blockMode: 'none',
};

/** postgres 系（含 gauss/人大金仓/vastbase 等兼容方言）共同切割语义 */
const pgSplitOptions: Required<SqlSplitOptions> = {
    ...defaultSplitOptions,
    backslashEscape: false,
    dollarQuote: true,
    escapeStringPrefix: true,
    nestedBlockComment: true,
    doubleQuoteAsIdentifier: true,
    blockMode: 'sql',
};

/** mysql 系共同切割语义 */
const mysqlSplitOptions: Required<SqlSplitOptions> = {
    ...defaultSplitOptions,
    backslashEscape: true,
    backtickQuote: true,
    hashComment: true,
    lineCommentNeedsWhitespace: true,
    executableComment: true,
    blockMode: 'procedural',
};

/**
 * 各方言切割语义（对齐服务端 dbm/sqlparser 的方言切割器选型），未注册方言按标准 SQL 处理。
 * 新增方言时在此注册即可，无需改动切割状态机（开闭原则）。
 */
const dialectSplitOptions: Record<string, Partial<SqlSplitOptions>> = {
    [DbType.mysql]: mysqlSplitOptions,
    [DbType.mariadb]: mysqlSplitOptions,
    [DbType.postgresql]: pgSplitOptions,
    [DbType.gauss]: pgSplitOptions,
    [DbType.kingbaseEs]: pgSplitOptions,
    [DbType.vastbase]: pgSplitOptions,
    // clickhouse：反引号标识符 + 反斜杠转义，# 与 -- 均为行注释（-- 不要求后随空白）
    [DbType.clickhouse]: { backslashEscape: true, backtickQuote: true, hashComment: true },
    // sqlite 兼容 MySQL 的反引号与方括号引用符，但反斜杠为普通字符
    [DbType.sqlite]: { backslashEscape: false, backtickQuote: true, bracketQuote: true, doubleQuoteAsIdentifier: true },
    // T-SQL：方括号标识符 + BEGIN..END/BEGIN TRY，IF、WHILE 无 END 闭合且 DECLARE 为独立语句，故不纳入过程关键字
    [DbType.mssql]: { backslashEscape: false, bracketQuote: true, doubleQuoteAsIdentifier: true, blockMode: 'sql' },
    // Oracle/达梦：q- quote 字面量 + PL-SQL 过程块（IF/LOOP/WHILE 均以 END <kw> 闭合）
    [DbType.oracle]: { backslashEscape: false, altQuoteLiteral: true, doubleQuoteAsIdentifier: true, blockMode: 'procedural' },
    [DbType.dm]: { backslashEscape: false, altQuoteLiteral: true, doubleQuoteAsIdentifier: true, blockMode: 'procedural' },
};

/**
 * 获取指定方言的切割选项。
 * @param dbType 数据库类型
 */
export function getSqlSplitOptions(dbType?: string): SqlSplitOptions {
    return dialectSplitOptions[dbType ?? ''] ?? {};
}

/** 合并调用方选项与默认值，得到完整切割语义（切割与光标区域判定共用） */
export function resolveSqlSplitOptions(opts?: SqlSplitOptions): Required<SqlSplitOptions> {
    return { ...defaultSplitOptions, ...opts };
}

/**
 * 读取 sql[i] 起始的 dollar-quote 开标签（$$ 或 $tag$），语义对齐服务端 pgsql_splitter.readDollarTag：
 * tag 体仅允许字母/数字/下划线，且 tag 内不允许换行；未命中返回空串。
 * 同时供补全模块（resolveCursorZone）复用，保证光标区域判定与切割语义一致。
 */
export function readDollarTag(sql: string, i: number): string {
    for (let j = i + 1; j < sql.length; j++) {
        const ch = sql[j];
        if (ch === '$') {
            // 校验 tag 体（i+1..j-1）为合法字符
            for (let k = i + 1; k < j; k++) {
                const t = sql[k];
                if (!(t >= 'a' && t <= 'z' || t >= 'A' && t <= 'Z' || t >= '0' && t <= '9' || t === '_')) {
                    return '';
                }
            }
            return sql.slice(i, j + 1);
        }
        // tag 内不允许换行（跨行的 $ 不是 dollar-quote 开标签）
        if (ch === '\n') {
            return '';
        }
    }
    return '';
}

/**
 * 通用 SQL 解析器，用于提取 SQL 语句及其位置信息。
 * 注释与字面量均保留原文（与后端 dbm/sqlparser 的切割语义一致）：
 * Oracle 的 /*+ hint *\/ 与 mysql 的 /*!40101 ... *\/ 会被服务端执行，剔除即丢语义，
 * 且剔除后与其紧邻的 token 粘连会改写 SQL；补全需要排除注释时请用 maskSqlComments。
 *
 * @param sql 完整的SQL文本
 * @param delimiter SQL语句分隔符，默认为分号
 * @param opts 切割方言选项，见 SqlSplitOptions
 */
export function splitSqlStatements(sql: string, delimiter: string = ';', opts: SqlSplitOptions = {}): SqlStatement[] {
    return scanSql(sql ?? '', delimiter, resolveSqlSplitOptions(opts), Number.POSITIVE_INFINITY).statements;
}

/**
 * 将注释内容替换为等长空白（换行保留），得到用于补全解析的「掩码文本」。
 *
 * 切割结果必须保留注释原文（否则 hint/可执行注释丢语义），而子句/表名等正则解析不得命中注释里的单词，
 * 故在补全侧掩码：长度与各行位置与原文完全一致，光标偏移与匹配下标仍可直接比较。
 * mysql 的 /*! ... *\/ 可执行注释内容是待执行 SQL，故不被掩码。
 *
 * @param sql 待掩码的文本（一般为单条语句）
 * @param opts 方言语义选项（与切割同一来源）
 */
export function maskSqlComments(sql: string, opts: SqlSplitOptions = {}): string {
    const text = sql ?? '';
    const { commentRanges } = scanSql(text, ';', resolveSqlSplitOptions(opts), Number.POSITIVE_INFINITY);
    if (!commentRanges.length) {
        return text;
    }
    let masked = '';
    let pos = 0;
    for (const range of commentRanges) {
        masked += text.slice(pos, range.start) + text.slice(range.start, range.end).replace(/[^\n\r]/g, ' ');
        pos = range.end;
    }
    return masked + text.slice(pos);
}

/** 扫描到的文本区域类型（标识符引用符内仍可代码提示，故与字符串区分） */
export type ScanZone = 'code' | 'string' | 'identifier' | 'comment';

type ScanState = 'normal' | 'string' | 'dollarQuote' | 'altQuote' | 'lineComment' | 'blockComment';

/** 单词字符：含 $ 以兼容 pg 标识符，含 Unicode 字母/数字以兼容中文标识符 */
const isWordChar = (ch: string | undefined): boolean => !!ch && /[\p{L}\p{N}_$]/u.test(ch);

/** 词首字符：$ 属 dollar-quote 判定范畴，故不纳入词首 */
const isWordStart = (ch: string | undefined): boolean => !!ch && /[\p{L}_]/u.test(ch);

/** 行注释中断符（mysql 要求双横线后至少一个空白/控制字符，换行与 EOF 均视为行尾） */
const isLineBreakChar = (ch: string | undefined): boolean => ch === undefined || /\s/.test(ch);

/** BEGIN 后紧跟这些词（或直接跟分隔符）时为开启事务语句，无对应 END，不入复合块 */
const txnBeginWords = new Set(['transaction', 'tran', 'work', 'distributed', 'deferred', 'immediate', 'exclusive', 'read', 'commented']);

/** 可与 END 构成一个闭合词的关键字（END IF / END LOOP / END TRY 等），随 END 一并消费，避免被当作块起始词再入栈 */
const endCloserWords = new Set(['if', 'loop', 'while', 'case', 'repeat', 'for', 'try', 'catch', 'block', 'function', 'procedure', 'proc', 'package', 'pkg', 'trigger', 'transaction', 'tran', 'handler', 'with']);

/** procedural 档位额外纳入的过程块起始词；不含 repeat/for（MySQL REPEAT() 同名函数；FOR 必携 LOOP，由 LOOP 计入） */
const proceduralOpeners = new Set(['if', 'loop', 'while']);

/**
 * 存储程序定义特征词：loop/while/if 在 MySQL、Oracle 系中均为非保留字（可作表名/列名），
 * 故过程块起始词仅在已处于复合块内、或当前语句确为存储程序/触发器/事件定义时才入块
 */
const spDefinitionWordRe = /\b(procedure|function|trigger|event)\b/i;

/** Oracle q-quote 括号型定界符配对，其余字符作定界符时首尾同字符 */
const altOpenCloseMap: Record<string, string> = { '[': ']', '{': '}', '(': ')', '<': '>' };

/** Oracle 系替代引用字面量的合法前缀词（q'[..]' / nq'[..]' / uq'[..]'） */
const altQuoteWords = new Set(['q', 'nq', 'uq']);

/** 读取 sql[i] 起始的整词（已转小写）及其结束位置（不含） */
function readWord(sql: string, i: number): { word: string; end: number } {
    let j = i;
    while (j < sql.length && isWordChar(sql[j])) {
        j++;
    }
    return { word: sql.slice(i, j).toLowerCase(), end: j };
}

/** 跳过空白，返回第一个非空白字符下标 */
function skipSpace(sql: string, i: number): number {
    let j = i;
    while (j < sql.length && /\s/.test(sql[j])) {
        j++;
    }
    return j;
}

/**
 * 判断引号前是否为 PG 转义字符串前缀 E（如 E'it\'s'）：
 * 仅适用于单引号（PG 无 E"..." 语法），且 E 必须是独立词（前一个字符不能是标识符字符），
 * 否则列名尾字母 e 会误开启转义语义
 */
function isEscapeStringQuote(sql: string, quoteIdx: number, options: Required<SqlSplitOptions>): boolean {
    if (!options.escapeStringPrefix || quoteIdx === 0 || sql[quoteIdx] !== "'") {
        return false;
    }
    const prefix = sql[quoteIdx - 1];
    return (prefix === 'E' || prefix === 'e') && !isWordChar(sql[quoteIdx - 2]);
}

/** IF 是否开启过程块：MySQL 的 IF 同时是函数名与 DDL 关键字（IF NOT EXISTS），以后续存在 THEN 为准 */
function hasThenBeforeDelimiter(sql: string, from: number, delimiter: string): boolean {
    const idx = sql.indexOf(delimiter, from);
    return /\bthen\b/i.test(sql.slice(from, idx < 0 ? sql.length : idx));
}

/** CASE/IF/LOOP/WHILE 等起始词是否存在与之配对的 END（无配对则视为普通标识符，避免误入块吞并后续语句） */
function createEndAheadChecker(sql: string): (from: number) => boolean {
    // 命中位置与已确认无 END 的起始位置均向后缓存，使全文总扫描量保持线性
    let endAheadIdx = -1;
    let exhaustedIdx = Number.POSITIVE_INFINITY;
    return (from: number): boolean => {
        if (endAheadIdx >= from) {
            return true;
        }
        // 此前的扫描已从更靠后位置走到文本末尾且未命中，则本位置同样不可能命中
        if (from >= exhaustedIdx) {
            return false;
        }
        endAheadIdx = -1;
        for (let j = from; j < sql.length; ) {
            if (!isWordStart(sql[j])) {
                j++;
                continue;
            }
            const word = readWord(sql, j);
            if (word.word === 'end') {
                endAheadIdx = j;
                return true;
            }
            j = word.end;
        }
        exhaustedIdx = from;
        return false;
    };
}

/** 扫描状态 -> 区域类型 */
function toZone(state: ScanState, stringIsIdentifier: boolean): ScanZone {
    if (state === 'lineComment' || state === 'blockComment') {
        return 'comment';
    }
    if (state === 'string') {
        return stringIsIdentifier ? 'identifier' : 'string';
    }
    if (state === 'altQuote') {
        return 'string';
    }
    // dollar-quote 体多为函数/脚本正文，光标在此仍允许代码提示（与历史行为一致）
    return 'code';
}

/** 前导空白长度（文本去除首尾空白后，其首字符相对原位置的偏移） */
const leadingBlankLen = (s: string): number => s.length - s.trimStart().length;

/**
 * SQL 文本单遍扫描：按方言语义产出语句列表、注释区域与 stopAt 偏移所处区域。
 * 切割（splitSqlStatements）、注释掩码（maskSqlComments）与光标区域判定（getSqlScanZone）共用本函数，保证语义单一来源：
 * 两套实现一旦漂移，会出现“补全判定的语句”与“实际执行的语句”不一致。
 *
 * @param sql 完整文本
 * @param delimiter 语句分隔符（单字符）
 * @param options 完整切割语义（见 resolveSqlSplitOptions）
 * @param stopAt 扫描到该偏移即中断（仅取区域时避免全文扫描），传 Infinity 表示全文
 */
function scanSql(
    sql: string,
    delimiter: string,
    options: Required<SqlSplitOptions>,
    stopAt: number
): { statements: SqlStatement[]; zone: ScanZone; commentRanges: { start: number; end: number }[] } {
    const statements: SqlStatement[] = [];
    let state: ScanState = 'normal';
    let buffer = '';
    let startPos = 0;
    // 当前语句是否含实质代码（非空白且非注释）：仅注释的分段不构成语句（如 dump 头部的注释块）
    let hasCode = false;
    // 当前字符串/标识符区上下文：闭合符、是否双写转义、反斜杠是否转义、是否为标识符（决定区域类型）
    let stringClose = '';
    let stringDoubling = false;
    let stringBackslash = false;
    let stringIsIdentifier = false;
    let dollarTag = ''; // dollar-quote 的完整闭合序列，如 $$ 或 $fn$
    let altClose = ''; // Oracle q-quote 的闭合序列，如 ]' 或 !'
    let blockCommentDepth = 0;
    const blockStack: string[] = []; // 复合块栈（元素为入块词，'declare' 表示 Oracle 匿名块声明段）
    const hasEndAhead = createEndAheadChecker(sql);
    let stopped = false;
    // 注释区域（[start, end) 不含行注释的换行符），供补全做等长掩码；可执行注释不属于此列
    const commentRanges: { start: number; end: number }[] = [];
    let commentStart = -1; // 当前注释起始偏移，-1 表示不在注释中
    let execComment = false; // 当前块注释是否为可执行注释（内容会被服务端执行）

    const pushStatement = (end: number) => {
        if (buffer.trim() && hasCode) {
            statements.push({ text: buffer.trim(), start: startPos, end, textStart: startPos + leadingBlankLen(buffer) });
        }
        buffer = '';
        hasCode = false;
        // 语句边界处重置块栈，防止未闭合块静默吞掉后续所有语句
        blockStack.length = 0;
        startPos = end + delimiter.length;
    };

    /** 进入字符串/标识符区（开闭符可异字符如 [x]，双写转义与反斜杠转义按方言语义各异），返回调用处需赋值的状态 */
    const enterString = (close: string, doubling: boolean, backslash: boolean, isIdentifier: boolean, openText: string): ScanState => {
        stringClose = close;
        stringDoubling = doubling;
        stringBackslash = backslash;
        stringIsIdentifier = isIdentifier;
        buffer += openText;
        hasCode = true;
        return 'string';
    };

    for (let i = 0; i < sql.length; i++) {
        if (i >= stopAt) {
            stopped = true;
            break;
        }
        const char = sql[i];
        const nextChar = sql[i + 1];

        if (state === 'normal') {
            if (char === '-' && nextChar === '-' && (!options.lineCommentNeedsWhitespace || isLineBreakChar(sql[i + 2]))) {
                state = 'lineComment';
                buffer += '--';
                commentStart = i;
                i++; // 跳过第二个 -
            } else if (char === '/' && nextChar === '*') {
                state = 'blockComment';
                blockCommentDepth = 1;
                buffer += '/*';
                commentStart = i;
                execComment = options.executableComment && sql[i + 2] === '!';
                if (execComment) {
                    // mysql 的 /*! 40101 ... */ 内容会被执行，既可单独成句也不参与注释掩码
                    hasCode = true;
                }
                i++;
            } else if (options.hashComment && char === '#') {
                state = 'lineComment';
                buffer += char;
                commentStart = i;
            } else if (char === "'" || char === '"') {
                // 双引号在标准 SQL 系为标识符引用符（仅影响光标区域归属），mysql 系为字符串字面量
                state = enterString(char, true, options.backslashEscape || isEscapeStringQuote(sql, i, options), char === '"' && options.doubleQuoteAsIdentifier, char);
            } else if (options.backtickQuote && char === '`') {
                // 反引号标识符内反斜杠为普通字符，转义靠双写 ``
                state = enterString(char, true, false, true, char);
            } else if (options.bracketQuote && char === '[') {
                // [x] 开闭异字符，]] 为转义 ]
                state = enterString(']', true, false, true, char);
            } else if (options.dollarQuote && char === '$') {
                const tag = readDollarTag(sql, i);
                if (tag) {
                    state = 'dollarQuote';
                    dollarTag = tag;
                    buffer += tag;
                    hasCode = true;
                    i += tag.length - 1;
                } else {
                    buffer += char;
                }
            } else if (char === delimiter) {
                if (blockStack.length) {
                    // 复合块内分号属于块体，不切割
                    buffer += char;
                } else {
                    pushStatement(i);
                }
            } else if ((options.blockMode !== 'none' || options.altQuoteLiteral) && isWordStart(char)) {
                // 整词消费：复合语句块感知依赖整词匹配（避免 CASE 命中 ASE 等子串误判）
                const { word, end } = readWord(sql, i);
                let wordEnd = end;
                if (word === 'begin' && options.blockMode !== 'none') {
                    const after = skipSpace(sql, end);
                    const nextWord = isWordStart(sql[after]) ? readWord(sql, after).word : '';
                    // BEGIN;/BEGIN TRANSACTION 等开启事务语句无对应 END，不能入块（否则后续语句被整段吞并）
                    if (
                        sql[after] !== undefined &&
                        sql[after] !== delimiter &&
                        !txnBeginWords.has(nextWord) &&
                        blockStack[blockStack.length - 1] !== 'declare' &&
                        hasEndAhead(end)
                    ) {
                        blockStack.push('begin');
                    }
                } else if (word === 'declare' && options.blockMode === 'procedural') {
                    // Oracle/MySQL PL-SQL 匿名块：DECLARE 变量段内的分号属声明结尾，仅语句起始且无块上下文时入块
                    // （T-SQL 的 DECLARE 是独立语句，'sql' 档位不处理）
                    if (!blockStack.length && !buffer.trim()) {
                        blockStack.push('declare');
                    }
                } else if (word === 'end') {
                    if (blockStack.length) {
                        blockStack.pop();
                        const after = skipSpace(sql, end);
                        if (isWordStart(sql[after])) {
                            const closer = readWord(sql, after);
                            if (endCloserWords.has(closer.word)) {
                                wordEnd = closer.end;
                            }
                        }
                    }
                } else if (word === 'case') {
                    // 表达式 CASE..END 与语句 CASE..END CASE 均以一个 END 闭合，计数自然平衡
                    if (options.blockMode !== 'none' && hasEndAhead(end)) {
                        blockStack.push('case');
                    }
                } else if (options.blockMode === 'procedural' && proceduralOpeners.has(word) && (blockStack.length || spDefinitionWordRe.test(buffer))) {
                    // 过程块起始词需有块上下文（已在块内，或本句为存储程序定义），且 IF 需后续存在 THEN
                    if (word !== 'if' || hasThenBeforeDelimiter(sql, end, delimiter)) {
                        blockStack.push(word);
                    }
                } else if (options.altQuoteLiteral && altQuoteWords.has(word) && sql[end] === "'" && sql[end + 1] && sql[end + 1] !== '\n') {
                    // Oracle q'[..]'：定界符内分号与单引号均为字面量内容
                    const open = sql[end + 1];
                    altClose = (altOpenCloseMap[open] ?? open) + "'";
                    state = 'altQuote';
                    wordEnd = end + 2;
                }
                buffer += sql.slice(i, wordEnd);
                hasCode = true;
                i = wordEnd - 1;
            } else {
                buffer += char;
                if (!/\s/.test(char)) {
                    hasCode = true;
                }
            }
        } else if (state === 'string') {
            if (stringBackslash && char === '\\' && nextChar !== undefined) {
                // 反斜杠转义：\' 不结束字符串，\\ 后的引号同样不结束
                buffer += char + nextChar;
                i++;
            } else if (char === stringClose) {
                if (stringDoubling && nextChar === stringClose) {
                    // 双写转义（'' / "" / `` / ]]）：仍属于同一字符串/标识符
                    buffer += char + nextChar;
                    i++;
                } else {
                    buffer += char;
                    stringClose = '';
                    state = 'normal';
                }
            } else {
                buffer += char;
            }
        } else if (state === 'dollarQuote') {
            if (sql.startsWith(dollarTag, i)) {
                buffer += dollarTag;
                i += dollarTag.length - 1;
                state = 'normal';
            } else {
                buffer += char;
            }
        } else if (state === 'altQuote') {
            if (sql.startsWith(altClose, i)) {
                buffer += altClose;
                i += altClose.length - 1;
                state = 'normal';
            } else {
                buffer += char;
            }
        } else if (state === 'lineComment') {
            // 注释原文保留于语句文本（不粘连、不改写 SQL），仅另记区域供补全掩码使用
            buffer += char;
            if (char === '\n') {
                commentRanges.push({ start: commentStart, end: i });
                commentStart = -1;
                state = 'normal';
            }
        } else if (state === 'blockComment' && char === '/' && nextChar === '*' && options.nestedBlockComment) {
            // PG 块注释可嵌套，内层 /* 需额外一层闭合
            blockCommentDepth++;
            buffer += '/*';
            i++;
        } else if (state === 'blockComment' && char === '*' && nextChar === '/') {
            blockCommentDepth--;
            buffer += '*/';
            i++; // 跳过闭合符 /
            if (blockCommentDepth === 0) {
                if (!execComment) {
                    commentRanges.push({ start: commentStart, end: i + 1 });
                }
                commentStart = -1;
                execComment = false;
                state = 'normal';
            }
        } else if (state === 'blockComment') {
            buffer += char;
        }
    }

    // 仅全文扫描需要产出语句（区域判定已提前中断，buffer 不完整）
    if (!stopped) {
        if (commentStart >= 0) {
            // 文本在注释内结束（未闭合注释）：区域掩码至末尾，避免注释内容参与解析
            commentRanges.push({ start: commentStart, end: sql.length });
        }
        if (buffer.trim() && hasCode) {
            if (blockStack.length) {
                // 块起始词到文末仍未闭合（如 CASE 表达式缺 END、向前探测把注释里的 END 当作配对）：
                // 说明本段并非复合块，关闭块感知重切当前语句，避免整段脚本被并为一条执行
                const tailStart = startPos;
                const tail = scanSql(sql.slice(tailStart), delimiter, { ...options, blockMode: 'none' }, Infinity).statements;
                for (const sub of tail) {
                    statements.push({ text: sub.text, start: sub.start + tailStart, end: sub.end + tailStart, textStart: sub.textStart + tailStart });
                }
            } else {
                // 最后一个语句（没有以分号结尾的情况）
                statements.push({ text: buffer.trim(), start: startPos, end: sql.length, textStart: startPos + leadingBlankLen(buffer) });
            }
        }
    }

    return { statements: stopped ? [] : statements, zone: toZone(state, stringIsIdentifier), commentRanges };
}

/**
 * 获取指定偏移量在 SQL 文本中所处的区域（与切割共用同一扫描器）。
 *
 * @param sql 完整文本
 * @param offset 偏移量（越界自动限制到 [0, length]）
 * @param opts 切割方言选项，见 SqlSplitOptions
 */
export function getSqlScanZone(sql: string, offset: number, opts: SqlSplitOptions = {}): ScanZone {
    const text = sql ?? '';
    return scanSql(text, ';', resolveSqlSplitOptions(opts), Math.min(Math.max(offset, 0), text.length)).zone;
}

/**
 * 获取光标所在的SQL语句
 * @param fullSql 完整的SQL文本
 * @param position 光标位置
 * @param model Monaco编辑器模型
 * @param opts 切割方言选项，见 SqlSplitOptions
 */
export function getCurrentStatement(fullSql: string, position: Position, model: editor.ITextModel, opts: SqlSplitOptions = {}): string | null {
    // 使用通用SQL解析器来分割SQL语句，并记录每个语句的位置
    const statements = splitSqlStatements(fullSql, ';', opts);

    // 根据光标位置找到对应的SQL语句
    if (position) {
        const offset = model.getOffsetAt(position);

        // 遍历所有语句，找到光标所在的语句
        for (let i = 0; i < statements.length; i++) {
            const stmt = statements[i];
            // 光标在语句范围内（包括末尾分号）
            if (offset >= stmt.start && offset <= stmt.end) {
                return stmt.text;
            }
            // 光标在语句分号后一个位置
            if (offset === stmt.end + 1) {
                return stmt.text;
            }
        }

        // 如果光标处没有SQL，则执行光标前的最后一个SQL
        for (let i = statements.length - 1; i >= 0; i--) {
            const stmt = statements[i];
            if (offset > stmt.end) {
                return stmt.text;
            }
        }
    }

    return null;
}
