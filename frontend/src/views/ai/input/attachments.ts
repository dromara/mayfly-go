/**
 * 附件读取与分流（对齐 tokhub packages/chat/src/attachments.ts）
 *
 * 按类型分流：
 * - 图片 → base64 dataURL
 * - 文本类文件（md/txt/json/代码等）→ 前端读取文本，内联到消息
 * - 其它文件（pdf/docx 等）→ 本阶段不支持，预留 kind:'file' 与上传 API 扩展点
 */
import type { MessageAttachment } from '../protocol/types';

/** 可按文本解码的扩展名（mime 缺失时兜底） */
const TEXT_EXTENSIONS = new Set([
    'txt',
    'md',
    'markdown',
    'json',
    'jsonl',
    'csv',
    'tsv',
    'xml',
    'yaml',
    'yml',
    'html',
    'htm',
    'css',
    'js',
    'jsx',
    'ts',
    'tsx',
    'py',
    'rs',
    'go',
    'java',
    'c',
    'cpp',
    'h',
    'hpp',
    'sh',
    'sql',
    'toml',
    'ini',
    'conf',
    'log',
]);

/** 单个附件大小上限（dataURL/文本内联传输的上限保护） */
export const MAX_ATTACHMENT_SIZE = 10 * 1024 * 1024;

/** 文件选择框 accept 值（图片 + 文本类） */
export const ATTACHMENT_ACCEPT = [
    'image/*',
    'text/*',
    'application/json',
    'application/xml',
    ...Array.from(TEXT_EXTENSIONS).map((ext) => `.${ext}`),
].join(',');

/** 字节大小格式化 */
export function formatAttachmentSize(size: number): string {
    if (size < 1024) return `${size} B`;
    if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
    return `${(size / 1024 / 1024).toFixed(1)} MB`;
}

/** 提取扩展名（小写） */
function fileExt(file: File): string {
    return file.name.split('.').pop()?.toLowerCase() ?? '';
}

/** 判断是否可按文本解码 */
function isTextual(file: File): boolean {
    if (file.type.startsWith('text/')) return true;
    if (file.type === 'application/json' || file.type === 'application/xml') return true;
    return TEXT_EXTENSIONS.has(fileExt(file));
}

/** 判断文件是否需要上传到服务端（非图片、非文本）——本阶段不支持 */
export function needsUpload(file: File): boolean {
    return !file.type.startsWith('image/') && !isTextual(file);
}

/** 是否可作为附件发送 */
export function isSupportedAttachment(file: File): boolean {
    return !needsUpload(file) && file.size <= MAX_ATTACHMENT_SIZE;
}

/** 读取单个文件为消息附件 */
export async function readChatAttachment(file: File): Promise<MessageAttachment> {
    const base = { name: file.name, mime: file.type || undefined, size: file.size };

    if (file.type.startsWith('image/')) {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = () => resolve({ ...base, kind: 'image', dataUrl: String(reader.result) });
            reader.onerror = () => reject(reader.error ?? new Error('read error'));
            reader.readAsDataURL(file);
        });
    }

    if (isTextual(file)) {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = () => resolve({ ...base, kind: 'text', text: String(reader.result) });
            reader.onerror = () => reject(reader.error ?? new Error('read error'));
            reader.readAsText(file);
        });
    }

    // 扩展点：接入上传 API 后返回 fileKey，走 kind:'file' 链路
    return { ...base, kind: 'file' };
}

/**
 * 将附件内容拼进发送文本（后端零改动）：
 * - 文本类内联原文（AI 可直接读取）
 * - 图片/其它文件以说明占位（展示元数据由消息附件渲染承担）
 */
export function buildMessageContent(text: string, attachments?: MessageAttachment[]): string {
    if (!attachments?.length) return text;
    let content = text;
    for (const att of attachments) {
        if (att.kind === 'text' && att.text) {
            content += `\n\n--- 附件：${att.name} ---\n${att.text}`;
        } else if (att.kind === 'image') {
            content += `\n\n[图片：${att.name}]`;
        } else {
            content += `\n\n[附件：${att.name}]`;
        }
    }
    return content;
}
