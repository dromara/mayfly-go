/**
 * 行内注释补全：在 `-- 描述` 的注释行上按 Tab 补一条 SQL。
 *
 * 定位与代价：
 *  - 这是「快速试探」，不是准确路径。它只做一次模型调用（跳过表选择题），上下文只来自
 *    「注释里出现的标识符 + 注释相关表」，所以**只展示通过校验（verified）的建议**；
 *    想拿到带依据、可自纠的结果，走对话框那条路。
 *  - 因此默认关闭，由用户在工具条显式开启；同一行同一段时间内只请求一次，避免打字即扣费。
 *
 * 注册约束（与 completion provider 同一坑）：monaco 的 provider 按语言全局唯一，
 * 多编辑器共存时必须按 model uri 路由到各自的库上下文，否则会拿 A 库建议补到 B 库。
 */
import { ensureInlineCompletionsFeature, languages } from '@/components/monaco/setup';
import type { CancellationToken, editor, languages as MonacoLanguages, Position } from 'monaco-editor';
import { sqlDraftApi } from './api';

/** 每个编辑器实例的补全上下文 */
export interface InlineDraftScope {
    dbId: number;
    dbName: string;
    enabled: boolean;
}

const scopes = new Map<string, InlineDraftScope>();
const THROTTLE_MS = 30_000;
/** 节流表容量上界（见 LineThrottle） */
const THROTTLE_MAX_ENTRIES = 200;
let installed = false;

/**
 * 行级节流表：同一行文本在窗口期内只放行一次请求。
 *
 * 做成一个类型而不是「模块级 Map + 散落的读写」：这张表必须有容量上界，而上界这条规则
 * 只有和读写逻辑写在一起才不会被后来的调用点绕过。key 含行文本，用户在一份长脚本里逐行
 * 触发就会逐行留一条记录，而编辑器 tab 常常一开一整天——无界的 Map 等于一条缓慢的内存泄漏。
 */
export class LineThrottle {
    private readonly entries = new Map<string, number>();

    constructor(
        private readonly windowMs: number,
        private readonly maxEntries: number
    ) {}

    /** 当前记账条数（供测试与诊断） */
    get size(): number {
        return this.entries.size;
    }

    /** 作废一次记账：请求失败后让重试不必白等一个窗口 */
    forget(key: string): void {
        this.entries.delete(key);
    }

    /**
     * 判定并记账：返回 true 表示放行（本次应当发请求）。
     *
     * 判定与记账必须在同一次调用里完成。分开写就会出现「判定放行了但忘了记账」，
     * 于是同一行每次触发都发请求——节流静默失效，界面上完全看不出来，只有账单能看出来。
     */
    allow(key: string, now: number): boolean {
        // 必须区分「从未记账」与「记账于某个时刻」：用 `?? 0` 兜底会把两者混为一谈，
        // 未记账的键在 now 较小时会被判成"刚刚才放行过"，于是首次触发被静默吞掉
        const last = this.entries.get(key);
        if (last !== undefined && now - last < this.windowMs) {
            return false;
        }
        this.evict(now);
        this.entries.set(key, now);
        return true;
    }

    /**
     * 容量收敛：先清掉已过期条目（多数情况下这一步就够了），仍满则按插入顺序淘汰最早的一半。
     * Map 的迭代顺序就是插入顺序，因此不需要额外维护时间戳堆或排序。
     */
    private evict(now: number): void {
        if (this.entries.size < this.maxEntries) {
            return;
        }
        for (const [key, at] of this.entries) {
            if (now - at >= this.windowMs) {
                this.entries.delete(key);
            }
        }
        if (this.entries.size < this.maxEntries) {
            return;
        }
        let drop = this.entries.size - Math.floor(this.maxEntries / 2);
        for (const key of this.entries.keys()) {
            if (drop <= 0) {
                break;
            }
            this.entries.delete(key);
            drop -= 1;
        }
    }
}

/** 同一行文本的请求节流（key = uri|lineText），避免连续输入重复计费 */
const recent = new LineThrottle(THROTTLE_MS, THROTTLE_MAX_ENTRIES);

/** 登记某编辑器实例的补全上下文（父组件在 ready / 切库 / 开关时调用） */
export function setInlineDraftScope(uri: string, scope: InlineDraftScope) {
    scopes.set(uri, scope);
}

/** 注销（编辑器销毁时调用，避免 map 永远增长） */
export function removeInlineDraftScope(uri: string) {
    scopes.delete(uri);
}

