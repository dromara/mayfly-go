/**
 * 表格数据导出 / 生成能力
 *
 * 所有格式统一抽象为 TableExportStrategy，并注册到下方两张注册表：
 * - export 类：直接下载文件（Excel / CSV / SQL）
 * - gen    类：生成文本，由调用方弹窗展示（INSERT SQL / JSON）
 *
 * 表格组件只读注册表来构建右键子菜单，因此新增格式无需改动组件——
 * 与本目录方言注册表一致，都走「实现策略 + 自注册」这一条扩展路径。
 */
import { exportCsv, exportExcel, exportFile } from '@/common/utils/export';
import { formatDate } from '@/common/utils/format';
import { DbInst } from '../../db';
import type { TableExportContext, TableExportStrategy } from '../../types';

interface UseTableExportOptions {
    dbId: () => number;
    db: () => string;
    table: () => string;
    datas: () => Record<string, unknown>[];
    columns: () => { columnName?: string; show?: boolean; key: string; title: string }[];
}

// ==================== 内置策略实现 ====================

function buildFilename(table: string): string {
    return `Data-${table}-${formatDate(new Date(), 'YYYYMMDDHHmm')}`;
}

function getVisibleColumnNames(ctx: TableExportContext): string[] {
    return ctx.columns.filter((c) => c.show).map((c) => c.columnName ?? '');
}

/** 按可见列把行数据投影为 JSON 文本 */
function buildJsonText(ctx: TableExportContext): string {
    const jsonObj = ctx.datas.map((row) => {
        const obj: Record<string, unknown> = {};
        for (const col of ctx.columns) {
            if (col.show) {
                obj[col.title] = row[col.key];
            }
        }
        return obj;
    });
    return JSON.stringify(jsonObj, null, 4);
}

/** Excel 导出策略 */
const excelStrategy: TableExportStrategy = {
    key: 'excel',
    labelI18nKey: 'db.exportExcel',
    permission: 'db:data:export',
    execute: async (ctx) => {
        await exportExcel(buildFilename(ctx.table), [{ name: 'Data', columns: getVisibleColumnNames(ctx), datas: ctx.datas }]);
    },
};

/** CSV 导出策略 */
const csvStrategy: TableExportStrategy = {
    key: 'csv',
    labelI18nKey: 'db.exportCsv',
    permission: 'db:data:export',
    execute: (ctx) => {
        exportCsv(buildFilename(ctx.table), getVisibleColumnNames(ctx), ctx.datas);
    },
};

/** SQL (INSERT) 文件导出策略 */
const sqlStrategy: TableExportStrategy = {
    key: 'sql',
    labelI18nKey: 'db.exportSql',
    requireTable: true,
    permission: 'db:data:export',
    execute: async (ctx) => {
        const dbInst = DbInst.getInst(ctx.dbId);
        exportFile(`${buildFilename(ctx.table)}.sql`, await dbInst.genInsertSql(ctx.db, ctx.table, ctx.datas));
    },
};

/** INSERT SQL 生成策略（返回文本，不落盘） */
const insertSqlStrategy: TableExportStrategy = {
    key: 'insertSql',
    labelI18nKey: 'db.genSql',
    requireTable: true,
    resultTitle: 'SQL',
    execute: async (ctx) => await DbInst.getInst(ctx.dbId).genInsertSql(ctx.db, ctx.table, ctx.datas),
};

/** JSON 生成策略（返回文本，不落盘） */
const jsonStrategy: TableExportStrategy = {
    key: 'json',
    labelI18nKey: 'db.genJson',
    resultTitle: 'JSON',
    execute: (ctx) => buildJsonText(ctx),
};

// ==================== 策略注册表 ====================

/** 下载类策略，驱动右键菜单「导出」子菜单 */
const exportStrategies: TableExportStrategy[] = [];

/** 生成类策略，驱动右键菜单「生成」子菜单 */
const genStrategies: TableExportStrategy[] = [];

/**
 * 注册导出/生成策略（开闭原则：新增格式只需注册，无需修改表格组件）
 * @param strategy 导出策略
 * @param category 注册类别：'export' 追加到「导出」子菜单，'gen' 追加到「生成」子菜单
 */
export function registerExportStrategy(strategy: TableExportStrategy, category: 'export' | 'gen' = 'export') {
    (category === 'export' ? exportStrategies : genStrategies).push(strategy);
}

// 内置策略自注册（数组顺序即子菜单展示顺序）
[excelStrategy, csvStrategy, sqlStrategy].forEach((s) => registerExportStrategy(s));
[insertSqlStrategy, jsonStrategy].forEach((s) => registerExportStrategy(s, 'gen'));

// ==================== Composable ====================

export function useTableExport(options: UseTableExportOptions) {
    /**
     * 构建策略执行上下文
     * @param datas 覆盖行数据（如仅导出选中行），缺省为当前全部行
     */
    const buildContext = (datas?: Record<string, unknown>[]): TableExportContext => ({
        dbId: options.dbId(),
        db: options.db(),
        table: options.table(),
        datas: datas ?? options.datas(),
        columns: options.columns(),
    });

    /** 执行指定策略，返回生成类策略的文本结果 */
    const executeStrategy = async (strategy: TableExportStrategy, datas?: Record<string, unknown>[]) => {
        return await strategy.execute(buildContext(datas));
    };

    /** 按 key 执行策略（key 即右键菜单项 id），返回生成类策略的文本结果 */
    const exportByKey = async (key: string, datas?: Record<string, unknown>[]) => {
        const strategy = [...exportStrategies, ...genStrategies].find((s) => s.key === key);
        return strategy ? await executeStrategy(strategy, datas) : undefined;
    };

    /** 获取全部下载类策略 */
    const getExportStrategies = () => exportStrategies;

    /** 获取全部生成类策略 */
    const getGenStrategies = () => genStrategies;

    return { executeStrategy, exportByKey, getExportStrategies, getGenStrategies };
}
