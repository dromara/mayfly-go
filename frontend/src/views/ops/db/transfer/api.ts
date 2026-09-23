import Api from '@/common/Api';
import type { PageResult } from '@/types/common';
import type { DbTransferFile, DbTransferLogListVO, DbTransferLogRunInfo, DbTransferTaskListVO } from '@/views/ops/db/types';

/** 执行导出文件的入参 (对应后端 form.DbTransferFileRunForm) */
export interface DbTransferFileRunForm {
    /** 文件 ID */
    id: number;
    /** 需要执行 sql 的数据库 id */
    targetDbId: number;
    /** 需要执行 sql 的数据库名 */
    targetDbName: string;
    /** 客户端唯一 id，用于执行进度消息回传与断点续跑 */
    clientId: string;
}

export const dbTransferApi = {
    // 数据库迁移相关
    /** 迁移任务列表：后端返回 vo.DbTransferTaskListVO（不含 taskKey/concurrency） */
    dbTransferTasks: Api.newGet<PageResult<DbTransferTaskListVO>>('/dbTransfer'),
    saveDbTransferTask: Api.newPost<void>('/dbTransfer/save'),
    deleteDbTransferTask: Api.newDelete<void>('/dbTransfer/{taskId}/del'),
    updateDbTransferTaskStatus: Api.newPost<void>('/dbTransfer/{taskId}/status'),
    /** 立即执行：返回本次执行的日志 id */
    runDbTransferTask: Api.newPost<number>('/dbTransfer/{taskId}/run'),
    /** 数据校验：异步执行并返回日志 id */
    verifyDbTransferTask: Api.newPost<number>('/dbTransfer/{taskId}/verify'),
    stopDbTransferTask: Api.newPost<void>('/dbTransfer/{taskId}/stop'),
    /** 执行日志列表 (对应 vo.DbTransferLogListVO，不含 runLog) */
    dbTransferTaskLogs: Api.newGet<PageResult<DbTransferLogListVO>>('/dbTransfer/{taskId}/logs'),
    /** 单条执行日志的运行日志内容 (对应 vo.DbTransferLogRunVO) */
    dbTransferTaskLogRun: Api.newGet<DbTransferLogRunInfo>('/dbTransfer/logs/{logId}/run'),

    // 导出文件管理
    dbTransferFileList: Api.newGet<PageResult<DbTransferFile>>('/dbTransfer/files/{taskId}'),
    dbTransferFileDel: Api.newPost<void>('/dbTransfer/files/del/{fileId}'),
    dbTransferFileRun: Api.newPost<void, DbTransferFileRunForm>('/dbTransfer/files/run'),
};
