/**
 * SQL 草稿的纯逻辑：请求装配、结论呈现、写回文本。
 *
 * 全部做成纯函数：这些规则（能不能写回、多语句怎么拼）是「业务判定」，
 * 放进组件后只能靠 DOM 断言验证，放在这里可以用表驱动单测钉住。
 */
import type { DraftContextTable, DraftFinding, DraftIntentId, DraftSeverity, DraftStatement, DraftVerification, SqlDraftRequest } from './types';

/** 描述长度上限（超出由界面提示，不做静默截断——半句描述会生成更离谱的 SQL） */
export const MAX_PROMPT_LENGTH = 2000;
/**
 * 材料总长上限（描述 + 选区 + 光标语句），与后端 `maxMaterialRunes` 同值同口径。
 *
 * 两侧都要有：只放后端，用户要等一个完整往返才知道"选区太长"；只放前端，绕过界面
 * 直调接口就能把超长材料送进模型（后端是权威闸门，前端只是提前拦住并给出可操作提示）。
 */
export const MAX_MATERIAL_LENGTH = 8000;

export interface DraftPayloadInput {
    db: string;
    intent: DraftIntentId;
    prompt: string;
    selection?: string;
    statement?: string;
    /** 上一版草稿：非空即「在它基础上继续」，与「重新生成」互斥 */
    refine?: string;
    includePlans?: boolean;
}

/** 组装请求体：只带「用户说了什么 + 编辑器材料」，结构由后端自己取 */
export function buildDraftPayload(input: DraftPayloadInput): SqlDraftRequest {
    const payload: SqlDraftRequest = {
        db: input.db,
        intent: input.intent,
        prompt: input.prompt.trim(),
        includePlans: input.includePlans === true,
    };
    const selection = (input.selection ?? '').trim();
    if (selection) {
        payload.selection = selection;
    }
    const statement = (input.statement ?? '').trim();
    if (statement) {
        payload.anchorStatement = statement;
    }
    const refine = (input.refine ?? '').trim();
    if (refine) {
        payload.refineOf = refine;
    }
    return payload;
}

/** 提交前的输入材料（描述 + 编辑器材料） */
export interface DraftInput {
    prompt: string;
    selection?: string;
    statement?: string;
    /** 上一版草稿（多轮细化）：也是材料，必须计入长度闸门，否则这条入口能绕过上界 */
    refine?: string;
    /** 当前意图是否必须有 SQL 材料（fix / optimize / explain） */
    requiresSelection?: boolean;
    /** 当前意图是否允许零描述（与后端 Spec.PromptOptional 同口径） */
    promptOptional?: boolean;
}

/** 一条阻止提交的原因：i18n key + 插值参数（由界面负责翻译，纯逻辑层不碰 i18n 实例） */
export interface DraftInputIssue {
    key: string;
    params?: Record<string, number>;
}

/**
 * 材料总长（字符数），与后端 `materialsOf` 同口径：各段 trim 后按换行拼接。
 *
 * 按码点计而不是 `.length`：后端按 rune 计数，含 emoji 之类代理对的描述下
 * UTF-16 的 length 会多算，两侧判定就会在临界值上给出不同答案。
 */
export function materialLength(input: DraftInput): number {
    const parts = [input.prompt, input.selection, input.statement, input.refine].map((text) => (text ?? '').trim()).filter((text) => text.length > 0);
    const runes = parts.reduce((total, text) => total + [...text].length, 0);
    // 段之间的换行分隔符也占长度
    return runes + Math.max(0, parts.length - 1);
}

/**
 * 提交前校验：返回第一条阻止提交的原因，通过时返回 null。
 *
 * 收成一个入口而不是让界面串几个判定：「什么算有 SQL 材料」「多长算超长」都是业务规则，
 * 漏在组件里就只能靠 DOM 断言验证，而且每个调用点都要重新推一遍。
 */
export function validateDraftInput(input: DraftInput): DraftInputIssue | null {
    const prompt = input.prompt.trim();
    if (!prompt) {
        // 零输入意图（解释/优化/改写）不需要描述：材料里的 SQL 就是需求对象
        return input.promptOptional ? null : { key: 'db.draft.promptRequired' };
    }
    if ([...prompt].length > MAX_PROMPT_LENGTH) {
        return { key: 'db.draft.promptTooLong', params: { max: MAX_PROMPT_LENGTH } };
    }
    const hasSqlMaterial = (input.selection ?? '').trim().length > 0 || (input.statement ?? '').trim().length > 0;
    if (input.requiresSelection && !hasSqlMaterial) {
        return { key: 'db.draft.sqlRequired' };
    }
    if (materialLength(input) > MAX_MATERIAL_LENGTH) {
        return { key: 'db.draft.materialTooLong', params: { max: MAX_MATERIAL_LENGTH } };
    }
    return null;
}

/** 结论是否可以写回编辑器：invalid 一律不写回；散文产物（verification 为空）无语句可写 */
export function canWriteback(verification: DraftVerification): boolean {
    if (verification === '') {
        return false;
    }
    return verification !== 'invalid';
}

