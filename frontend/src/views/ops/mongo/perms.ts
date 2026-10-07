/**
 * Mongo 模块的权限码常量。
 *
 * 字面值与后端 `mongodoc`（数据面分级）及 API 路由声明（实例管理、导出）同源：
 * 各组件里再各写一份字符串，改权限时就会出现「界面按钮可见、接口 403」这种最难查的失配。
 */
export const perms = {
    /** 实例登记与删除（平台侧元数据，不代表能读数据） */
    mongoSave: 'mongo:save',
    mongoDel: 'mongo:del',
    /** 集合与索引等结构变更 */
    ddlSave: 'mongo:ddl:save',
    ddlDel: 'mongo:ddl:del',
    /** 文档写入与删除 */
    dataSave: 'mongo:data:save',
    dataDel: 'mongo:data:del',
    /** 导出：把数据带走的路径，默认不授予公共角色 */
    dataExport: 'mongo:data:export',
} as const;
