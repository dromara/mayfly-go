import { ContextmenuItem } from '@/components/contextmenu';
import { hasPerm } from '@/components/auth/auth';
import { FILE_PERM } from '../constants';
import { isFile } from './useFileOperations';
import type { FileRowVM, MachineFileInfo } from '../../types';

/** 操作可以出现的界面位置：行内动作组 / 行右键菜单 / 底部批量动作条 */
export type ActionSurface = 'row' | 'menu' | 'batch';

/**
 * 文件操作描述：一个操作的全部知识（文案、图标、权限、适用条件、执行动作、出现在哪些位置）
 * 只在此处声明一次，三处 UI 均从描述表派生。
 */
export interface FileActionDef {
    id: string;
    /** 文案 i18n key：按钮文字、tooltip、菜单项、aria-label 同源 */
    labelKey: string;
    icon: string;
    /** 需要的权限码，未声明表示不额外限权 */
    permission?: string;
    /** 该操作出现在哪些位置 */
    surfaces: ActionSurface[];
    /** 危险动作（删除类）：按钮走 danger 样式 */
    danger?: boolean;
    /** 对给定行集是否适用；不适用时该位置不渲染（而不是渲染成禁用） */
    applicable?: (rows: MachineFileInfo[], surface: ActionSurface) => boolean;
    run: (rows: FileRowVM[]) => void | Promise<void>;
}

/**
 * 系统关键目录只在「目标明确是它」的位置隐去动作（行内 / 菜单）。
 *
 * 批量位置不按此过滤：否则「勾选里混进一个系统目录」会让整排按钮凭空消失，
 * 用户无从知道原因；能不能删交给确认框（列出完整路径）与后端判定。
 */
const notProtectedExceptBatch =
    (isProtected: (row: MachineFileInfo) => boolean) =>
    (rows: MachineFileInfo[], surface: ActionSurface) =>
        rows.length > 0 && (surface === 'batch' || !rows.some(isProtected));

/** 执行动作由外部注入，本模块只负责「有哪些操作、在什么条件下出现在哪里」 */
export interface UseFileActionsDeps {
    rename: (row: FileRowVM) => void;
    download: (row: MachineFileInfo) => void;
    remove: (rows: MachineFileInfo[]) => Promise<void>;
    copy: (rows: MachineFileInfo[]) => void;
    move: (rows: MachineFileInfo[]) => void;
    /** 系统关键目录判定：命中的路径不可删除/复制/移动 */
    isProtected: (row: MachineFileInfo) => boolean;
}

/**
 * 行内与菜单位置都是单行操作，统一包成接收行集的签名。
 *
 * 参数定成 FileRowVM：只读后端事实的处理函数（入参 MachineFileInfo）因参数逆变自然可赋值，
 * 需要渲染态的处理函数（入参 FileRowVM）则精硬匹配，无需断言。
 */
const forSingleRow = (fn: (row: FileRowVM) => void | Promise<void>) => (rows: FileRowVM[]) => {
    const row = rows[0];
    if (row) {
        fn(row);
    }
};

export function useFileActions(deps: UseFileActionsDeps) {
    const { isProtected } = deps;

    /**
     * 操作清单：新增一个操作（如 chmod、压缩、复制路径）只需在此追加一条描述，
     * 行内按钮、右键菜单、批量动作条与权限/适用条件会自动跟进。
     */
    const actions: FileActionDef[] = [
        {
            id: 'rename',
            labelKey: 'common.rename',
            icon: 'EditPen',
            permission: FILE_PERM.write,
            surfaces: ['menu'],
            run: forSingleRow(deps.rename),
        },
        {
            id: 'download',
            labelKey: 'machine.download',
            icon: 'Download',
            permission: FILE_PERM.write,
            surfaces: ['row', 'menu'],
            // 目录没有可下载的内容，只有普通文件提供下载
            applicable: (rows) => rows.length === 1 && isFile(rows[0]),
            run: forSingleRow(deps.download),
        },
        {
            id: 'copy',
            labelKey: 'machine.copy',
            icon: 'CopyDocument',
            permission: FILE_PERM.rm,
            surfaces: ['batch'],
            applicable: notProtectedExceptBatch(isProtected),
            run: deps.copy,
        },
        {
            id: 'move',
            labelKey: 'machine.move',
            icon: 'Rank',
            permission: FILE_PERM.rm,
            surfaces: ['batch'],
            applicable: notProtectedExceptBatch(isProtected),
            run: deps.move,
        },
        {
            id: 'delete',
            labelKey: 'common.delete',
            icon: 'Delete',
            permission: FILE_PERM.rm,
            danger: true,
            surfaces: ['row', 'menu', 'batch'],
            applicable: notProtectedExceptBatch(isProtected),
            run: deps.remove,
        },
    ];

    const isApplicable = (action: FileActionDef, rows: MachineFileInfo[], surface: ActionSurface) =>
        !action.applicable || action.applicable(rows, surface);

    /** 权限 + 适用条件同时满足才出现 */
    const isVisible = (action: FileActionDef, rows: MachineFileInfo[], surface: ActionSurface) =>
        (!action.permission || hasPerm(action.permission)) && isApplicable(action, rows, surface);

    /** 某个位置上、给定行集下可见的操作（行内与批量条用） */
    const actionsOn = (surface: ActionSurface, rows: MachineFileInfo[]) =>
        actions.filter((action) => action.surfaces.includes(surface) && isVisible(action, rows, surface));

    /**
     * 行右键菜单项：清单与上面同源，权限交给菜单组件的 v-auth，适用条件用 hideFunc 按当前行判定
     * （菜单在打开时才拿到目标行，无法在构建期用行集过滤）。
     */
    const menuItems = (): ContextmenuItem[] =>
        actions
            .filter((action) => action.surfaces.includes('menu'))
            .map((action) =>
                new ContextmenuItem(action.id, action.labelKey)
                    .withIcon(action.icon)
                    .withPermission(action.permission ?? '')
                    .withHideFunc((row: MachineFileInfo) => !isApplicable(action, [row], 'menu'))
                    .withOnClick((row: FileRowVM) => action.run([row]))
            );

    return { actions, actionsOn, menuItems };
}
