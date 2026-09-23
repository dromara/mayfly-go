import Api from '@/common/Api';
import { encryptField } from '@/views/ops/db/api';
import type { PageResult } from '@/types/common';
import type { DataSyncLogListVO, DataSyncLogRunInfo, DataSyncTask, DataSyncTaskListVO } from '@/views/ops/db/types';

export const dbSyncApi = {
    // 数据同步相关
    /** 任务列表：后端返回 vo.DataSyncTaskListVO，cron 的键名为 `cron`（详情实体里是 `taskCron`） */
    datasyncTasks: Api.newGet<PageResult<DataSyncTaskListVO>>('/datasync/tasks'),
    saveDatasyncTask: Api.newPost<void>('/datasync/tasks/save').withBeforeHandler(async (param: any) => await encryptField(param, 'dataSql')),
    /** 任务详情：返回完整实体，含 dataSql/fieldMap 等同步配置 */
    getDatasyncTask: Api.newGet<DataSyncTask>('/datasync/tasks/{taskId}'),
    deleteDatasyncTask: Api.newDelete<void>('/datasync/tasks/{taskId}/del'),
    updateDatasyncTaskStatus: Api.newPost<void>('/datasync/tasks/{taskId}/status'),
    runDatasyncTask: Api.newPost<void>('/datasync/tasks/{taskId}/run'),
    stopDatasyncTask: Api.newPost<void>('/datasync/tasks/{taskId}/stop'),
    /** 执行日志列表 (对应 vo.DataSyncLogListVO，不含 runLog/dataSqlFull) */
    datasyncLogs: Api.newGet<PageResult<DataSyncLogListVO>>('/datasync/tasks/{taskId}/logs'),
    /** 单条执行日志的运行日志内容 (对应 vo.DataSyncLogRunVO) */
    datasyncLogRun: Api.newGet<DataSyncLogRunInfo>('/datasync/tasks/logs/{logId}/run'),
};
