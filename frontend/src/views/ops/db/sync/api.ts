import Api from '@/common/Api';
import { encryptField } from '@/views/ops/db/api';

export const dbSyncApi = {
    // 数据同步相关
    datasyncTasks: Api.newGet('/datasync/tasks'),
    saveDatasyncTask: Api.newPost('/datasync/tasks/save').withBeforeHandler(async (param: any) => await encryptField(param, 'dataSql')),
    getDatasyncTask: Api.newGet('/datasync/tasks/{taskId}'),
    deleteDatasyncTask: Api.newDelete('/datasync/tasks/{taskId}/del'),
    updateDatasyncTaskStatus: Api.newPost('/datasync/tasks/{taskId}/status'),
    runDatasyncTask: Api.newPost('/datasync/tasks/{taskId}/run'),
    stopDatasyncTask: Api.newPost('/datasync/tasks/{taskId}/stop'),
    datasyncLogs: Api.newGet('/datasync/tasks/{taskId}/logs'),
};
