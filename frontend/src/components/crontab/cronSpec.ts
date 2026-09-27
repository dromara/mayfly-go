/**
 * cron 表达式模型：解析、序列化、校验与运行时间推算（纯函数，不依赖 Vue）。
 *
 * 语法以后端调度器为准（`server/pkg/scheduler` 用 robfig/cron v3，解析器为
 * `SecondOptional | Minute | Hour | Dom | Month | Dow | Descriptor`）：
 * - 接受 5 段（分 时 日 月 周）或 6 段（秒 分 时 日 月 周）；5 段时秒按 0 补齐，与后端默认一致
 * - 每段支持 `*`、`?`、`a`、`a-b`、`a/n`、`a-b/n`、逗号列表，以及月/周的英文名（`jan`、`mon-fri`）
 * - 取值边界：秒 0-59、分 0-59、时 0-23、日 1-31、月 1-12、周 0-6（0=周日，后端不做 7→0 归一）
 * - 区间不支持回绕（`22-2` 后端直接报错）；步长必须是正整数
 * - **不支持** `L`、`W`、`#` 与第 7 段年份：后端解析失败只写日志、任务静默不执行，所以面板不提供、校验判错
 *
 * star 位语义（决定「日/周」是取交集还是并集，后端 `dayMatches`）：
 * `*`、`?`、`*`/1 这类「未限定」写法带 star 位，日与周按「且」匹配；两边都限定时按「或」匹配。
 * 因此「覆盖全部取值」不等于「未限定」——`1-31`、`0-59/1` 都不带 star 位，模型必须保留这个区别，
 * 否则手敲的表达式在面板里预览出的时间会与后端实际触发时间不一致。
 *
 * 有意比后端严格的两处（都是后端接受但明显写错的形态，宁可拦下）：
 * - 空列表项（`1,`、`0,,5`）：后端 `FieldsFunc` 会静默丢弃空串，面板判错
 * - `@every 0s` 与负值时长：后端不校验，前者会把调度打成忙循环
 *
 * 本文件的语法口径与 `server/pkg/scheduler/cron_parser_test.go` 的判定表成对维护：
 * 后端保存入口会用同一解析器再校验一次（`scheduler.ValidateSpec`），改一侧语法必须同步另一侧与两边用例。
 */

/** 字段键，顺序即表达式中的段序 */
export const CRON_FIELD_KEYS = ['second', 'min', 'hour', 'day', 'month', 'week'] as const;

export type CronFieldKey = (typeof CRON_FIELD_KEYS)[number];

/**
 * 规则类型：
 * - `all` 未限定（`*`，或后端的 `*`/1）
 * - `none` 不指定（`?`，仅日/周，与 `all` 同为「不带约束」）
 * - `cycle` 闭区间（`from-to`）
 * - `step` 自起始值起按间隔递增（`start/every`）
 * - `list` 离散取值（逗号列表）
 *
 * 消费点一律用穷尽 switch，新增变体时编译期即报错，避免漏改某处导致语义静默缺失。
 */
export type CronRule =
    | { kind: 'all' }
    | { kind: 'none' }
    | { kind: 'cycle'; from: number; to: number }
    | { kind: 'step'; start: number; every: number }
    | { kind: 'list'; values: number[] };

export type CronRuleKind = CronRule['kind'];

export type CronSpec = Record<CronFieldKey, CronRule>;

/** 规则类型的展示文案 key，与 CronRule 同处登记 */
export const RULE_KIND_LABEL_KEYS: Record<CronRuleKind, string> = {
    all: 'components.crontab.ruleAll',
    none: 'components.crontab.ruleNone',
    cycle: 'components.crontab.ruleCycle',
    step: 'components.crontab.ruleStep',
    list: 'components.crontab.ruleList',
};

