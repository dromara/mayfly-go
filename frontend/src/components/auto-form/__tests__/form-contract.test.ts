import { describe, expect, it } from 'vitest';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';

/**
 * 架构守护：表单类型契约（AutoForm 宿主与页面之间的类型收敛点唯一化）
 *
 * 背景：AutoForm 以 `AutoFormData = Record<string, any>` 在宿主（Dialog/Drawer）、字段配置回调、
 * 提交动作之间传递表单数据。若页面按 `AutoFormItem[]` 声明字段、又用 `ref<AutoFormData>` 暂存宿主
 * 内部表单，则每个读取点都要 `rawForm as XxxForm` 断言一次，类型在整条链上反复丢失（历史上散落 80+ 处）。
 *
 * 单一真源：类型只在两个边界入口收敛一次——
 * - components/auto-form/types.ts 的 defineFormItems<TForm>：字段配置回调拿到业务表单类型；
 * - hooks/useAutoFormModel.ts 的 useAutoFormModel<TForm>：@opened 接管宿主内部表单。
 * 因此页面侧不允许再出现「表单袋断言为业务表单」的写法，也不允许把业务表单往袋上拓宽。
 */
const VIEWS_DIR = join(import.meta.dirname, '../../../views');

const listVueFiles = (dir: string): string[] =>
    readdirSync(dir).flatMap((name) => {
        const full = join(dir, name);
        return statSync(full).isDirectory() ? listVueFiles(full) : name.endsWith('.vue') ? [full] : [];
    });

/** 从源码中挑出匹配指定正则的行（附带文件路径，便于定位） */
const findLines = (files: string[], pattern: RegExp): string[] =>
    files.flatMap((file) =>
        readFileSync(file, 'utf-8')
            .split('\n')
            .map((line, index) => ({ line, index }))
            .filter(({ line }) => pattern.test(line))
            .map(({ line, index }) => `${file.replace(VIEWS_DIR, 'views')}:${index + 1}: ${line.trim()}`)
    );

describe('表单类型契约守护（views 下禁止散落表单袋断言）', () => {
    const vueFiles = listVueFiles(VIEWS_DIR);

    it('禁止把表单袋拓宽断言为 AutoFormData（Record<string, any> 天然兼容，断言不产生约束）', () => {
        expect(findLines(vueFiles, /\bas\s+AutoFormData\b/)).toEqual([]);
    });

    it('禁止在页面里以表单袋声明变量/形参/泛型实参（ref<AutoFormData>、computed<AutoFormData>、: AutoFormData 等）', () => {
        // 字段回调交给 defineFormItems，内部表单交给 useAutoFormModel，回填数据交给业务表单（或其 Partial）
        expect(findLines(vueFiles, /[:<]\s*AutoFormData\b/)).toEqual([]);
    });

    it('禁止把 AutoForm 表单袋断言回业务表单类型（as XxxForm 形态）', () => {
        // 允许：`Object as PropType<XxxForm | null>`（props 契约声明）；`[] as XxxForm[]`（空数组形状种子，不是袋断言）
        const violations = findLines(vueFiles, /\bas\s+[A-Za-z][\w.]*Form(?!\[\])\b(?!\s*\[)/).filter((text) => !text.includes('PropType<'));
        expect(violations).toEqual([]);
    });

    it('禁止用本地 type FormData 承接表单形状（遮蔽 DOM 全局 FormData，且与后端 form 无对照）', () => {
        expect(findLines(vueFiles, /^type\s+FormData\s*=/m)).toEqual([]);
    });
});
