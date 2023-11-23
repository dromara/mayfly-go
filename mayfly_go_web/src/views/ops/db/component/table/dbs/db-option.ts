export interface sqlType {
    udtName: string;
    dataType: string;
    desc: string;
    space: string;
    range?: string;
}

export interface DbOption {
    /** 生成字段类型列表 */
    getTypeList(): sqlType[];
    /** 生成创建表sql */
    getCreateTableSql(tableData: any): string;
    /** 生成创建索引sql */
    getCreateIndexSql(tableData: any): string;
    /** 生成编辑列sql */
    getModifyColumnSql(tableName: string, changeData: { del: any[]; add: any[]; upd: any[] }): string;
    /** 生成编辑索引sql */
    getModifyIndexSql(tableName: string, changeData: { del: any[]; add: any[]; upd: any[] }): string;
}

export const DbType = {
    mysql: 'mysql',
    postgresql: 'postgres',
};
