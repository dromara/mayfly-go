import { reactive } from 'vue';

/**
 * 全局执行计划对话框的共享状态（模块级单例）。
 *
 * 由 layout 挂载的一个 SqlPlanDialog 消费（主渲染树，全局注册的 el-* 与 i18n 均可用）。
 * 任何调用方——包括无 appContext 的独立 render 树（SqlExecBox 执行确认框）——都通过 openSqlPlan
 * 唤起这一个对话框，而不必自己内嵌 SqlPlanDialog（那样在 render 树里 el-* 组件不会解析）。
 */
export const sqlPlanState = reactive({
    visible: false,
    dbId: 0,
    db: '',
    sql: '',
});

/** 打开执行计划对话框：给定库与 SQL 文本，多条时对话框内可逐条选中查看 */
export function openSqlPlan(dbId: number, db: string, sql: string) {
    sqlPlanState.dbId = dbId;
    sqlPlanState.db = db;
    sqlPlanState.sql = sql;
    sqlPlanState.visible = true;
}
