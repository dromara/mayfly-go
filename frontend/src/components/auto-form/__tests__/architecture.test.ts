import { describe, expect, it } from 'vitest';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';

/**
 * 架构守护：auto-form 分层边界（UI 框架无关性契约）
 *
 * 分层规则：
 * - core 层 = 目录下全部纯 .ts 文件（types 契约 / shared 数据与校验编排 / json 编译与条件求值）：
 *   禁止 import 任何 UI 框架（element-plus 等）。替换 UI 框架（如 shadcn-vue）时 core 层零改动迁移，
 *   调用点的 items/schema/rules 契约全部保持不变。
 * - 适配层 = 全部 .vue 组件（控件渲染 AutoFormControl、表单壳 AutoForm、栅格 AutoFormFieldCol、
 *   弹层宿主 AutoFormDialog/AutoFormDrawer、Schema 编辑工具）：当前为 element-plus 实现，
 *   换框架时仅需重写该层模板与控件分支，对外 props/事件/AutoFormInstance 契约不变。
 */
const CORE_DIR = join(import.meta.dirname, '..');

const listCoreTsFiles = (dir: string): string[] =>
    readdirSync(dir).flatMap((name) => {
        const full = join(dir, name);
        if (statSync(full).isDirectory()) {
            return name === '__tests__' ? [] : listCoreTsFiles(full);
        }
        return name.endsWith('.ts') ? [full] : [];
    });

describe('auto-form 架构守护（core 层 UI 框架无关）', () => {
    it('全部纯 .ts（core 层）禁止 import 任何 UI 框架', () => {
        const violations = listCoreTsFiles(CORE_DIR)
            .map((file) => ({ file, source: readFileSync(file, 'utf-8') }))
            .filter(({ source }) => /from\s+['"]element-plus['"]/.test(source))
            .map(({ file }) => file);
        expect(violations).toEqual([]);
    });

    it('core 层禁止出现 el-* 组件标签（模板化 UI 依赖）', () => {
        const violations = listCoreTsFiles(CORE_DIR)
            .map((file) => ({ file, source: readFileSync(file, 'utf-8') }))
            .filter(({ source }) => /<el-[a-z-]+/.test(source))
            .map(({ file }) => file);
        expect(violations).toEqual([]);
    });
});
