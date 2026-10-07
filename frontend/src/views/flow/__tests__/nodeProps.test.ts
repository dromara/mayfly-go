/**
 * 流程图属性抽屉必须深拷贝节点属性。
 *
 * 属性里带条件树这类嵌套结构：与画布节点共享引用时，编辑器就地改动（切换「且/或」、
 * 增删条件行）会立刻污染节点数据，点「取消」回不去，之后一次保存流程还会把这些
 * 已取消的改动静默写进流程定义
 */
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';

import { cloneNodeProperties } from '../components/flowdesign/node/nodeProps';

type Condition = { logic: string; items: unknown[] };

describe('节点属性拷贝', () => {
    it('嵌套对象与数组都不共享引用', () => {
        const properties = {
            condition: { logic: 'all', items: [{ kind: 'condition', field: 'sql', op: 'contains', value: 'x' }] },
            candidates: ['1'] as string[],
        };
        const copy = cloneNodeProperties(properties);
        const copyCondition = copy.condition as Condition;

        copyCondition.logic = 'any';
        copyCondition.items.push({ kind: 'segment', ref: 'other' });
        (copy.candidates as string[]).push('2');

        expect(properties.condition.logic).toBe('all');
        expect(properties.condition.items).toHaveLength(1);
        expect(properties.candidates).toEqual(['1']);
    });

    it('无属性时给出空对象而不是 undefined', () => {
        expect(cloneNodeProperties(undefined)).toEqual({});
        expect(cloneNodeProperties(null)).toEqual({});
    });

    it('保留普通字段取值', () => {
        const copy = cloneNodeProperties({ completionCondition: { kind: 'condition', field: 'nrOfCompletedRate', op: 'gte', value: 1 } });
        expect(copy.completionCondition).toEqual({ kind: 'condition', field: 'nrOfCompletedRate', op: 'gte', value: 1 });
    });
});

// 光测工具函数不够：抽屉若忘了调用它（退回 { ...n.properties } 浅拷贝），
// 上面的用例照样全绿，而「取消不回滚」的污染会回来。这里按源码接线断言守住
describe('属性抽屉的接线', () => {
    const drawer = readFileSync(join(import.meta.dirname, '../components/flowdesign/node/PropSettingDrawer.vue'), 'utf-8');

    it('打开抽屉时用深拷贝装载节点属性', () => {
        expect(drawer).toMatch(/form\.value = cloneNodeProperties\(n\.properties\)/);
        expect(drawer).not.toMatch(/form\.value = \{ \.\.\.n\.properties \}/);
    });
});
