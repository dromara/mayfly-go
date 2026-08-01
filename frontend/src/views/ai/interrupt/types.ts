/**
 * 中断信息组件统一类型定义
 */

import type { SessionMessage } from '../api';

/** 中断恢复信息（已处理/待提交的操作记录） */
export interface ResumeInfo {
    action: string;
    payload?: Record<string, unknown>;
    timestamp?: string;
}

/** 中断内容数据（extra.content 的结构） */
export interface InterruptContent {
    title?: string;
    description?: string;
    type?: string;
    toolInfo?: { name: string; [key: string]: unknown };
    arguments?: unknown;
    options?: Array<{ value: string; label: string }>;
    [key: string]: unknown;
}

/**
 * 内部消息类型（包含 extra 字段）
 */
export type InternalMessage = SessionMessage & {
    codeId?: string;
    extra?: {
        type?: 'interrupt' | 'notification' | string;
        content?: InterruptContent;
        toolStatus?: string;
        interruptId?: string;
        actionId?: string;
        turnId?: string;
        resumeInfo?: ResumeInfo;
        pendingResumeInfo?: ResumeInfo;
        [key: string]: unknown;
    };
};

/**
 * 中断操作事件数据
 */
export interface InterruptActionEvent {
    turnId: string; // 轮次Id
    interruptId: string; // 中断ID
    interruptType: string; // 中断类型（如 'APPROVAL', 'PARAM_COMPLETION' 等）
    action: string; // 操作（如 'approve', 'reject', 'complete', 'cancel' 等）
    payload?: Record<string, unknown>; // 操作携带的额外数据
}

/**
 * 中断组件事件处理器类型
 */
export type InterruptActionHandler = (action: InterruptActionEvent) => void | Promise<void>;

/**
 * 中断组件 Props 接口
 * 所有中断组件必须遵循此接口
 */
export interface InterruptComponentProps {
    data: InternalMessage; // 完整的内部消息对象
    readonly?: boolean; // 是否只读模式
}
