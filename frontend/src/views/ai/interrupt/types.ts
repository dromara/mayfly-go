/**
 * 中断模块统一类型定义
 * 使用新协议类型，不再依赖旧 SessionMessage
 */

import type { InterruptEvent, TurnItem } from '../protocol/types';

/** 中断操作事件数据（组件事件回调） */
export interface InterruptActionEvent {
    /** 轮次 ID */
    turnId: string;
    /** 中断 ID */
    interruptId: string;
    /** 中断类型 */
    interruptType: string;
    /** 操作（approve/reject/complete/cancel 等） */
    action: string;
    /** 操作携带的额外数据 */
    payload?: Record<string, unknown>;
    /** 关联的工具调用 ID（用于精确匹配移除） */
    toolCallId?: string;
}

/** 中断组件事件处理器类型 */
export type InterruptActionHandler = (action: InterruptActionEvent) => void | Promise<void>;

/** 中断组件 Props 接口（所有中断组件必须遵循） */
export interface InterruptComponentProps {
    /** 中断状态 */
    interrupt: InterruptEvent;
    /** 所属 turn */
    turnId: string;
    /** 是否只读模式 */
    readonly?: boolean;
    /** 操作回调 */
    onAction: InterruptActionHandler;
}
