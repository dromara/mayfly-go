/**
 * 导出文件的浏览器落地。
 *
 * 不复用 `common/utils/export.ts` 的 `exportFile`：它固定加 UTF-8 BOM 且写死 `text/plain`。
 * BOM 会让 NDJSON 首行变成 `﻿{"_id"...}`，`head -1 | jq` 直接报错；
 * 而导出的本来就是带类型的文档文件，媒体类型应当如实回给浏览器。
 */

/** 服务端返回的媒体类型，缺省时退回按文件名后缀猜 */
const FALLBACK_TYPES: Record<string, string> = {
    json: 'application/x-ndjson; charset=utf-8',
    ndjson: 'application/x-ndjson; charset=utf-8',
    csv: 'text/csv; charset=utf-8',
};

function guessType(fileName: string): string {
    const ext = fileName.slice(fileName.lastIndexOf('.') + 1).toLowerCase();
    return FALLBACK_TYPES[ext] ?? 'application/octet-stream';
}

/**
 * 触发一次下载并在完成后释放 Blob。
 *
 * 不 revoke 会让整份导出内容驻留在页面内存里，导出几万条后再操作别的集合就是白占的几百 MB。
 */
export function downloadTextFile(fileName: string, content: string, contentType?: string) {
    const blob = new Blob([content], { type: contentType || guessType(fileName) });
    const link = document.createElement('a');
    link.href = URL.createObjectURL(blob);
    link.download = fileName;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(link.href);
}
