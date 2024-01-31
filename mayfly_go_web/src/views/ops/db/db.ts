/* eslint-disable no-unused-vars */
import { dbApi } from './api';
import { getTextWidth } from '@/common/utils/string';
import SqlExecBox from './component/sqleditor/SqlExecBox';
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';
import { editor, languages, Position } from 'monaco-editor';

import { registerCompletionItemProvider } from '@/components/monaco/completionItemProvider';
import { DbDialect, EditorCompletionItem, getDbDialect } from './dialect';
import { type RemovableRef, useLocalStorage } from '@vueuse/core';

const hintsStorage: RemovableRef<Map<string, any>> = useLocalStorage('db-table-hints', new Map());
const tableStorage: RemovableRef<Map<string, any>> = useLocalStorage('db-tables', new Map());

const dbInstCache: Map<number, DbInst> = new Map();

export class DbInst {
    /**
     * 标签路径
     */
    tagPath: string;

    /**
     * 实例id
     */
    id: number;

    /**
     * ip:port
     */
    host: string;

    /**
     * 实例名
     */
    name: string;

    /**
     * 数据库类型, mysql postgres
     */
    type: string;

    /**
     * dbName -> db
     */
    dbs: Map<string, Db> = new Map();

    /** 数据库，多个用空格隔开 */
    databases: string[];

    /**
     * 默认查询分页数量
     */
    static DefaultLimit = 25;

    /**
     * 获取指定数据库实例，若不存在则新建并缓存
     * @param dbName 数据库名
     * @returns db实例
     */
    getDb(dbName: string) {
        if (!dbName) {
            throw new Error('dbName不能为空');
        }
        let key = `${this.id}_${dbName}`;
        let db = this.dbs.get(key);
        if (db) {
            return db;
        }
        console.info(`new db -> dbId: ${this.id}, dbName: ${dbName}`);
        db = new Db();
        db.name = dbName;
        this.dbs.set(key, db);
        return db;
    }

    // 获取数据库实例方言
    getDialect(): DbDialect {
        return getDbDialect(this.type);
    }

    /**
     * 加载数据库表信息
     * @param dbName 数据库名
     * @param reload 是否重新请求接口获取数据
     * @returns 表信息
     */
    async loadTables(dbName: string, reload?: boolean) {
        const db = this.getDb(dbName);
        let key = this.dbTablesKey(dbName);
        let tables = tableStorage.value.get(key);
        // 优先从 table 缓存中获取
        if (!reload && tables) {
            db.tables = tables;
            return tables;
        }
        // 重置列信息缓存与表提示信息
        db.columnsMap?.clear();
        console.log(`load tables -> dbName: ${dbName}`);
        tables = await dbApi.tableInfos.request({ id: this.id, db: dbName });
        tableStorage.value.set(key, tables);
        db.tables = tables;

        // 异步加载表提示信息
        this.loadDbHints(dbName, true).then(() => {});
        return tables;
    }

    async loadTableSuggestions(dbDialect: DbDialect, dbName: string, range: any, reload?: boolean) {
        const tables = await this.loadTables(dbName, reload);
        // 表名联想
        let suggestions: languages.CompletionItem[] = [];
        tables?.forEach((tableMeta: any, index: any) => {
            const { tableName, tableComment } = tableMeta;
            suggestions.push({
                label: {
                    label: tableName + ' - ' + tableComment,
                    description: 'table',
                },
                kind: monaco.languages.CompletionItemKind.File,
                detail: tableComment,
                insertText: dbDialect.quoteIdentifier(tableName) + ' ',
                range,
                sortText: 300 + index + '',
            });
        });
        return { suggestions };
    }

    /** 加载列信息提示 */
    async loadTableColumnSuggestions(dbDialect: DbDialect, db: string, tableName: string, range: any) {
        let dbHits = await this.loadDbHints(db);
        let columns = dbHits[tableName];
        let suggestions: languages.CompletionItem[] = [];
        columns?.forEach((a: string, index: any) => {
            // 字段数据格式  字段名 字段注释，  如： create_time  [datetime][创建时间]
            const nameAndComment = a.split('  ');
            const fieldName = nameAndComment[0];
            suggestions.push({
                label: {
                    label: a,
                    description: 'column',
                },
                kind: monaco.languages.CompletionItemKind.Property,
                detail: '', // 不显示detail, 否则选中时备注等会被遮挡
                insertText: dbDialect.quoteIdentifier(fieldName) + ' ', // create_time
                range,
                sortText: 100 + index + '', // 使用表字段声明顺序排序,排序需为字符串类型
            });
        });

        return { suggestions };
    }

