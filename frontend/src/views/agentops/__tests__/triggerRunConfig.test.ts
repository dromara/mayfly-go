import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

/**
 * 触发器「关联任务」的显示接线守卫。
 *
 * 存在理由（真机走出来的）：触发器列表的「运行配置」列原来渲染的是 `关联任务 #18`——一个在
 * 任务列表里根本找不到对应物的数字；删掉任务后打开编辑抽屉，「关联任务」下拉也因无匹配 option
 * 而退化成直接显示裸值。两处都不是报错，只是把「我绑的是哪个任务」这个信息丢给了用户。
 *
 * 「model 值对 ≠ UI 渲染对」：接口回传 taskId 是对的，但界面对它的渲染方式错了照样是不可用；
 * 这类只有真机点开才看得见的问题，修完必须把接线钉住，否则下一次重构又会退回去。
 */
const readSource = (name: string) => readFileSync(resolve(__dirname, '..', name), 'utf8');

const readI18n = (lang: string) => readFileSync(resolve(__dirname, '..', '..', '..', 'i18n', lang, 'agentops.ts'), 'utf8');

/** 截取片段源码；匹配不到直接抛错，避免「没匹配到 → 断言恒真」 */
function slice(source: string, re: RegExp, what: string): string {
    const m = source.match(re);
    if (!m) {
        throw new Error(`未找到${what}（改名/删除必须同步本守卫）`);
    }
    return m[0];
}

it('列表的关联任务分支显示任务名，任务已删除时给占位文案而不是裸 id', () => {
    const slot = slice(readSource('TriggerList.vue'), /<template #runConfig="[\s\S]*?<\/template>/, '运行配置列插槽');

    expect(slot, '关联任务分支必须消费后端回传的 taskName').toContain('data.taskName');
    expect(slot, 'taskName 为空时必须落到「已删除的任务」文案').toContain("$t('agentops.bindTaskMissing'");
    // 裸 id 直显的形态是 `#{{ data.taskId }}`；新写法里 taskId 只出现在文案参数中
    expect(slot, '不得再把关联任务渲染成裸 #id').not.toMatch(/#\{\{ data\.taskId \}\}/);
    // 保留「关联任务」前缀：该列同时承载内联档位，去掉前缀就看不出这一行走的是哪种模式
    expect(slot, '关联任务分支必须保留模式前缀').toContain("$t('agentops.bindTask')");
});

it('编辑抽屉必须为查不到的绑定任务补一条带文案的选项', () => {
    const fn = slice(readSource('TriggerEdit.vue'), /const loadTaskOptions = async \(\) => \{[\s\S]*?\n\};/, 'loadTaskOptions');

    expect(fn, '必须读当前触发器的 taskId（而不是表单值）才能判定是否悬空').toContain('props.data?.taskId');
    expect(fn, '绑定任务不在选项里时必须补一条带文案的选项').toMatch(/options\.push\(\{ value: boundTaskId, label: t\('agentops\.bindTaskMissing'/);
});

describe('bindTaskMissing 文案中英成对登记', () => {
    for (const lang of ['zh-cn', 'en']) {
        it(`${lang} 有该键`, () => {
            expect(readI18n(lang), `${lang}/agentops.ts 缺少 bindTaskMissing`).toContain('bindTaskMissing:');
        });
    }
});
