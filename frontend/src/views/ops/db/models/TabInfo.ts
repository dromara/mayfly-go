import { DbInst } from '../db';

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

/**
 * tab 参数信息（getNowDbInfo 返回的实例信息 + 各 tab 类型附加字段）
 */
export interface TabParams {
    tagPath: string;
    id: number;
    name: string;
    type: string;
    host: string;
    dbName: string;
    /** 表数据tab: 表名 */
    table?: string;
    /** 查询tab: sql模板名 */
    sqlName?: string;
    /** 表操作tab: 库名 */
    db?: string;
    /** 查询tab: 该实例所有库名 */
    dbs?: string[];
    [key: string]: unknown;
}

/**
 * tab 组件 ref（各子组件通过 defineExpose 暴露的方法集合）
 */
export interface TabComponentRef {
    /** 激活当前tab（DbTableDataOp/DbSqlEditor 暴露） */
    active?: () => void;
    [key: string]: unknown;
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
    params: TabParams;

    /**
     * 组件ref
     */
    componentRef: TabComponentRef | null = null;

    getNowDbInst() {
        return DbInst.getInst(this.dbId);
    }

    getNowDb() {
        return this.getNowDbInst().getDb(this.db);
    }
}
