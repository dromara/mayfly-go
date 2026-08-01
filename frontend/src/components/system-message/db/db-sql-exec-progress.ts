import syssocket from '@/common/syssocket';
import { nextTick, reactive } from 'vue';
import { activeNotifications, completeNotification, createOrUpdateNotification } from '../global-notification-manager';
import DbSqlExecProgress from './DbSqlExecProgress.vue';

// 存储SQL执行任务的取消方法
interface SqlExecProgressMessage {
    id: string;
    executedStatements?: number;
    terminated?: boolean;
    status?: string;
}

const sqlExecAborters = new Map<string, { abort: () => void; progress?: SqlExecProgress }>();

// 存储待注册的 abort 方法（等待 WebSocket 消息到达）
const pendingSqlExecAborters = new Map<string, () => void>();

export interface SqlExecProgress {
    id: string;
    title: string;
    dbCode: string;
    dbName: string;
    executedStatements: number;
    terminated: boolean;
    status: string;
    clientId: string;
}

const sqlExecStates = reactive<Map<string, SqlExecProgress>>(new Map());

/**
 * 注册数据库SQL执行进度消息处理
 */
export async function registerDbSqlExecProgress() {
    await syssocket.registerMsgHandler('sqlScriptRunProgress', function (message: Record<string, unknown>) {
        const content = message.params as SqlExecProgressMessage;
        const id = content.id;

        const progress = sqlExecStates.get(id);
        if (!progress) {
            return;
        }

        // SQL执行完成
        if (content.terminated) {
            completeNotification(id, 1000);
            sqlExecAborters.delete(id);
            return;
        }

        progress.executedStatements = content.executedStatements || 0;
        progress.terminated = content.terminated || false;
        progress.status = content.status || '';
        return;
    });
}

/**
 * 创建SQL执行进度通知
 * @param id 执行ID
 * @param data 进度数据
 */
export function createSqlExecNotification(id: string, data: SqlExecProgress) {
    // 构建组件props
    const props = {
        progress: data,
        onCancel: () => {
            const aborter = sqlExecAborters.get(id);
            if (aborter) {
                aborter.abort();

                // 更新通知状态为取消
                if (aborter.progress) {
                    const prog = aborter.progress;
                    nextTick(() => {
                        prog.status = 'cancelled';
                        prog.terminated = true;
                    });

                    // 延迟后关闭通知
                    setTimeout(() => {
                        completeNotification(id, 1000);
                        sqlExecAborters.delete(id);
                    }, 1500);
                } else {
                    sqlExecAborters.delete(id);
                }
            }
        },
    };

    // 创建或更新通知
    createOrUpdateNotification(id, 'sqlScriptRun', data, DbSqlExecProgress, props, {
        title: data.title || 'db.sqlExecute',
        onCancel: props.onCancel,
    });

    sqlExecStates.set(id, data);

    // 如果有待注册的 abort 方法，现在注册
    const pendingAbort = pendingSqlExecAborters.get(id);
    if (pendingAbort) {
        sqlExecAborters.set(id, { abort: pendingAbort, progress: props.progress });
        pendingSqlExecAborters.delete(id);
    }
}

/**
 * 注册SQL执行任务的取消方法
 * @param execId 执行ID
 * @param abort 取消方法
 */
export function registerSqlExecAborter(execId: string, abort: () => void) {
    // 先检查通知是否已经存在
    const task = activeNotifications.get(execId);
    const progress = (task?.componentProps?.progress || null) as SqlExecProgress | null;

    if (progress) {
        // 通知已存在，直接注册
        sqlExecAborters.set(execId, { abort, progress });
    } else {
        // 通知还未创建，保存为 pending
        pendingSqlExecAborters.set(execId, abort);
    }
}
