import syssocket from '@/common/syssocket';
import { reactive } from 'vue';
import { completeNotification, createOrUpdateNotification } from '../global-notification-manager';
import DbDataImportProgress from './DbDataImportProgress.vue';

/** 数据文件导入进度状态（与后端 dataImportProgress Ws 消息字段对齐） */
export interface DataImportProgress {
    uploadId: string;
    title: string;
    dbCode: string;
    dbName: string;
    table: string;
    imported: number;
    total: number;
    terminated: boolean;
    status: string;
}

const importStates = reactive<Map<string, DataImportProgress>>(new Map());

/**
 * 注册数据文件导入进度消息处理：按 uploadId 更新对应通知，terminated 时关闭
 */
export async function registerDbDataImportProgress() {
    await syssocket.registerMsgHandler('dataImportProgress', function (message: Record<string, unknown>) {
        const content = message.params as Record<string, unknown>;
        const uploadId = content.uploadId as string;
        if (!uploadId) {
            return;
        }

        const progress = importStates.get(uploadId);
        if (!progress) {
            return;
        }

        progress.imported = (content.imported as number) || 0;
        if (content.total) {
            progress.total = content.total as number;
        }
        if (content.status) {
            progress.status = content.status as string;
        }
        if (content.terminated) {
            progress.terminated = true;
            completeNotification(uploadId, 1200);
            importStates.delete(uploadId);
        }
    });
}

/**
 * 创建/更新导入进度通知（前端在发起导入请求时调用，随后由 Ws 消息驱动刷新）
 */
export function createDataImportNotification(data: DataImportProgress) {
    const props = { progress: data };
    createOrUpdateNotification(data.uploadId, 'dataImport', data, DbDataImportProgress, props, {
        title: data.title || 'db.importData',
    });
    importStates.set(data.uploadId, data);
}

/**
 * 导入请求结束（成功/失败/异常）时兜底关闭通知：即使 terminated 的 Ws 消息丢失也不会悬挂
 */
export function finishDataImportNotification(uploadId: string) {
    const progress = importStates.get(uploadId);
    if (!progress) {
        return;
    }
    progress.terminated = true;
    completeNotification(uploadId, 600);
    importStates.delete(uploadId);
}
