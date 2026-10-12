import syssocket from '@/common/syssocket';
import { nextTick, reactive } from 'vue';
import { completeNotification, createOrUpdateNotification } from '../global-notification-manager';
import DbSqlExecProgress from './DbSqlExecProgress.vue';

// 存储SQL执行任务的取消方法
interface SqlExecProgressMessage {
    id: string;
    executedStatements?: number;
    terminated?: boolean;
    status?: string;
}

// 取消方法在发起侧登记一次，用户点取消时按 id 取用；不预存 progress 引用，避免与状态表双份持有
const sqlExecAborters = new Map<string, () => void>();

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

        // SQL执行完成：走统一收尾（置终态 + 关闭通知 + 清掉状态表），
        // 否则 sqlExecStates 只进不出，每执行完一个脚本就留一条永不释放的记录
        if (content.terminated) {
            finishSqlExecNotification(id);
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
            // 取消是一次性动作：先注销再断链，重复点击不会二次 abort
            const abort = sqlExecAborters.get(id);
            sqlExecAborters.delete(id);
            abort?.();

            // 断的是上传/响应通道，执行侧是否停下由服务端收尾。先把通知置为取消态
            //（经状态表代理解包，UI 才能响应）并给一个短暂可见窗口，随后统一收尾——
            // 不能什么都不做，否则通知会一直挂着等一个可能永远不来的终态消息
            const prog = sqlExecStates.get(id);
            if (prog) {
                nextTick(() => {
                    prog.status = 'cancelled';
                    prog.terminated = true;
                });
            }
            setTimeout(() => finishSqlExecNotification(id), 1500);
        },
    };

    // 创建或更新通知
    createOrUpdateNotification(id, 'sqlScriptRun', data, DbSqlExecProgress, props, {
        title: data.title || 'db.sqlExecute',
        onCancel: props.onCancel,
    });

    sqlExecStates.set(id, data);
}

/**
 * 结束并清理 SQL 执行进度通知。
 *
 * 通知的终态有三个来源：执行完成的 Ws 消息、上传请求自身的成功/失败回执、用户点取消。
 * 无论哪条先到，都必须把通知、取消器与状态表一并收掉；只关通知不删状态表，
 * 每执行/取消一次就在内存里留下一条永不释放的记录
 */
export function finishSqlExecNotification(id: string) {
    const progress = sqlExecStates.get(id);
    if (!progress) {
        return;
    }
    // 先置终态再等延迟关闭：终态让取消按钮立即失效，否则关闭前的窗口里还能再点一次取消
    progress.terminated = true;
    completeNotification(id, 600);
    sqlExecStates.delete(id);
    sqlExecAborters.delete(id);
}

/**
 * 注册SQL执行任务的取消方法（通知创建后同步调用；取消动作在用户点击时才按 id 取用）
 * @param execId 执行ID
 * @param abort 取消方法
 */
export function registerSqlExecAborter(execId: string, abort: () => void) {
    sqlExecAborters.set(execId, abort);
}
