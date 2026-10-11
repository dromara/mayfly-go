import type { SqlPlanNode } from '../types';

/**
 * 执行计划的呈现模型。
 *
 * 这里只做「把后端给的父子下标摊成可渲染的行」与「按估算行数算相对热度」，
 * 不重算任何判定：哪个节点算全表扫描、行数是估算还是真实值，都由方言与后端给出，
 * 前端按 nodeKind 文本再猜一遍就是把方言判定复制成第二套真源（且只对已知方言正确）。
 */

/** 展平后的一行计划节点 */
export interface PlanRow {
    /** 后端 Plan 数组里的下标，作稳定 key */
    index: number;
    node: SqlPlanNode;
    /** 缩进层级，根为 0 */
    depth: number;
    /** 该节点不是任何节点的子（平铺形态下全为 true） */
    root: boolean;
    /** 该节点声明了子边，但有下标越界/自引用而被忽略 */
    droppedEdge: boolean;
}

/** 该计划是否给出了父子关系。全空不等于「都是叶子」，而是该方言没给结构（只能平铺） */
export function planHasTree(nodes: SqlPlanNode[]): boolean {
    return nodes.some((node) => (node.children?.length ?? 0) > 0);
}

/** 有效子边：剔除越界与自引用下标，防止前端按下标取节点时取到 undefined */
function validChildren(nodes: SqlPlanNode[], index: number): { children: number[]; dropped: boolean } {
    const declared = nodes[index].children ?? [];
    const children = declared.filter((child) => child >= 0 && child < nodes.length && child !== index);
    return { children, dropped: children.length !== declared.length };
}

/**
 * 前序展平：从「不是任何人的孩子」的节点起走，越界边忽略，
 * 环里剩下的节点补在末尾。一个节点都不能丢——丢节点等于把计划的一部分藏起来。
 */
export function flattenPlan(nodes: SqlPlanNode[]): PlanRow[] {
    if (!nodes.length) {
        return [];
    }

    const parented = new Set<number>();
    nodes.forEach((_, index) => {
        validChildren(nodes, index).children.forEach((child) => parented.add(child));
    });

    const rows: PlanRow[] = [];
    const visited = new Set<number>();
    const walk = (index: number, depth: number) => {
        if (visited.has(index)) {
            // 只可能来自环（parented 判定已保证正常树不重复访问），跳过而非死循环
            return;
        }
        visited.add(index);
        const { children, dropped } = validChildren(nodes, index);
        rows.push({ index, node: nodes[index], depth, root: depth === 0, droppedEdge: dropped });
        children.forEach((child) => walk(child, depth + 1));
    };

    nodes.forEach((_, index) => {
        if (!parented.has(index)) {
            walk(index, 0);
        }
    });
    // 成环导致没被任何根走到的节点：仍要呈现出来，不能静默消失
    nodes.forEach((_, index) => walk(index, 0));
    return rows;
}

/** 本计划内的最大估算行数；全为 0（方言未给行数）时返回 0 */
export function maxPlanRows(nodes: SqlPlanNode[]): number {
    return nodes.reduce((max, node) => Math.max(max, node.estRows || 0), 0);
}

/**
 * 行热区强度（0~1）：按本计划内最大估算行数归一的**相对**深浅。
 *
 * 刻意不给绝对数值语义：这些行数是优化器估算，拿它算「占全表百分比」是在给估算镀金
 */
export function planHeat(node: SqlPlanNode, nodes: SqlPlanNode[]): number {
    const max = maxPlanRows(nodes);
    if (!max || !node.estRows || node.estRows <= 0) {
        return 0;
    }
    return Math.min(1, node.estRows / max);
}
