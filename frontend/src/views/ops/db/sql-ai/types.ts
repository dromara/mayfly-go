/**
 * SQL 草稿（自然语言 → SQL）的前后端契约。
 *
 * 与后端 `dbc/application/sqldraft.Draft` 对齐：这里只描述事实，文案一律按 code 在
 * 前端渲染——后端已回本地化 message，但界面需要按级别/码上色与分组，不能只靠文本。
 */

/** 意图 id：与后端 RegisterIntent 的注册 id 一一对应（契约测试核对集合一致） */
export type DraftIntentId = 'draft' | 'fix' | 'optimize' | 'explain';

/**
 * 允许直接下发的意图（含行内补全用的 suggest）。
 *
 * suggest 不在意图注册表里出现：它不是用户可选的动作，而是注释行补全的内部通道，
 * 表现形式也不同（不给卡片、只往编辑器里塞一段 ghost 文本选中的内容）。
 */
export type DraftRequestIntent = DraftIntentId | 'suggest';

/** 生成请求 */
export interface SqlDraftRequest {
    db: string;
    intent: DraftRequestIntent;
    prompt: string;
    selection?: string;
    anchorStatement?: string;
    /** 上一版生成的语句：非空表示「在它基础上继续」，与「重新生成」是两种意图 */
    refineOf?: string;
    includePlans?: boolean;
}

/** 业务词典条目（语义层）：术语 → 表 + 口径说明 */
export interface AiTerm {
    id?: number;
    dbId: number;
    /** 库名；空串表示该库资源下所有库共用 */
    schema?: string;
    term: string;
    tables?: string[];
    details?: string;
}

/** 校验结论级别：invalid 不可写回，unverified 可写回但必须标出，note 仅提示 */
export type DraftSeverity = 'invalid' | 'unverified' | 'note';

/** 一条校验结论 */
export interface DraftFinding {
    severity: DraftSeverity;
    code: string;
    message: string;
    identifier?: string;
    candidates?: string[];
}

/** 一条生成的语句 */
export interface DraftStatement {
    sql: string;
    notes?: string;
    assumptions?: string[];
}

/** 上下文里带上的表（界面展示「依据」，并说明为什么带上） */
export interface DraftContextTable {
    name: string;
    comment?: string;
    columns: number;
    reason?: string;
}

/** 结论：由后端确定性校验推导，不使用模型自评；散文产物（explain）为空串 */
export type DraftVerification = 'verified' | 'unverified' | 'invalid' | '';

/** 生成结果 */
export interface SqlDraftResult {
    intent: DraftRequestIntent;
    statements: DraftStatement[];
    /** 散文产物（explain 意图）：不是 SQL，不参与写回 */
    explanation?: string;
    verification: DraftVerification;
    checks: DraftFinding[];
    contextTables: DraftContextTable[];
    warnings?: string[];
    repairs: number;
    /** 未通过校验时实际补采的候选数（0 表示没补采）：多花了一次模型调用必须让用户看见 */
    candidates: number;
    dialect: string;
    usage?: {
        promptTokens: number;
        completionTokens: number;
        totalTokens: number;
    };
}
