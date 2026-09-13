import type { languages, IRange } from '@/components/monaco/setup';
import type { DbInst } from '../db';
import { buildColumnSuggestion, buildTableSuggestion } from './format';

/**
 * 表 / 字段联想建议的读取（数据来自 DbInst，形状由 format.ts 统一）
 *
 * 为什么是本模块的函数而不是 DbInst 的方法：建议项的形状（monaco CompletionItem、左侧图标 kind、
 * 右侧灰显 description）属于编辑器层，DbInst 只负责数据加载与缓存。这两者一旦混在 db.ts，
 * 补全所需 monaco 枚举就会顺着「db 模块入口 → db.ts」静态回流，
 * 使表数据页、资源树、实例列表等所有 DB 页面都被迫下载编辑器主体（详见 components/monaco/setup.ts）。
 */

/**
 * 表名联想：insertText 为裸表名，由贡献者统一按方言包裹引用符。
 */
export async function tableSuggestions(dbInst: DbInst, dbName: string, range: IRange): Promise<languages.CompletionItem[]> {
    const tables = await dbInst.loadTables(dbName);
    return (tables ?? []).map((tableMeta, index) => buildTableSuggestion(tableMeta, index, range));
}

/** 字段联想：数据取自库级字段提示缓存， insertText 同样为裸字段名 */
export async function columnSuggestions(dbInst: DbInst, db: string, tableName: string, range: IRange): Promise<languages.CompletionItem[]> {
    const hints = await dbInst.loadDbHints(db);
    // 字段提示原始串格式：`字段名  [类型][注释]`，如 `create_time  [datetime][创建时间]`
    return (hints[tableName] ?? []).map((raw, index) => buildColumnSuggestion(raw, index, range));
}
