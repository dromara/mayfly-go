/**
 * 「保存流程定义」前必须先过策略编辑器的校验。
 *
 * 策略编辑器已经实时算出不合法项并显示在问题面板里，但如果提交路径不看它，
 * 用户仍要发一次请求、等后端拒绝才知道哪里没填对——这类断线不会报错，只会让
 * 前面的校验白做，所以按源码接线断言守住（与 layering.test.ts 同一手法：
 * onConfirm 依赖弹层宿主与一堆 api，单元里不便整体挂载）
 */
import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

const source = readFileSync(join(import.meta.dirname, '../ProcdefEdit.vue'), 'utf-8');

describe('流程定义提交的接线', () => {
    it('提交前先向策略编辑器要校验结果', () => {
        expect(source).toMatch(/policyEditorRef\.value\?\.validate\(\)/);
    });

    it('有问题时按 confirmApi 契约抛错中止（正常返回会被宿主当成提交成功）', () => {
        const confirmBlock = source.slice(source.indexOf('const onConfirm'), source.indexOf('const onConfirm') + 600);
        expect(confirmBlock).toContain('if (issue)');
        expect(confirmBlock).toMatch(/throw new Error\(issue\)/);
        // 拦截必须发生在真正发请求之前
        expect(confirmBlock.indexOf('if (issue)')).toBeLessThan(confirmBlock.indexOf('await saveFlowDefExec()'));
    });

    it('生效资源的可治理类型只认 schema 下发的清单，不留第二份真源', () => {
        expect(source).toMatch(/governTagTypesOf\(policySchema\.value\?\.governPaths\)/);
        // 保底清单就是第二份「谁能被治理」，写死过一次就把机器场景关在了门外
        expect(source).not.toMatch(/TagResourceTypePath\.Db\s*,\s*TagResourceTypeEnum\.Redis\.value\s*\]/);
    });
});
