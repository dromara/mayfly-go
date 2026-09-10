import { describe, expect, it } from 'vitest';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';

/**
 * 架构守护：auto-form 分层边界（UI 框架无关性契约）
 *
 * 分层规则：
 * - core 层 = 纯 .ts 文件（types 契约 / shared 数据与校验编排 / json 编译与条件求值）：
 *   禁止 import 任何 UI 框架（element-plus 等）。替换 UI 框架（如 shadcn-vue）时 core 层零改动迁移，
 *   调用点的 items/schema/rules 契约全部保持不变。
 *   注意：fields/useFieldControl.ts 虽在 fields/ 下，仍属 core 层（纯 .ts，仅依赖 types.ts）。
 * - 适配层 = ui/adapter.ts（element-plus 唯一入口，组件与类型再导出）+ 全部 .vue 组件：
 *   换框架时仅需替换 ui/adapter.ts 实现并重写 .vue 模板中的组件标签，
 *   对外 props/事件/AutoFormInstance 契约不变。
 */
const CORE_DIR = join(import.meta.dirname, '..');

/** core 层扫描排除的目录：__tests__（测试）/ ui（适配层，允许 import element-plus） */
const EXCLUDED_DIRS = new Set(['__tests__', 'ui']);

const listCoreTsFiles = (dir: string): string[] =>
    readdirSync(dir).flatMap((name) => {
        const full = join(dir, name);
        if (statSync(full).isDirectory()) {
            return EXCLUDED_DIRS.has(name) ? [] : listCoreTsFiles(full);
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
