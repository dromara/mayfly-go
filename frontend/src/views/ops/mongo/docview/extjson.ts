/**
 * Mongo 文档的 Extended JSON 展示层。
 *
 * 后端按「普通 JSON 能否无损表达该文档」决定返回形态：
 *  - mode=plain    文档所有字段都能用 JSON 无损表达，直接展示即可
 *  - mode=extjson  含 Date/ObjectID/Decimal128/Binary 等 BSON 专有类型，用 $ 包装显式承载
 *
 * 本模块只做一件事：把 $ 包装翻译成人类可读的显示值与类型名，并找出文档里所有带类型标注的位置。
 * 显示与写回是分离的——编辑器始终以原文为本源，写回统一交给后端解析（普通 JSON 是 Extended JSON
 * 的真子集，两种模式共用同一条解码路径），所以这里的一切判定都只影响「怎么看」，不影响「存什么」。
 */

/** 文档编码模式，取值与后端 mongodoc.Mode 字面一致 */
export type DocMode = 'plain' | 'extjson';

export const DOC_MODE_PLAIN = 'plain';
export const DOC_MODE_EXT_JSON = 'extjson';

/** relaxed Extended JSON 的类型包装键 → BSON 类型名 */
const WRAPPER_TYPES: Record<string, string> = {
    $oid: 'ObjectID',
    $date: 'Date',
    $numberLong: 'NumberLong',
    $numberInt: 'Int32',
    $numberDouble: 'Double',
    $numberDecimal: 'Decimal128',
    $binary: 'Binary',
    $uuid: 'UUID',
    $regex: 'Regex',
    $regularExpression: 'Regex',
    $timestamp: 'Timestamp',
    $code: 'Code',
    $dbPointer: 'DBPointer',
    $minKey: 'MinKey',
    $maxKey: 'MaxKey',
    $undefined: 'Undefined',
};

/** 字段路径与其上的 BSON 类型 */
export interface TypedField {
    /** 点号路径，如 `meta.createdAt`，与脱敏规则、索引字段的表达一致 */
    path: string;
    /** BSON 类型名，如 Date、Decimal128 */
    type: string;
    /** 人类可读的当前值 */
    display: string;
}

