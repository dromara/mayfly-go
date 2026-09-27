import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));
vi.mock('@/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }));
vi.mock('@/hooks/useI18n', () => ({
    Msg: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), deleteSuccess: vi.fn() },
    useI18nDeleteConfirm: vi.fn(),
}));
vi.mock('@/views/ops/machine/api', () => ({
    machineApi: {},
    uploadFile: vi.fn(),
    uploadFolder: vi.fn(),
    buildFileDownloadUrl: vi.fn(() => 'url'),
}));

// 权限判定可控：声明表按 permission 过滤，测试里通过 perms 决定放行哪些权限码
const perms = new Set<string>();
vi.mock('@/components/auth/auth', () => ({ hasPerm: (code: string) => !code || perms.has(code) }));

import type { FileRowVM, MachineFileInfo } from '../../../types';
import { FILE_PERM } from '../../constants';
import { useFileActions, type UseFileActionsDeps } from '../useFileActions';

const row = (path: string, type: string = '-'): FileRowVM => ({
    name: path.split('/').pop() ?? path,
    path,
    type,
    size: 1,
    mode: '',
    modTime: '',
    uid: 0,
    gid: 0,
    isFolder: type === 'd',
    icon: 'file',
    nameEdit: false,
    dirSize: '',
    loadingDirSize: false,
    stat: '',
    loadingStat: false,
});

function build(deps: Partial<UseFileActionsDeps> = {}) {
    const stub = vi.fn();
    return useFileActions({
        rename: stub,
        download: stub,
        remove: stub,
        copy: stub,
        move: stub,
        isProtected: (file: MachineFileInfo) => file.path === '/etc',
        ...deps,
    });
}

beforeEach(() => {
    perms.add(FILE_PERM.write);
    perms.add(FILE_PERM.rm);
});

describe('操作声明表的结构约束', () => {
    it('id 唯一，且每个操作都声明了出现位置', () => {
        const { actions } = build();
        const ids = actions.map((a) => a.id);

        expect(new Set(ids).size).toBe(ids.length);
        for (const action of actions) {
            expect(action.surfaces.length).toBeGreaterThan(0);
            expect(action.labelKey).toBeTruthy();
            expect(action.icon).toBeTruthy();
        }
    });

    /** 位置划分是本模块的交互契约：重命名只走右键菜单（行内铅笔因语义歧义被否决） */
    it('每个操作出现在约定的位置', () => {
        const { actions } = build();
        const surfacesOf = (id: string) => actions.find((a) => a.id === id)?.surfaces ?? [];

        expect(surfacesOf('rename')).toEqual(['menu']);
        expect(surfacesOf('download')).toEqual(['row', 'menu']);
        expect(surfacesOf('copy')).toEqual(['batch']);
        expect(surfacesOf('move')).toEqual(['batch']);
        expect(surfacesOf('delete')).toEqual(['row', 'menu', 'batch']);
    });
});

describe('按位置与条件派生可见操作', () => {
    it('目录行不提供下载，普通文件行提供下载与删除', () => {
        const { actionsOn } = build();

        const fileRow = actionsOn('row', [row('/srv/a.txt')]);
        expect(fileRow.map((a) => a.id)).toEqual(['download', 'delete']);

        const dirRow = actionsOn('row', [row('/srv/logs', 'd')]);
        expect(dirRow.map((a) => a.id)).toEqual(['delete']);
    });

    /**
     * 保护目录的隐去只作用于「目标明确」的行内位置；批量动作照常出现。
     * 否则勾选里混进一个系统目录会让整排按钮凭空消失，而菜单/行内单目标时不给删除才是可理解的。
     */
    it('系统关键目录在行内不给删除，但批量动作不因此消失', () => {
        const { actionsOn } = build();

        expect(actionsOn('row', [row('/etc', 'd')]).map((a) => a.id)).toEqual([]);
        expect(actionsOn('batch', [row('/etc', 'd'), row('/srv/a.txt')]).map((a) => a.id)).toEqual(['copy', 'move', 'delete']);
    });

    it('批量位置对空集合不给任何动作', () => {
        const { actionsOn } = build();
        expect(actionsOn('batch', [])).toEqual([]);
    });

    /**
     * 权限码与后端一致：cp/mv/rm 三个端点都要求 machine:file:rm，
     * 所以缺 rm 时删除与批量复制/移动会一并消失；下载属写操作，受 machine:file:write 控制。
     */
    it('按权限码过滤所有位置', () => {
        perms.delete(FILE_PERM.write);
        expect(build().actionsOn('row', [row('/srv/a.txt')]).map((a) => a.id)).toEqual(['delete']);

        perms.add(FILE_PERM.write);
        perms.delete(FILE_PERM.rm);
        const withoutRm = build();
        expect(withoutRm.actionsOn('row', [row('/srv/a.txt')]).map((a) => a.id)).toEqual(['download']);
        expect(withoutRm.actionsOn('batch', [row('/srv/a.txt')])).toEqual([]);
    });
});

describe('右键菜单项与声明表同源', () => {
    it('菜单项 id 与声明表中声明了 menu 位置的操作一一对应', () => {
        const { actions, menuItems } = build();
        const menuIds = actions.filter((a) => a.surfaces.includes('menu')).map((a) => a.id);

        expect(menuItems().map((item) => item.clickId)).toEqual(menuIds);
    });

    it('菜单项携带权限码，由菜单组件的 v-auth 二次把关', () => {
        const { menuItems } = build();
        const rename = menuItems().find((item) => item.clickId === 'rename');

        expect(rename?.permission).toBe(FILE_PERM.write);
    });

    /** hideFunc 返回 true 表示该项不渲染：目录行不该出现下载 */
    it('菜单按当前行判定适用条件', () => {
        const { menuItems } = build();
        const download = menuItems().find((item) => item.clickId === 'download');

        expect(download?.hideFunc?.(row('/srv/logs', 'd'))).toBe(true);
        expect(download?.hideFunc?.(row('/srv/a.txt'))).toBe(false);
    });
});

describe('执行动作委派给注入依赖', () => {
    it('行内下载只把该行交给处理函数', () => {
        const download = vi.fn();
        const { actionsOn } = build({ download });
        const target = row('/srv/a.txt');

        actionsOn('row', [target])
            .find((a) => a.id === 'download')!
            .run([target]);

        expect(download).toHaveBeenCalledWith(target);
    });

    it('批量删除把整个勾选集合交给处理函数', () => {
        const remove = vi.fn().mockResolvedValue(undefined);
        const { actionsOn } = build({ remove });
        const rows = [row('/srv/a.txt'), row('/srv/b.txt')];

        actionsOn('batch', rows)
            .find((a) => a.id === 'delete')!
            .run(rows);

        expect(remove).toHaveBeenCalledWith(rows);
    });
});
