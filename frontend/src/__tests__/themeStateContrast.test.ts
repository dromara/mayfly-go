import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

/**
 * 状态色可读性修复必须「主题成对、作用域明确」。
 *
 * 亮色下 EP 拿语义色中阶做文字（徽标 2.52:1、行内链接 2.78:1），压暗即可；
 * 但同一批压暗值不分主题盖上去，暗色下会从 6.93:1 掉到 3.38:1；
 * 而盖到 dark 变体（EP 给实心底的是白字）又变成深字压同色实底，只剩 2.05:1。
 * 两类都是「只在一套主题/一个变体下成立」的覆盖，故按作用域各守一条。
 */
describe('主题作用域与动效降级', () => {
    const css = readFileSync(join(process.cwd(), 'src/theme/element.scss'), 'utf8');

    /** 取某个选择器块的字符区间（按花括号配对，块内可含嵌套） */
    function scopes(pattern: RegExp): [number, number][] {
        const ranges: [number, number][] = [];
        const scanner = new RegExp(pattern.source, pattern.flags.includes('g') ? pattern.flags : pattern.flags + 'g');
        for (let m = scanner.exec(css); m; m = scanner.exec(css)) {
            let depth = 0;
            for (let i = m.index + m[0].length - 1; i < css.length; i++) {
                if (css[i] === '{') depth++;
                else if (css[i] === '}' && --depth === 0) {
                    ranges.push([m.index, i + 1]);
                    break;
                }
            }
        }
        return ranges;
    }

    const lightScopes = () => scopes(/html:not\(\.dark\)\s*\{/g);
    const darkScopes = () => scopes(/html\.dark\s*\{/g);
    const inRange = ([start, end]: [number, number], pos: number) => pos >= start && pos <= end;
    /** 亮色专用作用域判据 */
    const inLightScope = (pos: number) => lightScopes().some((range) => inRange(range, pos));
    /** 任一主题作用域判据：EP 变量覆盖不允许裸挂在全局 */
    const inAnyThemeScope = (pos: number) => [...lightScopes(), ...darkScopes()].some((range) => inRange(range, pos));

    it('压暗的标签文字色限定主题作用域，且亮色分支排除 dark 变体', () => {
        expect(lightScopes().length, '需要 html:not(.dark) 作用域').toBeGreaterThan(0);

        const declared = [...css.matchAll(/--el-tag-text-color:/g)];
        expect(declared.length, '至少应有标签文字色覆盖').toBeGreaterThan(0);
        for (const match of declared) {
            expect(inAnyThemeScope(match.index), '标签文字色未限定主题作用域，会把一套主题的值带到另一套').toBe(true);
        }

        // 亮色分支里每条改文字色的规则都必须排除 dark 变体：实心底上 EP 用的是白字
        for (const match of css.matchAll(/([^{}]+)\{[^{}]*--el-tag-text-color:/g)) {
            // 用匹配位置而不是「选择器文本首次出现的位置」：同名选择器在亮暗两块里都出现时，
            // indexOf 会把暗色那条规则查到亮色块上，反过来把有意为之的暗色规则误判成外溢
            if (!inLightScope(match.index)) continue;
            const name = match[1].trim().split('\n').pop();
            expect(match[1], `「${name}」未排除 dark 标签，白字会被压成深字压深底`).toContain(':not(.el-tag--dark)');
        }

        const links = [...css.matchAll(/--el-button-text-color:\s*#2b68ad/g)];
        expect(links.length).toBeGreaterThan(0);
        for (const match of links) expect(inLightScope(match.index), '链接按钮文字色必须留在亮色作用域').toBe(true);
    });

    it('可读语义色阶明暗成对，且状态线索处不再用中阶语义色或单侧色条', () => {
        for (const name of ['success', 'warning', 'danger']) {
            const token = `--app-color-${name}-readable`;
            // 只给一档，另一套主题就会「深字压深底 / 浅字压浅底」，因此两档都必须存在
            expect(css, `${token} 缺少亮色取值`).toMatch(new RegExp(`:root\\s*\\{[^}]*${token}:\\s*#[0-9a-f]{6}`, 's'));
            expect(css, `${token} 缺少暗色取值`).toMatch(new RegExp(`html\\.dark\\s*\\{[^}]*${token}:\\s*#[0-9a-f]{6}`, 's'));
        }

        const history = readFileSync(join(process.cwd(), 'src/views/flow/components/ProcdefPolicyHistory.vue'), 'utf8');
        // 增删改三行是给人读的文字，EP 中阶语义色在白底只有 2.2~2.9:1
        expect(history).toContain('var(--app-color-success-readable)');
        expect(history).not.toMatch(/color:\s*var\(--el-color-(success|danger|warning)\)/);

        for (const file of ['src/views/ops/db/table-editor/ValidationResult.vue', 'src/views/ops/milvus/components/SystemInfo.vue']) {
            const text = readFileSync(join(process.cwd(), file), 'utf8');
            // 单侧色条是项目 UI 规范点名的反面样式，且只给一侧上颜色在暗色下更像残缺
            expect(text, `${file} 又用回了单侧色条`).not.toMatch(/border-left:\s*\d+px\s+solid/);
        }

        // dark 变体是实心底 + 白字：只能压底色（改文字色会得到深字压深底，实测 2.05:1）。
        // 逐块断言而不是「全文有一处即可」：亮暗两块退回任一块都不该溜过去
        const solidSuccess = [...css.matchAll(/\.el-tag--dark\.el-tag--success\s*\{([^}]*)\}/g)];
        expect(solidSuccess.length, '亮色与暗色两块实底 success 标签规则都得在').toBeGreaterThanOrEqual(2);
        for (const block of solidSuccess) {
            expect(block[1], '实底 success 标签未压深底色').toContain('--el-tag-bg-color: var(--app-color-success-readable)');
        }
    });

    it('弹窗动画不整体过渡 all，且提供 reduced-motion 降级', () => {
        // 不得再用 `transition: all`（会把布局属性一起动画），只允许 opacity/transform 参与
        expect(css).not.toMatch(/\.dialog-bounce[^{]*\{[^}]*transition:[^;]*\ball\b/);
        expect(css).toMatch(/transition:\s*\n?\s*opacity 0\.2s ease,\s*\n?\s*transform 0\.24s/);
        // 过冲曲线保留在 transform 上（是刻意的质感），但必须留减少动效分支
        expect(css).toMatch(/prefers-reduced-motion/);
        expect(css).toMatch(/cubic-bezier\(0\.175,\s*0\.885,\s*0\.32,\s*1\.275\)/);
    });
});
