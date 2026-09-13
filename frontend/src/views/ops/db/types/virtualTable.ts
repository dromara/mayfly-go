/**
 * DB 模块虚拟表格扩展类型定义
 *
 * 导出策略是表格右键菜单「导出 / 生成」子菜单的唯一数据源：
 * 菜单项的文案、图标、权限、可见性全部由策略自描述，组件不硬编码任何格式。
 */

/** 导出策略：新增导出/生成格式只需 registerExportStrategy 注册，无需改动表格组件（开闭原则） */
export interface TableExportStrategy {
    /** 策略唯一 key，如 'csv'、'excel'、'insertSql' */
    key: string;
    /** i18n key，用于右键菜单文案 */
    labelI18nKey: string;
    /** 图标名 (Element Plus icon 或 SvgIcon name)，缺省则不展示图标 */
    icon?: string;
    /** 是否需要表名（无表名时自动隐藏，如 SQL 执行结果集无归属表） */
    requireTable?: boolean;
    /** 需要的权限标识，无权限时菜单项自动隐藏 */
    permission?: string;
    /** 生成类策略的结果弹窗标题（如 'SQL'、'JSON'）；下载类策略无需设置 */
    resultTitle?: string;
    /**
     * 执行策略。
     * 下载类策略直接落盘并返回 void；生成类策略返回文本，由调用方决定如何展示。
     */
    execute: (ctx: TableExportContext) => string | void | Promise<string | void>;
}

/** 导出策略执行上下文 */
export interface TableExportContext {
    dbId: number;
    db: string;
    table: string;
    datas: Record<string, unknown>[];
    columns: { columnName?: string; key: string; title: string; show?: boolean }[];
}
