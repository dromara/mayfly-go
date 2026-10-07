import { describe, expect, it } from 'vitest';
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';

/**
 * AutoFormDrawer/AutoFormDialog 的提交数据源门禁。
 *
 * 宿主把**内部实时表单**传给 `confirm-api`：要么直接绑 api（`xxxApi.save.request`）、要么处理函数收 form 形参、
 * 要么用 `useAutoFormModel().requireForm()` 读取接管后的内部表单。
 * 若处理函数不收形参、又去读宿主外面那份「打开时回填用的 `xxxForm.value` 快照」，用户编辑的内容会被整体丢弃：
 * 新建时提交空表单（后端报“xxx cannot be empty”），编辑时提交旧值（改了等于没改），且**不报任何错**。
 *
 * 实测就漏过一例：某新建抽屉填好名称/机器/远程地址后点确定，提交的仍是全空的默认表单。
 */

const CONFIRM_API_RE = /:confirm-api="([^"]+)"/g;

function staleSnapshotSubmits(source: string): string[] {
    const hits: string[] = [];
    for (const m of source.matchAll(CONFIRM_API_RE)) {
        const name = m[1].trim();
        // 直接把 api 交给宿主驱动（含 useApi 的执行函数）：宿主用内部表单调用，天然正确
        if (/\.request$/.test(name) || /useApi\(\)/.test(name)) continue;
        // 接管了宿主内部表单：读取点必然正确
        if (source.includes('requireForm')) continue;
        const decl = new RegExp(`(?:const|function)\\s+${name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\s*(?:=\\s*)?(?:async\\s*)?\\(([^)]*)\\)`).exec(source);
        // 收 form 形参：宿主传入的是内部实时表单
        if (decl && decl[1].trim()) continue;
        const body = new RegExp(`(?:const|function)\\s+${name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\b[\\s\\S]{0,400}?(?:\\n\\};|\\n\\})`).exec(source);
        // 不收形参、又读外面那份快照 —— 提交内容必然与用户编辑脱节
        if (body && /\b\w*[Ff]orm\w*\.value/.test(body[0])) {
            hits.push(`:confirm-api="${name}" 的提交读外部 *.value 快照，未接管宿主内部表单`);
        }
    }
    return hits;
}

function listVueFiles(dir: string, into: string[] = []): string[] {
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

describe('AutoForm 宿主提交数据源', () => {
    const files = listVueFiles(join(process.cwd(), 'src'));

    it('扫到了足量 confirm-api 用法（防止判据本身失效导致空跑）', () => {
        const total = files.reduce((sum, file) => sum + [...readFileSync(file, 'utf8').matchAll(CONFIRM_API_RE)].length, 0);
        expect(total).toBeGreaterThanOrEqual(25);
    });

    it('判据能识别出“提交读外部快照”（内置夹具反向验证）', () => {
        const bad = `<template><auto-form-drawer :confirm-api="onSubmitForm" /></template>
<script lang="ts" setup>
const editForm = ref(null);
const onSubmitForm = async () => {
    await api.save.request(editForm.value);
};
</script>`;
        expect(staleSnapshotSubmits(bad)).toHaveLength(1);
        // 三种合法写法都不该误报
        expect(staleSnapshotSubmits(`<auto-form-drawer :confirm-api="machineApi.save.request" />`)).toEqual([]);
        expect(staleSnapshotSubmits(bad.replace('const onSubmitForm = async () => {', 'const onSubmitForm = async (form) => {'))).toEqual([]);
        expect(staleSnapshotSubmits(bad.replace('const onSubmitForm', 'const requireForm = 1;\nconst onSubmitForm'))).toEqual([]);
    });

    it('全仓不存在“提交丢弃用户编辑、发送外部快照”的表单宿主', () => {
        const problems: string[] = [];
        for (const file of files) {
            for (const hit of staleSnapshotSubmits(readFileSync(file, 'utf8'))) {
                problems.push(`${file.replace(process.cwd() + '/', '')}: ${hit}`);
            }
        }
        expect(problems, '以下表单提交会静默丢弃用户编辑:\n' + problems.join('\n')).toEqual([]);
    });
});
