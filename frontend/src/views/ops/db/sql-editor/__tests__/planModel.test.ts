import { describe, expect, it } from 'vitest';
import type { SqlPlanNode } from '../../types';
import { flattenPlan, maxPlanRows, planHasTree, planHeat } from '../planModel';

/**
 * 执行计划呈现模型的守卫
 *
 * 存在理由：后端给的是「节点数组 + 子节点下标」，前端按下标取节点画层级。
 * 下标越界、自引用、成环这三类畸形输入只要出现一次，渲染就会拿 undefined 取属性；
 * 而「为了不死循环顺手丢掉几个节点」更危险——那是把计划的一部分藏起来。
 */

const node = (overrides: Partial<SqlPlanNode> = {}): SqlPlanNode => ({ nodeKind: 'Scan', estRows: 0, ...overrides });

describe('planModel 计划展平', () => {
    it('planHasTree：全空 children 不等于都是叶子，而是该方言没给结构', () => {
        expect(planHasTree([node(), node()])).toBe(false);
        expect(planHasTree([node({ children: [1] }), node()])).toBe(true);
        // 声明了但全是无效下标，仍算「没给出结构」
        expect(planHasTree([node({ children: [] }), node()])).toBe(false);
    });

    it('树按前序展平并带层级，父一定早于子', () => {
        const nodes = [
            node({ nodeKind: 'Join', children: [1, 2] }),
            node({ nodeKind: 'Seq Scan' }),
            node({ nodeKind: 'Index Scan', children: [3] }),
            node({ nodeKind: 'Materialize' }),
        ];
        const rows = flattenPlan(nodes);
        expect(rows.map((r) => [r.index, r.depth])).toEqual([
            [0, 0],
            [1, 1],
            [2, 1],
            [3, 2],
        ]);
        expect(rows[0].root).toBe(true);
        expect(rows[1].root).toBe(false);
    });

    it('平铺形态保持库侧输出顺序，全部记为根', () => {
        const rows = flattenPlan([node({ nodeKind: 'ALL' }), node({ nodeKind: 'ref' }), node({ nodeKind: 'const' })]);
        expect(rows.map((r) => r.node.nodeKind)).toEqual(['ALL', 'ref', 'const']);
        expect(rows.every((r) => r.depth === 0 && r.root)).toBe(true);
    });

    it('越界与自引用下标：忽略该边并标记，但节点一个都不丢', () => {
        const nodes = [node({ nodeKind: 'root', children: [9, 0, 1] }), node({ nodeKind: 'child' })];
        const rows = flattenPlan(nodes);
        expect(rows.map((r) => r.index).sort()).toEqual([0, 1]);
        const root = rows.find((r) => r.index === 0);
        expect(root?.droppedEdge).toBe(true);
        // 自引用边被剔除后只剩 [1]，子节点层级为 1
        expect(rows.find((r) => r.index === 1)?.depth).toBe(1);
    });

    it('成环不死循环，环里的节点仍然呈现', () => {
        const nodes = [node({ nodeKind: 'a', children: [1] }), node({ nodeKind: 'b', children: [0] })];
        const rows = flattenPlan(nodes);
        // 两个节点互为父子 → 没有根，靠补走兜底，仍要全部出现
        expect(rows.map((r) => r.index).sort()).toEqual([0, 1]);
    });

    it('空计划返回空数组，不抛', () => {
        expect(flattenPlan([])).toEqual([]);
    });
});

describe('planModel 行热区', () => {
    it('按本计划内最大估算行数归一，只表达相对深浅', () => {
        const nodes = [node({ estRows: 100 }), node({ estRows: 25 })];
        expect(planHeat(nodes[0], nodes)).toBe(1);
        expect(planHeat(nodes[1], nodes)).toBeCloseTo(0.25);
    });

    it('方言未给行数（全 0）时热区为 0，不靠猜上色', () => {
        const nodes = [node(), node()];
        expect(maxPlanRows(nodes)).toBe(0);
        expect(planHeat(nodes[0], nodes)).toBe(0);
    });
});
