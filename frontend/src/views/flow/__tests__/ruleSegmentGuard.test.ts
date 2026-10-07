import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

/**
 * 条件组管理页的三条不变量守卫。
 *
 * 这三条都是实测修出来的行为，且失效方式都是「界面看着正常、数据其实不对」：
 * 空条件组存进去会让引用它的规则恒真或恒假；保存后不重取 schema 会让本页摘要降级成
 * 「未配置条件」且其他编辑器看不到新条件组；装载不深拷贝则「取消」不回滚、脏数据被下次保存落库。
 * 页面依赖 IOC 与接口，挂载成本高，这里按源码文本守住接线位置
 */
const source = readFileSync(join(process.cwd(), 'src/views/flow/RuleSegmentList.vue'), 'utf8');

const bodyOf = (name: string) => {
    const start = source.indexOf(`const ${name} =`);
    expect(start, `${name} 不存在`).toBeGreaterThan(-1);
    const end = source.indexOf('\n};', start);
    return source.slice(start, end);
};

describe('条件组页的保存与装载不变量', () => {
    it('空条件组与校验未通过都在提交前挡住', () => {
        const onSave = bodyOf('onSave');
        expect(onSave).toContain('if (!state.ruleNode)');
        expect(onSave).toContain('flow.ruleSegment.conditionRequired');
        // 有问题就中止：只提示不拦截会让非法条件树落库，引用它的规则运行期才报错
        expect(onSave).toMatch(/if \(issues\.value\.length > 0\)[\s\S]{0,160}return;/);
        expect(onSave).toMatch(/ruleSegmentApi\.save\.request/);
        // 校验必须在发请求之前
        expect(onSave.indexOf('ruleSegmentApi.save.request')).toBeGreaterThan(onSave.indexOf('issues.value.length'));
    });

    it('保存与删除成功后都重取 policy-schema', () => {
        // 条件组清单随 schema 一起下发，漏掉重取会让「条件内容」摘要降级、其他编辑器看不见改动
        expect(bodyOf('onSave')).toContain('await reloadPolicySchema()');
        expect(bodyOf('onDelete')).toContain('await reloadPolicySchema()');
        // 顺序：先重取再提示成功，避免提示已弹、清单还是旧的
        const onSave = bodyOf('onSave');
        expect(onSave.indexOf('reloadPolicySchema')).toBeLessThan(onSave.indexOf('Msg.saveSuccess'));
    });

    it('编辑装载必须深拷贝条件树', () => {
        const edit = bodyOf('onEdit');
        // 直接引用户行对象的 ruleNode 会让就地编辑污染列表数据，取消也不回滚
        expect(edit).toContain('cloneRuleNode(');
        expect(edit).not.toMatch(/state\.ruleNode\s*=\s*data\.ruleNode\s*;/);
    });

    it('新建时给出可用的空组而不是空壳（点了添加就得能填条件）', () => {
        const start = bodyOf('startEditing');
        expect(start).toContain('createGroupNode');
        // 建组即给一行：只给空壳会让校验当场报「条件组至少需要一个条件」，一点按钮就飘红
        expect(start).toMatch(/items\s*=\s*\[createConditionNode/);
    });
});