    /**
     * 获取表的所有列信息
     * @param dbName 数据库名
     * @param table 表名
     */
    async loadColumns(dbName: string, table: string) {
        const db = this.getDb(dbName);
        // 优先从 table map中获取
        let columns = db.getColumns(table);
        if (columns) {
            return columns;
        }
        console.log(`load columns -> dbName: ${dbName}, table: ${table}`);
        columns = await dbApi.columnMetadata.request({
            id: this.id,
            db: dbName,
            tableName: table,
        });
        db.columnsMap.set(table, columns);
        return columns;
    }

    /**
     * 获取指定表的指定信息
     * @param table 表名
     */
    async loadTableColumn(dbName: string, table: string, columnName?: string) {
        // 确保该表的列信息都已加载
        await this.loadColumns(dbName, table);
        return this.getDb(dbName).getColumn(table, columnName);
    }

    dbTableHintsKey(dbName: string) {
        return `db-table-hints_${this.id}_${dbName}`;
    }

    dbTablesKey(dbName: string) {
        return `db-tables_${this.id}_${dbName}`;
    }

    /**
     * 获取库信息提示
     */
    async loadDbHints(dbName: string, reload?: boolean) {
        const db = this.getDb(dbName);
        let key = this.dbTableHintsKey(dbName);
        let hints = hintsStorage.value.get(key);
        if (!reload && hints) {
            db.tableHints = hints;
            return hints;
        }
        console.log(`load db-hits -> dbName: ${dbName}`);
        hints = await dbApi.hintTables.request({ id: this.id, db: db.name });
        db.tableHints = hints;
        hintsStorage.value.set(key, hints);
        return hints;
    }

    /**
     * 执行sql
     *
     * @param sql sql
     * @param remark 执行备注
     */
    async runSql(dbName: string, sql: string, remark: string = '') {
        return await dbApi.sqlExec.request({
            id: this.id,
            db: dbName,
            sql: sql.trim(),
            remark,
        });
    }

    /**
     * 执行sql(可取消的)
     *
     * @param sql sql
     * @param remark 执行备注
     */
    execSql(dbName: string, sql: string, remark: string = '') {
        let dbId = this.id;
        return dbApi.sqlExec.useApi({
            id: dbId,
            db: dbName,
            sql: sql.trim(),
            remark,
        });
    }

    /**
     * 获取count sql
     * @param table 表名
     * @param condition 条件
     * @returns count sql
     */
    getDefaultCountSql = (table: string, condition?: string) => {
        return `SELECT COUNT(*) count FROM ${this.wrapName(table)} ${condition ? 'WHERE ' + condition : ''}`;
    };

    // 获取指定表的默认查询sql
    getDefaultSelectSql(db: string, table: string, condition: string, orderBy: string, pageNum: number, limit: number = DbInst.DefaultLimit) {
        return getDbDialect(this.type).getDefaultSelectSql(db, table, condition, orderBy, pageNum, limit);
    }

    /**
     * 生成指定数据的insert语句
     * @param dbName 数据库名
     * @param table 表名
     * @param datas 要生成的数据
     */
    async genInsertSql(dbName: string, table: string, datas: any[], skipNull = false) {
        if (!datas) {
            return '';
        }

        const columns = await this.loadColumns(dbName, table);
        const sqls = [];
        for (let data of datas) {
            let colNames = [];
            let values = [];
            for (let column of columns) {
                const colName = column.columnName;
                if (skipNull && data[colName] == null) {
                    continue;
                }
                colNames.push(this.wrapName(colName));
                values.push(DbInst.wrapValueByType(data[colName]));
            }
            sqls.push(`INSERT INTO ${this.wrapName(table)} (${colNames.join(', ')}) VALUES(${values.join(', ')})`);
        }
        return sqls.join(';\n') + ';';
    }

    /**
     * 生成根据主键更新语句
     * @param dbName 数据库名
     * @param table 表名
     * @param columnValue 要更新的列以及对应的值 field->columnName; value->columnValue
     * @param rowData 表的一行完整数据（需要获取主键信息）
     */
    async genUpdateSql(dbName: string, table: string, columnValue: {}, rowData: {}) {
        let schema = '';
        let dbArr = dbName.split('/');
        if (dbArr.length == 2) {
            schema = this.wrapName(dbArr[1]) + '.';
        }

        let sql = `UPDATE ${schema}${this.wrapName(table)} SET `;
        // 主键列信息
        const primaryKey = await this.loadTableColumn(dbName, table);
        let primaryKeyType = primaryKey.columnType;
        let primaryKeyName = primaryKey.columnName;
        let primaryKeyValue = rowData[primaryKeyName];
        const dialect = this.getDialect();
        for (let k of Object.keys(columnValue)) {
            const v = columnValue[k];
            // 更新字段列信息
            const updateColumn = await this.loadTableColumn(dbName, table, k);
            sql += ` ${this.wrapName(k)} = ${DbInst.wrapColumnValue(updateColumn.columnType, v, dialect)},`;
        }

        sql = sql.substring(0, sql.length - 1);
        return (sql += ` WHERE ${this.wrapName(primaryKeyName)} = ${DbInst.wrapColumnValue(primaryKeyType, primaryKeyValue)} ;`);
    }

