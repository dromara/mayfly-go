import { i18n } from '@/i18n';
import { ElMessageBox } from 'element-plus';

/** 「仅提醒」命中后操作者的选择：直接执行、提交工单审批、或先不动 */
export type WarnAckChoice = 'run' | 'ticket' | 'skip';

/**
 * 「仅提醒」级策略命中后的确认框。
 *
 * 只有命令/语句**还没执行**时才有意义：这一级别本身不阻断，一旦跑完再提单，审批通过就会执行第二遍。
 * 所以后端在需要确认时直接不执行、返回确认码，由这里决定是带确认重试还是转工单。
 *
 * 两个能弹确认的入口（SQL 控制台、Redis 命令台）共用同一个确认框，措辞与按钮只在这里改一次。
 * 机器命令没有审批通道，确认了也无处可转，所以它不弹确认框、只在终端回显一行提醒
 *
 * @param message 后端下发的命中提示（含命中的规则原因），已按请求语言渲染
 */
export async function confirmWarnAck(message: string, options: { ticket?: boolean } = {}): Promise<WarnAckChoice> {
    const t = i18n.global.t;
    // 该入口能否就地转工单：key 面板的类型化操作不能（工单要带原始命令，结构化表单反推不出等价命令）。
    // 不能时不给「提交工单审批」按钮，取消与关闭都算「先不执行」；后端此时也换成不提转审批的措辞，
    // 两边都跟着入口能力走，避免出现点了没下文的按钮
    const ticket = options.ticket !== false;
    try {
        await ElMessageBox.confirm(message, t('flow.warnAckTitle'), {
            confirmButtonText: t('flow.executeDirectly'),
            cancelButtonText: ticket ? t('flow.submitTicket') : t('common.cancel'),
            // 必须区分「点了提交工单」与「关掉弹窗」：前者要给出提单入口，后者只是暂缓
            distinguishCancelAndClose: ticket,
            type: 'warning',
        });
        return 'run';
    } catch (reason) {
        if (!ticket) {
            return 'skip';
        }
        // element-plus 用 'cancel' 表示点了取消按钮，'close'（右上角/ESC）表示关掉弹窗
        return reason === 'cancel' ? 'ticket' : 'skip';
    }
}
