/**
 * 本地组件注册守卫（全仓 SFC）
 *
 * 本项目没有全局组件注册（main.ts 只注册了 SvgIcon 与 Element Plus 图标），组件必须在
 * 使用方 SFC 的 `<script setup>` 里导入才能被模板解析。漏导入时：
 *   - 模板把 `<auto-form>` 当成未知原生元素渲染，**字段整块静默消失**（用户看到空弹层）；
 *   - vue-tsc / eslint / vite build 全部放行；
 *   - 连控制台告警都没有——main.ts 里 `app.config.warnHandler = () => null` 静音了
 *     「Failed to resolve component」。
 *
 * 机器文件「新建文件或目录」弹层没有名称输入框就是这个成因：只写了
 * `import type { AutoFormItem }`（类型导入不产生组件绑定）。
 *
 * 判据按 src/components/**\/index.ts 的导出名推导，新增组件零登记即被覆盖。
 */
import { describe, expect, it } from 'vitest';
import * as fs from 'node:fs';
import * as path from 'node:path';

const SRC_ROOT = path.resolve(__dirname, '..');

/** 全局注册的组件，模板里无需导入 */
const GLOBAL_COMPONENTS = new Set(['SvgIcon', 'ElWatermark']);

/** 收集本项目自有组件名：只认 `export { default as Xxx }` 这种桶文件导出 */
function collectOwnComponents(): Set<string> {
    const names = new Set<string>();
    const walk = (dir: string) => {
        for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
            const full = path.join(dir, entry.name);
            if (entry.isDirectory()) {
                // components/ui 是 shadcn-vue 生成的第三方组件，由使用方显式具名导入，不参与本判据
                if (entry.name === 'ui' || entry.name === 'node_modules') continue;
                walk(full);
            } else if (entry.name.endsWith('.ts') && !entry.name.endsWith('.d.ts')) {
                const src = fs.readFileSync(full, 'utf-8');
                for (const match of src.matchAll(/export\s*\{\s*default\s+as\s+([A-Z][A-Za-z0-9]*)/g)) {
                    names.add(match[1]);
                }
            }
        }
    };
    walk(path.join(SRC_ROOT, 'components'));
    return names;
}

const OWN_COMPONENTS = collectOwnComponents();

/** 组件名 → 模板里允许的标签写法（PascalCase 与 kebab-case） */
const tagToComponent = new Map<string, string>();
for (const name of OWN_COMPONENTS) {
    tagToComponent.set(name, name);
    tagToComponent.set(
        name.replace(/([a-z0-9])([A-Z])/g, '$1-$2').toLowerCase(),
        name
    );
}

function listSfc(dir: string): string[] {
    return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
        const full = path.join(dir, entry.name);
        if (entry.isDirectory()) {
            return entry.name === 'node_modules' ? [] : listSfc(full);
        }
        return entry.name.endsWith('.vue') ? [full] : [];
    });
}

/** 取 `<script>` 块（模板部分不参与绑定判定） */
function scriptOf(src: string): string {
    const scripts = [...src.matchAll(/<script\b[^>]*>([\s\S]*?)<\/script>/g)].map((m) => m[1]);
    return scripts.join('\n');
}

/**
 * 模板部分：剥掉 script 块与 HTML 注释。
 *
 * 不能用「截到第一个 </template>」：模板里有嵌套的 `<template #header>` 等具名插槽，
 * 第一个闭合标签就会把后面的组件标签剪掉，造成漏检。
 */
function templateOf(src: string): string {
    return src.replace(/<script\b[\s\S]*?<\/script>/g, '').replace(/<!--[\s\S]*?-->/g, '');
}

describe('模板里用到的自有组件必须在 SFC 内导入', () => {
    it('桶文件确实扫到了组件名（防止判据失效导致空跑）', () => {
        expect(OWN_COMPONENTS.size).toBeGreaterThan(10);
        expect(OWN_COMPONENTS.has('AutoForm')).toBe(true);
    });

    it('每个组件标签都有对应的导入绑定或全局注册', () => {
        const violations: string[] = [];

        for (const file of listSfc(SRC_ROOT)) {
            const src = fs.readFileSync(file, 'utf-8');
            const script = scriptOf(src);
            const rel = path.relative(SRC_ROOT, file);

            for (const [, tag] of templateOf(src).matchAll(/<([A-Za-z][A-Za-z0-9]*(?:-[a-z0-9]+)*)[\s/>]/g)) {
                const component = tagToComponent.get(tag);
                if (!component || GLOBAL_COMPONENTS.has(component)) continue;

                // 绑定来源：具名/默认导入、异步组件常量、局部注册对象
                const bound = new RegExp(
                    `(import[^\\n]*\\b${component}\\b[^\\n]*from|\\bconst\\s+${component}\\s*=|\\b${component}\\s*:\\s*${component}\\b)`
                ).test(script);
                if (!bound) {
                    violations.push(`${rel}: <${tag}> 未导入 ${component}`);
                }
            }
        }

        expect(violations).toEqual([]);
    });
});
