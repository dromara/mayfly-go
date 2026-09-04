/**
 * 附件读取、上传与分流（对齐 tokhub packages/chat/src/attachments.ts）
 *
 * 按类型分流（内容均经统一文件服务落 local/S3，消息体只随 fileKey 轻量引用）：
 * - 图片 → 本地压缩预览 dataURL，发送时上传压缩产物（与 LLM 输入一致）
 * - 文本类文件（md/txt/json/代码等）→ 前端读取文本，内联到消息正文，附件另上传留档回显
 * - 其它文件（pdf/docx 等）→ 原始 File 随附件内存持有，发送时上传
 */
import { getUploadFileUrl } from '@/common/request';
import type { ContentSegment, MessageAttachment } from '../protocol/types';

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

/** 单个附件大小上限（上传文件服务的大小保护） */
export const MAX_ATTACHMENT_SIZE = 10 * 1024 * 1024;

/** 文件选择框 accept 值（不限类型：任意文件均经文件服务上传） */
export const ATTACHMENT_ACCEPT = '*/*';

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

/** 是否可作为附件发送（任意类型均可上传文件服务，仅限制大小） */
export function isSupportedAttachment(file: File): boolean {
    return file.size <= MAX_ATTACHMENT_SIZE;
}

/** 图片压缩参数：长边上限与 JPEG 质量（对齐主流多模态输入建议，控制 dataURL 体积） */
const IMAGE_MAX_EDGE = 1568;
const IMAGE_JPEG_QUALITY = 0.85;
/** 小于该体积（字节）的图片直接原样内联，跳过重绘 */
const IMAGE_INLINE_THRESHOLD = 512 * 1024;

/** 将图片重绘到长边上限内并转 JPEG（透明底填白）；失败回退原图 dataURL */
function compressImage(dataUrl: string): Promise<string> {
    return new Promise((resolve) => {
        const img = new Image();
        img.onload = () => {
            try {
                const scale = Math.min(1, IMAGE_MAX_EDGE / Math.max(img.width, img.height));
                const canvas = document.createElement('canvas');
                canvas.width = Math.max(1, Math.round(img.width * scale));
                canvas.height = Math.max(1, Math.round(img.height * scale));
                const ctx = canvas.getContext('2d');
                if (!ctx) return resolve(dataUrl);
                ctx.fillStyle = '#fff';
                ctx.fillRect(0, 0, canvas.width, canvas.height);
                ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
                resolve(canvas.toDataURL('image/jpeg', IMAGE_JPEG_QUALITY));
            } catch {
                resolve(dataUrl);
            }
        };
        img.onerror = () => resolve(dataUrl);
        img.src = dataUrl;
    });
}

/** 读取单个文件为消息附件 */
export async function readChatAttachment(file: File): Promise<MessageAttachment> {
    const base = { name: file.name, mime: file.type || undefined, size: file.size };

    if (file.type.startsWith('image/')) {
        const raw = await new Promise<string>((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = () => resolve(String(reader.result));
            reader.onerror = () => reject(reader.error ?? new Error('read error'));
            reader.readAsDataURL(file);
        });
        // GIF 保留原样以尽量保留动画；小图免重绘；大图压缩控制传输/落库体积
        const dataUrl =
            file.type === 'image/gif' || file.size <= IMAGE_INLINE_THRESHOLD
                ? raw
                : await compressImage(raw);
        return { ...base, kind: 'image', dataUrl };
    }

    if (isTextual(file)) {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = () => resolve({ ...base, kind: 'text', text: String(reader.result) });
            reader.onerror = () => reject(reader.error ?? new Error('read error'));
            reader.readAsText(file);
        });
    }

    // 任意文件经文件服务上传；原 File 随附件内存持有（仅发送时上传用，不参与序列化/持久化）
    return { ...base, kind: 'file', rawFile: file };
}

// ==================== 上传（统一文件服务：按配置路由 local/S3） ====================

/** dataURL 转 Blob（图片压缩产物上传用） */
async function dataUrlToBlob(dataUrl: string): Promise<Blob> {
    const res = await fetch(dataUrl);
    return res.blob();
}

/** 取附件上传内容：图片用压缩后的 dataUrl（与 LLM 输入一致），文本用内联内容，其它用原始 File */
async function attachmentToBlob(att: MessageAttachment): Promise<Blob> {
    if (att.kind === 'image' && att.dataUrl) return dataUrlToBlob(att.dataUrl);
    if (att.kind === 'text' && att.text != null) return new Blob([att.text], { type: 'text/plain' });
    if (att.rawFile) return att.rawFile;
    throw new Error(`attachment content missing: ${att.name}`);
}

/**
 * 上传单个附件到统一文件服务（POST /sys/files/upload，返回 fileKey）。
 * 内容以原始文件名上传（后端按扩展名识别 MIME 与下载名）；
 * 幂等：已带 fileKey（队列编辑回填重发等场景）直接返回，不重复上传
 */
export async function uploadChatAttachment(att: MessageAttachment): Promise<string> {
    if (att.fileKey) return att.fileKey;
    const blob = await attachmentToBlob(att);
    const form = new FormData();
    form.append('file', blob, att.name);
    const res = await fetch(getUploadFileUrl(), { method: 'POST', body: form });
    const result = (await res.json()) as { code: number; msg: string; data?: string };
    if (!res.ok || result.code !== 200 || !result.data) {
        throw new Error(result.msg || `upload failed: ${att.name}`);
    }
    return result.data;
}

/** 批量上传附件并回填 fileKey（本地预览字段 dataUrl/text 保留，发送前即时回显不受影响） */
export async function uploadChatAttachments(attachments: MessageAttachment[]): Promise<MessageAttachment[]> {
    return Promise.all(
        attachments.map(async (att) => ({ ...att, fileKey: await uploadChatAttachment(att) })),
    );
}

/**
 * 将附件内容拼进发送文本：
 * - 文本类内联原文（AI 可直接读取）
 * - 图片以 image 段贯穿（见 buildImageSegments），不再以文本占位重复描述
 * - 其它文件以说明占位（展示元数据由消息附件渲染承担）
 */
export function buildMessageContent(text: string, attachments?: MessageAttachment[]): string {
    if (!attachments?.length) return text;
    let content = text;
    for (const att of attachments) {
        if (att.kind === 'text' && att.text) {
            content += `\n\n--- 附件：${att.name} ---\n${att.text}`;
        } else if (att.kind === 'file') {
            content += `\n\n[附件：${att.name}]`;
        }
    }
    return content;
}

/**
 * 将图片附件转为 image 内容段（对齐后端 protocol.ContentSegmentImage：
 * fileKey 引用存 extra 贯穿持久化/回显，展示端以 /sys/files/{fileKey} 访问；
 * LLM 侧由后端解析为 base64 data URL 后转 UserInputImage block；
 * text 为 i18n 占位说明随段持久化）
 */
export function buildImageSegments(
    attachments?: MessageAttachment[],
    placeholder = '',
): ContentSegment[] {
    if (!attachments?.length) return [];
    return attachments
        .filter((att): att is MessageAttachment => att.kind === 'image' && !!att.fileKey)
        .map((att) => ({
            type: 'image' as const,
            text: placeholder,
            extra: { fileKey: att.fileKey, name: att.name },
        }));
}
