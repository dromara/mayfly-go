/**
 * 写操作文本的解析与形状校验。
 *
 * 这些判据只用于「提交前拦住明显写错的形状」并把话说清楚，不是安全边界：
 * 后端对同一份文本还会各校验一次（且更严），前端拦住的目的是省一次往返与一句能看懂的提示。
 *
 * 返回的 issue.key 是语言包键而不是成品句子，调用方用 `t()` 出词，
 * 以便与全站提示语的语气、占位参数保持一致。
 */
export interface JsonIssue {
    /** 语言包键 */
    key: string;
    params?: Record<string, unknown>;
}

export type ParsedJson<T> = { ok: true; value: T } | { ok: false; issue: JsonIssue };

function invalid(key: string, params?: Record<string, unknown>): ParsedJson<never> {
    return { ok: false, issue: { key, params } };
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
    return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/** 读不到的哨兵值：`null` 本身是合法 JSON，不能拿它当「解析失败」的标志 */
const READ_FAILED = Symbol('mongo.json.readFailed');

/** 解析 JSON 文本本身：失败只说明「不是合法 JSON」，形状由调用方继续判 */
function read(text: string): unknown {
    const trimmed = (text ?? '').trim();
    if (!trimmed) {
        return READ_FAILED;
    }
    try {
        return JSON.parse(trimmed);
    } catch {
        return READ_FAILED;
    }
}

/** JSON 对象（条件、文档、单条命令都用它） */
export function parseJsonObject(text: string, fieldKey = 'mongo.docMustBeObject'): ParsedJson<Record<string, unknown>> {
    const value = read(text);
    if (!isPlainObject(value)) {
        return invalid(fieldKey);
    }
    return { ok: true, value };
}

/** 非空 JSON 数组（聚合管道、索引定义都用它） */
export function parseJsonArray(text: string, issueKey: string): ParsedJson<unknown[]> {
    const value = read(text);
    if (!Array.isArray(value) || !value.length) {
        return invalid(issueKey);
    }
    return { ok: true, value };
}

/**
 * 索引定义数组：每项必须是含 `key` 的文档。
 *
 * `name`、`unique`、`expireAfterSeconds` 等选项一律不校验也不补默认：
 * 服务端把这份原文交给 Mongo，前端替它猜名字反而会让「复制现有定义改一改」这条路失配。
 */
export function parseIndexSpecs(text: string): ParsedJson<Record<string, unknown>[]> {
    const parsed = parseJsonArray(text, 'mongo.indexSpecsInvalid');
    if (!parsed.ok) {
        return parsed;
    }
    const specs = parsed.value;
    if (!specs.every((spec) => isPlainObject(spec) && Object.hasOwn(spec, 'key'))) {
        return invalid('mongo.indexSpecsInvalid');
    }
    return { ok: true, value: specs as Record<string, unknown>[] };
}

/** 单条索引定义（详情里的「以现有定义新建」会传一份 spec 数组给 createIndexes） */
export function indexSpecTemplate(field = 'field'): string {
    return JSON.stringify([{ key: { [field]: 1 }, name: `${field}_1` }], null, 4);
}

/**
 * 更新内容：`{$set: {...}}` 这类操作符文档，或 `[{$set: {...}}]` 更新管道。
 *
 * 判据是「所有键以 `$` 开头」：Mongo 拒绝裸替换文档与操作符混用，
 * 而把 `{"status": "paid"}` 当成替换文档提交会整条覆盖文档（丢掉所有其他字段），必须拦住。
 */
export function parseUpdateSpec(text: string): ParsedJson<unknown> {
    const value = read(text);
    if (value === READ_FAILED) {
        return invalid('mongo.updateSpecInvalid');
    }
    if (Array.isArray(value)) {
        const pipelineOk = value.length > 0 && value.every((stage) => isPlainObject(stage) && Object.keys(stage).every((key) => key.startsWith('$')));
        return pipelineOk ? { ok: true, value } : invalid('mongo.updateSpecInvalid');
    }
    if (isPlainObject(value)) {
        const keys = Object.keys(value);
        if (keys.length && keys.every((key) => key.startsWith('$'))) {
            return { ok: true, value };
        }
    }
    return invalid('mongo.updateSpecInvalid');
}
