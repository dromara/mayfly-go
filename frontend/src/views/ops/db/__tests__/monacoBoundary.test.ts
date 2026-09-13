import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

/**
 * 重依赖边界守卫（monaco 编辑器主体、sql-formatter）
 *
 * 编辑器主体约 3.9M（gz 约 1M）、sql-formatter 约 200K（gz 约 55K），一旦被 db 模块的公共入口
 * （db.ts）或按需挂载的页面壳静态触及，就会顺着依赖链回流到表数据页、资源树、实例列表等所有 DB 页面。
 * 这类退化不改变任何功能，只体现在构建产物上，故用源码静态可达性断言锁住。
 */

const DB_ROOT = path.resolve(__dirname, '..');
const SRC_ROOT = path.resolve(__dirname, '../../../..');

/**
 * 编辑器主体的说明符：裸包 `monaco-editor` 与它的任意深路径（setup.ts 就是按深路径逐个注册的）。
 *
 * 例外是 `monaco-editor/nls/**`——那只是约 97KB 的中文文案表，main.ts 必须最先引它（monaco 的部分
 * 本地化常量在模块加载期就求值），它不进编辑器 chunk，故不算污染。
 *
 * 守卫以「能否解析到仓库内文件」划分：解析得到的说明符一律继续下钻，只有落到裸包时才算命中。
 * 这样 `@/components/monaco/MonacoEditorBox`（只是弹窗外壳，编辑器在其异步子组件里）不会被误判，
 * 而「外壳里把编辑器改成静态引入」这种退化会被下钻到 MonacoEditor.vue → setup.ts → 裸包时抓到。
 */
const MONACO_PKG = /^monaco-editor(?!\/nls\/)/;

/** 惰性注册入口；它的静态引用方即「需要 SQL 联想但不能吃编辑器」的页面清单 */
const LAZY_ENTRY = path.join(DB_ROOT, 'completion', 'lazy.ts');

/** 命令式编辑器弹窗入口；被 api 模块等底层文件静态引用（含首页资源卡片），静态图上不得出现 monaco */
const MONACO_BOX_ENTRY = path.join(SRC_ROOT, 'components', 'monaco', 'MonacoEditorBox.ts');

/** 递归收集源码文件（跳过测试目录） */
function collect(dir: string): string[] {
    return readdirSync(dir).flatMap((name) => {
        const full = path.join(dir, name);
        if (statSync(full).isDirectory()) {
            return name === '__tests__' ? [] : collect(full);
        }
        return /\.(ts|vue)$/.test(name) ? [full] : [];
    });
}

/**
 * 预处理：去注释 + 把跨行的具名导入塌成一行。
 *
 * 去注释是为了不把注释里的 import 示例当成真依赖；塌行是因为 prettier 换行的
 * `import {\n a,\n b\n} from 'x'` 若按行匹配会被整个漏掉，守卫就此失效。
 */
