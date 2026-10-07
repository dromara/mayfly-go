import { describe, expect, it } from 'vitest';
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';

/**
 * `<script setup>` 组件标签与同名 state 变量的撞名门禁。
 *
 * Vue 编译器把 kebab 标签 `<a-b-c>` 解析成组件时，**先按 camelCase `aBC` 找 setup 绑定**，
 * 找到就直接用它；只有找不到才回退到 PascalCase `ABC`。
 * 因此当某个对话框的状态变量正好叫 `aBC`、而组件常量叫 `ABC` 时，标签会被解析成那个
 * 普通响应式对象，Vue 把非组件对象当组件挂载 —— 渲染成空注释、**不报任何错**，
 * 于是 `visible` 怎么改都不开关，界面表现就是「按钮点了完全没反应」。
 *
 * 实测就漏过两个：状态变量 `batchRunDialog` / `batchFileDialog` 与组件常量
 * `BatchRunDialog` / `BatchFileDialog` 撞名，导致批量执行、批量文件两个入口全成死按钮。
 * vue-tsc / eslint / build 全都发现不了。
 */

const SCRIPT_SETUP_RE = /<script[^>]*\bsetup\b[^>]*>([\s\S]*?)<\/script>/;
const DECL_RE = /\b(?:const|let|var|function)\s+([A-Za-z_$][\w$]*)/g;
const KEBAB_TAG_RE = /<([a-z][a-z0-9]*(?:-[a-z0-9]+)+)(?=[\s/>])/g;

const toCamel = (tag: string) => {
    const parts = tag.split('-');
    return parts[0] + parts.slice(1).map((p) => p[0].toUpperCase() + p.slice(1)).join('');
};
const toPascal = (tag: string) => tag.split('-').map((p) => p[0].toUpperCase() + p.slice(1)).join('');

/** 返回该文件里「kebab 标签被同名 state 变量抢走解析」的冲突清单 */
function tagBindingCollisions(source: string): string[] {
    const scriptMatch = source.match(SCRIPT_SETUP_RE);
    if (!scriptMatch) return [];
    const template = source.slice(0, scriptMatch.index);
    const bindings = new Set([...scriptMatch[1].matchAll(DECL_RE)].map((m) => m[1]));
    const hits: string[] = [];
    for (const tag of new Set([...template.matchAll(KEBAB_TAG_RE)].map((m) => m[1]))) {
        const camel = toCamel(tag);
        const pascal = toPascal(tag);
        // 只有「PascalCase 常量确实存在（本意是挂组件）」才算冲突；纯原生/EP 标签两边都没声明，跳过
        if (camel !== pascal && bindings.has(camel) && bindings.has(pascal)) {
            hits.push(`<${tag}> 解析到 state '${camel}' 而非组件 '${pascal}'`);
        }
    }
    return hits;
}

function listVueFiles(dir: string, into: string[] = []): string[] {
    if (!readdirSyncExists(dir)) return into;
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
        const full = join(dir, entry.name);
        if (entry.isDirectory()) {
            if (entry.name !== 'node_modules' && entry.name !== '__tests__') listVueFiles(full, into);
        } else if (/\.vue$/.test(entry.name)) {
            into.push(full);
        }
    }
    return into;
}

function readdirSyncExists(dir: string): boolean {
    try {
        readdirSync(dir);
        return true;
    } catch {
        return false;
    }
}

describe('<script setup> 组件标签撞名', () => {
    const files = listVueFiles(join(process.cwd(), 'src'));

    it('扫到了足量 .vue 与 kebab 标签（防止判据本身失效导致空跑）', () => {
        expect(files.length).toBeGreaterThan(100);
        const tagTotal = files.reduce(
            (sum, file) => sum + new Set([...readFileSync(file, 'utf8').matchAll(KEBAB_TAG_RE)].map((m) => m[1])).size,
            0,
        );
        expect(tagTotal).toBeGreaterThan(200);
    });

    it('判据能识别出冲突（内置夹具反向验证，避免检测器腐化后静默全绿）', () => {
        const fixture = `<template><foo-bar-dialog v-model:visible="fooBarDialog.visible" /></template>
<script lang="ts" setup>
const FooBarDialog = () => null;
const fooBarDialog = ref({ visible: false });
</script>`;
        expect(tagBindingCollisions(fixture)).toHaveLength(1);
        // PascalCase 未声明时不该误报（原生/EP 标签场景）
        expect(tagBindingCollisions(`<template><el-button /></template><script setup>const elButton = 1;</script>`)).toEqual([]);
    });

    it('全仓不存在标签被同名 state 抢走解析的组件', () => {
        const problems: string[] = [];
        for (const file of files) {
            for (const hit of tagBindingCollisions(readFileSync(file, 'utf8'))) {
                problems.push(`${file.replace(process.cwd() + '/', '')}: ${hit}`);
            }
        }
        expect(problems, '以下功能入口会静默失效（点击无反应、无任何报错）:\n' + problems.join('\n')).toEqual([]);
    });
});