/** 结论标签样式 */
export function verificationTagType(verification: DraftVerification): 'success' | 'warning' | 'danger' {
    switch (verification) {
        case '':
            // 散文产物没有校验结论，调用方按 `verification === ''` 直接不渲染标签
            return 'warning';
        case 'verified':
            return 'success';
        case 'unverified':
            return 'warning';
        default:
            return 'danger';
    }
}

/** 校验结论标签样式 */
export function severityTagType(severity: DraftSeverity): 'danger' | 'warning' | 'info' {
    switch (severity) {
        case 'invalid':
            return 'danger';
        case 'unverified':
            return 'warning';
        default:
            return 'info';
    }
}

/** 校验结论计数（界面按级别汇总，避免把 3 条 note 渲染成「有 3 个问题」） */
export function summarizeChecks(checks: DraftFinding[] | null | undefined): Record<DraftSeverity, number> {
    const summary: Record<DraftSeverity, number> = { invalid: 0, unverified: 0, note: 0 };
    for (const check of checks ?? []) {
        summary[check.severity] += 1;
    }
    return summary;
}

/**
 * 把模型返回的清单拼成一句中文列举。
 *
 * 模型给的每条假设/说明常自带句末标点（「……。」），直接用「；」连接会拼出「。；」
 * 这种断句（真机截图坐实）。先归一掉句末标点再连接，末尾统一补一个句号。
 */
export function joinList(items: string[] | null | undefined): string {
    const cleaned = (items ?? []).map((item) => item.trim().replace(/[。；;.]$/, '')).filter((item) => item.length > 0);
    return cleaned.length > 0 ? `${cleaned.join('；')}。` : '';
}

/**
 * 把多条语句拼成可粘贴的脚本。
 *
 * 每条补一个分号：编辑器按分号切分语句，缺分号的多语句粘进去会被当成一条，
 * 执行时直接语法错误——这属于「写回即破坏」的缺陷，不能留给用户自己发现。
 */
export function joinStatements(statements: DraftStatement[] | null | undefined): string {
    // 后端对散文意图可能返回 null（JSON null）：这里必须容错，否则 map() 抛异常会让
    // 整个结果卡片渲染中断、界面停在 loading（真机踩到过）
    const parts = (statements ?? [])
        .map((statement) => statement.sql.trim())
        .filter((sql) => sql.length > 0)
        .map((sql) => (sql.endsWith(';') ? sql : `${sql};`));
    return parts.join('\n\n');
}

/** 拼接结果是否为空（空结果不允许触发写回） */
export function isEmptyDraft(statements: DraftStatement[] | null | undefined): boolean {
    return joinStatements(statements).length === 0;
}

/** 最近使用描述的保留条数与存储键前缀（按库资源隔离，避免跨库串提示） */
export const RECENT_PROMPT_LIMIT = 5;
export const RECENT_PROMPT_KEY_PREFIX = 'db-sql-ai-recent';

/**
 * 读最近使用的需求描述。
 *
 * 坏值（非数组、非字符串项、JSON 解析失败）一律当「没有历史」而不是抛错：
 * 这是便利功能，不能因为 localStorage 脏了就挡住生成
 */
export function readRecentPrompts(storage: Storage | undefined, dbId: number): string[] {
    if (!storage) {
        return [];
    }
    try {
        const raw = storage.getItem(`${RECENT_PROMPT_KEY_PREFIX}:${dbId}`);
        if (!raw) {
            return [];
        }
        const parsed: unknown = JSON.parse(raw);
        if (!Array.isArray(parsed)) {
            return [];
        }
        return parsed.filter((item): item is string => typeof item === 'string' && item.trim().length > 0).slice(0, RECENT_PROMPT_LIMIT);
    } catch {
        return [];
    }
}

/** 记录一条本次使用的需求描述，返回新的最近列表（去重、置顶、限量） */
export function pushRecentPrompt(storage: Storage | undefined, dbId: number, prompt: string): string[] {
    const text = prompt.trim();
    if (!storage || !text) {
        return readRecentPrompts(storage, dbId);
    }
    const next = [text, ...readRecentPrompts(storage, dbId).filter((item) => item !== text)].slice(0, RECENT_PROMPT_LIMIT);
    try {
        storage.setItem(`${RECENT_PROMPT_KEY_PREFIX}:${dbId}`, JSON.stringify(next));
    } catch {
        // 写失败（隐私模式、配额）不影响本次生成：便利功能不能变成故障源
    }
    return next;
}

/**
 * 参考表是否值得渲染备注列。
 *
 * 整列都空时（本项目自有库的表注释普遍为空）渲染空列只会白占宽度，把真正有信息量的
 * 「选用原因」挤到截断（真机截图坐实）。判定收在这里而不是模板里，才能被表驱动单测钉住。
 */
export function hasAnyComment(tables: DraftContextTable[] | null | undefined): boolean {
    return (tables ?? []).some((table) => (table.comment ?? '').trim().length > 0);
}