function isObject(value: unknown): value is Record<string, unknown> {
    return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/**
 * 报告该值是否为一个类型包装文档，返回其 BSON 类型名。
 *
 * 判据是「只有一个键且该键是已知的 $ 包装」：Extended JSON 的类型包装都是单键文档，
 * 因此数据里恰好含 $ 开头的普通字段不会被误认成类型标注。
 */
export function wrapperTypeOf(value: unknown): string | undefined {
    if (!isObject(value)) {
        return undefined;
    }
    const keys = Object.keys(value);
    if (keys.length !== 1) {
        return undefined;
    }
    const [key] = keys;
    if (!Object.prototype.hasOwnProperty.call(WRAPPER_TYPES, key)) {
        return undefined;
    }
    // $binary/$timestamp/$regularExpression 等的内层是结构体而不是标量，仍属于同一个包装
    return WRAPPER_TYPES[key];
}

/** 把类型包装翻译成可读值 */
export function displayWrapper(key: string, payload: unknown): string {
    const text = (v: unknown) => (v === null || v === undefined ? '' : String(v));

    switch (key) {
        case '$oid':
            return text(payload);
        case '$date': {
            // relaxed 形态是不带毫秒的 ISO 字符串，canonical 形态是 {"$numberLong":"毫秒"}
            if (isObject(payload)) {
                return formatDate(Number(text(payload.$numberLong)));
            }
            const asNumber = Number(payload);
            return Number.isFinite(asNumber) ? formatDate(asNumber) : text(payload);
        }
        case '$numberLong':
        case '$numberInt':
        case '$numberDecimal':
            return text(payload);
        case '$numberDouble':
            // Infinity/NaN 在 JSON 词法里无法表达，后端以此字符串承载，展示时原样给出
            return text(payload);
        case '$binary': {
            if (isObject(payload)) {
                const { subType, base64 } = payload as { subType?: unknown; base64?: unknown };
                const size = estimateBase64Bytes(text(base64));
                return `BinData(${formatSubType(text(subType))}, ${size} bytes)`;
            }
            return text(payload);
        }
        case '$uuid':
            return `UUID(${text(payload)})`;
        case '$timestamp': {
            if (isObject(payload)) {
                const { t, i } = payload as { t?: unknown; i?: unknown };
                return `Timestamp(${text(t)}, ${text(i)})`;
            }
            return text(payload);
        }
        case '$regex':
            return `/${text(payload)}/`;
        case '$regularExpression': {
            if (isObject(payload)) {
                const { pattern, options } = payload as { pattern?: unknown; options?: unknown };
                return `/${text(pattern)}/${text(options)}`;
            }
            return text(payload);
        }
        case '$code':
            return `Code(${text(payload)})`;
        case '$dbPointer':
            return `DBPointer(${text(payload)})`;
        case '$minKey':
            return 'MinKey';
        case '$maxKey':
            return 'MaxKey';
        case '$undefined':
            return 'undefined';
        default:
            return text(payload);
    }
}

/**
 * 收集文档里所有带 BSON 类型标注的字段。
 *
 * 用途是「一眼看出这份文档为什么不是普通 JSON」：编辑器上方列出类型清单后，用户删掉 `$date`
 * 这类包装会造成字段类型变化就不再是无意识的动作（Date→String 是本次重构要消灭的损坏形态）。
 */
export function collectTypedFields(doc: unknown, prefix = ''): TypedField[] {
    const fields: TypedField[] = [];

    const walk = (value: unknown, path: string) => {
        const type = wrapperTypeOf(value);
        if (type) {
            const [key] = Object.keys(value as Record<string, unknown>);
            fields.push({ path, type, display: displayWrapper(key, (value as Record<string, unknown>)[key]) });
            return;
        }
        if (Array.isArray(value)) {
            value.forEach((item, index) => walk(item, `${path}[${index}]`));
            return;
        }
        if (isObject(value)) {
            Object.keys(value).forEach((key) => walk(value[key], path ? `${path}.${key}` : key));
        }
    };

    walk(doc, prefix);
    return fields;
}

/** 类型读数字符串，如 `Date ×1、ObjectID ×1`；无类型标注时返回空串 */
export function typedFieldsSummary(fields: TypedField[]): string {
    if (!fields.length) {
        return '';
    }
    const counts = new Map<string, number>();
    fields.forEach((field) => counts.set(field.type, (counts.get(field.type) ?? 0) + 1));
    return Array.from(counts.entries())
        .map(([type, count]) => `${type} ×${count}`)
        .join('、');
}

/**
 * 把带类型包装的值换成可读形态：`{"$date":"..."}` → `2026-09-01 10:00:00`。
 *
 * 只用于展示：写回一律以原文为本源，否则「看得到改不动」与「改得动但类型丢了」会互相替换。
 */
export function displayCell(value: unknown): string {
    const type = wrapperTypeOf(value);
    if (type) {
        const [key] = Object.keys(value as Record<string, unknown>);
        return displayWrapper(key, (value as Record<string, unknown>)[key]);
    }
    if (value === null || value === undefined) {
        return 'null';
    }
    if (typeof value !== 'object') {
        return String(value);
    }
    return JSON.stringify(displayForm(value));
}

/** displayCell 的递归准备：先把嵌套层的类型包装换成标量，再交给 JSON 序列化 */
export function displayForm(value: unknown): unknown {
    const type = wrapperTypeOf(value);
    if (type) {
        const [key] = Object.keys(value as Record<string, unknown>);
        return displayWrapper(key, (value as Record<string, unknown>)[key]);
    }
    if (Array.isArray(value)) {
        return value.map((item) => displayForm(item));
    }
    if (value !== null && typeof value === 'object') {
        const res: Record<string, unknown> = {};
        Object.keys(value).forEach((key) => {
            res[key] = displayForm((value as Record<string, unknown>)[key]);
        });
        return res;
    }
    return value;
}

/**
 * 稳定缩进。
 *
 * 直接用 JSON.stringify：对象的字符串键保持插入顺序，因此后端返回的字段顺序得以沿用；
 * 唯一例外是纯数字字段名（如 `{"0": ...}`）会被 JS 引擎提前，只影响显示顺序，不影响写回内容
 * （文档字段顺序在 BSON 语义上不参与比较）。
 */
export function prettyDoc(doc: unknown): string {
    try {
        return JSON.stringify(doc ?? null, null, 4);
    } catch {
        // 循环引用等异常形状不应让页面崩，退回占位文本；真实数据不会走到这里
        return '{}';
    }
}

/** 编辑器文本 → 文档对象；解析失败返回 null，由调用方提示而不是静默按空文档处理 */
export function parseDocText(text: string): unknown | null {
    if (!text.trim()) {
        return null;
    }
    try {
        return JSON.parse(text);
    } catch {
        return null;
    }
}

function formatDate(ms: number): string {
    if (!Number.isFinite(ms)) {
        return '';
    }
    const date = new Date(ms);
    const pad = (v: number) => String(v).padStart(2, '0');
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}

/**
 * base64 字符串对应的字节数（忽略换行与填充误差，取整即可）
 */
function estimateBase64Bytes(base64: string): number {
    const cleaned = base64.replace(/\s/g, '');
    if (!cleaned) {
        return 0;
    }
    const padding = cleaned.endsWith('==') ? 2 : cleaned.endsWith('=') ? 1 : 0;
    return Math.max(0, Math.floor((cleaned.length * 3) / 4) - padding);
}

/**
 * 子类型按十进制显示。
 *
 * Extended JSON 的 subType 是十六进制字符串（UUID 为 `"04"`），而 mongosh 与运维习惯都写成
 * `BinData(4, ...)`；直接透出 `04` 会让人误以为子类型是十进制的 4 以外的东西。
 */
function formatSubType(subType: string): string {
    if (!subType) {
        return '';
    }
    const parsed = Number.parseInt(subType, 16);
    return Number.isNaN(parsed) ? subType : String(parsed);
}