/** 字段元数据：取值边界、别名与面板可选的规则类型 */
export interface CronField {
    key: CronFieldKey;
    /** 字段名 i18n key */
    labelKey: string;
    min: number;
    max: number;
    /** 是否支持 `?`（仅日/周；其余字段的 `?` 与 `*` 等价） */
    allowNone: boolean;
    /** 可选规则类型，顺序即面板中的展示顺序 */
    kinds: CronRuleKind[];
    /** 取值标签 i18n key（周字段用星期名），下标即取值；缺省展示数字 */
    valueLabelKeys?: string[];
    /** 字面量别名 → 取值，与后端 `parseIntOrName` 的 names 表一致（大小写不敏感） */
    aliases?: Record<string, number>;
    /** 指定取值的 chip 网格列数 */
    columns: number;
}

const BASE_KINDS: CronRuleKind[] = ['all', 'cycle', 'step', 'list'];

/** 日/周额外支持「不指定」（`?`），排在最前便于直接选到 */
const DAY_KINDS: CronRuleKind[] = ['none', 'all', 'cycle', 'step', 'list'];

/** 周日为一周之首，下标与 JS `Date.getDay()` 及后端 dow 取值一致 */
const WEEK_LABEL_KEYS = [
    'components.crontab.sunday',
    'components.crontab.monday',
    'components.crontab.tuesday',
    'components.crontab.wednesday',
    'components.crontab.thursday',
    'components.crontab.friday',
    'components.crontab.saturday',
];

/** 后端 month/dow 的名称表，取小写字面量 */
const MONTH_ALIASES = { jan: 1, feb: 2, mar: 3, apr: 4, may: 5, jun: 6, jul: 7, aug: 8, sep: 9, oct: 10, nov: 11, dec: 12 };
const WEEK_ALIASES = { sun: 0, mon: 1, tue: 2, wed: 3, thu: 4, fri: 5, sat: 6 };

export const CRON_FIELDS: Record<CronFieldKey, CronField> = {
    second: { key: 'second', labelKey: 'components.crontab.second', min: 0, max: 59, allowNone: false, kinds: BASE_KINDS, columns: 12 },
    min: { key: 'min', labelKey: 'components.crontab.minute', min: 0, max: 59, allowNone: false, kinds: BASE_KINDS, columns: 12 },
    hour: { key: 'hour', labelKey: 'components.crontab.hour', min: 0, max: 23, allowNone: false, kinds: BASE_KINDS, columns: 12 },
    day: { key: 'day', labelKey: 'components.crontab.day', min: 1, max: 31, allowNone: true, kinds: DAY_KINDS, columns: 10 },
    month: { key: 'month', labelKey: 'components.crontab.month', min: 1, max: 12, allowNone: false, kinds: BASE_KINDS, aliases: MONTH_ALIASES, columns: 6 },
    week: {
        key: 'week',
        labelKey: 'components.crontab.week',
        min: 0,
        max: 6,
        allowNone: true,
        kinds: DAY_KINDS,
        valueLabelKeys: WEEK_LABEL_KEYS,
        aliases: WEEK_ALIASES,
        columns: 7,
    },
};

/** 面板页签顺序，与表达式段序一致 */
export const CRON_PANEL_FIELDS: CronField[] = CRON_FIELD_KEYS.map((key) => CRON_FIELDS[key]);

/** 预览条数；标题文案按此插值，改这一处即同步 */
export const PREVIEW_COUNT = 5;

/** 解析/校验错误，`key` 与 `field` 均为 i18n key，由展示侧翻译后插值 */
export interface CronError {
    key: string;
    /** 出错字段，展示侧据此翻译出字段名填入 `{field}` 插值 */
    field?: CronFieldKey;
    params?: Record<string, string | number>;
}

/** 表达式的可编辑性：`panel` 可面板编辑，`descriptor` 为只能原样保留的描述符 */
export type CronExprKind = 'panel' | 'descriptor';

export interface CronParseResult {
    kind: CronExprKind;
    /** 解析失败或为描述符时为默认规则集 */
    spec: CronSpec;
    error: CronError | null;
}

