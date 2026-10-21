import { describe, expect, it } from 'vitest';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, resolve } from 'node:path';

/**
 * token 展示口径的全局统一守卫。
 *
 * 存在理由：token 量级天然很大（一次编排动辄上百万），而展示格式有两个现成原语
 * （`formatThousands` 精确千分位、`formatCountCompact` 紧凑 K/M/B）。只要允许各页面自己挑，
 * 就一定会出现「概览卡挑了千分位」这种读不出量级的数值 —— 治理看板的「Token 消耗」显示成
 * `1,957,362` 就是这么来的。现在 token 只认 `formatTokenCount`（列表用紧凑、概览/详情附精确值），
 * 本守卫保证它不被绕过：新增 token 展示位时不需要再判断该用哪种格式，也不可能用错。
 */
const srcRoot = resolve(__dirname, '..');

/** 取 src 下所有参与渲染的源文件（跳过测试与格式化原语自身：它们本来就该直接引用原语） */
function sourceFiles(dir: string, acc: string[] = []): string[] {
    for (const entry of readdirSync(dir)) {
        const full = join(dir, entry);
        if (entry === 'node_modules' || entry === 'dist' || entry === '__tests__') continue;
        if (statSync(full).isDirectory()) {
            sourceFiles(full, acc);
            continue;
        }
        if (!/\.(vue|ts)$/.test(entry)) continue;
        // 唯一入口自己的实现与它内部调用原语是合法的
        if (full.endsWith(join('common', 'utils', 'format.ts'))) continue;
        acc.push(full);
    }
    return acc;
}

const files = sourceFiles(srcRoot);

/** 去掉行注释与块注释：说明注释里会原样写出被禁的旧写法作为反面例子，不剔就是拿自己的注释当缺陷 */
function stripComments(source: string): string {
    return source.replace(/\/\*[\s\S]*?\*\//g, '').replace(/\/\/[^\n]*/g, '');
}

/** 取某文件里所有「调用了 fn 且实参含 Tokens」的调用片段 */
function tokenCalls(source: string, fn: string): string[] {
    const pattern = new RegExp(`${fn}\\(([^)]*[Tt]okens[^)]*)\\)`, 'g');
    return [...source.matchAll(pattern)].map((m) => m[0]);
}

describe('token 展示必须走唯一入口 formatTokenCount', () => {
    it('扫描面不为空（防判据空跑）', () => {
        const direct = files.filter((file) => stripComments(readFileSync(file, 'utf8')).includes('formatTokenCount('));
        const columns = files.filter((file) => /TableColumn\.new\('[A-Za-z]*Tokens'/.test(stripComments(readFileSync(file, 'utf8'))));
        expect(direct.length, '一个 token 展示位都没扫到，说明扫描路径失效').toBeGreaterThanOrEqual(2);
        expect(columns.length, '一个 token 列都没扫到，说明扫描路径失效').toBeGreaterThanOrEqual(2);
    });

    it('任何页面都不得把 token 直接交给 formatThousands / formatCountCompact', () => {
        for (const file of files) {
            const source = stripComments(readFileSync(file, 'utf8'));
            for (const fn of ['formatThousands', 'formatCountCompact']) {
                const hits = tokenCalls(source, fn);
                expect(hits, `${file.replace(srcRoot, 'src')} 绕过统一入口，用 ${fn} 渲染 token`).toEqual([]);
            }
        }
    });

    it('token 列必须声明为 isToken（列宽与格式同源，不允许留在通用计数上）', () => {
        for (const file of files) {
            const source = stripComments(readFileSync(file, 'utf8'));
            for (const line of source.split('\n')) {
                if (!/TableColumn\.new\('[A-Za-z]*Tokens'/.test(line)) continue;
                expect(line, `${file.replace(srcRoot, 'src')} 的 token 列没走 isToken()`).toContain('.isToken()');
            }
        }
    });

    it('通用计数入口 isCount 不得被重新引入而不带真实消费者', () => {
        // isCount 曾因「全部计数列都是 token 列」而成为零消费者死代码，被 isToken 取代。
        // 重新引入必须先有非 token 的真实调用点，否则就是又开一个并行的格式入口
        const used = files.filter((file) => /\bisCount\(\)/.test(stripComments(readFileSync(file, 'utf8'))));
        const defined = files.filter((file) => /isCount\(\): TableColumn/.test(stripComments(readFileSync(file, 'utf8'))));
        expect(defined.length, 'table-column.ts 里不该再有 isCount 定义').toBe(0);
        expect(used.length).toBe(0);
    });
});
