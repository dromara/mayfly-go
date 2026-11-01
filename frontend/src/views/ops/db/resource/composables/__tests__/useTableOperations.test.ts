/**
 * 编辑表初始化时把「主键约束名」带进表编辑上下文。
 *
 * 为什么值得单测：pg/mssql 撤销主键必须 DROP CONSTRAINT <名字>，名字只能来自 t-keys 接口；
 * 这条接线断了不会报错，只会退化成猜默认名（`<表>_pkey` / `PK_<表>`），
 * 用户改主键时可能删不掉约束。这里钉住「接口返回的主键名 -> formData.primaryKeyName」。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ref } from 'vue';
import type { DbInst } from '../../../db';
import { useTableOperations, type TableCreateDialogState } from '../useTableOperations';

const tableIndex = vi.fn();
const columnMetadata = vi.fn();
const tableKeys = vi.fn();

vi.mock('../../../api', () => ({
    dbApi: {
        tableIndex: { request: (...args: unknown[]) => tableIndex(...args) },
        columnMetadata: { request: (...args: unknown[]) => columnMetadata(...args) },
        tableKeys: { request: (...args: unknown[]) => tableKeys(...args) },
    },
}));
vi.mock('@/views/ops/db/sql-editor/SqlExecBox', () => ({ default: vi.fn() }));
vi.mock('@/hooks/useI18n', () => ({
    Msg: { warning: vi.fn(), error: vi.fn(), success: vi.fn() },
    useI18nCreateTitle: () => '',
    useI18nEditTitle: () => '',
    useI18nDeleteConfirm: async () => true,
}));
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));

function setup() {
    const tableCreateDialog = ref<TableCreateDialogState>({
        visible: false,
        title: '',
        activeName: '1',
        dbId: 0,
        version: '',
        db: '',
        dbType: '',
        parentKey: '',
        data: null,
    });
    const { onEditTable } = useTableOperations({
        // 本用例只走 onEditTable，不触碰实例对象；给一个满足类型的占位
        nowDbInst: ref({ getDialect: () => ({}) } as unknown as DbInst),
        tableCreateDialog,
        ddlDialog: ref({ visible: false, ddl: '' }),
        propsDialog: ref({ visible: false, name: '', attrs: {}, args: null }),
        chooseTableName: ref(''),
        reloadNode: () => undefined,
    });
    return { onEditTable, tableCreateDialog };
}

const editParams = {
    params: {
        id: 3,
        db: 'e2e_probe/public',
        type: 'postgres',
        tableName: 't_len_probe',
        tableComment: '探针',
        parentKey: 'k',
        key: 'k.t',
        version: '16',
    },
} as never;

beforeEach(() => {
    tableIndex.mockReset();
    columnMetadata.mockReset();
    tableKeys.mockReset();
    tableIndex.mockResolvedValue([]);
    columnMetadata.mockResolvedValue([{ columnName: 'id', dataType: 'int8', isPrimaryKey: true, nullable: false }]);
});

describe('编辑表：主键约束名接线', () => {
    it('t-keys 返回主键时，把约束名带进 formData', async () => {
        tableKeys.mockResolvedValue([
            { name: 't_len_probe_pkey', type: 'primary', columns: [{ name: 'id', ordinal: 1 }] },
            { name: 'uk_name', type: 'unique', columns: [{ name: 'name', ordinal: 1 }] },
        ]);
        const { onEditTable, tableCreateDialog } = setup();

        await onEditTable(editParams);

        const data = tableCreateDialog.value.data as { formData: { primaryKeyName?: string } };
        expect(data.formData.primaryKeyName).toBe('t_len_probe_pkey');
    });

    it('t-keys 失败（老后端/无权限）不阻断编辑，约束名留空由方言回退默认命名', async () => {
        tableKeys.mockRejectedValue(new Error('404'));
        const { onEditTable, tableCreateDialog } = setup();

        await onEditTable(editParams);

        const data = tableCreateDialog.value.data as { formData: { primaryKeyName?: string; fields: { res: unknown[] } } };
        expect(data.formData.primaryKeyName).toBeUndefined();
        expect(data.formData.fields.res).toHaveLength(1);
    });
});
