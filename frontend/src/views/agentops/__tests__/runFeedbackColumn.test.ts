import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

/**
 * 「反馈」在运行列表的接线守卫（列 + 筛选项 + 类型 + 文案四处必须同源）。
 *
 * 存在理由：反馈是学习闭环的人工判定（点赞/点踩 + 备注，备注还会沉淀成记忆），后端一直把它
 * 存成独立列，注释里更写着「可按此筛选统计」——**但列表既不显示、也没有该筛选维度**：
 * 看不出哪条运行已给过反馈（于是会重复反馈），也拿不出「所有点踩过的运行」这个最该复盘的样本集。
 * 声明与实现分叉时编译、类型、单测全绿，只有对着界面才看得出来，所以钉成源码守卫。
 */
const read = (rel: string) => readFileSync(resolve(__dirname, '..', rel), 'utf8');

const runList = read('RunList.vue');
const enums = read('enums.ts');
const types = read('types.ts');

const readI18n = (lang: string) => readFileSync(resolve(__dirname, '..', '..', '..', 'i18n', lang, 'agentops.ts'), 'utf8');

describe('运行列表的反馈列与反馈筛选', () => {
    it('反馈既是一列也是筛选维度，且都取同一个枚举', () => {
        expect(runList).toContain("TableColumn.new('feedback', 'agentops.runFeedback').typeTag(RunFeedbackEnum)");
        expect(runList).toContain("SearchItem.select('feedback', 'agentops.runFeedback').withEnum(RunFeedbackEnum)");
        expect(runList).toContain('RunFeedbackEnum');
    });

    it('枚举值集合与后端 entity.FeedbackUp/FeedbackDown 完全一致，且不含空串哨兵', () => {
        const block = enums.match(/export const RunFeedbackEnum = \{[\s\S]*?\};/);
        if (!block) {
            throw new Error('未找到 RunFeedbackEnum 定义（改名或删除必须同步本守卫）');
        }
        const values = [...block[0].matchAll(/EnumValue\.of\('([^']*)'/g)].map((m) => m[1]);
        // 值必须逐个来自后端声明的两个取值：多一个空串就等于往库里塞哨兵值
        expect(new Set(values)).toEqual(new Set(['up', 'down']));
    });

    it('列表视图类型带 feedback 字段（后端 VO 有值但前端类型缺失会被静默读成 undefined）', () => {
        expect(types).toMatch(/feedback\?: string;/);
    });

    it('中英文两份文案都齐（列名 + 两个枚举标签），缺一个界面就直出裸键', () => {
        for (const lang of ['zh-cn', 'en']) {
            const dict = readI18n(lang);
            for (const key of ['runFeedback', 'feedbackUp', 'feedbackDown']) {
                expect(dict, `${lang} 缺少 agentops.${key}`).toContain(`${key}:`);
            }
        }
    });
});
