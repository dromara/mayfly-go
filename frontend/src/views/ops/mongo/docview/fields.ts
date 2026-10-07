/**
 * 文档的呈现派生：行标识、主键显示、字段清单。
 *
 * 都是纯计算，放在一处是为了让「表格、卡片、详情抽屉看到的是同一个身份」：
 * 行 key 若在两个视图各写一遍，切视图就会整列重建 DOM；主键类型若只在一个视图标注，
 * 另一个视图就会把 ObjectId 与「24 位十六进制的字符串」当成同一条。
 */
import type { MongoDoc } from '../types';
import { displayCell } from './extjson';
import { typeKindOf } from './schema';

/**
 * idKind → 显示名。
 *
 * 这些是 BSON/驱动的类型名而非文案，跨语言不需要翻译（翻译反而会让 mongosh 里对不上号），
 * 因此不走 i18n；后端 idKind 的取值集合与 mongodoc 的 classifyIDKind 同源。
 */
const ID_KIND_LABELS: Record<string, string> = {
    objectId: 'ObjectId',
    string: 'String',
    number: 'Number',
    date: 'ISODate',
    decimal: 'NumberDecimal',
    binary: 'BinData',
    regex: 'RegExp',
    timestamp: 'Timestamp',
    document: 'Document',
    array: 'Array',
    boolean: 'Boolean',
    null: 'Null',
    other: 'Other',
};

/**
 * 行 key：优先用主键令牌。
 *
 * 令牌对同一文档稳定不变，用数组下标会在删除/翻页后错位复用 DOM，
 * 用文档对象本身则每次查询都是新对象、整列表销毁重建。
 * 无令牌（投影排除了 _id）时退回内容特征，只为渲染稳定，不承诺唯一。
 */
export function docKey(doc: MongoDoc, index = -1): string {
    if (doc?.idToken) {
        return doc.idToken;
    }
    const source = doc?.doc ?? {};
    return `pos-${index}-${Object.keys(source).join(',')}`;
}

/** 主键类型显示名，未知类型原样透出（后端加了新分类时界面不至于显示 undefined） */
export function idKindLabel(kind: string): string {
    return ID_KIND_LABELS[kind] ?? kind;
}

/** 主键的文本形态（含类型包装的取包装里的可读值） */
export function idText(doc: MongoDoc): string {
    return displayCell(doc?.doc?._id);
}

/** 详情里的一行字段：路径、类型、可读值与原始值 */
export interface DocField {
    /** 展示路径，如 `buyer.name`；数组元素用 `tags[0]` */
    path: string;
    /**
     * 用于 Mongo 查询条件的路径（空串表示这个值不能做等值过滤）。
     *
     * 展示路径与查询路径必须分开：`tags[0]` 不是合法的 Mongo 字段路径（写了查不出任何东西），
     * 而数组元素真正该问的是「数组里有没有这个值」——Mongo 对数组字段做等值匹配天然命中任意元素，
     * 所以数组子项的查询路径是其所属数组本身；空文档/空数组没有可比的值，给空串由调用方禁用入口。
     */
    queryPath: string;
    /** 类型名（BSON 包装优先） */
    kind: string;
    /** 可读值，嵌套文档/数组给紧凑 JSON */
    display: string;
    /** 原始值（写过滤条件时要用它，而不是可读值） */
    value: unknown;
    /** 是否是类型包装字段：提示「删掉 $ 包装即改变类型」 */
    typed: boolean;
}

/** 递归深度上限：嵌套文档再深也不该无限展开，超过就当成紧凑值显示 */
const MAX_FIELD_DEPTH = 6;

/**
 * 摊平文档字段。
 *
 * 文档与数组继续下钻（运维要看的就是 `buyer.city` 这类路径），标量与 BSON 类型包装作为叶子；
 * 空文档与空数组留一行占位，否则「字段确实存在但没内容」会被当成缺字段。
 */
export function flattenFields(doc: Record<string, unknown> | undefined | null): DocField[] {
    const fields: DocField[] = [];

    const push = (value: unknown, path: string, queryPath: string, depth: number) => {
        const isNestedDoc = typeof value === 'object' && value !== null && !Array.isArray(value) && !isWrapper(value);
        if (isNestedDoc && depth < MAX_FIELD_DEPTH) {
            const entries = Object.entries(value as Record<string, unknown>);
            if (!entries.length) {
                // 空文档没有可等值匹配的值，查询路径留空
                fields.push({ path, queryPath: '', kind: 'Document', display: '{}', value, typed: false });
                return;
            }
            entries.forEach(([key, item]) => push(item, `${path}.${key}`, `${queryPath}.${key}`, depth + 1));
            return;
        }

        if (Array.isArray(value) && depth < MAX_FIELD_DEPTH) {
            if (!value.length) {
                fields.push({ path, queryPath: '', kind: 'Array', display: '[]', value, typed: false });
                return;
            }
            // 子项的查询路径指回数组本身：{ tags: 'vip' } 命中任意位置的该值
            value.forEach((item, index) => push(item, `${path}[${index}]`, queryPath, depth + 1));
            return;
        }

        fields.push({ path, queryPath, kind: typeKindOf(value), display: displayCell(value), value, typed: isWrapper(value) });
    };

    Object.entries(doc ?? {}).forEach(([key, value]) => push(value, key, key, 0));
    return fields;
}

/** 是否为 BSON 类型包装文档（单键且以 $ 开头即足够：真正的判据在 extjson 里） */
function isWrapper(value: unknown): boolean {
    if (typeof value !== 'object' || value === null || Array.isArray(value)) {
        return false;
    }
    const keys = Object.keys(value);
    return keys.length === 1 && keys[0].startsWith('$');
}