/**
 * 注释行判定（纯函数，便于单测）：
 * 返回注释正文；不是「光标位于注释行末尾」的形态时返回空串。
 */
export function commentTextAtLine(lineText: string, cursorColumn: number): string {
    const trimmedEnd = lineText.trimEnd();
    // 光标必须落在「行尾或行尾空白区间内」：在行中间请求补全会把后半行当成注释的一部分，
    // 而严格要求与 trimEnd 完全相等又会把"注释写完了但光标停在行尾空格上"这种常见情形漏掉
    const offset = cursorColumn - 1;
    if (offset < trimmedEnd.length || offset > lineText.length) {
        return '';
    }
    const match = /^\s*--\s*(.+)$/.exec(trimmedEnd);
    if (!match) {
        return '';
    }
    const text = match[1].trim();
    // 太短的注释几乎必然是随手输入（"-- 1"），不值得花一次模型调用
    return text.length >= 4 ? text : '';
}

/**
 * 生成建议的插入文本：**必须以换行开头**。
 *
 * 触发位置在注释行尾，若直接插入 `SELECT ...`，采纳后会变成 `-- 统计...SELECT ...`，
 * 整条 SQL 被注释掉——真机验收时正是踩到这一点。换行让 SQL 落在注释的下一行。
 */
export function ghostInsertText(sql: string): string {
    const trimmed = sql.trim().replace(/[\r\n]+$/, '');
    return trimmed ? `\n${trimmed}` : '';
}

/**
 * 安装 provider（幂等）。
 *
 * 注意两件事：
 *  1. ghost 视图来自 monaco 的 inlineCompletions 贡献模块，本项目的 monaco 装配入口没有引入它，
 *     只注册 provider 会「被调用但永远不渲染」——所以这里必须先把贡献模块装上；
 *  2. 装配走**动态 import**：静态引入会让该贡献模块在测试/其它页面也被求值，实测会污染
 *     无关用例（如表格编辑器用例里同一 mock 被多调一次）。按需加载只在真正启用补全时付费。
 */
export async function ensureInlineDraftProvider() {
    if (installed) {
        return;
    }
    installed = true;
    try {
        // 装配入口负责引包与副作用（db 模块不得直接值导入 monaco-editor）
        await ensureInlineCompletionsFeature();
    } catch {
        // 装配失败就退回普通补全，不影响编辑与手动生成
    }
    const provider: MonacoLanguages.InlineCompletionsProvider = {
        async provideInlineCompletions(
            model: editor.ITextModel,
            position: Position,
            _context: MonacoLanguages.InlineCompletionContext,
            token: CancellationToken
        ) {
            const scope = scopes.get(model.uri.toString());
            if (!scope?.enabled) {
                return { items: [] };
            }
            const prompt = commentTextAtLine(model.getLineContent(position.lineNumber), position.column);
            if (!prompt) {
                return { items: [] };
            }
            const key = `${model.uri.toString()}|${prompt}`;
            if (!recent.allow(key, Date.now())) {
                return { items: [] };
            }

            try {
                const result = await sqlDraftApi.draft.request({
                    dbId: scope.dbId,
                    db: scope.dbName,
                    intent: 'suggest',
                    prompt,
                });
                // 光标已移走 / 内容已变：丢弃，别把上一个位置的建议补到新位置
                if (token.isCancellationRequested) {
                    return { items: [] };
                }
                // 只接受通过校验的建议：不通过的一律不展示（否则等于把不确定的 SQL 塞进编辑器）
                if (result.verification !== 'verified') {
                    return { items: [] };
                }
                const insertText = ghostInsertText(result.statements?.[0]?.sql ?? '');
                if (!insertText) {
                    return { items: [] };
                }
                return {
                    items: [
                        {
                            insertText,
                            range: {
                                startLineNumber: position.lineNumber,
                                startColumn: model.getLineMaxColumn(position.lineNumber),
                                endLineNumber: position.lineNumber,
                                endColumn: model.getLineMaxColumn(position.lineNumber),
                            },
                        },
                    ],
                };
            } catch {
                // 补全是可选增强：失败不出提示、不打断输入
                recent.forget(key);
                return { items: [] };
            }
        },
        // monaco 0.57 要求实现释放钩子；本 provider 不持有额外资源
        disposeInlineCompletions() {
            return;
        },
    };
    languages.registerInlineCompletionsProvider('sql', provider);
}
