import type { languages, IRange } from '@/components/monaco/setup';
import type { CompletionDbInst } from './types';
import { buildColumnSuggestion, buildTableSuggestion, buildViewSuggestion } from './format';

/**
 * 表 / 字段联想建议的读取（数据来自实例的元数据加载能力，形状由 format.ts 统一）
 *
 * 为什么是本模块的函数而不是 DbInst 的方法：建议项的形状（monaco CompletionItem、左侧图标 kind、
 * 右侧灰显 description）属于编辑器层，DbInst 只负责数据加载与缓存。这两者一旦混在 db.ts，
 * 补全所需 monaco 枚举就会顺着「db 模块入口 → db.ts」静态回流，
 * 使表数据页、资源树、实例列表等所有 DB 页面都被迫下载编辑器主体（详见 components/monaco/setup.ts）。
 */

/**
 * 表名联想：insertText 为裸表名，由贡献者统一按方言包裹引用符。
 * 一并纳入视图（表与视图在 FROM/JOIN 位置语义等价），视图用不同图标区分。
 */
export async function tableSuggestions(dbInst: CompletionDbInst, dbName: string, range: IRange): Promise<languages.CompletionItem[]> {
    const [tables, views] = await Promise.all([dbInst.loadTables(dbName), dbInst.loadViews(dbName)]);
    const items = (tables ?? []).map((tableMeta, index) => buildTableSuggestion(tableMeta, index, range));
    const viewItems = views.map((view, index) => buildViewSuggestion(view, index, range));
    return [...items, ...viewItems];
}

/** 字段联想：数据取自按表结构化列元数据（c-metadata / loadColumns，按表缓存），insertText 为裸字段名 */
export async function columnSuggestions(dbInst: CompletionDbInst, db: string, tableName: string, range: IRange): Promise<languages.CompletionItem[]> {
    const columns = await dbInst.loadColumns(db, tableName);
    return columns.map((col, index) => buildColumnSuggestion(col, index, range));
}
