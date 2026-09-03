/**
 * AI Input 模块类型定义
 *
 * 触发器相关类型（TriggerMenuItem/TriggerDef 等）见 ./triggers/types.ts
 */
import type { ChipSegment } from './chipRegistry';
import type { MessageAttachment } from '../protocol/types';

/** 技能/触发项数据接口 */
export interface SkillItem {
    id: string;
    name: string;
    description?: string;
    icon?: string;
    [key: string]: unknown;
}

/** 发送事件数据 */
export interface ChatInputSubmitData {
    text: string;
    segments: ChipSegment[];
    /** 随消息发送的附件（图片 dataURL / 文本内联 / 文件元数据） */
    attachments?: MessageAttachment[];
}
