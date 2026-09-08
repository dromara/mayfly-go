/**
 * 终端多窗格布局树（递归二叉树，同 VS Code）的纯操作函数与类型
 *
 * 渲染采用扁平化方案：所有窗格（TerminalBody 实例）以 paneId 为 key 平铺渲染，
 * 布局树仅用于计算每个窗格的百分比矩形（layoutTree），保证拆分/移动时已有终端实例不被销毁重建。
 */

/** 拆分方向：left/right 为横向拆分，up/down 为纵向拆分 */
export type SplitDirection = 'left' | 'right' | 'up' | 'down';

/** 叶子节点：一个独立终端窗格（独立 WebSocket -> 独立 SSH 会话） */
export interface PaneLeaf {
    id: number;
    type: 'leaf';
}

/** 分裂节点：dir 决定子节点排列方向，ratio 为 first 节点占比 */
export interface PaneSplit {
    id: number;
    type: 'split';
    dir: 'row' | 'column';
    ratio: number;
    first: PaneNode;
    second: PaneNode;
}

export type PaneNode = PaneLeaf | PaneSplit;

/** 窗格矩形（百分比，相对于容器） */
export interface PaneRect {
    left: number;
    top: number;
    width: number;
    height: number;
}

/** 布局计算结果：窗格矩形 + 分裂节点及其矩形（用于渲染分隔条） */
export interface PaneLayout {
    panes: { id: number; rect: PaneRect }[];
    splits: { node: PaneSplit; rect: PaneRect }[];
}

/** 统计叶子（窗格）数量 */
export function countLeaves(node: PaneNode): number {
    return node.type === 'leaf' ? 1 : countLeaves(node.first) + countLeaves(node.second);
}

/**
 * 将目标叶子节点替换为 split 节点，新终端位于指定方向
 * @param genPaneId 窗格/分裂节点 id 生成器
 */
export function splitLeaf(node: PaneNode, id: number, direction: SplitDirection, genPaneId: () => number): PaneNode {
    if (node.type === 'leaf') {
        if (node.id !== id) {
            return node;
        }
        const newLeaf: PaneLeaf = { id: genPaneId(), type: 'leaf' };
        if (direction === 'left' || direction === 'up') {
            return { id: genPaneId(), type: 'split', dir: direction === 'left' ? 'row' : 'column', ratio: 0.5, first: newLeaf, second: node };
        }
        return { id: genPaneId(), type: 'split', dir: direction === 'right' ? 'row' : 'column', ratio: 0.5, first: node, second: newLeaf };
    }
    node.first = splitLeaf(node.first, id, direction, genPaneId);
    node.second = splitLeaf(node.second, id, direction, genPaneId);
    return node;
}

/**
 * 摘除指定叶子节点，其兄弟节点自动撑满；返回新子树根，若整棵树只有该叶子则返回 null
 */
export function removeLeaf(node: PaneNode, id: number): PaneNode | null {
    if (node.type === 'leaf') {
        return node.id === id ? null : node;
    }
    const first = removeLeaf(node.first, id);
    const second = removeLeaf(node.second, id);
    if (!first) {
        return second;
    }
    if (!second) {
        return first;
    }
    node.first = first;
    node.second = second;
    return node;
}

/**
 * 将叶子插入到目标叶子节点的指定方向（用于窗格移动）
 */
export function insertLeaf(node: PaneNode, targetId: number, leaf: PaneLeaf, position: SplitDirection, genPaneId: () => number): PaneNode {
    if (node.type === 'leaf') {
        if (node.id !== targetId) {
            return node;
        }
        if (position === 'left' || position === 'up') {
            return { id: genPaneId(), type: 'split', dir: position === 'left' ? 'row' : 'column', ratio: 0.5, first: leaf, second: node };
        }
        return { id: genPaneId(), type: 'split', dir: position === 'right' ? 'row' : 'column', ratio: 0.5, first: node, second: leaf };
    }
    node.first = insertLeaf(node.first, targetId, leaf, position, genPaneId);
    node.second = insertLeaf(node.second, targetId, leaf, position, genPaneId);
    return node;
}

/**
 * 遍历布局树，计算每个窗格与分裂节点的百分比矩形
 */
export function layoutTree(root: PaneNode): PaneLayout {
    const layout: PaneLayout = { panes: [], splits: [] };
    const walk = (node: PaneNode, rect: PaneRect) => {
        if (node.type === 'leaf') {
            layout.panes.push({ id: node.id, rect });
            return;
        }
        layout.splits.push({ node, rect });
        const { left, top, width, height } = rect;
        if (node.dir === 'row') {
            const firstWidth = width * node.ratio;
            walk(node.first, { left, top, width: firstWidth, height });
            walk(node.second, { left: left + firstWidth, top, width: width - firstWidth, height });
        } else {
            const firstHeight = height * node.ratio;
            walk(node.first, { left, top, width, height: firstHeight });
            walk(node.second, { left, top: top + firstHeight, width, height: height - firstHeight });
        }
    };
    walk(root, { left: 0, top: 0, width: 100, height: 100 });
    return layout;
}

/** 矩形转绝对定位样式 */
export function rectStyle(rect: PaneRect): Record<string, string> {
    return { left: `${rect.left}%`, top: `${rect.top}%`, width: `${rect.width}%`, height: `${rect.height}%` };
}

/** 根据落点在目标窗格内的相对位置，计算放置方向（距哪条边最近） */
export function getDropPosition(e: DragEvent, rect: DOMRect): SplitDirection {
    const x = (e.clientX - rect.left) / rect.width;
    const y = (e.clientY - rect.top) / rect.height;
    const dists: [SplitDirection, number][] = [
        ['left', x],
        ['right', 1 - x],
        ['up', y],
        ['down', 1 - y],
    ];
    dists.sort((a, b) => a[1] - b[1]);
    return dists[0][0];
}
