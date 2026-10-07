/**
 * 导入文件的解析与分批。
 *
 * 只接受 JSON 与 NDJSON：CSV 无法表达 BSON 类型（日期、ObjectId、Decimal128 会变成字符串），
 * 用它导入等于把字段类型悄悄改掉——那正是本次重构要消灭的损坏形态，宁可不支持也不假装支持。
 *
 * 解析全部在浏览器完成，出错行带定位信息，让用户改文件而不是猜哪一行没进去。
 */

/** 出错定位的说明：NDJSON 按下标给行号，JSON 数组形态给元素下标 +1 */
export interface ImportIssue {
    at: number;
    reason: 'invalidJson' | 'notObject';
}

export interface ParsedImport {
    docs: Record<string, unknown>[];
    issues: ImportIssue[];
}

function isDoc(value: unknown): value is Record<string, unknown> {
    return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/**
 * 解析导入文本。
 *
 * 先整体按 JSON 解析：能解析就说明是数组或单文档形态；失败再按 NDJSON 逐行解析。
 * 两种形态都不抛异常，坏行只记录原因，因为「1000 行里第 812 行写错」不该让前 811 行白工。
 */
export function parseImportText(text: string): ParsedImport {
    const trimmed = (text ?? '').trim();
    if (!trimmed) {
        return { docs: [], issues: [] };
    }

    try {
        const value: unknown = JSON.parse(trimmed);
        if (Array.isArray(value)) {
            const docs: Record<string, unknown>[] = [];
            const issues: ImportIssue[] = [];
            value.forEach((item, index) => {
                if (isDoc(item)) {
                    docs.push(item);
                    return;
                }
                issues.push({ at: index + 1, reason: 'notObject' });
            });
            return { docs, issues };
        }
        if (isDoc(value)) {
            return { docs: [value], issues: [] };
        }
        return { docs: [], issues: [{ at: 1, reason: 'notObject' }] };
    } catch {
        // 整体不是合法 JSON，按「每行一个文档」再解析一次
    }

    const docs: Record<string, unknown>[] = [];
    const issues: ImportIssue[] = [];
    text.split(/\r?\n/).forEach((line, index) => {
        if (!line.trim()) {
            return;
        }
        let value: unknown;
        try {
            value = JSON.parse(line);
        } catch {
            issues.push({ at: index + 1, reason: 'invalidJson' });
            return;
        }
        if (!isDoc(value)) {
            issues.push({ at: index + 1, reason: 'notObject' });
            return;
        }
        docs.push(value);
    });
    return { docs, issues };
}

/**
 * 切批大小。
 *
 * 一次提交全部文档会把整份文件压进一个请求：既容易顶到执行时间上限，失败时也得从头再来。
 * 200 条是让「单次请求体不大、批次数不至于太多」的折中值。
 */
export const IMPORT_BATCH_SIZE = 200;

/** 按固定长度切分，末批允许不满 */
export function chunk<T>(items: T[], size = IMPORT_BATCH_SIZE): T[][] {
    if (!items.length) {
        return [];
    }
    const batch = size > 0 ? size : items.length;
    const res: T[][] = [];
    for (let index = 0; index < items.length; index += batch) {
        res.push(items.slice(index, index + batch));
    }
    return res;
}
