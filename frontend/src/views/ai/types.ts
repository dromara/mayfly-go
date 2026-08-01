/**
 * AI 模块类型定义
 * 对应后端: ai/domain/entity/*
 */

// ==================== Entity ====================

/** AI 会话实体 (对应 entity.Session) */
export interface AiSession {
    id: number;
    sessionKey: string;
    title: string;
    summary: string;
    messageCount: number;
    tokenCount: number;
    extra?: Record<string, unknown>;
    createTime: string;
    creator: string;
}

/** AI 会话消息实体 (对应 entity.SessionMessage) */
export interface AiSessionMessage {
    id: number;
    sessionKey: string;
    turnId: string;
    role: string;
    msgType: string;
    content: string;
    toolCalls: string;
    actionId: string;
    toolCallId: string;
    extra?: Record<string, unknown>;
    createTime: string;
    creator: string;
}

// ==================== VO ====================

/** 会话列表项 (用于前端展示) */
export interface Session {
    sessionKey: string;
    title: string;
    createTime: string;
    updateTime: string;
}

/** 工具调用 */
export interface ToolCall {
    id: string;
    function: {
        name: string;
        arguments: string;
    };
}

/** 会话消息 (用于前端展示) */
export interface SessionMessage {
    turnId?: string;
    sessionId?: string;
    role: string;
    content: string;
    type?: string;
    time?: string;
    reasoningContent?: string;
    toolCalls?: ToolCall[];
    toolCallId?: string;
    actionId?: string;
    extra?: Record<string, unknown>;
}
