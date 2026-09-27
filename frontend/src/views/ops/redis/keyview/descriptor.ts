import type { RedisFormSchema, RedisKeyMember, RedisViewCaps, RedisViewColumn, RedisViewDescriptor } from '../types';

/**
 * 数据视角契约的前端解释器。
 *
 * 这里没有任何「按类型名分支」的代码：表格列、能力位、表单结构、操作按钮全部来自后端描述符，
 * 因此后端新增一个类型处理器，前端不需要改动本文件与任何组件
 */

/** 命令模板里的 key 占位符，与后端 ViewDescriptor.ConsoleHints 的写法是同一约定 */
export const CONSOLE_KEY_PLACEHOLDER = '{key}';

/** 成员区布局，取值与后端 ViewDescriptor.Layout 对应 */
export const LAYOUT_VALUE = 'value';

/** 该视角的可写能力位，未加载到视角时全部关闭（不给只读界面开出可点的按钮） */
export const EMPTY_CAPS: RedisViewCaps = {
    create: false,
    update: false,
    delete: false,
    batchDelete: false,
    keyword: false,
    rankPaging: false,
    cursorPaging: false,
    ops: false,
};

/**
 * 「还有更多」判据：能按下标分页的视角看「已读条数 vs 总数」，只有游标语义的视角
 * （hash / set 的 scan）才看游标是否归零。顺序不能反：zset 按排名读取不产生游标，
 * 先判游标会让它读完首屏就显示「没有更多」
 */
export function hasMoreMembers(caps: RedisViewCaps, cursor: string, loaded: number, total: number, searching = false): boolean {
    // 关键字过滤一律走 scan 游标链路（即使该视角平时按排名分页），因此只看游标是否归零
    if (searching || !caps.rankPaging) {
        return !!cursor;
    }
    return loaded < total;
}

export function findView(descriptors: RedisViewDescriptor[], view: string): RedisViewDescriptor | undefined {
    return descriptors.find((item) => item.view === view);
}

/** 原生类型对应的默认视角，决定类型标签的配色与图标 */
export function defaultViewOf(descriptors: RedisViewDescriptor[], keyType: string): RedisViewDescriptor | undefined {
    return descriptors.find((item) => item.default && item.types.includes(keyType)) ?? descriptors.find((item) => item.types.includes(keyType));
}

/** 可服务某原生类型的视角清单（视角切换器数据源） */
export function viewsOfType(descriptors: RedisViewDescriptor[], keyType: string): RedisViewDescriptor[] {
    return descriptors.filter((item) => item.types.includes(keyType));
}

/**
 * Member 的自有字段名表：后端 Column.Field 与表单 prop 只允许取这些值（或 extra 的 key），
 * 显式列出而非断言成索引签名，类型链上不会出现「任意字段名都能取」的失控面
 */
const MEMBER_FIELDS = ['index', 'field', 'value', 'score', 'id'] as const;

type MemberField = (typeof MEMBER_FIELDS)[number];

/** 取成员的自有字段值，非自有字段（派生列）返回 undefined */
function memberField(row: RedisKeyMember, name: string): string | number | undefined {
    if (!(MEMBER_FIELDS as readonly string[]).includes(name)) {
        return undefined;
    }
    return row[name as MemberField];
}

/**
 * 取 extra 里的派生列值。必须显式判定自有属性：extra 是普通对象，
 * 直接用 [] 取会拿到 Object.prototype 上的成员（如 constructor），把函数当数据渲染出去
 */
function extraValue(row: RedisKeyMember, name: string): string | undefined {
    const extra = row.extra;
    return extra && Object.prototype.hasOwnProperty.call(extra, name) ? extra[name] : undefined;
}

/**
 * 列取值：Member 自有字段优先，取不到再取 extra（geo 的经纬度、位图的字节值等派生列）
 */
export function columnValue(row: RedisKeyMember, column: RedisViewColumn): string {
    const own = memberField(row, column.field);
    if (typeof own === 'number') {
        return String(own);
    }
    return own || extraValue(row, column.field) || '';
}

