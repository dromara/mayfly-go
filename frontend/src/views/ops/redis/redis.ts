import { redisApi } from './api';

/**
 * Redis 实例连接句柄：只承载「在哪个实例的哪个库上执行命令」这一身份，
 * 数据面的读写一律走 redisApi 的类型化接口，本类仅保留命令控制台的原始命令通道
 */
export class RedisInst {
    /** 实例id */
    id: number;

    /** 库号 */
    db: number;

    /** 实例资源编码：被策略拦下时提单要靠它解析出审批流程 */
    code = '';

    /** 实例名称（提单表单展示用） */
    name = '';

    /** 实例标签路径（提单表单展示与权限归属用） */
    tagPath = '';

    /**
     * 执行原始命令（命令控制台专用）
     * @param cmd 命令列表如：['SET', 'key', 'value']
     * @returns 执行结果（随命令变化，由调用方泛型指定）
     */
    /**
     * 执行一条命令。ackWarn 仅在操作者已在确认框里选「直接执行」时传 true：
     * 后端据此跳过「仅提醒」的再次追问，但不会因此放过禁止执行或需审批的命令
     */
    async runCmd<T = unknown>(cmd: (string | number)[], ackWarn = false): Promise<T> {
        return (await redisApi.runCmd.request({
            id: this.id,
            db: this.db,
            cmd,
            ackWarn,
        })) as T;
    }
}
