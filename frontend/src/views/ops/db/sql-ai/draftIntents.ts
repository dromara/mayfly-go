/**
 * 意图注册表（前端侧）。
 *
 * 与后端 `sqldraft.RegisterIntent` 的 id 集合一一对应：后端决定「这条意图怎么生成」，
 * 前端决定「这条意图怎么呈现与何时可见」。刻意不让后端下发 UI 文案与顺序——那会把
 * 界面细节钉进服务端契约，改个菜单顺序都要动后端。
 *
 * 新增一条意图 = 这里加一条声明（+ 后端注册一条 spec），菜单与对话框自动出现。
 */
import type { DraftIntentId } from './types';

/** 编辑器当前的「材料」状态：决定哪些意图有意义 */
export interface DraftEditorContext {
    /** 是否有非空选区 */
    hasSelection: boolean;
    /** 光标所在完整语句（无语句时为空串） */
    statement: string;
}

export interface DraftIntentDef {
    id: DraftIntentId;
    /** 意图标题（i18n key） */
    titleKey: string;
    /** 输入框提示（i18n key） */
    hintKey: string;
    /** 是否必须有一份 SQL 材料才有意义 */
    requiresSelection: boolean;
    /**
     * 需求描述可以为空（选中 SQL 直接点即可）。
     *
     * 必须与后端 `sqldraft.Spec.PromptOptional` 一致：前端放宽而后端仍报错，用户看到的就是
     * 「点了没反应」；前端收紧则白白拦住一条本该可用的零输入路径
     */
    promptOptional: boolean;
    /** 菜单顺序（升序） */
    order: number;
    /** 默认是否取执行计划做只读校验（多一次 EXPLAIN，慢一点但更有说服力） */
    includePlansByDefault: boolean;
    /** 在该材料状态下是否可用 */
    available: (ctx: DraftEditorContext) => boolean;
}

export const DRAFT_INTENTS: readonly DraftIntentDef[] = Object.freeze([
    {
        id: 'draft',
        titleKey: 'db.draft.intentDraft',
        hintKey: 'db.draft.hintDraft',
        requiresSelection: false,
        promptOptional: false,
        order: 10,
        includePlansByDefault: false,
        available: () => true,
    },
    {
        id: 'optimize',
        titleKey: 'db.draft.intentOptimize',
        hintKey: 'db.draft.hintOptimize',
        requiresSelection: true,
        promptOptional: true,
        order: 30,
        includePlansByDefault: false,
        available: (ctx) => ctx.hasSelection || ctx.statement.trim().length > 0,
    },
    {
        id: 'explain',
        titleKey: 'db.draft.intentExplain',
        hintKey: 'db.draft.hintExplain',
        requiresSelection: true,
        promptOptional: true,
        order: 40,
        includePlansByDefault: false,
        available: (ctx) => ctx.hasSelection || ctx.statement.trim().length > 0,
    },
    {
        id: 'fix',
        titleKey: 'db.draft.intentFix',
        hintKey: 'db.draft.hintFix',
        requiresSelection: true,
        promptOptional: true,
        order: 20,
        includePlansByDefault: false,
        // 选区或光标所在语句任一存在即可：光标停在一条语句里时，「改写这条」是自然诉求
        available: (ctx) => ctx.hasSelection || ctx.statement.trim().length > 0,
    },
]);

/** 当前材料状态下可用的意图（按 order 排序） */
export function listDraftIntents(ctx: DraftEditorContext): DraftIntentDef[] {
    return DRAFT_INTENTS.filter((intent) => intent.available(ctx)).sort((a, b) => a.order - b.order);
}

/**
 * 当前材料下最自然的默认意图：有 SQL 材料就「改写这段」，没有就「按描述生成」。
 *
 * 只给工具条入口与快捷键用：选中代码后按下 AI 快捷键，行业语义（Cursor 内联编辑、Copilot inline chat）
 * 都是「改它」而不是「另起一段」；写死 'draft' 会让用户每次都要先手动切一下意图。
 */
export function defaultDraftIntent(ctx: DraftEditorContext): DraftIntentId {
    return ctx.hasSelection || ctx.statement.trim().length > 0 ? 'fix' : 'draft';
}

/** 按 id 取意图声明 */
export function findDraftIntent(id: DraftIntentId): DraftIntentDef | undefined {
    return DRAFT_INTENTS.find((intent) => intent.id === id);
}

/**
 * 零输入能否直接生成：该意图允许空描述，且它要求的 SQL 材料确实已在手。
 *
 * 判据必须含 requiresSelection：只看 promptOptional 会把「无材料时光标一按回车」也放行，
 * 而那条请求必被后端拒掉——界面表现就是「按了没反应」。面板用它决定要不要把结果自动跑起来
 * （与 validateDraftInput 同口径，两处不会判反）
 */
export function canRunWithoutPrompt(intent: DraftIntentDef | undefined, ctx: DraftEditorContext): boolean {
    if (!intent?.promptOptional) {
        return false;
    }
    return !intent.requiresSelection || ctx.hasSelection || ctx.statement.trim().length > 0;
}