/** 面板默认规则：`* * * * * ?`（每秒） */
export function defaultSpec(): CronSpec {
    return {
        second: { kind: 'all' },
        min: { kind: 'all' },
        hour: { kind: 'all' },
        day: { kind: 'all' },
        month: { kind: 'all' },
        week: { kind: 'none' },
    };
}

/** 是否限定了具体取值（`*` 与 `?` 都不算，等价于后端「无 star 位」） */
export function isRestricted(rule: CronRule): boolean {
    return rule.kind !== 'all' && rule.kind !== 'none';
}

/**
 * 写入某字段的规则，并维持「日/周只限定其一」的约束：
 * 限定了日就把周置为 `?`，反之同理，避免退化成后端的「或」匹配语义。
 */
export function withRule(spec: CronSpec, key: CronFieldKey, rule: CronRule): CronSpec {
    const next: CronSpec = { ...spec, [key]: rule };
    if (key === 'day' && isRestricted(rule)) {
        next.week = { kind: 'none' };
    } else if (key === 'week' && isRestricted(rule)) {
        next.day = { kind: 'none' };
    }
    return next;
}

export function formatRule(rule: CronRule): string {
    switch (rule.kind) {
        case 'none':
            return '?';
        case 'all':
            return '*';
        case 'cycle':
            return `${rule.from}-${rule.to}`;
        case 'step':
            return `${rule.start}/${rule.every}`;
        case 'list':
            return rule.values.join(',');
    }
}

/** 规则集 → 6 段表达式 */
export function buildExpression(spec: CronSpec): string {
    return CRON_FIELD_KEYS.map((key) => formatRule(spec[key])).join(' ');
}

/** 表达式 → 规则集；解析失败时回落默认规则并带回错误，调用方据此提示而不是静默改语义 */
export function parseCronExpression(expression: string): CronParseResult {
    const expr = (expression ?? '').trim();
    if (!expr) return { kind: 'panel', spec: defaultSpec(), error: null };
    if (expr.startsWith('@')) return { kind: 'descriptor', spec: defaultSpec(), error: null };

    const segments = expr.split(/\s+/);
    if (segments.length !== 5 && segments.length !== 6) {
        return {
            kind: 'panel',
            spec: defaultSpec(),
            error: { key: 'components.crontab.errFieldCount', params: { found: segments.length } },
        };
    }

    // 5 段缺秒，与后端 SecondOptional 的补默认行为一致
    const filled = segments.length === 6 ? segments : ['0', ...segments];
    const spec = defaultSpec();
    for (const key of CRON_FIELD_KEYS) {
        const parsed = parseSegment(CRON_FIELDS[key], filled[CRON_FIELD_KEYS.indexOf(key)]);
        if ('error' in parsed) return { kind: 'panel', spec: defaultSpec(), error: parsed.error };
        spec[key] = parsed.rule;
    }
    return { kind: 'panel', spec, error: null };
}

/** 输入框一行提示所需的全部信息，一次解析得出，避免消费点各自重复解析 */
export interface CronExpressionState {
    /** 还没敲完（段数不足 5 段），此时不该判定合法性 */
    incomplete: boolean;
    /** 后端能否接受；未敲完时为 true */
    valid: boolean;
    /** 下一次运行时间；描述符、未敲完或不合法时为 null */
    nextRun: Date | null;
}

export function inspectCronExpression(expression: string, from = new Date()): CronExpressionState {
    const expr = (expression ?? '').trim();
    if (!expr) return { incomplete: false, valid: true, nextRun: null };
    if (expr.startsWith('@')) return { incomplete: false, valid: isCronDescriptor(expr), nextRun: null };
    if (expr.split(/\s+/).length < 5) return { incomplete: true, valid: true, nextRun: null };

    const parsed = parseCronExpression(expr);
    if (parsed.error) return { incomplete: false, valid: false, nextRun: null };
    return { incomplete: false, valid: true, nextRun: nextRunTimes(parsed.spec, 1, from).times[0] ?? null };
}

