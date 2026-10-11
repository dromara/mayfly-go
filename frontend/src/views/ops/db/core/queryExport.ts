import { baseUrl, joinClientParams } from '@/common/request';
import { i18n } from '@/i18n';
import { Msg } from '@/hooks/useI18n';

/** 导出入参：与后端 form.QueryExportForm 一致 */
export interface QueryExportParam {
    dbId: number;
    db: string;
    sql: string;
    format: string;
}

/**
 * 全量导出一次查询的结果集。
 *
 * 不导出页面上已有的行：页面结果被行数上限截断过，拿它导出的是一份缺尾巴的 CSV，
 * 而文件看起来完全正常。这里让服务端重跑同一条语句并把全部行写进文件。
 *
 * 走原生 fetch 而不是 Api 封装：产物是文件流，而统一封装会把响应当 JSON 解析。
 * 失败时后端返回的是 JSON 业务错误（下载头还没写出），因此必须按 content-type 分流，
 * 否则用户会把一份错误 JSON 当数据文件存下来。
 */
export async function exportQueryResult(param: QueryExportParam): Promise<boolean> {
    const url = `${baseUrl}/dbs/${param.dbId}/export-query?${joinClientParams()}`;

    let res: Response;
    try {
        res = await fetch(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ db: param.db, sql: param.sql, format: param.format }),
        });
    } catch (e: unknown) {
        // 浏览器给的原因串本身不是文案键，作为参数嵌进自己的话术里：只报原因会让中文界面
        // 弹出一整句英文，只报「导出失败」又把唯一可定位的信息丢了
        Msg.error('db.exportSendFailed', { detail: e instanceof Error ? e.message : String(e) });
        return false;
    }

    if (!res.ok) {
        Msg.error('db.exportHttpFailed', { status: res.status });
        return false;
    }

    const contentType = res.headers.get('content-type') ?? '';
    if (contentType.includes('application/json')) {
        // 校验没过（非只读语句、格式不支持、被策略拦下）：把服务端的原话给出，不落成文件
        const body = (await res.json()) as { msg?: string };
        Msg.error(body?.msg || 'db.exportFailed');
        return false;
    }

    const filename = resolveFilename(res.headers.get('content-disposition'), param);
    const blob = await res.blob();
    if (!(await isCompleteGzip(blob))) {
        // 下载头已经写出，服务端在这之后的失败（脱敏计划不可用、中途断连等）只能混进已开始的
        // 流里：响应仍是 200 且 content-type 不是 JSON。此时流尾是服务端的原话，摘出来给操作者
        Msg.error('db.exportIncomplete', { detail: await tailText(blob) });
        return false;
    }
    saveBlob(blob, filename);
    return true;
}

/**
 * 产物必须是**能被完整解压**的 gzip，而不是只看魔术号。
 *
 * 服务端中途失败时写进流尾的是一段错误文本，文件开头仍是合法的 gzip 魔术号：只校验魔术号会把
 * 一份少了尾巴的内容存成数据文件，用户拿它做判断甚至导出去，而文件看起来完全正常。解压到结尾
 * 才能把「少了尾巴」这件事在下载当场变成一条错误。
 *
 * 完整解压不额外增加内存峰值：调用前整个响应已经被 res.blob() 取进内存，这里逐块丢弃解码结果。
 * 老环境没有 DecompressionStream 时退回魔术号判定——不能因为校验能力缺失就让人连导出都用不了
 */
async function isCompleteGzip(blob: Blob): Promise<boolean> {
    if (blob.size < 2 || typeof blob.stream !== 'function' || typeof DecompressionStream === 'undefined') {
        return blob.size >= 2 && isGzipHead(blob);
    }
    if (!(await isGzipHead(blob))) {
        return false;
    }
    try {
        const reader = blob.stream().pipeThrough(new DecompressionStream('gzip')).getReader();
        for (;;) {
            const { done } = await reader.read();
            if (done) {
                return true;
            }
        }
    } catch {
        return false;
    }
}

// isGzipHead 前两字节是否为 gzip 魔术号（1f 8b）：不靠文件名推格式
async function isGzipHead(blob: Blob): Promise<boolean> {
    if (blob.size < 2) {
        return false;
    }
    const head = new Uint8Array(await blob.slice(0, 2).arrayBuffer());
    return head[0] === 0x1f && head[1] === 0x8b;
}

// tailText 取流尾可读文本作为错误详情：只取末尾若干字节，gzip 二进制前缀对排查无意义
async function tailText(blob: Blob): Promise<string> {
    const slice = await blob.slice(Math.max(0, blob.size - 400)).text();
    const plain = slice.replace(/[^\u0020-\u007e\u4e00-\u9fff]+/g, ' ').trim();
    return plain || i18n.global.t('db.exportFailed');
}

/** 从响应头取文件名；拿不到就按库名+时间自己给一个，不能出现无文件名下载 */
export function resolveFilename(contentDisposition: string | null, param: QueryExportParam): string {
    const matched = contentDisposition?.match(/filename\*?=(?:UTF-8'')?"?([^";]+)"?/i);
    const name = matched?.[1]?.trim();
    if (name) {
        return decodeURIComponent(name);
    }
    const stamp = new Date().toISOString().slice(0, 19).replace(/[-:T]/g, '');
    return `${param.db || 'query'}-${stamp}.${param.format}.gz`;
}

function saveBlob(blob: Blob, filename: string) {
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    link.click();
    // 不 revoke 会让 blob 一直占内存，直到页面刷新
    URL.revokeObjectURL(url);
}
