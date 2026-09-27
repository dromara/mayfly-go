/**
 * 命令控制台的命令行能力：切分、当前输入位判定、命令名与 key 名提示。
 *
 * 命令目录由实例自描述下发（后端 COMMAND 回复），本文件不内置任何命令名单；
 * 唯一兼容的是目录未就绪时的 fail-closed 确认名单，因为不可逆命令不能因为一次请求失败就静默执行
 */

import type { RedisCommandSpec } from '../types';

/** 反斜杠转义表，与 redis-cli 的引号内转义语义一致 */
const ESCAPES: Record<string, string> = { n: '\n', r: '\r', t: '\t', b: '\b', a: '\x07' };

function unescape(text: string): string {
    return text.replace(/\\(.)/g, (_, char: string) => ESCAPES[char] ?? char);
}

const TOKEN_PATTERN = /"([^"\\]*(?:\\.[^"\\]*)*)"|'([^'\\]*(?:\\.[^'\\]*)*)'|(\S+)/g;

export function splitCommand(line: string): string[] {
    const args: string[] = [];
    // lastIndex 需要显式复位：该正则是模块级共享对象，多次调用会从上次的结束位置继续
    TOKEN_PATTERN.lastIndex = 0;

    let match = TOKEN_PATTERN.exec(line);
    while (match) {
        args.push(unescape(match[1] ?? match[2] ?? match[3] ?? ''));
        match = TOKEN_PATTERN.exec(line);
    }
    return args;
}

/** 控制台建议项：value 是选中后回填输入框的文本，其余只用于展示 */
export type RedisConsoleSuggestion = {
    value: string;
    name: string;
    kind: 'command' | 'key';
    write: boolean;
    confirm: boolean;
};

/**
 * 当前正在输入的位置：0 是命令名本身，>=1 是第 n 个参数。
 * 以空白结尾表示上一个参数已写完，正在开一个新的
 */
export function editingToken(line: string): { index: number; token: string } {
    const tokens = splitCommand(line.trimEnd());
    if (!tokens.length || /\s$/.test(line)) {
        return { index: tokens.length, token: '' };
    }
    return { index: tokens.length - 1, token: tokens[tokens.length - 1] ?? '' };
}

/**
 * 该参数位是否应给 key 名提示：用命令自描述的键参数位置算，而不是猜「第二个参数就是 key」。
 * 目录里查不到该命令（自定义模块命令等）时宽泛提示，提示错一个参数位只是多余，不影响执行
 */
export function isKeyArgument(spec: RedisCommandSpec | undefined, argIndex: number): boolean {
    if (argIndex < 1) {
        return false;
    }
    if (!spec) {
        return true;
    }
    if (spec.firstKey < 1 || argIndex < spec.firstKey) {
        return false;
    }
    const step = spec.step > 0 ? spec.step : 1;
    if ((argIndex - spec.firstKey) % step !== 0) {
        return false;
    }
    return spec.lastKey < 0 || argIndex <= spec.lastKey;
}

/** 命令名建议：前缀命中优先于包含命中，同为前缀时短名优先，让 GET 排在 GETRANGE 前面 */
export function suggestCommands(specs: RedisCommandSpec[], token: string, limit = 12): RedisConsoleSuggestion[] {
    const keyword = token.trim().toUpperCase();
    if (!keyword) {
        return [];
    }
    const weight = (name: string) => (name.startsWith(keyword) ? 0 : name.includes(keyword) ? 1 : 2);
    return specs
        .filter((spec) => weight(spec.name) < 2)
        .sort((a, b) => weight(a.name) - weight(b.name) || a.name.length - b.name.length || a.name.localeCompare(b.name))
        .slice(0, limit)
        .map((spec) => ({
            // 带尾空格：选中后直接可以接着输参数，不用手动补一个空格
            value: `${spec.name.toLowerCase()} `,
            name: spec.name,
            kind: 'command' as const,
            write: spec.flags.some((flag) => flag.toLowerCase() === 'write'),
            confirm: spec.needConfirm,
        }));
}

/** 命令目录未就绪时的兜底确认名单（fail-closed） */
const FALLBACK_CONFIRM = new Set([
    'flushall',
    'flushdb',
    'shutdown',
    'config',
    'debug',
    'script',
    'function',
    'cluster',
    'acl',
    'swapdb',
    'slaveof',
    'replicaof',
    'eval',
    'evalsha',
    'monitor',
]);

/**
 * 执行前是否需要二次确认：以实例命令目录的标志为准（与后端高危命令判定同源），
 * 目录未就绪时落到兜底名单——清空整库这类不可逆动作，不能因为一次请求失败就静默执行
 */
export function commandNeedsConfirm(spec: RedisCommandSpec | undefined, name: string): boolean {
    return spec?.needConfirm ?? FALLBACK_CONFIRM.has(name.toLowerCase());
}

/** key 名建议：只从已加载的 key 列表里找，避免每敲一个字符就去服务端扫一次库 */
export function suggestKeys(keys: string[], token: string, limit = 12): RedisConsoleSuggestion[] {
    const keyword = token.trim().toLowerCase();
    if (!keyword) {
        return [];
    }
    const prefix: string[] = [];
    const partial: string[] = [];
    const seen = new Set<string>();
    for (const key of keys) {
        if (seen.has(key)) {
            continue;
        }
        seen.add(key);
        const lower = key.toLowerCase();
        if (lower.startsWith(keyword)) {
            prefix.push(key);
        } else if (lower.includes(keyword)) {
            partial.push(key);
        }
    }
    return [...prefix, ...partial].slice(0, limit).map((key) => ({ value: key, name: key, kind: 'key' as const, write: false, confirm: false }));
}