const CRON_DESCRIPTORS = ['@yearly', '@annually', '@monthly', '@weekly', '@daily', '@midnight', '@hourly'];

/** Go `time.ParseDuration` 的形式：允许多段拼接与小数，如 `1h30m`、`0.5s` */
const DURATION_PATTERN = /^([0-9]*\.?[0-9]+(ns|us|µs|μs|ms|s|m|h))+$/;

function isCronDescriptor(expr: string): boolean {
    const tokens = expr.split(/\s+/);
    if (tokens.length === 1) return CRON_DESCRIPTORS.includes(tokens[0]);
    // 全零的时长（`0s`、`0.0m`）后端会接受并把调度打成忙循环，一并判错
    return tokens.length === 2 && tokens[0] === '@every' && /[1-9]/.test(tokens[1]) && DURATION_PATTERN.test(tokens[1]);
}

type SegmentParse = { rule: CronRule } | { error: CronError };

function parseSegment(field: CronField, raw: string): SegmentParse {
    const text = raw.trim();
    if (!text) return { error: { key: 'components.crontab.errEmptySegment', field: field.key } };
    if (text === '*') return { rule: { kind: 'all' } };
    // 后端把 `?` 等同 `*`；非日/周字段没有「不指定」语义，按未限定处理
    if (text === '?') return { rule: { kind: field.allowNone ? 'none' : 'all' } };

    const items = text.split(',');
    const compact: CronRule[] = [];
    const values: number[] = [];
    for (const item of items) {
        const parsed = parseRange(field, item.trim());
        if ('error' in parsed) return parsed;
        values.push(...parsed.values);
        if (parsed.rule) compact.push(parsed.rule);
    }

    // 单项且能整体表达为一条紧凑规则时保留原形，保证「读入再写出」不改变表达式形态
    if (items.length === 1 && compact.length === 1) return { rule: compact[0] };
    // 混合列表展开为离散取值；即便覆盖全量也不能归一成 `all`，那会凭空补出 star 位改变日/周的匹配方式
    return { rule: { kind: 'list', values: [...new Set(values)].sort((a, b) => a - b) } };
}

/** 区间形态，决定能否还原为紧凑规则 */
type RangeShape = 'star' | 'single' | 'bounds';

type RangeParse = { values: number[]; rule?: CronRule } | { error: CronError };

function parseRange(field: CronField, text: string): RangeParse {
    if (!text) return { error: { key: 'components.crontab.errEmptySegment', field: field.key } };

    const [body = '', stepText, ...extra] = text.split('/');
    if (extra.length || !body) return { error: segmentError(field, text) };
    const hasStep = stepText !== undefined;

    let shape: RangeShape = 'star';
    let from = field.min;
    let to = field.max;
    if (body !== '*' && body !== '?') {
        const bounds = body.split('-');
        if (bounds.length === 1) {
            const value = parseValue(field, bounds[0], text);
            if ('error' in value) return value;
            shape = 'single';
            from = value.value;
            // 后端语义：`a` 只取 a，`a/n` 表示自 a 至上界按 n 递增
            to = hasStep ? field.max : value.value;
        } else if (bounds.length === 2) {
            const lower = parseValue(field, bounds[0], text);
            if ('error' in lower) return lower;
            const upper = parseValue(field, bounds[1], text);
            if ('error' in upper) return upper;
            shape = 'bounds';
            from = lower.value;
            to = upper.value;
        } else {
            return { error: segmentError(field, text) };
        }
    }
    if (from > to) {
        return { error: { key: 'components.crontab.errRange', field: field.key, params: { from, to } } };
    }

    let every = 1;
    if (hasStep) {
        const step = parseStep(field, stepText);
        if ('error' in step) return step;
        every = step.value;
    }

    const values = toValues(from, to, every);
    // `*`/1 与 `*` 同样保留 star 位；`a/1`、`a-b/n` 等不带 star，即使覆盖全量
    if (shape === 'star' && every === 1) return { values, rule: { kind: 'all' } };
    if (shape === 'bounds') return hasStep ? { values } : { values, rule: { kind: 'cycle', from, to } };
    if (shape === 'single' && !hasStep) return { values, rule: { kind: 'list', values: [from] } };
    return { values, rule: { kind: 'step', start: from, every } };
}

