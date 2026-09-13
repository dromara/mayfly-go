/**
 * DB 模块类型定义出口
 *
 * 本目录按领域拆分，此文件是 db 模块内唯一的类型导入入口（消费方一律 `from '../types'`）：
 * - entity.ts        后端实体与列表查询参数
 * - vo.ts            视图对象：表信息、列元数据、表格列定义
 * - sqlExec.ts       SQL 执行结果
 * - virtualTable.ts  虚拟表格导出策略扩展契约
 * - schema.ts        表编辑子系统私有领域模型（不经此出口，由该子系统自行导入）
 *
 * 方言契约相关类型（DbDialect / RowDefinition / IndexDefinition / TableEditContext 等）
 * 属方言层内核，从 '@/views/ops/db/dialect' 导入，不在此聚合，
 * 否则方言层会反向依赖 db 模块类型而形成循环。
 */

// 实体类型
export type {
    DbInstInfo,
    Db,
    DbForm,
    DbNamesParam,
    DbInstance,
    DbSql,
    DbSqlExec,
    DbInstanceListParam,
    DbListParam,
    DataSyncTask,
    DataSyncLog,
    DbTransferTask,
    DbBackup,
    DbBackupHistory,
    DbRestore,
    DbInstanceServerInfo,
    DbMaskRule,
    DbMaskColumn,
    DbMaskRuleQuery,
    DbMaskRuleSaveForm,
    DbMaskColumnQuery,
    DbMaskColumnSaveForm,
} from './entity';

// VO 类型
export type {
    DbTableInfo,
    ColumnMetadata,
    TableColumnDef,
    RenderTableColumn,
    DbTableColumn,
    DbTableIndex,
    TableOpData,
    DbNodeParams,
    DbTableNodeParams,
    DbTreeNodeData,
    TreeNodeCallbackData,
} from './vo';

// SQL 执行结果类型
export type { SqlExecResColumn, SqlExecRes } from './sqlExec';

// 虚拟表格导出扩展契约
export type { TableExportStrategy, TableExportContext } from './virtualTable';