    /**
     * 生成根据主键删除的sql语句
     * @param table 表名
     * @param datas 要删除的记录
     */
    async genDeleteByPrimaryKeysSql(db: string, table: string, datas: any[]) {
        const primaryKey = await this.loadTableColumn(db, table);
        const primaryKeyColumnName = primaryKey.columnName;
        const ids = datas.map((d: any) => `${DbInst.wrapColumnValue(primaryKey.columnType, d[primaryKeyColumnName])}`).join(',');
        return `DELETE FROM ${this.wrapName(table)} WHERE ${this.wrapName(primaryKeyColumnName)} IN (${ids})`;
    }

    /*
     * 弹框提示是否执行sql
     */
    promptExeSql = (db: string, sql: string, cancelFunc: any = null, successFunc: any = null) => {
        SqlExecBox({
            sql,
            dbId: this.id,
            db,
            dbType: this.getDialect().getInfo().formatSqlDialect,
            runSuccessCallback: successFunc,
            cancelCallback: cancelFunc,
        });
    };

    /**
     * 包裹数据库表名、字段名等，避免使用关键字为字段名或表名时报错
     * @param table
     * @param condition
     * @returns
     */
    wrapName = (name: string) => {
        return this.getDialect().quoteIdentifier(name);
    };

    /**
     * 获取或新建dbInst，如果缓存中不存在则新建，否则直接返回
     * @param inst 数据库实例，后端返回的列表接口中的信息
     * @returns DbInst
     */
    static getOrNewInst(inst: any) {
        if (!inst) {
            throw new Error('inst不能为空');
        }
        let dbInst = dbInstCache.get(inst.id);
        if (dbInst) {
            return dbInst;
        }
        console.info(`new dbInst: ${inst.id}, tagPath: ${inst.tagPath}`);
        dbInst = new DbInst();
        dbInst.tagPath = inst.tagPath;
        dbInst.id = inst.id;
        dbInst.host = inst.host;
        dbInst.name = inst.name;
        dbInst.type = inst.type;
        dbInst.databases = inst.databases;

        dbInstCache.set(dbInst.id, dbInst);
        return dbInst;
    }

    /**
     * 获取数据库实例id，若不存在，则新建一个并缓存
     * @param dbId 数据库实例id
     * @param dbType 第一次获取时为必传项，即第一次创建时
     * @returns 数据库实例
     */
    static getInst(dbId?: number): DbInst {
        if (!dbId) {
            throw new Error('dbId不能为空');
        }
        let dbInst = dbInstCache.get(dbId);
        if (dbInst) {
            return dbInst;
        }
        throw new Error('dbInst不存在! 请在合适调用点使用DbInst.newInst()新建该实例');
    }

    /**
     * 清空所有实例缓存信息
     */
    static clearAll() {
        dbInstCache.clear();
    }

    /**
     * 根据返回值包装值，若值为字符串类型则添加''
     * @param val 值
     * @returns 包装后的值
     */
    static wrapValueByType = (val: any) => {
        if (val == null) {
            return 'NULL';
        }
        if (typeof val == 'number') {
            return val;
        }
        return `'${val}'`;
    };

    /**
     * 根据字段类型包装字段值，如为字符串等则添加‘’，数字类型则直接返回即可
     */
    static wrapColumnValue(columnType: string, value: any, dbDialect?: DbDialect) {
        if (this.isNumber(columnType)) {
            return value;
        }
        if (!dbDialect) {
            return `'${value}'`;
        }
        return dbDialect.wrapStrValue(columnType, value);
    }

    /**
     * 判断字段类型是否为数字类型
     * @param columnType 字段类型
     * @returns
     */
    static isNumber(columnType: string) {
        return columnType.match(/int|double|float|number|decimal|byte|bit/gi);
    }