/**
 * 「可见项是否都已被勾选」的判据。必须逐项比对而不是比长度：
 * 先勾了 A 类 20 项、再把筛选切到只剩 5 项的 B 类时，`20 >= 5` 会误判成已全选
 */
export function coversAll(visible: readonly string[], checked: readonly string[]): boolean {
    if (!visible.length) {
        return false;
    }
    const selected = new Set(checked);
    return visible.every((key) => selected.has(key));
}

/** 行主键：不同视角的行标识不同（下标 / field / entry id / 值），拼出稳定且唯一的字符串 */
export function rowKey(row: RedisKeyMember): string {
    return `${row.id}|${row.field}|${row.index}|${row.value}`;
}

/**
 * 表单值 → 后端 args：值一律转字符串，数组按换行拼接，与后端 splitLines / splitKeys 的解析方式对齐。
 * 后端处理器是唯一解释这些值的地方，前端不做任何类型相关的转换
 */
export function toArgs(form: Record<string, unknown>): Record<string, string> {
    const args: Record<string, string> = {};
    Object.entries(form).forEach(([prop, value]) => {
        if (value === undefined || value === null) {
            args[prop] = '';
            return;
        }
        args[prop] = Array.isArray(value) ? value.join('\n') : String(value);
    });
    return args;
}

/**
 * 编辑回填：表单 prop 与 Member 字段名（或 extra key）同名，因此按 schema 逐字段取值即可，
 * 不需要为每种类型写一份映射代码
 */
export function prefillForm(schema: RedisFormSchema | undefined, row?: RedisKeyMember | null): Record<string, unknown> {
    if (!schema || !row) {
        return {};
    }

    const data: Record<string, unknown> = {};
    schema.fields.forEach((field) => {
        const own = memberField(row, field.prop);
        if (own !== undefined && own !== '') {
            data[field.prop] = own;
            return;
        }
        const extra = extraValue(row, field.prop);
        if (extra !== undefined) {
            data[field.prop] = castOptionValue(field, extra);
        }
    });
    return data;
}

/** extra 里的值都是字符串，回填到数字/开关类控件时需要还原成控件真正接受的类型 */
function castOptionValue(field: RedisFormSchema['fields'][number], value: string): unknown {
    if (field.type === 'number') {
        return Number(value);
    }
    if (field.type === 'switch') {
        return value === 'true' || value === '1';
    }
    return value;
}

/**
 * 字段剩余过期秒数 → 读数（与 key 级 TTL 同一套刻度：天以上带 d，时以上 hh:mm:ss，分以下 mm:ss）。
 * 服务端给的是读取瞬间的快照，页面停留期间不倒计时，需要精确值时点刷新重取
 */
export function formatTtlSeconds(seconds: number): string {
    if (seconds <= 0) {
        return '';
    }
    const pad = (val: number) => String(val).padStart(2, '0');
    const days = Math.floor(seconds / 86400);
    const clock = [Math.floor((seconds % 86400) / 3600), Math.floor((seconds % 3600) / 60), seconds % 60].map(pad).join(':');
    return days > 0 ? `${days}d ${clock}` : clock.slice(clock.startsWith('00:') ? 3 : 0);
}

/**
 * 字段过期列的读数：-1 是「字段存在但没设过期」，-2 是「字段已不存在」（读取瞬间的竞态），
 * 拿不到值说明当前实例不支持字段过期，这几种情况不能混成一个显示
 */
export function ttlCellText(raw: string, permanent: string): string {
    if (raw === '') {
        return '';
    }
    const seconds = Number(raw);
    if (!Number.isFinite(seconds)) {
        return raw;
    }
    if (seconds === -1) {
        return permanent;
    }
    return seconds < 0 ? '' : formatTtlSeconds(seconds);
}

/** 操作结果里的成员行（读操作返回值随命令而定，统一转成可读文本） */
export function formatOpResult(result: unknown): string {
    if (result === null || result === undefined || result === '') {
        return '';
    }
    if (typeof result === 'string' || typeof result === 'number' || typeof result === 'boolean') {
        return String(result);
    }
    return JSON.stringify(result, null, 2);
}
