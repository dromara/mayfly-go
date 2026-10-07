import type { TicketPrefill } from './types';

/**
 * 全局提单入口：给「独立 render 树」里的函数式弹窗借道真实组件树提单用。
 *
 * SqlExecBox 体系（执行确认框）是用 render(vnode) 挂到 body 的，整棵树没有
 * appContext：inject 链为空，useI18n 会直接崩（appContext.app 为 null）、模板里的
 * $t 也解析不到全局属性，所以 WorkTicketSubmit → ProcInstEdit 这条深链没法挂在
 * 那棵树里。全局实例挂在 layout 根（常驻真实组件树）上，openTicket 转发即可——
 * 与各宿主挂载的 WorkTicketSubmit（SQL 编辑器结果区、Redis 控制台等）走同一组件、
 * 同一条校验与提交链路，只是挂载位置不同。
 */
let opener: ((prefill?: TicketPrefill) => void) | null = null;

/** layout 根挂载的全局 WorkTicketSubmit 就绪后注册；卸载时须注销，防止悬挂引用 */
export const registerTicketOpener = (open: (prefill?: TicketPrefill) => void) => {
    opener = open;
};

export const unregisterTicketOpener = () => {
    opener = null;
};

/** 打开全局提单抽屉；全局实例未挂载（如登录页等 layout 之外场景）时静默忽略 */
export const openTicket = (prefill?: TicketPrefill) => {
    opener?.(prefill);
};
