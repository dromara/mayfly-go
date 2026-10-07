import { beforeEach, describe, expect, it, vi } from 'vitest';
import { effect } from 'vue';

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));
vi.mock('@/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }));
vi.mock('@/hooks/useI18n', () => ({
    Msg: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), deleteSuccess: vi.fn() },
    useI18nDeleteConfirm: vi.fn().mockResolvedValue(undefined),
}));
vi.mock('@/views/ops/machine/api', () => ({
    machineApi: {
        renameFile: { request: vi.fn().mockResolvedValue(undefined) },
        rmFile: { request: vi.fn().mockResolvedValue(undefined) },
        createFile: { request: vi.fn().mockResolvedValue(undefined) },
        cpFile: { request: vi.fn().mockResolvedValue(undefined) },
        mvFile: { request: vi.fn().mockResolvedValue(undefined) },
    },
    uploadFile: vi.fn(),
    uploadFolder: vi.fn(),
    buildFileDownloadUrl: vi.fn(() => 'built-url'),
}));
// 全站统一的隐藏 iframe 下载入口
vi.mock('@/common/utils/file', () => ({ downloadFile: vi.fn() }));

import { downloadFile as downloadByIframe } from '@/common/utils/file';
import { machineApi } from '@/views/ops/machine/api';
import type { MachineFileInfo } from '../../../types';
import { useFileOperations } from '../useFileOperations';

const endpoints = machineApi as unknown as Record<string, { request: ReturnType<typeof vi.fn> }>;

const row = (name: string, path: string, type = '-'): MachineFileInfo => ({ name, path, type }) as MachineFileInfo;

/** 构造一份受控的 composable 实例：目录、刷新次数都可观察 */
function build(nowPath = '/srv') {
    const ctx = { nowPath };
    let refreshCount = 0;
    const api = useFileOperations({
        machineId: () => 1,
        authCertName: () => 'root',
        fileId: () => 7,
        protocol: () => 1,
        nowPath: () => ctx.nowPath,
        setLoading: () => {},
        refresh: () => {
            refreshCount++;
        },
        uploadMaxFileSize: () => '1GB',
    });
    return { api, ctx, refreshTimes: () => refreshCount };
}

beforeEach(() => {
    Object.values(endpoints).forEach((item) => item.request.mockClear());
});

describe('剪贴板状态必须可被响应式追踪', () => {
    /**
     * 回归：copyOrMvFile 曾是普通对象，push/清空不触发重渲染，
     * 表现为「点了复制但粘贴条不出现，要再碰一下表格才冒出来」。
     */
    it('复制与取消都会驱动依赖它的渲染副作用重跑', () => {
        const { api } = build();
        let runs = 0;

        effect(() => {
            void api.copyOrMvFile.paths.length;
            runs++;
        });
        expect(runs).toBe(1);

        api.copyFile([row('a', '/srv/a')]);
        expect(runs).toBe(2);
        expect(api.copyOrMvFile.paths).toEqual(['/srv/a']);

        api.cancelCopy();
        expect(runs).toBe(3);
        expect(api.copyOrMvFile.paths).toEqual([]);
    });

    it('复制与移动共用同一份剪贴板并记录来源目录', () => {
        const { api, ctx } = build('/srv');

        api.copyFile([row('a', '/srv/a')]);
        expect(api.isCpFile()).toBe(true);

        api.mvFile([row('b', '/srv/b')]);
        expect(api.isCpFile()).toBe(false);
        expect(api.copyOrMvFile.fromPath).toBe(ctx.nowPath);
    });
});

describe('粘贴', () => {
    it('同目录粘贴被前置拦下，不发请求', async () => {
        const { api } = build('/srv');
        api.copyFile([row('a', '/srv/a')]);

        await expect(api.pasteFile()).rejects.toThrow();
        expect(endpoints.cpFile.request).not.toHaveBeenCalled();
    });

    it('跨目录粘贴带上来源路径与目标目录，成功后清空剪贴板', async () => {
        const { api, ctx } = build('/srv');
        api.copyFile([row('a', '/srv/a')]);
        ctx.nowPath = '/srv/other';

        await api.pasteFile();

        expect(endpoints.cpFile.request).toHaveBeenCalledWith(expect.objectContaining({ paths: ['/srv/a'], toPath: '/srv/other', machineId: 1, fileId: 7 }));
        expect(api.copyOrMvFile.paths).toEqual([]);
    });
});

describe('重命名', () => {
    it('按当前目录拼出源与目标路径，不再用 parseInt 转换 id', async () => {
        const { api } = build('/srv');

        await api.fileRename(row('new', '/srv/new'), 'old');

        expect(endpoints.renameFile.request).toHaveBeenCalledWith(expect.objectContaining({ machineId: 1, fileId: 7, path: '/srv/old', newname: '/srv/new' }));
    });

    it('新名称为空时不发请求', async () => {
        const { api } = build('/srv');

        await expect(api.fileRename(row('', '/srv/x'), 'old')).rejects.toThrow();
        expect(endpoints.renameFile.request).not.toHaveBeenCalled();
    });
});

describe('新建文件/目录的前置校验', () => {
    it('名称为空不发请求', async () => {
        const { api } = build('/srv');
        await expect(api.createFile('', 'd')).rejects.toThrow();
        expect(endpoints.createFile.request).not.toHaveBeenCalled();
    });

    /** 名称带分隔符会拼出与用户预期不同的路径（甚至越级到别的目录），必须在前置拦下 */
    it('名称含路径分隔符不发请求', async () => {
        const { api } = build('/srv');
        await expect(api.createFile('../etc/x', 'd')).rejects.toThrow();
        expect(endpoints.createFile.request).not.toHaveBeenCalled();
    });

    it('校验通过后返回创建后的完整路径并刷新列表', async () => {
        const { api, refreshTimes } = build('/srv');

        const created = await api.createFile('app', 'd');

        expect(created).toBe('/srv/app');
        expect(endpoints.createFile.request).toHaveBeenCalledWith(expect.objectContaining({ path: '/srv/app', type: 'd' }));
        expect(refreshTimes()).toBe(1);
    });
});

describe('系统关键目录保护', () => {
    it('精确命中受保护路径，子目录不受影响', () => {
        const { api } = build();

        expect(api.dontOperate(row('/', '/'))).toBe(true);
        expect(api.dontOperate(row('etc', '/etc'))).toBe(true);
        expect(api.dontOperate(row('nginx', '/etc/nginx'))).toBe(false);
        expect(api.dontOperate(row('local', '/usr/local'))).toBe(false);
    });
});

describe('下载', () => {
    /**
     * 必须走全站统一的 iframe 下载入口：自己拼 `a[target=_blank]` 会先开新标签页
     * 再由浏览器关闭，表现为点一下下载整页闪动。
     */
    it('委派给统一下载入口，并交给它完整的下载地址', () => {
        const { api } = build('/srv');

        api.downloadFile(row('a.txt', '/srv/a.txt'));

        expect(downloadByIframe).toHaveBeenCalledWith('built-url');
    });
});
