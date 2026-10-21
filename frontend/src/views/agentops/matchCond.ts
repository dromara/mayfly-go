/**
 * 触发器匹配条件的解析/序列化纯函数（与后端 trigger.MatchCondition 同构）。
 *
 * 存在理由：匹配条件在库/接口里一直是 JSON 字符串，表单从手敲 JSON 改为可视化控件后，
 * 「字符串 ↔ 结构」的转换是唯一的落库关口，抽成纯函数才能表驱动单测钉死往返语义：
 * 空维度不写键（与后端 omitempty 对齐，否则 `{"metrics":[]}` 这类形态会把「不限定」
 * 错成「白名单为空=永不命中」）；未知键/非法 JSON 必须显式暴露而不是静默吞掉——
 * 吞掉等于用户保存一次就无痕丢掉他手写的扩展条件。
 */

/** 与后端 trigger.MatchCondition 同构的匹配条件 */
export interface MatchCond {
    /** 资源类型稳定键串（machine/db/...；空=不限定） */
    resourceType?: string;
    /** 资源 id 白名单（空=不限定） */
    resourceIds?: number[];
    /** 严重级白名单（P0..P3；空=不限定） */
    severities?: string[];
    /** 指标名白名单（空=不限定） */
    metrics?: string[];
}

/**
 * 维度清单的唯一真源：解析认哪些键、序列化写哪些键、按什么顺序写，全部由它派生。
 *
 * 为什么不能继续「接口 + KNOWN_KEYS + 四个手写 if」三处各写一遍：给 MatchCond 加一个维度、
 * 忘了加 serialize 分支时，parse 认它、控件能填、预览看不到、保存后该维度从 JSON 里静默消失，
 * 没有任何报错。新增维度只需在这里加一行（kind 标它是标量还是白名单）。
 */
const MATCH_FIELDS: ReadonlyArray<{ key: keyof MatchCond; kind: 'scalar' | 'list' }> = [
    { key: 'resourceType', kind: 'scalar' },
    { key: 'resourceIds', kind: 'list' },
    { key: 'severities', kind: 'list' },
    { key: 'metrics', kind: 'list' },
];

/** 后端契约认识的维度键；出现之外的键说明有人绕过表单塞了扩展条件，可视化编辑会丢它 */
const KNOWN_KEYS: (keyof MatchCond)[] = MATCH_FIELDS.map((f) => f.key);

/**
 * 解析表单外的历史形态时置 invalid：调用方据此降级为原始 JSON 编辑模式，
 * 而不是拿一个空对象覆盖掉用户的既有条件
 */
export function parseMatchCond(raw?: string | null): { cond: MatchCond; invalid: boolean } {
    const s = (raw ?? '').trim();
    if (!s) {
        return { cond: {}, invalid: false };
    }
    try {
        const obj = JSON.parse(s);
        if (!obj || typeof obj !== 'object' || Array.isArray(obj)) {
            return { cond: {}, invalid: true };
        }
        if (Object.keys(obj).some((k) => !KNOWN_KEYS.includes(k as keyof MatchCond))) {
            return { cond: {}, invalid: true };
        }
        return { cond: obj as MatchCond, invalid: false };
    } catch {
        return { cond: {}, invalid: true };
    }
}

/** 序列化为落库 JSON：空维度不写键（顺序取 MATCH_FIELDS）；全部不限定返回空串（后端按「命中该来源全部事件」处理） */
export function serializeMatchCond(cond: MatchCond): string {
    const out: Record<string, unknown> = {};
    for (const { key, kind } of MATCH_FIELDS) {
        const val = cond[key];
        // 白名单维度空数组等价于「不限定」：写进去会被后端判成「白名单为空=永不命中」，语义相反
        if (kind === 'list' ? Array.isArray(val) && val.length > 0 : Boolean(val)) {
            out[key] = val;
        }
    }
    return Object.keys(out).length > 0 ? JSON.stringify(out) : '';
}
