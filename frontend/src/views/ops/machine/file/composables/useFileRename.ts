import { getCurrentInstance, onBeforeUnmount } from 'vue';
import type { FileRowVM } from '../../types';

/** 行内重命名编辑态的根元素类名，用于判定 pointerdown 是否落在编辑框之外 */
export const RENAME_EDIT_CLASS = 'machine-file-rename';

/**
 * 进入重命名时的预选区间：只选文件名主干（不含扩展名），改主名不误伤后缀；
 * 无扩展名文件与隐藏文件（.gitignore，首个点不是分隔符）选中整个名称
 */
export const renameSelectRange = (name: string): [number, number] => {
    const dotIdx = name.lastIndexOf('.');
    return [0, dotIdx > 0 ? dotIdx : name.length];
};

export interface UseFileRenameOptions {
    /**
     * 执行重命名；失败必须抛错，由本状态机还原旧名并退出编辑态
     */
    rename: (row: FileRowVM, oldname: string) => Promise<void>;
}

/**
 * 机器文件的行内重命名状态机：同一时刻至多一行处于编辑态，进入时记录旧名，
 * 回车提交 / Esc 或点击编辑框之外取消。
 *
 * 退出信号取「编辑框之外的 pointerdown」而非输入框 blur：右键菜单等弹层卸载时，其
 * FocusScope 会把焦点还给弹层打开前的元素，那一下 blur 不是用户意图，若以 blur 判定
 * 取消，用户还没开始输入，编辑框就会被关掉。pointerdown 只能由真实交互产生，不受焦点管理影响。
 */
export function useFileRename(options: UseFileRenameOptions) {
    let editingRow: FileRowVM | null = null;
    let oldname = '';

    const onOutsidePointerdown = (event: Event) => {
        const row = editingRow;
        if (!row) {
            return;
        }
        const target = event.target as HTMLElement | null;
        // 落在编辑框内部的点击交给输入框自身处理（光标定位、选区、清除等）
        if (target?.closest(`.${RENAME_EDIT_CLASS}`)) {
            return;
        }
        cancel();
    };

    /** 结束编辑态并摘掉全局监听 */
    const close = () => {
        if (editingRow) {
            editingRow.nameEdit = false;
            editingRow = null;
        }
        oldname = '';
        document.removeEventListener('pointerdown', onOutsidePointerdown, true);
    };

    /** 放弃修改：还原旧名并退出编辑态 */
    const cancel = () => {
        if (editingRow && oldname) {
            editingRow.name = oldname;
        }
        close();
    };

    /** 进入编辑态；重复进入同一行不重置已输入的文本 */
    const start = (row: FileRowVM) => {
        if (row.nameEdit) {
            return;
        }
        // 切换编辑行时先还原上一行未提交的修改
        cancel();
        oldname = row.name;
        row.nameEdit = true;
        editingRow = row;
        document.addEventListener('pointerdown', onOutsidePointerdown, true);
    };

    /** 提交当前编辑行的名称；名称未变化时不发起请求 */
    const submit = async () => {
        const row = editingRow;
        if (!row) {
            return;
        }
        const from = oldname;
        if (row.name === from) {
            close();
            return;
        }
        try {
            await options.rename(row, from);
            // 成功后列表会被刷新重建行对象，这里只需摘掉监听
            close();
        } catch {
            cancel();
        }
    };

    // 组件卸载时兜底摘监听，避免编辑态未结束就切走导致监听泄漏；非组件环境（如单测）下跳过
    if (getCurrentInstance()) {
        onBeforeUnmount(close);
    }

    return { start, submit, cancel, close, editClass: RENAME_EDIT_CLASS };
}
