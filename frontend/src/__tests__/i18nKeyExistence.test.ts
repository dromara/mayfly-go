import { describe, expect, it } from 'vitest';
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';

// 与运行时同源：合并语言目录下的分文件得到两份完整字典
const zhModules = import.meta.glob<{ default: object }>('@/i18n/zh-cn/*.ts', { eager: true });
const enModules = import.meta.glob<{ default: object }>('@/i18n/en/*.ts', { eager: true });

function merge(modules: Record<string, { default: object }>): object {
    return Object.values(modules).reduce<Record<string, unknown>>((acc, mod) => Object.assign(acc, mod.default), {});
}

/**
 * 代码里写出的 i18n key 必须在两种语言的字典里都存在。
 *
 * vue-i18n 在本项目配了 missingWarn:false，缺 key 不报错而是**把 key 原样显示到界面上**，
 * 英文界面还会直接吐出 `flow.procDesign` 这样的串；类型检查、lint、构建都不管这件事。
 * 实测就漏过一个例子：流程设计抽屉的标题取 `flow.procDesign`，而字典里只有 `flow.flowDesign`。
 */
const dictionaries = { zh: merge(zhModules), en: merge(enModules) };

function listSourceFiles(dir: string, into: string[] = []): string[] {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
        const full = join(dir, entry.name);
        if (entry.isDirectory()) {
            if (entry.name !== 'node_modules' && entry.name !== '__tests__') listSourceFiles(full, into);
        } else if (/\.(vue|ts)$/.test(entry.name)) {
            into.push(full);
        }
    }
    return into;
}

/** 取 `$t('a.b.c')` / `t('a.b.c')` 里写死的 key（只认字面量单引号，动态拼接不在本判据内） */
function referencedKeys(source: string): string[] {
    // `\b` 保证命中的是独立的 t/$t 调用，不会把 emit(...) 这类词尾字母当成翻译函数
    return [...source.matchAll(/\$?\bt\(\s*'([a-z]+\.[A-Za-z0-9_.]+)'/g)].map((m) => m[1]);
}

function lookup(root: object, path: string): boolean {
    return (
        path.split('.').reduce<unknown>((node, key) => {
            if (!node || typeof node !== 'object') return undefined;
            return Reflect.get(node, key);
        }, root) !== undefined
    );
}

describe('i18n key 与文案存在性', () => {
    const files = listSourceFiles(join(process.cwd(), 'src'));

    it('扫到了足量引用（防止判据本身失效导致空跑）', () => {
        const total = files.reduce((sum, file) => sum + referencedKeys(readFileSync(file, 'utf8')).length, 0);
        expect(total).toBeGreaterThan(200);
    });

    it('写死的 key 在中文与英文字典里都能取到值', () => {
        const missing: string[] = [];
        for (const file of files) {
            for (const key of new Set(referencedKeys(readFileSync(file, 'utf8')))) {
                // 前缀不在字典顶层（动态拼出来的 key 不在本判据内）就跳过，其余要求两种语言都有值
                if (!(key.split('.')[0] in dictionaries.zh)) continue;
                for (const [locale, root] of Object.entries(dictionaries)) {
                    if (!lookup(root, key)) missing.push(`${file.replace(process.cwd() + '/', '')}: ${key}（${locale}）`);
                }
            }
        }
        expect(missing, '界面上会直接吐出裸 key:\n' + missing.join('\n')).toEqual([]);
    });
});
