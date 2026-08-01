import { registerCompletionItemProvider } from '@/components/monaco/completionItemProvider';
import { editor, languages, Position, type IRange } from 'monaco-editor';
import { DbInst } from '../db';
import { getDbDialect, type EditorCompletionItem } from '../dialect/index';
import type { DbTableInfo } from '../types';

function registerCompletions(
    completions: EditorCompletionItem[],
    suggestions: languages.CompletionItem[],
    kind: languages.CompletionItemKind,
    range: IRange
) {
    // mysql关键字
    completions.forEach((item: EditorCompletionItem) => {
        let { label, insertText, description } = item;
        suggestions.push({
            label: { label, description },
            kind,
            insertText: insertText || label,
            range,
        });
    });
}

/**
 * 注册数据库表、字段等信息提示
 *
 * @param dbId 数据库id
 * @param db 库名
 * @param dbs 该库所有库名
 * @param dbType 数据库类型
 */
export function registerDbCompletionItemProvider(dbId: number, db: string, dbs: string[] = [], dbType: string) {
    let dbDialect = getDbDialect(dbType);
    let dbDialectInfo = dbDialect.getInfo();
    let { keywords, operators, functions, variables } = dbDialectInfo.editorCompletions;
    registerCompletionItemProvider('sql', {
        triggerCharacters: ['.', ' '],
        provideCompletionItems: async (model: editor.ITextModel, position: Position): Promise<languages.CompletionList | null | undefined> => {
            let word = model.getWordUntilPosition(position);
            const dbInst = await DbInst.getInstA(dbId);
            const { lineNumber, column } = position;
            const { startColumn, endColumn } = word;

            // 当前行文本
            let lineContent = model.getLineContent(lineNumber);
            // 注释行不需要代码提示
            if (lineContent.startsWith('--')) {
                return { suggestions: [] };
            }

            let range = {
                startLineNumber: lineNumber,
                endLineNumber: lineNumber,
                startColumn,
                endColumn,
            };

            //  光标前文本
            const textBeforePointer = model.getValueInRange({
                startLineNumber: lineNumber,
                startColumn: 0,
                endLineNumber: lineNumber,
                endColumn: column,
            });
            // // const nextTokens = textAfterPointer.trim().split(/\s+/)
            // // const nextToken = nextTokens[0].toLowerCase()
            const tokens = textBeforePointer.trim().split(/\s+/);
            let lastToken = tokens[tokens.length - 1].toLowerCase();
            const secondToken = (tokens.length > 2 && tokens[tokens.length - 2].toLowerCase()) || '';

            // 获取光标所在行之前的所有文本内容
            const textBeforeCursor = model.getValueInRange({
                startLineNumber: 1,
                startColumn: 0,
                endLineNumber: lineNumber,
                endColumn: column,
            });

            // 获取光标所在行之后的所有文本内容
            const textAfterCursor = model.getValueInRange({
                startLineNumber: lineNumber,
                startColumn: column,
                endLineNumber: model.getLineCount(),
                endColumn: model.getLineMaxColumn(model.getLineCount()),
            });

            // 检测光标前后文本中的分号位置，确定完整 SQL 语句的范围
            const start = textBeforeCursor.lastIndexOf(';');
            const end = textAfterCursor.indexOf(';');

            let sqlStatement = '';
            // 如果光标前后都有分号，则取二者之间的文本作为完整 SQL 语句
            if (start !== -1 && end !== -1) {
                sqlStatement = textBeforeCursor.substring(start + 1) + textAfterCursor.substring(0, end);
            }
            // 如果只有光标前面有分号，则取分号后的文本作为完整 SQL 语句
            else if (start !== -1) {
                sqlStatement = textBeforeCursor.substring(start + 1) + textAfterCursor;
            }
            // 如果只有光标后面有分号，则取分号前的文本作为完整 SQL 语句
            else if (end !== -1) {
                sqlStatement = textBeforeCursor + textAfterCursor.substring(0, end);
            }
            // 如果光标前后都没有分号，则取整个文本作为完整 SQL 语句
            else {
                sqlStatement = textBeforeCursor + textAfterCursor;
            }

            let suggestions: languages.CompletionItem[] = [];

            // 库名提示
            if (dbs && dbs.length > 0) {
                dbs.forEach((a: string) => {
                    suggestions.push({
                        label: {
                            label: a,
                            description: 'schema',
                        },
                        kind: languages.CompletionItemKind.Folder,
                        insertText: dbDialect.quoteIdentifier(a),
                        range,
                    });
                });
            }

            let alias = '';
            if (lastToken.indexOf('.') > -1 || secondToken.indexOf('.') > -1) {
                // 如果是.触发代码提示，则进行【 库.表名联想 】 或 【 表别名.表字段联想 】
                alias = lastToken.substring(0, lastToken.lastIndexOf('.'));
                if (lastToken.trim().startsWith('.')) {
                    alias = secondToken;
                }
                if (!alias && secondToken.indexOf('.') > -1) {
                    alias = secondToken.substring(secondToken.indexOf('.') + 1);
                }

                // 如果字符串粘连起了如:'a.creator,a.',需要重新取出别名
                let aliasArr = lastToken.split(',');
                if (aliasArr.length > 1) {
                    lastToken = aliasArr[aliasArr.length - 1];
                    alias = lastToken.substring(0, lastToken.lastIndexOf('.'));
                    if (lastToken.trim().startsWith('.')) {
                        alias = secondToken;
                    }
                }

                // 如果是【库.表名联想】.前的字符串是库名
                if (dbs?.filter((a) => alias === a?.toLowerCase()).length > 0) {
                    let dbName = alias;
                    if (db.indexOf('/') > 0) {
                        dbName = db.substring(0, db.indexOf('/') + 1) + alias;
                    }
                    return await dbInst.loadTableSuggestions(dbDialect, dbName, range);
                }
                // 表下列名联想  .前的字符串是表名或表别名
                const sqlInfo = getTableName4SqlCtx(sqlStatement, alias, db);
                // 提出到表名，则将表对应的字段也添加进提示建议
                if (sqlInfo) {
                    return await dbInst.loadTableColumnSuggestions(dbDialect, sqlInfo.db, sqlInfo.tableName, range);
                }
            }

            // 空格触发也会提示字段信息
            const sqlInfo = getTableName4SqlCtx(sqlStatement, alias, db);
            if (sqlInfo) {
                const columnSuggestions = await dbInst.loadTableColumnSuggestions(dbDialect, sqlInfo.db, sqlInfo.tableName, range);
                suggestions.push(...columnSuggestions.suggestions);
            }

            // 当前库的表名联想
            const tables = await dbInst.loadTables(db);
            tables.forEach((tableMeta: DbTableInfo, index: number) => {
                const { tableName, tableComment } = tableMeta;
                suggestions.push({
                    label: {
                        label: tableName + ' - ' + tableComment,
                        description: 'table',
                    },
                    kind: languages.CompletionItemKind.File,
                    detail: tableComment,
                    insertText: dbDialect.quoteIdentifier(tableName),
                    range,
                    sortText: 300 + index + '',
                });
            });

            registerCompletions(keywords, suggestions, languages.CompletionItemKind.Keyword, range);
            registerCompletions(operators, suggestions, languages.CompletionItemKind.Operator, range);
            registerCompletions(functions, suggestions, languages.CompletionItemKind.Function, range);
            registerCompletions(variables, suggestions, languages.CompletionItemKind.Variable, range);

            // 默认提示
            return {
                suggestions: suggestions,
            };
        },
    });
}

function getTableName4SqlCtx(
    sql: string,
    alias: string = '',
    defaultDb: string
):
    | {
          tableName: string;
          tableAlias: string;
          db: string;
      }
    | undefined {
    // 去除多余的换行、空格和制表符
    sql = sql.replace(/[\r\n\s\t]+/g, ' ');

    // 提取所有可能的表名和别名
    const regex = /(?:FROM|JOIN|UPDATE)\s+(\S+)\s+(?:AS\s+)?(\S+)/gi;
    let matches;
    const tables = [];

    // 使用正则表达式匹配所有的表和别名
    while ((matches = regex.exec(sql)) !== null) {
        let tableName = matches[1].replace(/[`"]/g, '');
        let db = defaultDb;
        if (tableName.indexOf('.') >= 0) {
            let info = tableName.split('.');
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
        // 如果指定了别名参数，则返回对应的表名
        return tables.find((t) => t.tableAlias === alias);
    } else {
        // 如果未指定别名参数，则返回第一个表名
        return tables.length > 0 ? tables[0] : undefined;
    }
}
