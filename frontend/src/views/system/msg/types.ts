/**
 * Msg 模块类型定义
 * 对应后端: msg/domain/entity/*
 */

// ==================== Entity ====================

/** 消息实体 (对应 entity.Msg) */
export interface Msg {
    id: number;
    type: number;
    subtype: string;
    status: number;
    msg: string;
    recipientId: number;
    extra?: Record<string, unknown>;
    createTime: string;
    creator: string;
}

/** 消息渠道实体 (对应 entity.MsgChannel) */
export interface MsgChannel {
    id: number;
    name: string;
    code: string;
    type: string;
    url: string;
    status: number;
    remark?: string;
    extra?: Record<string, unknown>;
    createTime: string;
    creator: string;
}

/** 消息模板实体 (对应 entity.MsgTmpl) */
export interface MsgTemplate {
    id: number;
    name: string;
    code: string;
    title: string;
    tmpl: string;
    msgType: string;
    status: number;
    remark?: string;
    extra?: Record<string, unknown>;
    createTime: string;
    creator: string;
}

/** 消息模板渠道关联 (对应 entity.MsgTmplChannel) */
export interface MsgTmplChannel {
    id: number;
    tmplId: number;
    channelId: number;
    createTime: string;
    creator: string;
}
