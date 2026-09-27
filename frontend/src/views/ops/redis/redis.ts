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

    /**
     * 执行原始命令（命令控制台专用）
     * @param cmd 命令列表如：['SET', 'key', 'value']
     * @returns 执行结果（随命令变化，由调用方泛型指定）
     */
    async runCmd<T = unknown>(cmd: (string | number)[]): Promise<T> {
        return (await redisApi.runCmd.request({
            id: this.id,
            db: this.db,
            cmd,
        })) as T;
    }
}