    /**
     *
     * @param str 字符串
     * @param tableData 表数据
     * @param flag 标志
     * @returns 列宽度
     */
    static flexColumnWidth = (prop: any, tableData: any) => {
        if (!prop || !prop.length || prop.length === 0 || prop === undefined) {
            return;
        }

        // 获取列名称的长度 加上排序图标长度、abc为字段类型简称占位符
        const columnWidth: number = getTextWidth(prop + 'abc') + 23;
        // prop为该列的字段名(传字符串);tableData为该表格的数据源(传变量);
        if (!tableData || !tableData.length || tableData.length === 0 || tableData === undefined) {
            return columnWidth;
        }

        // 获取该列中最长的数据(内容)
        let maxWidthText = '';
        // 获取该列中最长的数据(内容)
        for (let i = 0; i < tableData.length; i++) {
            let nowValue = tableData[i][prop];
            if (!nowValue) {
                continue;
            }
            // 转为字符串比较长度
            let nowText = nowValue + '';
            if (nowText.length > maxWidthText.length) {
                maxWidthText = nowText;
            }
        }
        const contentWidth: number = getTextWidth(maxWidthText) + 15;
        const flexWidth: number = contentWidth > columnWidth ? contentWidth : columnWidth;
        return flexWidth > 500 ? 500 : flexWidth;
    };
}

/**
 * 数据库实例信息
 */
class Db {
    name: string; // 库名
    tables: []; // 数据库实例表信息
    columnsMap: Map<string, any> = new Map(); // table -> columns
    tableHints: any = null; // 提示词

    /**
     * 获取指定表列信息（前提需要dbInst.loadColumns）
     * @param table 表名
     */
    getColumns(table: string) {
        return this.columnsMap.get(table);
    }

    /**
     * 获取指定表中的指定列名信息，若列名为空则默认返回主键
     * @param table 表名
     * @param columnName 列名
     */
    getColumn(table: string, columnName: string = '') {
        const cols = this.getColumns(table);
        if (!columnName) {
            const col = cols.find((c: any) => c.isPrimaryKey);
            return col || cols[0];
        }
        return cols.find((c: any) => c.columnName == columnName);
    }
}

export enum TabType {
    /**
     * 表数据
     */
    TableData,

    /**
     * 查询框
     */
    Query,

    /**
     * 表操作
     */
    TablesOp,
}

export class TabInfo {
    label: string;

    /**
     * tab唯一key。与name都一致
     */
    key: string;

    /**
     * 菜单树节点key
     */
    treeNodeKey: string;

    /**
     * 数据库实例id
     */
    dbId: number;

    /**
     * 库名
     */
    db: string = '';

    /**
     * tab 类型
     */
    type: TabType;

    /**
     * tab需要的其他信息
     */
    params: any;

    getNowDbInst() {
        return DbInst.getInst(this.dbId);
    }

    getNowDb() {
        return this.getNowDbInst().getDb(this.db);
    }
}

function registerCompletions(
    completions: EditorCompletionItem[],
    suggestions: languages.CompletionItem[],
    kind: monaco.languages.CompletionItemKind,
    range: any
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
export function registerDbCompletionItemProvider(dbId: number, db: string, dbs: any[] = [], dbType: string) {
    let dbDialect = getDbDialect(dbType);
    let dbDialectInfo = dbDialect.getInfo();
    let { keywords, operators, functions, variables } = dbDialectInfo.editorCompletions;
    registerCompletionItemProvider('sql', {
        triggerCharacters: ['.', ' '],
        provideCompletionItems: async (model: editor.ITextModel, position: Position): Promise<languages.CompletionList | null | undefined> => {
            let word = model.getWordUntilPosition(position);
            const dbInst = DbInst.getInst(dbId);
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
                dbs.forEach((a: any) => {
                    suggestions.push({
                        label: {
                            label: a,
                            description: 'schema',
                        },
                        kind: monaco.languages.CompletionItemKind.Folder,
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
            tables.forEach((tableMeta: any, index: any) => {
                const { tableName, tableComment } = tableMeta;
                suggestions.push({
                    label: {
                        label: tableName + ' - ' + tableComment,
                        description: 'table',
                    },
                    kind: monaco.languages.CompletionItemKind.File,
                    detail: tableComment,
                    insertText: dbDialect.quoteIdentifier(tableName) + ' ',
                    range,
                    sortText: 300 + index + '',
                });
            });

            registerCompletions(keywords, suggestions, monaco.languages.CompletionItemKind.Keyword, range);
            registerCompletions(operators, suggestions, monaco.languages.CompletionItemKind.Operator, range);
            registerCompletions(functions, suggestions, monaco.languages.CompletionItemKind.Function, range);
            registerCompletions(variables, suggestions, monaco.languages.CompletionItemKind.Variable, range);

            // 默认提示
            return {
                suggestions: suggestions,
            };
        },
    });
}

function getTableName4SqlCtx(sql: string, alias: string = '', defaultDb: string): { tableName: string; tableAlias: string; db: string } | undefined {
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
