import { afterEach, describe, expect, it, vi } from 'vitest';
import type { FileRowVM } from '../../../types';
import { RENAME_EDIT_CLASS, renameSelectRange, useFileRename } from '../useFileRename';

/** 构造一个完整行视图模型：渲染态字段由 lsFile 一次性补齐，测试也按完整形状构造，不靠断言凑类型 */
const makeRow = (name: string): FileRowVM => ({
    name,
    path: `/${name}`,
    size: 10,
    type: '-',
    mode: '-rw-r--r--',
    modTime: '2026-01-01 00:00:00',
    uid: 0,
    gid: 0,
    isFolder: false,
    icon: 'file',
    nameEdit: false,
    dirSize: '',
    loadingDirSize: false,
    stat: '',
    loadingStat: false,
});

/** 在文档里搭出「编辑框内部」与「编辑框外部」两个点击落点 */
function mountPoints() {
    const editor = document.createElement('div');
    editor.className = RENAME_EDIT_CLASS;
    const input = document.createElement('input');
    editor.appendChild(input);
    document.body.appendChild(editor);

    const outside = document.createElement('td');
    document.body.appendChild(outside);

    return {
        input,
        outside,
        cleanup: () => {
            editor.remove();
            outside.remove();
        },
    };
}

const pointerdown = (el: HTMLElement) => el.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }));

describe('renameSelectRange', () => {
    it('只选文件名主干，不把扩展名包进去', () => {
        expect(renameSelectRange('nginx.conf')).toEqual([0, 5]);
        expect(renameSelectRange('index.html.gz')).toEqual([0, 10]);
    });

    it('无扩展名文件与隐藏文件选中整个名称', () => {
        expect(renameSelectRange('LICENSE')).toEqual([0, 7]);
        // 首个点是隐藏文件前缀而非分隔符，不能选成空区间
        expect(renameSelectRange('.gitignore')).toEqual([0, 10]);
    });
});

describe('useFileRename', () => {
    let cleanupPoints: (() => void) | null = null;
    let rename: ReturnType<typeof useFileRename> | null = null;

    afterEach(() => {
        // 兜底结束编辑态，避免全局监听跨用例残留
        rename?.close();
        rename = null;
        cleanupPoints?.();
        cleanupPoints = null;
    });

    it('进入编辑态并记录旧名', () => {
        const row = makeRow('nginx.conf');
        rename = useFileRename({ rename: vi.fn() });
        rename.start(row);

        expect(row.nameEdit).toBe(true);
    });

    it('编辑框内部的 pointerdown 不取消编辑态', () => {
        const points = mountPoints();
        cleanupPoints = points.cleanup;

        const row = makeRow('nginx.conf');
        rename = useFileRename({ rename: vi.fn() });
        rename.start(row);

        pointerdown(points.input);
        expect(row.nameEdit).toBe(true);
        expect(row.name).toBe('nginx.conf');
    });

    it('编辑框外部的 pointerdown 取消编辑态并还原旧名', () => {
        const points = mountPoints();
        cleanupPoints = points.cleanup;

        const row = makeRow('nginx.conf');
        rename = useFileRename({ rename: vi.fn() });
        rename.start(row);

        // 用户改了名但未提交，点击别处应放弃修改
        row.name = 'nginx2.conf';
        pointerdown(points.outside);

        expect(row.nameEdit).toBe(false);
        expect(row.name).toBe('nginx.conf');
    });

    it('名称未变化时提交不发请求', async () => {
        const doRename = vi.fn();
        const row = makeRow('nginx.conf');
        rename = useFileRename({ rename: doRename });
        rename.start(row);

        await rename.submit();

        expect(doRename).not.toHaveBeenCalled();
        expect(row.nameEdit).toBe(false);
    });

    it('提交成功带上旧名，退出编辑态', async () => {
        const doRename = vi.fn().mockResolvedValue(undefined);
        const row = makeRow('nginx.conf');
        rename = useFileRename({ rename: doRename });
        rename.start(row);
        row.name = 'nginx2.conf';

        await rename.submit();

        expect(doRename).toHaveBeenCalledTimes(1);
        expect(doRename).toHaveBeenCalledWith(row, 'nginx.conf');
        expect(row.nameEdit).toBe(false);
    });

    it('提交失败还原旧名并退出编辑态', async () => {
        const doRename = vi.fn().mockRejectedValue(new Error('rename failed'));
        const row = makeRow('nginx.conf');
        rename = useFileRename({ rename: doRename });
        rename.start(row);
        row.name = 'nginx2.conf';

        await rename.submit();

        expect(row.name).toBe('nginx.conf');
        expect(row.nameEdit).toBe(false);
    });

    it('切换到另一行时还原上一行未提交的修改', () => {
        const points = mountPoints();
        cleanupPoints = points.cleanup;

        const first = makeRow('a.conf');
        const second = makeRow('b.conf');
        rename = useFileRename({ rename: vi.fn() });

        rename.start(first);
        first.name = 'changed.conf';
        rename.start(second);

        expect(first.nameEdit).toBe(false);
        expect(first.name).toBe('a.conf');
        expect(second.nameEdit).toBe(true);

        // 监听已随上一行退出而摘除，外部点击只作用于当前编辑行
        pointerdown(points.outside);
        expect(second.nameEdit).toBe(false);
    });

    it('重复进入同一行不重置已输入的文本', () => {
        const row = makeRow('nginx.conf');
        rename = useFileRename({ rename: vi.fn() });
        rename.start(row);
        row.name = 'nginx2.conf';

        rename.start(row);

        expect(row.name).toBe('nginx2.conf');
        expect(row.nameEdit).toBe(true);
    });
});