function normalize(src: string): string {
    return src
        .replace(/\/\*[\s\S]*?\*\//g, '')
        .replace(/^\s*\/\/.*$/gm, '')
        .replace(/\{[^{}]*\}/g, (block) => (block.includes('\n') ? block.replace(/\s+/g, ' ') : block));
}

/** 具名列表内全部说明符都带 type 修饰时，该语句同样被编译擦除 */
function isTypeOnlyList(clause: string): boolean {
    const named = /^\{(.*)\}$/.exec(clause.trim());
    if (!named) {
        return false;
    }
    const names = named[1]
        .split(',')
        .map((n) => n.trim())
        .filter(Boolean);
    return names.length > 0 && names.every((n) => n.startsWith('type '));
}

/** 源码文本缓存（本文件会多轮全量遍历 src，避免重复 IO） */
const sourceCache = new Map<string, string>();

function readSource(file: string): string {
    let src = sourceCache.get(file);
    if (src === undefined) {
        src = normalize(readFileSync(file, 'utf8'));
        sourceCache.set(file, src);
    }
    return src;
}

/**
 * 提取静态导入的模块说明符。
 * 只认 `import x from '...'` / `import '...'` / `export ... from '...'`，
 * 排除 `import type`（编译期擦除，不产生运行时边）与 `import('...')`（动态，按需加载）。
 */
function staticSpecifiers(file: string): string[] {
    const src = readSource(file);
    const specs: string[] = [];
    // 副作用导入：import './x'
    for (const m of src.matchAll(/^[ \t]*import[ \t]+['"]([^'"]+)['"]/gm)) {
        specs.push(m[1]);
    }
    // 带说明符的导入/再导出：import ... from './x'、export * from './x'
    for (const m of src.matchAll(/^[ \t]*(?:import|export)[ \t]+(?!type[ \t])([^'"]*?)[ \t]+from[ \t]+['"]([^'"]+)['"]/gm)) {
        if (!isTypeOnlyList(m[1])) {
            specs.push(m[2]);
        }
    }
    return specs;
}

/** 将说明符解析为仓库内文件；裸包（monaco-editor 等）返回 null */
function resolveSpec(spec: string, fromFile: string): string | null {
    let target: string;
    if (spec.startsWith('@/')) target = path.join(SRC_ROOT, spec.slice(2));
    else if (spec.startsWith('.')) target = path.resolve(path.dirname(fromFile), spec);
    else return null;

    const candidates = [target, `${target}.ts`, `${target}.vue`, `${target}.js`, path.join(target, 'index.ts'), path.join(target, 'index.vue')];
    return candidates.find((c) => existsSync(c) && statSync(c).isFile()) ?? null;
}

/**
 * 自入口出发做静态导入闭包，返回首个触及 monaco 的路径；未触及返回 null。
 */
function findMonacoPath(entry: string): string[] | null {
    const prev = new Map<string, string | null>([[entry, null]]);
    const queue = [entry];
    const hit = (() => {
        while (queue.length) {
            const cur = queue.shift()!;
            for (const spec of staticSpecifiers(cur)) {
                const next = resolveSpec(spec, cur);
                // 解析不到仓库内文件的说明符即裸包，落到 monaco-editor 就是编辑器主体本身
                if (!next) {
                    if (MONACO_PKG.test(spec)) return [...reconstruct(cur), spec];
                    continue;
                }
                if (prev.has(next)) continue;
                prev.set(next, cur);
                queue.push(next);
            }
        }
        return null;
    })();
    return hit;

    function reconstruct(file: string): string[] {
        const chain: string[] = [];
        let cur: string | null = file;
        while (cur) {
            chain.unshift(path.relative(SRC_ROOT, cur));
            cur = prev.get(cur) ?? null;
        }
        return chain;
    }
}

/** 引用了惰性联想入口的文件（这些页面一旦被改成静态引入 completion 桶，首屏就会吃编辑器） */
function lazyScopeUsers(): string[] {
    return collect(SRC_ROOT)
        .filter((f) => staticSpecifiers(f).some((s) => resolveSpec(s, f) === LAZY_ENTRY))
        .map((f) => path.relative(SRC_ROOT, f));
}

describe('db 模块 monaco 依赖边界', () => {
    // 这些入口自身不渲染编辑器（编辑器在其异步子组件/弹窗里），静态闭包必须与 monaco 无关。
    // 手工登记：判据是「页面壳不吃编辑器」，与是否申领 SQL 联想无关（如审批流 SQL 表单确实要编辑器）
    const lazyEntries = ['db.ts', 'completion/lazy.ts', 'resource/DbDataOp.vue', 'sql-editor/SqlExecDialog.vue', 'table-editor/DbTablesOp.vue', 'sync/SyncTaskEdit.vue'];

    it.each(lazyEntries)('%s 的静态导入闭包不含 monaco（编辑器不得回流进首屏）', (entry) => {
        const badPath = findMonacoPath(path.join(DB_ROOT, entry));
        expect(badPath, badPath ? `静态依赖链: ${badPath.join(' → ')}` : '').toBeNull();
    });

    it('标签页容器与命令式 SQL 弹窗不直接静态引入编辑器运行时', () => {
        // 这三处是低频/按需场景：编辑器与补全子系统必须留到真正用到的那一刻才加载
        const lazyShells = ['resource/DbDataOp.vue', 'sql-editor/SqlExecDialog.vue', 'data-grid/DbTableDataOp.vue'];
        // MonacoEditor.vue 与 setup.ts 任一环静态引入都会顺到 monaco-editor 裸包，把编辑器并进本文件所在 chunk
        const EDITOR_RUNTIME = /(MonacoEditor\.vue|setup\.ts)$/;
        for (const shell of lazyShells) {
            const file = path.join(DB_ROOT, shell);
            const staticEditor = staticSpecifiers(file).filter((s) => EDITOR_RUNTIME.test(resolveSpec(s, file) ?? '') || MONACO_PKG.test(s));
            expect(staticEditor, `${shell} 不应静态导入编辑器运行时: ${JSON.stringify(staticEditor)}`).toEqual([]);
        }
    });

    it('completion 桶对外只暴露惰性入口（禁止跨目录静态 import completion/index）', () => {
        const completionDir = path.join(DB_ROOT, 'completion') + path.sep;
        const barrel = path.join(DB_ROOT, 'completion', 'index.ts');
        const violators = collect(SRC_ROOT)
            .filter((f) => !f.startsWith(completionDir))
            .filter((f) => staticSpecifiers(f).some((s) => resolveSpec(s, f) === barrel))
            .map((f) => path.relative(SRC_ROOT, f));
        expect(violators).toEqual([]);
    });

    it('db 模块内不得值导入裸包 monaco-editor（值导入必须走装配入口 setup）', () => {
        const violators = collect(DB_ROOT).filter((file) => staticSpecifiers(file).some((s) => MONACO_PKG.test(s)));
        expect(violators.map((f) => path.relative(SRC_ROOT, f))).toEqual([]);
    });

    it('db 模块内不得静态 import sql-formatter（须经 sql-editor/utils/formatSql 惰性取用）', () => {
        // 格式化只在用户点击后用到，静态引入会把它并入所有 DB 页面共享的 db chunk
        const helper = path.join(DB_ROOT, 'sql-editor', 'utils', 'formatSql.ts');
        const violators = collect(DB_ROOT)
            .filter((file) => file !== helper && staticSpecifiers(file).includes('sql-formatter'))
            .map((file) => path.relative(SRC_ROOT, file));
        expect(violators).toEqual([]);
    });

    it('申领 SQL 联想的文件必须成对释放，否则全局 provider 会随卸载方残留', () => {
        // 补全 provider 按语言全局唯一，只申请不释放会让联想停留在已销毁页面的上下文上
        const users = lazyScopeUsers();
        expect(users.length, '未找到任何 completion/lazy 使用方，守卫可能已失效').toBeGreaterThan(0);
        const unpaired = users.filter((f) => !/\.release\(\)/.test(readSource(path.join(SRC_ROOT, f))));
        expect(unpaired).toEqual([]);
    });
});

/** 取出文件里的动态导入说明符（`import('...')`），即运行时才产生的懒边 */
function dynamicSpecifiers(file: string): string[] {
    return [...readSource(file).matchAll(/import\(\s*['"]([^'"]+)['"]\s*\)/g)].map((m) => m[1]);
}

/**
 * 编辑器弹窗与首屏的加载边界
 *
 * 编辑器主体是约 967KB(gzip) 的独立 chunk，由 layout 在进入控制台后空闲预取（点开即秒出）。
 * 预取是运行时的动态导入，不改静态图：谁若把它改成静态 import，整份 monaco 就会挂到首屏渲染的关键路径上。
 */
describe('编辑器与首屏的加载边界', () => {
    it('MonacoEditorBox 的静态闭包不含 monaco（编辑器只能是弹窗内的异步组件）', () => {
        // 该入口被 api 模块等底层文件静态引用，弹窗里的编辑器一旦改成静态引入，monaco 就顺着这条边回流首屏
        const badPath = findMonacoPath(MONACO_BOX_ENTRY);
        expect(badPath, badPath ? `静态依赖链: ${badPath.join(' → ')}` : '').toBeNull();
    });

    it('layout 空闲预取的正是弹窗异步挂载的那个编辑器', () => {
        // 「点开秒出」完全依赖这条预取边：删掉它或指错文件，既没有报错也不会有其他用例失败，只是每次都退化成等 967KB
        // 预取挂在 layout 而不是 main.ts：登录页用不到编辑器，不该为它下载这份内容
        const shellFile = path.join(SRC_ROOT, 'layout', 'index.vue');
        const dialogFile = path.join(SRC_ROOT, 'components', 'monaco', 'MonacoEditorDialog.vue');
        const shellSrc = readSource(shellFile);
        // 必须是空闲时机：写成顶层立即 import() 会和页面自身首屏请求抢带宽，写成静态 import 则会挂进关键路径（由下面的闭包用例拦）
        expect(shellSrc, 'layout 的预取未走 requestIdleCallback').toContain('requestIdleCallback');
        const prefetched = dynamicSpecifiers(shellFile).map((s) => resolveSpec(s, shellFile));
        const lazilyMounted = dynamicSpecifiers(dialogFile).map((s) => resolveSpec(s, dialogFile));
        expect(lazilyMounted.length, '弹窗里已找不到异步引入的编辑器，本用例需随之调整').toBe(1);
        expect(prefetched).toEqual(expect.arrayContaining(lazilyMounted));
    });

    // 资源模块的页面入口按目录结构推导，新增模块零登记即被覆盖；判据是「进入该模块的第一站不吃编辑器」
    const resourceEntries = () =>
        readdirSync(path.join(SRC_ROOT, 'views', 'ops'))
            .map((name) => path.join(SRC_ROOT, 'views', 'ops', name, 'resource', 'index.ts'))
            .filter((f) => existsSync(f))
            .map((f) => path.relative(SRC_ROOT, f));

    it('资源模块入口清单非空（否则下面的按目录推导已失效）', () => {
        expect(resourceEntries().length).toBeGreaterThanOrEqual(5);
    });

    it.each(['main.ts', 'router/index.ts', 'layout/index.vue', ...resourceEntries()])('%s 的静态闭包不含 monaco（首屏不等编辑器）', (entry) => {
        const badPath = findMonacoPath(path.join(SRC_ROOT, entry));
        expect(badPath, badPath ? `静态依赖链: ${badPath.join(' → ')}` : '').toBeNull();
    });
});
