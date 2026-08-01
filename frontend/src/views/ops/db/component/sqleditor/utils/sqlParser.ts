import { editor, Position } from 'monaco-editor';

export interface SqlStatement {
    text: string;
    start: number;
    end: number;
}

/**
 * 通用SQL解析器，用于提取SQL语句及其位置信息
 * @param sql 完整的SQL文本
 * @param delimiter SQL语句分隔符，默认为分号
 */
export function splitSqlStatements(sql: string, delimiter: string = ';'): SqlStatement[] {
    let state = 'normal';
    let buffer = '';
    let result: SqlStatement[] = [];
    let inString: string | null = null; // 用于记录当前字符串的引号类型（' 或 "）
    let startPos = 0;

    for (let i = 0; i < sql.length; i++) {
        const char = sql[i];
        const nextChar = sql[i + 1];

        if (state === 'normal') {
            if (char === '-' && nextChar === '-') {
                state = 'singleLineComment';
                i++; // 跳过下一个字符
            } else if (char === '/' && nextChar === '*') {
                state = 'multiLineComment';
                i++; // 跳过下一个字符
            } else if (char === "'" || char === '"') {
                state = 'string';
                inString = char;
                buffer += char;
            } else if (char === delimiter) {
                if (buffer.trim()) {
                    result.push({
                        text: buffer.trim(),
                        start: startPos,
                        end: i,
                    });
                }
                buffer = '';
                startPos = i + 1;
            } else {
                buffer += char;
            }
        } else if (state === 'string') {
            buffer += char;
            if (char === '\\') {
                // 处理转义字符
                buffer += nextChar;
                i++;
            } else if (char === inString) {
                state = 'normal';
                inString = null;
            }
        } else if (state === 'singleLineComment') {
            if (char === '\n') {
                state = 'normal';
            }
        } else if (state === 'multiLineComment') {
            if (char === '*' && nextChar === '/') {
                buffer += nextChar;
                state = 'normal';
                i++; // 跳过下一个字符
            }
        }
    }

    // 处理最后一个语句（没有以分号结尾的情况）
    if (buffer.trim()) {
        result.push({
            text: buffer.trim(),
            start: startPos,
            end: sql.length,
        });
    }

    return result;
}

/**
 * 获取光标所在的SQL语句
 * @param fullSql 完整的SQL文本
 * @param position 光标位置
 * @param model Monaco编辑器模型
 */
export function getCurrentStatement(fullSql: string, position: Position, model: editor.ITextModel): string | null {
    // 使用通用SQL解析器来分割SQL语句，并记录每个语句的位置
    const statements = splitSqlStatements(fullSql);

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
