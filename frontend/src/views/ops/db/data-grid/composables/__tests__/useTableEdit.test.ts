/**
 * useTableEdit 的编辑态语义测试
 *
 * 重点守 hasUpdatedFields：表数据页工具条与 SQL 结果页签的「提交/取消」按钮显隐完全由它决定。
 * 这条链曾因事件 payload 漏传而静默失效（按钮再也不出现，且不报错），故把状态迁移逐条钉住：
 * 改值 → 出现按钮；改回原值 → 按钮消失；取消/提交 → 按钮消失。
 */
import { beforeEach, describe, expect, it, vi } from 'vitest';

const genUpdateSql = vi.fn().mockResolvedValue('UPDATE `t` SET `name` = 1;');
// promptExeSql 真实实现里要等用户确认并执行完才回调，这里直接触发成功分支
const promptExeSql = vi.fn((_db: string, _sql: string, _unknown: undefined, onSuccess: () => void) => onSuccess());

vi.mock('../../../db', () => ({
    DbInst: {
        getInst: vi.fn(() => ({ genUpdateSql, promptExeSql })),
    },
}));

import { useTableEdit } from '../useTableEdit';

const createEdit = () =>
    useTableEdit({
        dbId: () => 7,
        db: () => 'testdb',
        table: () => 'tbl',
    });

/** 模拟一次单元格编辑：进入编辑态（记录旧值）→ 写入新值 → 退出编辑态 */
const editCell = (edit: ReturnType<typeof createEdit>, rowData: Record<string, unknown>, key: string, newValue: unknown, rowIndex = 0) => {
    edit.onEnterEditMode(rowData, { key }, rowIndex, 1);
    rowData[key] = newValue;
    edit.onExitEditMode(rowData, { key }, rowIndex);
};

describe('useTableEdit', () => {
    beforeEach(() => {
        genUpdateSql.mockClear();
        promptExeSql.mockClear();
    });

    it('无变更时 hasUpdatedFields 为 false', () => {
        expect(createEdit().hasUpdatedFields.value).toBe(false);
    });

    it('改值后为 true，且原值被记录（供取消还原与生成 WHERE 条件）', () => {
        const edit = createEdit();
        const row = { id: 1, name: 'a' };
        editCell(edit, row, 'name', 'b');

        expect(edit.hasUpdatedFields.value).toBe(true);
        expect(edit.isUpdated(0, 'name')?.oldValue).toBe('a');
    });

    it('未真正改变值（同值退出）不算变更', () => {
        const edit = createEdit();
        const row = { id: 1, name: 'a' };
        editCell(edit, row, 'name', 'a');

        expect(edit.hasUpdatedFields.value).toBe(false);
    });

    it('改回原值后按钮要消失，且该行不留空记录', () => {
        const edit = createEdit();
        const row = { id: 1, name: 'a' };
        editCell(edit, row, 'name', 'b');
        expect(edit.hasUpdatedFields.value).toBe(true);

        // 再次编辑回原值：该列记录移除后按钮应消失（若只清列不判行数，size 会残留导致按钮不消失）
        editCell(edit, row, 'name', 'a');
        expect(edit.hasUpdatedFields.value).toBe(false);
        expect(edit.isUpdated(0, 'name')).toBeUndefined();
    });

    it('取消变更恢复原值并清空待提交集合', () => {
        const edit = createEdit();
        const row = { id: 1, name: 'a' };
        editCell(edit, row, 'name', 'b');

        const onSuccess = vi.fn();
        edit.cancelUpdateFields(onSuccess);

        expect(row.name).toBe('a');
        expect(edit.hasUpdatedFields.value).toBe(false);
        expect(onSuccess).toHaveBeenCalledTimes(1);
    });

    it('提交时按行生成 UPDATE SQL，成功后清空待提交集合', async () => {
        const edit = createEdit();
        const row1 = { id: 1, name: 'a' };
        const row2 = { id: 2, name: 'c' };
        editCell(edit, row1, 'name', 'b', 0);
        editCell(edit, row2, 'name', 'd', 1);
        expect(edit.hasUpdatedFields.value).toBe(true);

        const onSuccess = vi.fn();
        await edit.submitUpdateFields(onSuccess);

        expect(genUpdateSql).toHaveBeenCalledTimes(2);
        expect(promptExeSql).toHaveBeenCalledTimes(1);
        expect(edit.hasUpdatedFields.value).toBe(false);
        expect(onSuccess).toHaveBeenCalledTimes(1);
    });

    it('无待提交变更时提交不发任何 SQL', async () => {
        const edit = createEdit();
        const onSuccess = vi.fn();

        await edit.submitUpdateFields(onSuccess);

        expect(genUpdateSql).not.toHaveBeenCalled();
        expect(promptExeSql).not.toHaveBeenCalled();
        expect(onSuccess).not.toHaveBeenCalled();
    });

    it('同一行多列变更时，撤销单列不影响其余列的待提交状态', () => {
        const edit = createEdit();
        const row = { id: 1, name: 'a', age: 10 };
        editCell(edit, row, 'name', 'b');
        editCell(edit, row, 'age', 20);

        // 撤销其中一列：仍有另一列待提交，按钮必须留着
        editCell(edit, row, 'name', 'a');
        expect(edit.hasUpdatedFields.value).toBe(true);
        expect(edit.isUpdated(0, 'name')).toBeUndefined();
        expect(edit.isUpdated(0, 'age')?.oldValue).toBe(10);
    });

    it('切换表（table 为空）时不进入编辑态', () => {
        const edit = useTableEdit({ dbId: () => 7, db: () => 'testdb', table: () => '' });
        const row = { id: 1, name: 'a' };
        editCell(edit, row, 'name', 'b');

        expect(edit.hasUpdatedFields.value).toBe(false);
    });
});
