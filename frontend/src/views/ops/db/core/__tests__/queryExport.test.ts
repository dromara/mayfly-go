import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const errors = vi.fn();

vi.mock('@/hooks/useI18n', () => ({
    Msg: {
        error: (...args: unknown[]) => errors(...args),
        success: vi.fn(),
    },
}));

vi.mock('@/common/request', () => ({
    baseUrl: '/api',
    joinClientParams: () => 'token=t&clientId=c',
}));

import { exportQueryResult, resolveFilename, type QueryExportParam } from '../queryExport';

const param: QueryExportParam = { dbId: 1, db: 'mayfly_go', sql: 'select 1', format: 'csv' };

/** 造一个响应：content-type 决定走下载还是走错误分支 */
function response(init: { status?: number; contentType: string; json?: unknown; body?: Blob }, headers?: Record<string, string>) {
    return {
        ok: (init.status ?? 200) < 400,
        status: init.status ?? 200,
        headers: { get: (name: string) => headers?.[name.toLowerCase()] ?? init.contentType },
        json: async () => init.json,
        blob: async () => init.body ?? new Blob([CSV_GZ]),
    } as unknown as Response;
}

/** gzip 魔术号开头：产物真伪靠它判，不看文件名 */
function gzipBlob(parts: BlobPart[]): Blob {
    return new Blob([new Uint8Array([0x1f, 0x8b]), ...parts]);
}

/**
 * 「a,b\n1,2\n」的真实 gzip 字节（node zlib 生成）。
 *
 * 成功用例必须是能被完整解压的产物：校验已从「看魔术号」收紧成「解压到结尾」，
 * 拼一个假头会让成功路径自己测不过
 */
const CSV_GZ = new Uint8Array([
    31, 139, 8, 0, 0, 0, 0, 0, 0, 19, 75, 212, 73, 226, 50, 212, 49, 226, 2, 0, 123, 7, 151, 10, 8, 0, 0, 0,
]);

describe('queryExport', () => {
    let saved: string[] = [];
    let createObjectUrl: ReturnType<typeof vi.fn>;

    beforeEach(() => {
        errors.mockReset();
        saved = [];
        createObjectUrl = vi.fn(() => 'blob:mock');
        vi.stubGlobal('URL', { createObjectURL: createObjectUrl, revokeObjectURL: vi.fn() });
        vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
            saved.push(this.download);
        });
    });

    afterEach(() => {
        vi.unstubAllGlobals();
        vi.restoreAllMocks();
    });

    it('校验失败返回的是 JSON 业务错误：必须报错而不是存成一个数据文件', async () => {
        vi.stubGlobal(
            'fetch',
            vi.fn(async () => response({ contentType: 'application/json', json: { code: 400, msg: '只允许查询类语句' } }, {}))
        );

        const ok = await exportQueryResult(param);
        expect(ok).toBe(false);
        expect(errors).toHaveBeenCalledWith('只允许查询类语句');
        expect(createObjectUrl).not.toHaveBeenCalled();
        expect(saved).toHaveLength(0);
    });

    it('成功时按响应头的文件名落盘', async () => {
        vi.stubGlobal(
            'fetch',
            vi.fn(async () =>
                response(
                    { contentType: 'application/octet-stream', body: new Blob([CSV_GZ]) },
                    { 'content-disposition': 'attachment; filename=mayfly_go-20261008.csv.gz' }
                )
            )
        );

        expect(await exportQueryResult(param)).toBe(true);
        expect(saved).toEqual(['mayfly_go-20261008.csv.gz']);
        expect(errors).not.toHaveBeenCalled();
    });

    it('产物不是 gzip 时不落盘：下载头写出后的失败只会混进流里', async () => {
        vi.stubGlobal(
            'fetch',
            vi.fn(async () =>
                response(
                    { contentType: 'application/octet-stream', body: new Blob(['{"code":400,"msg":"导出失败：脱敏不可用"}']) },
                    { 'content-disposition': 'attachment; filename=broken.csv.gz' }
                )
            )
        );

        expect(await exportQueryResult(param)).toBe(false);
        expect(saved).toHaveLength(0);
        expect(errors).toHaveBeenCalledWith('db.exportIncomplete', expect.objectContaining({ detail: expect.stringContaining('脱敏不可用') }));
    });

    it('有合法 gzip 头但解压不完（尾巴被错误文本接管）也不落盘：只看魔术号会把少了尾巴的内容当数据文件存下来', async () => {
        vi.stubGlobal(
            'fetch',
            vi.fn(async () =>
                response(
                    {
                        contentType: 'application/octet-stream',
                        body: gzipBlob(['{"code":500,"msg":"脱敏计划不可用，导出中止"}']),
                    },
                    { 'content-disposition': 'attachment; filename=half.csv.gz' }
                )
            )
        );

        expect(await exportQueryResult(param)).toBe(false);
        expect(saved).toHaveLength(0);
        expect(errors).toHaveBeenCalledWith('db.exportIncomplete', expect.objectContaining({ detail: expect.stringContaining('脱敏计划不可用') }));
    });

    it('HTTP 层失败的话术走 i18n 键并带上状态码：裸串会让中文界面弹出一截英文', async () => {
        vi.stubGlobal(
            'fetch',
            vi.fn(async () => response({ status: 500, contentType: 'text/html' }))
        );

        expect(await exportQueryResult(param)).toBe(false);
        expect(errors).toHaveBeenCalledWith('db.exportHttpFailed', { status: 500 });
    });

    it('请求根本没发出去时也是 i18n 键 + 原因参数', async () => {
        vi.stubGlobal(
            'fetch',
            vi.fn(async () => {
                throw new Error('Failed to fetch');
            })
        );

        expect(await exportQueryResult(param)).toBe(false);
        expect(errors).toHaveBeenCalledWith('db.exportSendFailed', { detail: 'Failed to fetch' });
    });

    it('没有响应头时也要给出文件名：无文件名的下载会让用户不知道拿到了什么', () => {
        const name = resolveFilename(null, param);
        expect(name.startsWith('mayfly_go-')).toBe(true);
        expect(name.endsWith('.csv.gz')).toBe(true);
        // 库名为空也不能留下 "-时间.csv" 这种没有主体名的文件
        expect(resolveFilename('', { ...param, db: '' }).startsWith('query-')).toBe(true);
    });
});
