import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

/**
 * 任务/触发器两个编辑表单的「结果通知」接线守卫。
 *
 * 存在理由（真机复现过的缺陷）：通知模板字段的显示条件是「notifyType ≠ 不通知」，而新建态
 * 若不给 notifyType 预置值，undefined ≠ 0 会判真 → 通知模板凭空变成必填，用户无法在不配通知
 * 的情况下保存。后端 notifyType 零值语义就是 None，因此「未填」与「不通知」必须同义。
 *
 * 这类「同一字段在两个表单里默认值不一致」的漂移，编译、类型、界面都不会红，只能钉住源码。
 */
const readForm = (name: string) => readFileSync(resolve(__dirname, '..', name), 'utf8');

describe('结果通知字段的两个表单同源', () => {
    it.each(['TaskEdit.vue', 'TriggerEdit.vue'])('%s 新建态必须显式预置 notifyType', (file) => {
        const source = readForm(file);
        // 取 editData 的「新建态」返回分支（props.data 为空时的那段对象字面量）
        const createBranch = source.slice(source.lastIndexOf('return {'));
        expect(createBranch, `${file} 新建默认值缺少 notifyType`).toMatch(/notifyType\s*:/);
    });

    it('触发器的通知模板显示条件必须把「未填」与「不通知」判为同义', () => {
        const source = readForm('TriggerEdit.vue');
        const whenLine = source.split('\n').find((l) => l.includes('prop: ') === false && l.includes('notifyType') && l.includes('taskId'));
        expect(whenLine, '未找到通知模板的 when 条件').toBeDefined();
        // 缺 != null 判空时，undefined 会被判成「要通知」而把模板拖成必填
        expect(whenLine).toContain('notifyType != null');
    });
});
