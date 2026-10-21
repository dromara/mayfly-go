import type { NoticePart } from '../protocol/types';

/**
 * 轮次终止提示的种类（镜像后端 `entity.TurnNoticeKind*`，值域真源在后端）。
 *
 * 为什么单独成文件而不是放 `protocol/types.ts`：协议层是与后端线格式一一对应的字段定义，
 * 「值 → i18n 文案键 + 展示归类」属于展示语义，本仓这类登记表的可归属性先例是
 * `registries/statuses.ts`（`RESUME_STATUS_BADGES` / `FALLBACK_RESUME_BADGE`）。混进协议层会让
 * 同一个文件承载「线格式」与「展示」两个变更理由。
 */
export type TurnNoticeKind = 'stopped';

/**
 * 种类 → 文案键。键类型取后端值域而不是前端 part 联合，且**不用 Partial**：
 * 新增一种 kind 时这里漏写键会直接编译不过，把「静默归进 stopped 桶」变成响亮失败
 */
export const TURN_NOTICE_TEXT_KEY: Record<TurnNoticeKind, string> = {
    stopped: 'ai.chat.turnStopped',
};

/**
 * 按落库种类解析出「展示归类 + 文案键」。
 *
 * 未登记的新种类不静默丢掉、也不硬塞进某个已知桶：丢掉就等于回到「刷新后看不出来这一轮被终止过」
 * 那个本已修掉的缺陷；归类取最接近的 stopped（它确实是「本轮被终止」），正文回显原始种类，
 * 让人一眼看出是前端还没登记的新值（i18n 对不存在的键原样返回，正好用作回显）
 */
export function resolveNoticeTextKey(rawKind: string | undefined): { kind: NoticePart['kind']; textKey: string } {
    const kind = rawKind ?? '';
    if (kind in TURN_NOTICE_TEXT_KEY) {
        return { kind: kind as TurnNoticeKind, textKey: TURN_NOTICE_TEXT_KEY[kind as TurnNoticeKind] };
    }
    return { kind: 'stopped', textKey: kind };
}
