import { describe, it, expect, vi, beforeEach } from 'vitest';

import { exportCsv, exportExcel, exportFile } from '@/common/utils/export';

vi.mock('@/common/utils/export', () => ({
    exportCsv: vi.fn(),
    exportExcel: vi.fn().mockResolvedValue(undefined),
    exportFile: vi.fn(),
}));

const genInsertSql = vi.fn().mockResolvedValue("INSERT INTO `t` (`id`) VALUES (1);");

vi.mock('../../../db', () => ({
    DbInst: { getInst: vi.fn(() => ({ genInsertSql })) },
}));

import { DbInst } from '../../../db';
import { registerExportStrategy, useTableExport } from '../useTableExport';
import type { TableExportStrategy } from '../../../types';

const columns = [
    { columnName: 'id', key: 'id', title: 'ID', show: true },
    { columnName: 'name', key: 'name', title: '名称', show: true },
    // 隐藏列不应出现在任何导出结果里
    { columnName: 'secret', key: 'secret', title: '密码', show: false },
];

const allRows = [
    { id: 1, name: 'a', secret: 'x' },
    { id: 2, name: 'b', secret: 'y' },
];

const createExport = () =>
    useTableExport({
        dbId: () => 7,
        db: () => 'testdb',
        table: () => 'tbl',
        datas: () => allRows,
        columns: () => columns,
    });

beforeEach(() => {
    vi.clearAllMocks();
});

describe('导出策略注册表驱动右键菜单', () => {
    it('内置下载类策略按 Excel → CSV → SQL 顺序自注册（数组顺序即子菜单顺序）', () => {
        const { getExportStrategies } = createExport();
        expect(getExportStrategies().map((s) => s.key)).toEqual(['excel', 'csv', 'sql']);
    });

    it('内置生成类策略按 生成SQL → 生成JSON 顺序自注册', () => {
        const { getGenStrategies } = createExport();
        expect(getGenStrategies().map((s) => s.key)).toEqual(['insertSql', 'json']);
    });

    it('策略自带菜单渲染所需的全部元数据，组件无需认识具体格式', () => {
        const { getExportStrategies, getGenStrategies } = createExport();
        for (const s of [...getExportStrategies(), ...getGenStrategies()]) {
            expect(s.labelI18nKey, `${s.key} 缺 i18n 文案 key`).toMatch(/^db\./);
            expect(typeof s.execute, `${s.key} 缺 execute`).toBe('function');
        }
        // 导出属受权限管控的操作，任一策略漏声明 permission 都会让菜单项绕过鉴权
        expect(getExportStrategies().every((s) => s.permission === 'db:data:export')).toBe(true);
        // 依赖具体表的策略必须声明 requireTable，组件据此在库级节点隐藏菜单项
        expect(getExportStrategies().find((s) => s.key === 'sql')?.requireTable).toBe(true);
        expect(getGenStrategies().find((s) => s.key === 'insertSql')?.requireTable).toBe(true);
        // 生成类策略的结果要弹窗展示，必须有标题；下载类不需要
        expect(getGenStrategies().every((s) => !!s.resultTitle)).toBe(true);
        expect(getExportStrategies().every((s) => !s.resultTitle)).toBe(true);
    });

    it('registerExportStrategy 是可用扩展点：注册后立刻出现在对应子菜单并能被派发', async () => {
        const execute = vi.fn().mockReturnValue('custom-result');
        const custom: TableExportStrategy = { key: 'custom-md', labelI18nKey: 'db.exportMd', execute };

        const { getGenStrategies, exportByKey } = createExport();
        registerExportStrategy(custom, 'gen');
        try {
            expect(getGenStrategies().map((s) => s.key)).toContain('custom-md');
            expect(await exportByKey('custom-md')).toBe('custom-result');
            expect(execute).toHaveBeenCalledWith(expect.objectContaining({ dbId: 7, db: 'testdb', table: 'tbl' }));
        } finally {
            // getGenStrategies() 返回的是注册表本身，测试内注册的策略须自行摘除，避免污染其他用例
            getGenStrategies().splice(getGenStrategies().indexOf(custom), 1);
        }
        expect(getGenStrategies().map((s) => s.key)).toEqual(['insertSql', 'json']);
    });
});