function segmentError(field: CronField, segment: string): CronError {
    return { key: 'components.crontab.errSegment', field: field.key, params: { segment } };
}

function parseValue(field: CronField, text: string, segment: string): { value: number } | { error: CronError } {
    const named = field.aliases?.[text.toLowerCase()];
    // 别名表里的值本身就在字段边界内，无需再判界
    if (named !== undefined) return { value: named };
    if (!/^\d+$/.test(text)) return { error: segmentError(field, segment) };
    const value = Number(text);
    if (value < field.min || value > field.max) {
        return {
            error: {
                key: 'components.crontab.errValueRange',
                field: field.key,
                params: { min: field.min, max: field.max, value },
            },
        };
    }
    return { value };
}

/** 步长只要求正整数，上不封顶（超界步长后端等价于只取起始值，语义仍可表达） */
function parseStep(field: CronField, text: string): { value: number } | { error: CronError } {
    if (!/^\d+$/.test(text) || Number(text) < 1) {
        return { error: { key: 'components.crontab.errStep', field: field.key } };
    }
    return { value: Number(text) };
}

function toValues(from: number, to: number, step: number): number[] {
    const values: number[] = [];
    for (let value = from; value <= to; value += step) values.push(value);
    return values;
}

/** 规则覆盖到的全部取值，用于运行时间推算 */
export function ruleValues(field: CronField, rule: CronRule): number[] {
    switch (rule.kind) {
        case 'all':
        case 'none':
            return toValues(field.min, field.max, 1);
        case 'cycle':
            return toValues(rule.from, rule.to, 1);
        case 'step':
            return toValues(rule.start, field.max, rule.every);
        case 'list':
            return [...rule.values];
    }
}

/** 某字段的合法取值列表（chip 网格的数据源） */
export function fieldValues(field: CronField): number[] {
    return toValues(field.min, field.max, 1);
}

/** 取值的展示文本 i18n key（周字段为星期名），无专属名称时返回空串 */
export function ruleValueLabelKey(field: CronField, value: number): string {
    return field.valueLabelKeys?.[value] ?? '';
}

/** 规则的一句话说明，`unit` 插值由展示侧用字段名翻译后填入 */
export interface CronRuleDescription {
    key: string;
    params?: Record<string, string | number>;
}

export function describeRule(field: CronField, rule: CronRule): CronRuleDescription {
    switch (rule.kind) {
        case 'all':
            return { key: 'components.crontab.descAll' };
        case 'none':
            return { key: field.key === 'day' ? 'components.crontab.descDayNone' : 'components.crontab.descWeekNone' };
        case 'cycle':
            return { key: 'components.crontab.descCycle', params: { from: rule.from, to: rule.to } };
        case 'step':
            return { key: 'components.crontab.descStep', params: { start: rule.start, every: rule.every } };
        case 'list':
            return { key: 'components.crontab.descList', params: { count: rule.values.length, values: rule.values.join(', ') } };
    }
}

export interface CronPreview {
    /** 按时间升序的运行时间 */
    times: Date[];
    /** 检索窗口内不足请求条数（含 0 条） */
    exhausted: boolean;
}

/** 检索窗口：50 年，足以覆盖「2 月 29 日 + 指定星期」这类稀疏组合；提示文案按此插值 */
export const PREVIEW_YEARS = 50;

/** 跳转步数上限，防御始终无法命中的表达式 */
const PREVIEW_MAX_STEPS = 1_000_000;

