import { splitSqlStatements } from '../../component/sqleditor/utils/sqlParser';
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
 * 提取光标偏移量所在的完整 SQL 语句文本（感知字符串/注释中的分号）。
 * 复用通用解析器 splitSqlStatements，替代旧的简单 lastIndexOf(';') 切分，
 * 避免 `WHERE remark = 'a;b'` 等语句被错误截断。
 *
 * @param fullSql 编辑器完整文本
 * @param offset 光标在全文中的偏移量
 * @returns 光标所在语句文本；光标位于语句间隙（分号后未输入）时返回空串
 */
export function extractStatementAt(fullSql: string, offset: number): string {
    const statements = splitSqlStatements(fullSql ?? '');
    for (const stmt of statements) {
        // 光标位于语句范围内（含结束分号所在位置）
        if (offset >= stmt.start && offset <= stmt.end) {
            return stmt.text;
        }
    }
    return '';
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
 * 从 SQL 语句中解析表上下文（表名、别名、库名）。
 * 支持 FROM/JOIN/UPDATE 子句、`db.table` 形式及标识符引用符包裹的名称。
 *
 * @param sql 完整 SQL 语句文本
 * @param alias 指定匹配的表别名；为空时返回语句中的第一个表
 * @param defaultDb 默认库名
 */
export function resolveTableContext(sql: string, alias: string = '', defaultDb: string): TableCtx | undefined {
    // 去除多余的换行、空格和制表符
    sql = sql.replace(/[\r\n\s\t]+/g, ' ');

    // 提取所有可能的表名和别名
    const regex = /(?:FROM|JOIN|UPDATE)\s+(\S+)\s+(?:AS\s+)?(\S+)/gi;
    let matches;
    const tables: TableCtx[] = [];

    while ((matches = regex.exec(sql)) !== null) {
        let tableName = matches[1].replace(/[`"]/g, '');
        let db = defaultDb;
        if (tableName.indexOf('.') >= 0) {
            const info = tableName.split('.');
            db = info[0];
            if (defaultDb.indexOf('/') > 0) {
                db = defaultDb.substring(0, defaultDb.indexOf('/') + 1) + db;
            }
            tableName = info[1];
        }
        const tableAlias = matches[2] ? matches[2].replace(/[`"]/g, '') : tableName;
        tables.push({ tableName, tableAlias, db });
    }

    if (alias) {
        // 如果指定了别名参数，则返回对应的表名（忽略大小写，兼容 FROM Users U + U. 等写法）
        const lowerAlias = alias.toLowerCase();
        return tables.find((t) => t.tableAlias?.toLowerCase() === lowerAlias || t.tableName?.toLowerCase() === lowerAlias);
    }
    // 如果未指定别名参数，则返回第一个表名
    return tables.length > 0 ? tables[0] : undefined;
}

/** 光标所处区域 */
export type CursorZone = 'code' | 'string' | 'comment';

/**
 * 判断光标偏移量所处区域：代码、字符串字面量（单引号）或注释。
 * 字符串/注释内不提供代码提示，避免噪音建议。
 * 规则与 splitSqlStatements 一致：支持 \' 转义、-- 行注释与 /* *\/ 块注释；
 * 双引号/反引号不视为字符串（多数方言中为标识符引用符）。
 *
 * @param fullSql 编辑器完整文本
 * @param offset 光标在全文中的偏移量
 */
export function resolveCursorZone(fullSql: string, offset: number): CursorZone {
    const sql = fullSql ?? '';
    const end = Math.min(Math.max(offset, 0), sql.length);
    let state: 'normal' | 'string' | 'singleLineComment' | 'multiLineComment' = 'normal';

    for (let i = 0; i < end; i++) {
        const char = sql[i];
        const nextChar = sql[i + 1];
        if (state === 'normal') {
            if (char === '-' && nextChar === '-') {
                state = 'singleLineComment';
                i++;
            } else if (char === '/' && nextChar === '*') {
                state = 'multiLineComment';
                i++;
            } else if (char === "'") {
                state = 'string';
            }
        } else if (state === 'string') {
            if (char === '\\') {
                i++;
            } else if (char === "'") {
                state = 'normal';
            }
        } else if (state === 'singleLineComment') {
            if (char === '\n') {
                state = 'normal';
            }
        } else if (char === '*' && nextChar === '/') {
            state = 'normal';
            i++;
        }
    }
    return state === 'normal' ? 'code' : state === 'string' ? 'string' : 'comment';
}
