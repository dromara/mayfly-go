import { isWarnAckError } from '@/common/request';
import { confirmWarnAck } from '@/views/flow/warnAck';

/**
 * 带确认的资源操作结果。
 *
 * `executed: false` 时调用方必须跳过后续动作，两种原因的出路不同：
 * `ticket` 为真是操作者主动选了「提交工单审批」（要给他提单入口），为假只是暂缓（保留现状即可）
 */
export type WarnAckOutcome<T> = { executed: true; result: T } | { executed: false; ticket: boolean };

/**
 * key 面板类型化操作的「仅提醒」确认。
 *
 * 这些操作（成员写入、视角操作、设置 TTL、重命名、复制、删除 key）和命令控制台一样是用户点出来的，
 * 命中提醒时同样要先问再执行；此前它们悄悄执行完、只在服务端日志留一行，管理员配的「仅提醒」等于没配。
 *
 * 与命令台的区别是没有「提交工单审批」这一选项：工单必须带上原始命令，而面板只有结构化表单，
 * 反推不出等价命令。需要走审批请到命令控制台执行同等命令（后端「需审批」提示也是这么指路的）。
 *
 * @param run 执行操作，入参是随请求带上的确认位
 * @param options.ticket 该入口能否就地转工单提单。面板写操作不能（结构化表单反推不出等价命令）；
 * 读内容能——它给得出「申请查看」，所以按可提单口径出对话框与话术
 */
export async function execWithWarnAck<T>(run: (ackWarn: boolean) => Promise<T>, options: { ticket?: boolean } = {}): Promise<WarnAckOutcome<T>> {
    try {
        return { executed: true, result: await run(false) };
    } catch (error) {
        if (!isWarnAckError(error)) {
            throw error;
        }
        // 提示文本由后端按请求语言渲染（含命中的规则原因），这里不再自造一句
        const choice = await confirmWarnAck((error as Error).message, { ticket: options.ticket === true });
        if (choice === 'run') {
            return { executed: true, result: await run(true) };
        }
        return { executed: false, ticket: choice === 'ticket' };
    }
}
