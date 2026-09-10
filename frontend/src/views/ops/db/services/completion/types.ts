import type { editor, languages, Position, IRange } from 'monaco-editor';
import type { DbDialect } from '../../dialect';
import type { QuotePair } from './quoter';
import type { CursorClause } from './sqlContext';
import type { DbInst } from '../../db';

/**
 * SQL 补全上下文：由 context.ts 统一构建，
 * 贡献者只依赖上下文编程，不直接接触编辑器模型（单一职责，便于扩展与测试）
 */
export interface SqlCompletionContext {
    /** 编辑器模型 */
    model: editor.ITextModel;
    /** 光标位置 */
    position: Position;
    /** 光标所在单词及其替换范围 */
    word: editor.IWordAtPosition;
    range: IRange;
    /** 当前行内容 */
    lineContent: string;
    /** 光标所在完整 SQL 语句 */
    statement: string;
    /** 光标相对语句起点的偏移（用于 JOIN 多表场景就近取表） */
    statementCursorOffset: number;
    /** 光标前行文本的最后/次后令牌（小写） */
    lastToken: string;
    secondToken: string;
    /** 是否为 `.` 触发补全（如 t. 或 db.） */
    isDotTrigger: boolean;
    /** `.` 触发时解析出的别名/库名 */
    dotAlias: string;
    /** 光标所处子句类型（table：表名期望位置；column：字段/表达式位置；free：全量兜底） */
    clause: CursorClause;
    /** 数据库实例 */
    dbInst: DbInst;
    /** 当前库名 */
    db: string;
    /** 可切换的所有库名 */
    dbs: string[];
    /** 数据库方言 */
    dialect: DbDialect;
    /** 数据库类型（mysql/postgres/...，用于切割与光标区域判定的方言选项） */
    dbType: string;
    /** 方言接受的标识符引用符 */
    quotePairs: QuotePair[];
    /** 光标单词是否已被引用符包裹 */
    isWordQuoted: boolean;
    /**
     * 补全场景标识符引用器：生成表名/字段名等插入文本。
     * 光标处已被包裹时返回裸名，避免 `` `id` `` -> `` ``id`` `` 二次包裹
     */
    quoteIdentifier(name: string): string;
}

/**
 * 贡献者产出结果
 */
export interface ContributorResult {
    suggestions: languages.CompletionItem[];
    /**
     * 命中专属补全场景（如 `.` 触发的库.表/别名.字段联想）时为 true，
     * 贡献者链提前收敛，仅返回该结果（与旧版提前 return 行为一致）
     */
    exclusive?: boolean;
}

/**
 * 补全建议贡献者：每类建议（库名/表/字段/关键字…）一个实现，
 * 新增建议类型只需新增贡献者并注册到贡献者链，无需修改核心逻辑（开闭原则）
 */
export interface SuggestionContributor {
    /** 贡献者名称，用于调试与排序 */
    name: string;
    /** 未命中该贡献者场景时返回 undefined */
    contribute(ctx: SqlCompletionContext): Promise<ContributorResult | undefined>;
}