describe('策略派发', () => {
    it('exportByKey 派发 CSV：只导出可见列，文件名含表名', async () => {
        const { exportByKey } = createExport();
        await exportByKey('csv');
        expect(exportCsv).toHaveBeenCalledTimes(1);
        const [filename, cols, datas] = (exportCsv as unknown as ReturnType<typeof vi.fn>).mock.calls[0];
        expect(filename).toMatch(/^Data-tbl-\d{12}$/);
        expect(cols).toEqual(['id', 'name']);
        expect(datas).toBe(allRows);
    });

    it('exportByKey 派发 Excel：以工作表形式包装可见列与数据', async () => {
        const { exportByKey } = createExport();
        await exportByKey('excel');
        expect(exportExcel).toHaveBeenCalledTimes(1);
        const [filename, sheets] = (exportExcel as unknown as ReturnType<typeof vi.fn>).mock.calls[0];
        expect(filename).toMatch(/^Data-tbl-\d{12}$/);
        expect(sheets).toEqual([{ name: 'Data', columns: ['id', 'name'], datas: allRows }]);
    });

    it('exportByKey 派发 SQL：走方言实例生成 INSERT 语句并落盘为 .sql', async () => {
        const { exportByKey } = createExport();
        await exportByKey('sql');
        expect(DbInst.getInst).toHaveBeenCalledWith(7);
        expect(genInsertSql).toHaveBeenCalledWith('testdb', 'tbl', allRows);
        expect(exportFile).toHaveBeenCalledTimes(1);
        const [filename, content] = (exportFile as unknown as ReturnType<typeof vi.fn>).mock.calls[0];
        expect(filename).toMatch(/^Data-tbl-\d{12}\.sql$/);
        expect(content).toContain('INSERT INTO');
    });

    it('未知 key 静默返回 undefined，不抛错', async () => {
        const { exportByKey } = createExport();
        expect(await exportByKey('no-such-format')).toBeUndefined();
        expect(exportCsv).not.toHaveBeenCalled();
        expect(exportFile).not.toHaveBeenCalled();
    });
});

describe('生成类策略：文本结果供弹窗展示', () => {
    it('生成 JSON：按可见列的标题投影，隐藏列不外泄', async () => {
        const { exportByKey } = createExport();
        const txt = await exportByKey('json');
        expect(typeof txt).toBe('string');
        const parsed = JSON.parse(txt as string);
        expect(parsed).toEqual([
            { ID: 1, 名称: 'a' },
            { ID: 2, 名称: 'b' },
        ]);
    });

    it('生成 INSERT SQL：返回方言生成的语句文本', async () => {
        const { exportByKey } = createExport();
        expect(await exportByKey('insertSql')).toContain('INSERT INTO');
        // 生成类只返回文本，不应触发任何下载
        expect(exportFile).not.toHaveBeenCalled();
    });

    it('datas 覆盖参数生效：仅对选中行生成，而非当前全部行', async () => {
        const { exportByKey, executeStrategy, getGenStrategies } = createExport();
        const selected = [{ id: 2, name: 'b', secret: 'y' }];

        expect(JSON.parse((await exportByKey('json', selected)) as string)).toEqual([{ ID: 2, 名称: 'b' }]);

        // 右键「生成」子菜单走的是 executeStrategy（直接传策略对象），同样要支持覆盖
        const jsonStrategy = getGenStrategies().find((s) => s.key === 'json')!;
        const txt = await executeStrategy(jsonStrategy, selected);
        expect(JSON.parse(txt as string)).toEqual([{ ID: 2, 名称: 'b' }]);
    });

    it('未传 datas 时缺省导出全部行', async () => {
        const { exportByKey } = createExport();
        const json = JSON.parse((await exportByKey('json')) as string);
        expect(json).toHaveLength(allRows.length);
    });
});
