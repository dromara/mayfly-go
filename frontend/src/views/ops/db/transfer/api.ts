import Api from '@/common/Api';

export const dbTransferApi = {
    // 数据库迁移相关
    dbTransferTasks: Api.newGet('/dbTransfer'),
    saveDbTransferTask: Api.newPost('/dbTransfer/save'),
    deleteDbTransferTask: Api.newDelete('/dbTransfer/{taskId}/del'),
    updateDbTransferTaskStatus: Api.newPost('/dbTransfer/{taskId}/status'),
    runDbTransferTask: Api.newPost('/dbTransfer/{taskId}/run'),
    stopDbTransferTask: Api.newPost('/dbTransfer/{taskId}/stop'),
    dbTransferTaskLogs: Api.newGet('/dbTransfer/{taskId}/logs'),
    dbTransferFileList: Api.newGet('/dbTransfer/files/{taskId}'),
    dbTransferFileDel: Api.newPost('/dbTransfer/files/del/{fileId}'),
    dbTransferFileRun: Api.newPost('/dbTransfer/files/run'),
    dbTransferFileDown: Api.newGet('/dbTransfer/files/down/{fileUuid}'),
};