/**
 * 推算 `from` 之后的前 `count` 次运行时间。
 *
 * 按「月→日→时→分→秒」逐级跳过不命中的区间，因此稀疏表达式（如每年一次）也只有少量迭代。
 * 日的匹配复刻后端 `dayMatches`：任一侧未限定则取交集（由另一侧决定），两侧都限定则取并集。
 */
export function nextRunTimes(spec: CronSpec, count = PREVIEW_COUNT, from = new Date()): CronPreview {
    const seconds = new Set(ruleValues(CRON_FIELDS.second, spec.second));
    const minutes = new Set(ruleValues(CRON_FIELDS.min, spec.min));
    const hours = new Set(ruleValues(CRON_FIELDS.hour, spec.hour));
    const days = new Set(ruleValues(CRON_FIELDS.day, spec.day));
    const months = new Set(ruleValues(CRON_FIELDS.month, spec.month));
    const weeks = new Set(ruleValues(CRON_FIELDS.week, spec.week));
    const dayUnrestricted = !isRestricted(spec.day);
    const weekUnrestricted = !isRestricted(spec.week);

    const cursor = new Date(from.getTime());
    cursor.setMilliseconds(0);
    cursor.setSeconds(cursor.getSeconds() + 1);

    const until = new Date(cursor.getTime());
    until.setFullYear(until.getFullYear() + PREVIEW_YEARS);

    const times: Date[] = [];
    for (let steps = 0; steps < PREVIEW_MAX_STEPS && times.length < count && cursor.getTime() < until.getTime(); steps++) {
        if (!months.has(cursor.getMonth() + 1)) {
            cursor.setDate(1);
            cursor.setHours(0, 0, 0, 0);
            cursor.setMonth(cursor.getMonth() + 1);
            continue;
        }
        const dayOfMonth = cursor.getDate();
        const dayOfWeek = cursor.getDay();
        const dayMatch = dayUnrestricted || weekUnrestricted ? days.has(dayOfMonth) && weeks.has(dayOfWeek) : days.has(dayOfMonth) || weeks.has(dayOfWeek);
        if (!dayMatch) {
            cursor.setDate(dayOfMonth + 1);
            cursor.setHours(0, 0, 0, 0);
            continue;
        }
        if (!hours.has(cursor.getHours())) {
            cursor.setHours(cursor.getHours() + 1, 0, 0, 0);
            continue;
        }
        if (!minutes.has(cursor.getMinutes())) {
            cursor.setMinutes(cursor.getMinutes() + 1, 0, 0);
            continue;
        }
        if (!seconds.has(cursor.getSeconds())) {
            cursor.setSeconds(cursor.getSeconds() + 1, 0);
            continue;
        }
        times.push(new Date(cursor.getTime()));
        cursor.setSeconds(cursor.getSeconds() + 1, 0);
    }

    return { times, exhausted: times.length < count };
}

/** 常用预设：一个按钮对应一条完整表达式，点即生效 */
export interface CronPreset {
    /** 预设标识 */
    key: string;
    /** 按钮文案 i18n key */
    labelKey: string;
    expr: string;
}

export const CRON_PRESETS: CronPreset[] = [
    { key: 'everyMinute', labelKey: 'components.crontab.presetEveryMinute', expr: '0 * * * * ?' },
    { key: 'every5Minutes', labelKey: 'components.crontab.presetEvery5Minutes', expr: '0 0/5 * * * ?' },
    { key: 'everyHour', labelKey: 'components.crontab.presetEveryHour', expr: '0 0 * * * ?' },
    { key: 'everyDay', labelKey: 'components.crontab.presetEveryDay', expr: '0 0 0 * * ?' },
    { key: 'everyWeekdayMorning', labelKey: 'components.crontab.presetEveryWeekdayMorning', expr: '0 0 9 ? * 1-5' },
    { key: 'everyMonthFirstDay', labelKey: 'components.crontab.presetEveryMonthFirstDay', expr: '0 0 0 1 * ?' },
];
