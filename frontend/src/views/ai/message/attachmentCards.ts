/**
 * 用户消息附件卡片装配（UserMessageBubble 使用）。
 *
 * 事实源优先级：
 * - 图片：image 段是持久化的唯一事实源；但历史数据里存在「只有 attachments、没有 image 段」的旧消息，
 *   这类图片此前会被整体丢弃（卡片区为空，也不报错）。这里把它们补回卡片列表，
 *   以 fileKey 去重（段卡片已代表的图片不重复出现）。
 * - 文本/文件：只存在于 attachments。
 * - 本地 dataUrl 预览按 name 匹配到 image 段，避免刚上传完还等文件服务响应。
 */
import type { ContentSegment, MessageAttachment } from '../protocol/types';
import { IMAGE_SEGMENT_TYPE } from '../protocol/types';

export function buildCardAttachments(
    segments: ContentSegment[] | undefined,
    attachments: MessageAttachment[] | undefined,
    fallbackImageName: string
): MessageAttachment[] {
    const localPreviews = new Map<string, string>();
    for (const a of attachments ?? []) {
        if (a.kind === 'image' && a.dataUrl && a.name) localPreviews.set(a.name, a.dataUrl);
    }

    const imageCards: MessageAttachment[] = [];
    const segmentFileKeys = new Set<string>();
    for (const seg of segments ?? []) {
        if (seg.type !== IMAGE_SEGMENT_TYPE) continue;
        const fileKey = typeof seg.extra?.fileKey === 'string' ? seg.extra.fileKey : '';
        if (!fileKey) continue;
        segmentFileKeys.add(fileKey);
        const name = typeof seg.extra?.name === 'string' && seg.extra.name ? seg.extra.name : fallbackImageName;
        const dataUrl = localPreviews.get(name);
        imageCards.push({ name, kind: 'image', fileKey, ...(dataUrl ? { dataUrl } : {}) });
    }

    // 旧消息兜底：attachments 里带 fileKey、但没有对应 image 段的图片也要显示
    const legacyImageCards = (attachments ?? []).filter(
        (a) => a.kind === 'image' && (!a.fileKey || !segmentFileKeys.has(a.fileKey))
    );

    const others = (attachments ?? []).filter((a) => a.kind !== 'image');
    return [...imageCards, ...legacyImageCards, ...others];
}
