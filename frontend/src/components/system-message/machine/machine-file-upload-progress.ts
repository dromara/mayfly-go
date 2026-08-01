import syssocket from '@/common/syssocket';
import { nextTick, reactive } from 'vue';
import { activeNotifications, completeNotification, createOrUpdateNotification } from '../global-notification-manager';
import MachineFileUploadProgress from './MachineFileUploadProgress.vue';

// 存储上传任务的取消方法
const uploadAborters = new Map<string, { abort: () => void; progress?: FileUploadProgress }>();

// 存储待注册的 abort 方法（等待 WebSocket 消息到达）
const pendingAborters = new Map<string, () => void>();

export interface FileUploadProgress {
    authCertName: string;
    path: string;
    filename: string;
    totalSize?: number; // 文件总大小
    uploadedSize?: number; // 已上传大小
    status?: 'uploading' | 'complete' | 'error';
    timestamp?: number;
}

const fileStates = reactive<Map<string, FileUploadProgress>>(new Map());

/**
 * 注册机器文件上传进度消息处理
 */
export async function registerMachineFileUploadProgress() {
    await syssocket.registerMsgHandler('machineFileUploadProgress', function (message: Record<string, unknown>) {
        const content = message.params as Record<string, unknown>;
        const uploadId = content.uploadId as string;

        const progress = fileStates.get(uploadId);
        if (!progress) {
            return;
        }

        // 上传完成或失败
        const status = content.status as string;
        if (status === 'complete' || status === 'error') {
            completeNotification(uploadId, 1000);
            uploadAborters.delete(uploadId);
            return;
        }

        progress.status = (status || 'uploading') as FileUploadProgress['status'];
        progress.uploadedSize = content.uploadedSize as number;
        progress.timestamp = content.timestamp as number;
        progress.totalSize = content.totalSize as number;
        return;
    });
}

export function createUploadFileNotification(uploadId: string, data: FileUploadProgress) {
    // 构建组件props
    const props = {
        progress: data,
        onCancel: () => {
            const aborter = uploadAborters.get(uploadId);
            if (aborter) {
                aborter.abort();

                // 更新通知状态为取消
                if (aborter.progress) {
                    const progress = aborter.progress;
                    nextTick(() => {
                        progress.status = 'error';
                        progress.filename = '已取消: ' + (progress.filename || '');
                    });

                    // 延迟后关闭通知
                    setTimeout(() => {
                        completeNotification(uploadId, 1000);
                        uploadAborters.delete(uploadId);
                    }, 1500);
                } else {
                    uploadAborters.delete(uploadId);
                }
            }
        },
    };

    // 创建或更新上传通知
    createOrUpdateNotification(uploadId, 'machineFileUpload', data, MachineFileUploadProgress, props, {
        title: 'machine.fileUpload',
    });

    fileStates.set(uploadId, data);

    // 如果有待注册的 abort 方法，现在注册
    const pendingAbort = pendingAborters.get(uploadId);
    if (pendingAbort) {
        uploadAborters.set(uploadId, { abort: pendingAbort, progress: props.progress });
        pendingAborters.delete(uploadId);
    }
}

/**
 * 注册上传任务的取消方法
 * @param uploadId 上传ID
 * @param abort 取消方法
 */
export function registerUploadFileAborter(uploadId: string, abort: () => void) {
    // 先检查通知是否已经存在
    const task = activeNotifications.get(uploadId);
    const progress = (task?.componentProps?.progress || null) as FileUploadProgress | null;

    if (progress) {
        // 通知已存在，直接注册
        uploadAborters.set(uploadId, { abort, progress });
    } else {
        // 通知还未创建，保存为 pending
        pendingAborters.set(uploadId, abort);
    }
}
