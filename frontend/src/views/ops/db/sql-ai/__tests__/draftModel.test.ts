import { describe, expect, it } from 'vitest';
import {
    MAX_MATERIAL_LENGTH,
    MAX_PROMPT_LENGTH,
    RECENT_PROMPT_LIMIT,
    buildDraftPayload,
    canWriteback,
    hasAnyComment,
    isEmptyDraft,
    joinList,
    joinStatements,
    materialLength,
    pushRecentPrompt,
    readRecentPrompts,
    severityTagType,
    summarizeChecks,
    validateDraftInput,
    verificationTagType,
} from '../draftModel';

/** 内存版 Storage：不依赖真 localStorage，也便于注入「存坏了」的脏值 */
function memoryStorage(initial: Record<string, string> = {}): Storage {
    const data = new Map(Object.entries(initial));
    return {
        get length() {
            return data.size;
        },
        key: (index: number) => [...data.keys()][index] ?? null,
        getItem: (key: string) => data.get(key) ?? null,
        setItem: (key: string, value: string) => void data.set(key, value),
        removeItem: (key: string) => void data.delete(key),
        clear: () => data.clear(),
    };
}

describe('SQL 草稿纯逻辑', () => {
    it('请求体只带用户输入与材料，空字段不占位', () => {
        const payload = buildDraftPayload({
            db: 'demo',
            intent: 'draft',
            prompt: '  统计近 7 天订单  ',
            selection: '   ',
            statement: 'select 1',
        });
        expect(payload.prompt).toBe('统计近 7 天订单');
        expect(payload.selection).toBeUndefined();
        expect(payload.anchorStatement).toBe('select 1');
        expect(payload.includePlans).toBe(false);
    });

    it('includePlans 严格取布尔真值，不被字符串之类的脏值点亮', () => {
        // 后端不会因为多点一次 EXPLAIN 出错，但「以为是 false 却传了 true」会白跑一次计划
        expect(buildDraftPayload({ db: 'd', intent: 'draft', prompt: 'p', includePlans: true }).includePlans).toBe(true);
        expect(buildDraftPayload({ db: 'd', intent: 'draft', prompt: 'p' }).includePlans).toBe(false);
    });

    it('输入校验：空描述、超长描述、缺 SQL 材料各自有明确原因', () => {
        expect(validateDraftInput({ prompt: '   ' })?.key).toBe('db.draft.promptRequired');
        expect(validateDraftInput({ prompt: 'x'.repeat(MAX_PROMPT_LENGTH + 1) })?.key).toBe('db.draft.promptTooLong');
        expect(validateDraftInput({ prompt: '改成倒序', requiresSelection: true })?.key).toBe('db.draft.sqlRequired');
        // 选区为空串但光标语句有内容，同样算「有 SQL 材料」
        expect(validateDraftInput({ prompt: '改成倒序', selection: '  ', statement: 'select 1', requiresSelection: true })).toBeNull();
        expect(validateDraftInput({ prompt: '统计订单' })).toBeNull();
    });

    it('超长描述的原因带上上限值（界面要把它插进提示文案）', () => {
        expect(validateDraftInput({ prompt: 'x'.repeat(MAX_PROMPT_LENGTH + 1) })?.params).toEqual({ max: MAX_PROMPT_LENGTH });
    });

    it('材料总长按「各段 trim 后换行拼接」计数，与后端 materialsOf 同口径', () => {
        expect(materialLength({ prompt: ' 统计 ' })).toBe(2);
        expect(materialLength({ prompt: '统计', selection: 'select 1', statement: 'select 2' })).toBe(2 + 8 + 8 + 2);
        // 空段不参与拼接，也就不贡献分隔符
        expect(materialLength({ prompt: '统计', selection: '   ' })).toBe(2);
        // 码点计数：代理对算 1 个字符（.length 会算 2，与后端 rune 计数不一致）
        expect(materialLength({ prompt: '\u{1F600}' })).toBe(1);
    });

    it('材料总长超上限时拒绝提交，正好等于上限时放行', () => {
        // 临界值要按段凑：描述单独受 MAX_PROMPT_LENGTH 管，把 8000 字全塞进描述
        // 会先撞上描述上限，测的就不是材料总长这条规则了
        const atLimit = {
            prompt: 'a'.repeat(MAX_PROMPT_LENGTH),
            selection: 'b'.repeat(3000),
            statement: 'c'.repeat(MAX_MATERIAL_LENGTH - MAX_PROMPT_LENGTH - 3000 - 2),
        };
        expect(materialLength(atLimit)).toBe(MAX_MATERIAL_LENGTH);
        expect(validateDraftInput(atLimit)).toBeNull();

        const over = { prompt: '统计', selection: 'a'.repeat(MAX_MATERIAL_LENGTH) };
        expect(validateDraftInput(over)?.key).toBe('db.draft.materialTooLong');
        expect(validateDraftInput(over)?.params).toEqual({ max: MAX_MATERIAL_LENGTH });
    });

    it('结论为 invalid 时禁止写回，其余允许', () => {
        expect(canWriteback('invalid')).toBe(false);
        expect(canWriteback('unverified')).toBe(true);
        expect(canWriteback('verified')).toBe(true);
    });

    it('后端返回 null 数组时按空处理（散文意图的 statements/checks 为 null）', () => {
        // 真机踩到过：null.map 抛异常导致结果卡片整块不渲染、界面停在 loading
        expect(joinStatements(null)).toBe('');
        expect(joinStatements(undefined)).toBe('');
        expect(isEmptyDraft(null)).toBe(true);
        expect(summarizeChecks(null)).toEqual({ invalid: 0, unverified: 0, note: 0 });
    });

    it('多语句补分号并按空行分隔（缺少分号会被编辑器当成一条语句）', () => {
        const text = joinStatements([{ sql: 'select 1' }, { sql: '  select 2;  ' }, { sql: '   ' }]);
        expect(text).toBe('select 1;\n\nselect 2;');
        expect(isEmptyDraft([{ sql: '   ' }])).toBe(true);
    });

    it('按级别汇总校验结论，note 不混进「问题」计数', () => {
        const summary = summarizeChecks([
            { severity: 'invalid', code: 'unknown_column', message: 'x' },
            { severity: 'note', code: 'write_statement', message: 'y' },
            { severity: 'note', code: 'policy_ticket', message: 'z' },
        ]);
        expect(summary).toEqual({ invalid: 1, unverified: 0, note: 2 });
    });

    it('标签样式覆盖三种结论与三种级别（防止新增取值时静默落到默认色）', () => {
        expect(verificationTagType('verified')).toBe('success');
        expect(verificationTagType('unverified')).toBe('warning');
        expect(verificationTagType('invalid')).toBe('danger');
        expect(severityTagType('invalid')).toBe('danger');
        expect(severityTagType('unverified')).toBe('warning');
        expect(severityTagType('note')).toBe('info');
    });

    it('清单拼接归一模型自带的句末标点，不出现「。；」断句', () => {
        // 真机截图坐实：模型每条假设自带「。」，直接 join('；') 会拼出「。；」
        expect(joinList(['对应表 t_db（通过 instance_id 关联）。', '默认过滤软删除数据：is_deleted = 0。'])).toBe(
            '对应表 t_db（通过 instance_id 关联）；默认过滤软删除数据：is_deleted = 0。'
        );
        expect(joinList(['only one'])).toBe('only one。');
        expect(joinList([])).toBe('');
        expect(joinList(null)).toBe('');
        // 句末是括号/表名等非标点时原样保留，只补连接符
        expect(joinList(['计数为 0）', '排序按数量降序'])).toBe('计数为 0）；排序按数量降序。');
    });

    it('细化轮：上一版草稿进 payload 且计入材料长度', () => {
        const payload = buildDraftPayload({ db: 'demo', intent: 'draft', prompt: '再按月分组', refine: '  SELECT 1  ' });
        expect(payload.refineOf).toBe('SELECT 1');
        // 空白的上一版不占位：后端按「有没有 refineOf」区分细化与重做
        expect(buildDraftPayload({ db: 'demo', intent: 'draft', prompt: 'x', refine: '   ' }).refineOf).toBeUndefined();

        // 上一版单独超长也必须被闸门拦下：否则这条入口能绕过材料上界
        const tooLong = { prompt: '再按月分组', refine: 'x'.repeat(MAX_MATERIAL_LENGTH) };
        expect(materialLength(tooLong)).toBeGreaterThan(MAX_MATERIAL_LENGTH);
        expect(validateDraftInput(tooLong)?.key).toBe('db.draft.materialTooLong');
    });

    it('备注列显隐只看是否存在非空备注（空白串不算，避免渲染整列空表头）', () => {
        expect(hasAnyComment(null)).toBe(false);
        expect(hasAnyComment([])).toBe(false);
        expect(hasAnyComment([{ name: 't_db', columns: 1 }])).toBe(false);
        expect(hasAnyComment([{ name: 't_db', columns: 1, comment: '   ' }])).toBe(false);
        expect(
            hasAnyComment([
                { name: 't_db', columns: 1, comment: '' },
                { name: 't_x', columns: 2, comment: '库' },
            ])
        ).toBe(true);
    });

    it('最近使用：去重置顶、限量、按库隔离，且脏值不影响生成', () => {
        const storage = memoryStorage();
        expect(pushRecentPrompt(storage, 1, ' 统计订单 ')).toEqual(['统计订单']);
        expect(pushRecentPrompt(storage, 1, '查最近 7 天')).toEqual(['查最近 7 天', '统计订单']);
        // 重复提问顶到最前而不是追加两份
        expect(pushRecentPrompt(storage, 1, '统计订单')).toEqual(['统计订单', '查最近 7 天']);
        // 按库隔离：跨库提示会误导（同一个说法在另一张库里对应的表完全不同）
        expect(readRecentPrompts(storage, 2)).toEqual([]);

        for (let i = 0; i < RECENT_PROMPT_LIMIT + 3; i++) {
            pushRecentPrompt(storage, 1, `需求 ${i}`);
        }
        expect(readRecentPrompts(storage, 1)).toHaveLength(RECENT_PROMPT_LIMIT);

        expect(pushRecentPrompt(storage, 1, '   ')).toEqual(readRecentPrompts(storage, 1));
        expect(readRecentPrompts(undefined, 1)).toEqual([]);
        // 坏值一律当「没有历史」：这是便利功能，不能因为 localStorage 脏了就挡住生成
        expect(readRecentPrompts(memoryStorage({ 'db-sql-ai-recent:1': 'not-json' }), 1)).toEqual([]);
        expect(readRecentPrompts(memoryStorage({ 'db-sql-ai-recent:1': '[1,"  ","ok"]' }), 1)).toEqual(['ok']);
    });
});
