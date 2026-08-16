import syssocket from '@/common/syssocket';
import { nextTick, reactive } from 'vue';
import { i18n } from '@/i18n';
import { activeNotifications, completeNotification, createOrUpdateNotification } from '../global-notification-manager';
import MachineFolderUploadProgress from './MachineFolderUploadProgress.vue';
import { formatByteSize } from '@/common/utils/format';

// 存储上传任务的取消方法
const folderUploadAborters = new Map<string, { abort: () => void; progress?: FolderUploadProgress }>();

// 存储待注册的 abort 方法（等待 WebSocket 消息到达）
const pendingFolderAborters = new Map<string, () => void>();

export interface FolderUploadProgress {
    authCertName: string;
    path: string;
    folderName: string;
    totalFiles: number; // 文件夹总文件数
    uploadedFiles: number; // 已上传文件数
    finishedFiles: number; // 已完成（成功或失败）的文件数
    totalSize: number; // 文件夹总大小
    uploadedSize: number; // 已上传大小
    status: 'uploading' | 'complete' | 'error';

    files: Map<
        string, // 文件路径
        {
            path: string;
            status: 'waiting' | 'uploading' | 'complete' | 'error'; // 文件状态
            progress: number;
            currentSize: number; // 当前已上传大小
            totalSize: number; // 文件总大小
            timestamp: number; // 后端推送的时间戳（用于前端计算速度）
            speed?: string;
        }
    >;
}

const folderStates = reactive<Map<string, FolderUploadProgress>>(new Map());

/**
 * 更新文件夹上传进度（处理后端推送的单文件进度消息）
 */
export function handleUploadFolderProgress(content: Record<string, unknown>) {
    const uploadId = content.uploadId as string;
    const filename = content.filename as string;
    const path = content.path as string;
    const uploadedSize = content.uploadedSize as number;
    const totalSize = content.totalSize as number;
    const status = content.status as string;
    const timestamp = content.timestamp as number;

    if (!uploadId || !filename) {
        return;
    }

    const folderState = folderStates.get(uploadId);
    if (!folderState) {
        return;
    }

    // 构建文件完整路径（后端推送的 path + filename）
    const backendFilePath = path ? `${path}/${filename}` : filename;

    // 查找匹配的文件
    let matchedFile = folderState.files.get(backendFilePath);

    if (!matchedFile) {
        return;
    }

    // 更新 Map 中的状态
    if (status === 'uploading' && totalSize > 0) {
        // 如果已经是 complete 或 error，不要覆盖
        if (matchedFile.status === 'complete' || matchedFile.status === 'error') {
            return;
        }

        matchedFile.status = 'uploading';
        matchedFile.progress = Math.round((uploadedSize / totalSize) * 100);
        matchedFile.currentSize = uploadedSize;
        matchedFile.totalSize = totalSize;

        // 计算传输速度（使用后端推送的 timestamp）
        if (timestamp) {
            // 第一次推送时初始化
            if (!matchedFile.timestamp) {
                matchedFile.timestamp = timestamp;
            } else {
                const timeDiff = (timestamp - matchedFile.timestamp) / 1000; // 转换为秒
                if (timeDiff > 0) {
                    const sizeDiff = uploadedSize - matchedFile.currentSize;
                    if (sizeDiff > 0) {
                        const speed = sizeDiff / timeDiff;
                        matchedFile.speed = formatByteSize(speed);
                    }
                }
                matchedFile.timestamp = timestamp;
            }
        }
    } else if (status === 'complete') {
        matchedFile.status = 'complete';
        matchedFile.progress = 100;
        matchedFile.currentSize = matchedFile.totalSize;
        folderState.uploadedFiles = (folderState.uploadedFiles || 0) + 1;
        folderState.finishedFiles = (folderState.finishedFiles || 0) + 1;
        folderState.uploadedSize += uploadedSize;
    } else if (status === 'error') {
        matchedFile.status = 'error';
        matchedFile.progress = 0;
        folderState.finishedFiles = (folderState.finishedFiles || 0) + 1;
    }

    // 文件夹上传完成条件：所有文件都已完成（成功或失败）
    if (folderState.finishedFiles === folderState.totalFiles) {
        completeNotification(uploadId, 1000);
    }
}

/**
 * 创建文件夹上传通知
 */
export function createUploadFolderNotification(uploadId: string, data: FolderUploadProgress) {
    const props = {
        progress: data,
        onCancel: () => {
            const aborter = folderUploadAborters.get(uploadId);
            if (aborter) {
                aborter.abort();

                if (aborter.progress) {
                    const progress = aborter.progress;
                    nextTick(() => {
                        progress.status = 'error';
                        progress.folderName = i18n.global.t('machine.uploadCancelled') + ': ' + (progress.folderName || '');
                    });

                    setTimeout(() => {
                        completeNotification(uploadId, 1000);
                        folderUploadAborters.delete(uploadId);
                        folderStates.delete(uploadId);
                    }, 1500);
                } else {
                    folderUploadAborters.delete(uploadId);
                    folderStates.delete(uploadId);
                }
            }
        },
    };

    createOrUpdateNotification(uploadId, 'machineFolderUpload', data, MachineFolderUploadProgress, props, {
        title: 'machine.folderUpload',
    });

    folderStates.set(uploadId, data);

    // 注册 aborter
    const pendingAbort = pendingFolderAborters.get(uploadId);
    if (pendingAbort) {
        folderUploadAborters.set(uploadId, { abort: pendingAbort, progress: props.progress });
        pendingFolderAborters.delete(uploadId);
    }
}

/**
 * 注册文件夹上传进度消息处理
 */
export async function registerFolderUploadProgressHandler() {
    await syssocket.registerMsgHandler('machineFolderUploadProgress', function (message: Record<string, unknown>) {
        const content = message.params as Record<string, unknown>;
        const uploadId = content.uploadId;

        if (!uploadId) {
            return;
        }

        // 文件夹上传：处理单文件进度和完成消息
        // 注意：文件夹上传时，单个文件的 complete/error 不关闭通知，只标记文件状态
        handleUploadFolderProgress(content);
    });
}

/**
 * 注册文件夹上传任务的取消方法
 * @param uploadId 上传ID
 * @param abort 取消方法
 */
export function registerUploadFolderAborter(uploadId: string, abort: () => void) {
    // 先检查通知是否已经存在
    const task = activeNotifications.get(uploadId);
    const progress = (task?.componentProps?.progress || null) as FolderUploadProgress | null;

    if (progress) {
        // 通知已存在，直接注册
        folderUploadAborters.set(uploadId, { abort, progress });
    } else {
        // 通知还未创建，保存为 pending
        pendingFolderAborters.set(uploadId, abort);
    }
}
