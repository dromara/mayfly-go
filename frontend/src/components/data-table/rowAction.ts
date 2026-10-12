import { i18n } from '@/i18n';
import { useI18nConfirm } from '@/hooks/useI18n';
import type { ActionItem } from './index';

/**
 * 解析一格按钮的文案（按行插值）。
 * 直显按钮（RowActionEntry）与「更多」菜单项（RowActions）共用本函数，
 * 避免同一个 label 字段两处各自实现取文案逻辑（一处改了另一处静默不一致）。
 */
export function actionLabel(item: ActionItem, data: Record<string, unknown>): string {
    return i18n.global.t(item.label, item.labelParams?.(data) ?? {});
}

/**
 * 这一格当前是否不可操作。
 * 直显按钮不用判：el-button 的 disabled/loading 自己会禁掉点击。
 * 但「更多」菜单项不是按钮，必须显式把 disabled 与 loading 都折算成不可选，
 * 否则进行中的动作能在菜单里重复触发。
 */
export function isActionBlocked(item: ActionItem, data: Record<string, unknown>): boolean {
    return !!item.disabled?.(data) || !!item.loading?.(data);
}

/**
 * 执行一格行内操作：确认门禁只写这一处，直显按钮与「更多」菜单项共用。
 *
 * 就地气泡确认（confirm.pop）由气泡自己的 confirm 事件驱动，不经过这里；
 * 但同一项被折进「更多」时菜单里弹不下气泡，此时退回确认框——危险动作不能被静默执行。
 */
export async function runRowAction(
    item: ActionItem,
    data: Record<string, unknown>,
    opts: { event?: MouseEvent; folded?: boolean } = {}
): Promise<void> {
    const confirm = item.confirm;
    if (confirm && (opts.folded || !confirm.pop)) {
        if (!(await useI18nConfirm(confirm.title, confirm.params?.(data) ?? {}))) {
            return;
        }
    }
    item.onClick(data, opts.event);
}
