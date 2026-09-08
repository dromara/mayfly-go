import { describe, expect, it } from 'vitest';
import { countLeaves, getDropPosition, insertLeaf, layoutTree, removeLeaf, splitLeaf, type PaneLeaf, type PaneNode, type PaneSplit } from '../paneTree';

/** 固定 id 生成器：100 起自增，与测试中手工构造的节点 id 不冲突 */
const genPaneId = (() => {
    let i = 100;
    return () => ++i;
})();

const leaf = (id: number): PaneLeaf => ({ id, type: 'leaf' });

/** 断言为 split 节点并收窄类型 */
const expectSplit = (node: PaneNode): PaneSplit => {
    expect(node.type).toBe('split');
    return node as PaneSplit;
};

describe('countLeaves', () => {
    it('单叶子为 1', () => {
        expect(countLeaves(leaf(1))).toBe(1);
    });

    it('嵌套树正确累计', () => {
        const root = splitLeaf(splitLeaf(leaf(1), 1, 'right', genPaneId), 1, 'down', genPaneId);
        expect(countLeaves(root)).toBe(3);
    });
});

describe('splitLeaf', () => {
    it('向右拆分：原窗格在左，新窗格在右，方向为 row', () => {
        const root = expectSplit(splitLeaf(leaf(1), 1, 'right', genPaneId));
        expect(root.dir).toBe('row');
        expect(root.first).toEqual(leaf(1));
        expect(root.second.type).toBe('leaf');
        expect((root.second as PaneLeaf).id).not.toBe(1);
        expect(root.ratio).toBe(0.5);
    });

    it('向左拆分：新窗格在左，原窗格在右', () => {
        const root = expectSplit(splitLeaf(leaf(1), 1, 'left', genPaneId));
        expect(root.dir).toBe('row');
        expect(root.first.type).toBe('leaf');
        expect((root.first as PaneLeaf).id).not.toBe(1);
        expect(root.second).toEqual(leaf(1));
    });

    it('向上/向下拆分方向为 column', () => {
        expect(expectSplit(splitLeaf(leaf(1), 1, 'up', genPaneId)).dir).toBe('column');
        expect(expectSplit(splitLeaf(leaf(1), 1, 'down', genPaneId)).dir).toBe('column');
    });

    it('目标窗格不存在时原树原样返回', () => {
        const origin = leaf(1);
        expect(splitLeaf(origin, 999, 'right', genPaneId)).toBe(origin);
    });

    it('可对深层叶子拆分', () => {
        const one = expectSplit(splitLeaf(leaf(1), 1, 'right', genPaneId));
        const two = expectSplit(splitLeaf(one, (one.second as PaneLeaf).id, 'down', genPaneId));
        expect(countLeaves(two)).toBe(3);
    });
});

describe('removeLeaf', () => {
    it('整棵树只有该叶子时返回 null', () => {
        expect(removeLeaf(leaf(1), 1)).toBeNull();
    });

    it('移除一端叶子后兄弟节点提升为根', () => {
        const root = expectSplit(splitLeaf(leaf(1), 1, 'right', genPaneId));
        const newRoot = removeLeaf(root, 1);
        expect(newRoot).toEqual(root.second);
    });

    it('移除深层叶子后对应分裂节点被回收', () => {
        // root = split(1, split(2, 3))
        const l1 = expectSplit(splitLeaf(leaf(1), 1, 'right', genPaneId));
        const l2 = expectSplit(splitLeaf(l1, (l1.second as PaneLeaf).id, 'right', genPaneId));
        const newRoot = removeLeaf(l2, 1);
        // 1 被移除后，其兄弟子树整体提升
        expect(newRoot).toEqual(l1.second);
        expect(countLeaves(newRoot!)).toBe(2);
    });

    it('目标不存在时原树原样返回', () => {
        const origin = leaf(1);
        expect(removeLeaf(origin, 999)).toBe(origin);
    });
});

describe('insertLeaf', () => {
    it('插入到目标左侧：新 split 在前', () => {
        const root = insertLeaf(leaf(2), 2, leaf(9), 'left', genPaneId);
        const split = expectSplit(root);
        expect(split.dir).toBe('row');
        expect(split.first).toEqual(leaf(9));
        expect(split.second).toEqual(leaf(2));
    });

    it('插入到目标下方：方向为 column', () => {
        expect(expectSplit(insertLeaf(leaf(2), 2, leaf(9), 'down', genPaneId)).dir).toBe('column');
    });

    it('目标不存在时返回原树', () => {
        const origin = leaf(1);
        expect(insertLeaf(origin, 999, leaf(9), 'left', genPaneId)).toBe(origin);
    });
});

describe('layoutTree', () => {
    it('单窗格铺满容器', () => {
        const layout = layoutTree(leaf(1));
        expect(layout.panes).toEqual([{ id: 1, rect: { left: 0, top: 0, width: 100, height: 100 } }]);
        expect(layout.splits).toHaveLength(0);
    });

    it('row 拆分：两矩形左右相邻互补', () => {
        const root = splitLeaf(leaf(1), 1, 'right', genPaneId);
        const layout = layoutTree(root);
        expect(layout.panes).toHaveLength(2);
        const [a, b] = layout.panes;
        expect(a.rect).toEqual({ left: 0, top: 0, width: 50, height: 100 });
        expect(b.rect).toEqual({ left: 50, top: 0, width: 50, height: 100 });
        expect(a.rect.left + a.rect.width).toBeCloseTo(b.rect.left);
        expect(a.rect.width + b.rect.width).toBeCloseTo(100);
    });

    it('column 拆分：两矩形上下相邻互补', () => {
        const root = splitLeaf(leaf(1), 1, 'down', genPaneId);
        const [, b] = layoutTree(root).panes;
        expect(b.rect).toEqual({ left: 0, top: 50, width: 100, height: 50 });
    });

    it('ratio 生效', () => {
        const root = expectSplit(splitLeaf(leaf(1), 1, 'right', genPaneId));
        root.ratio = 0.3;
        const [a, b] = layoutTree(root).panes;
        expect(a.rect.width).toBeCloseTo(30);
        expect(b.rect.left).toBeCloseTo(30);
        expect(b.rect.width).toBeCloseTo(70);
    });

    it('嵌套拆分：矩形面积不重叠且铺满', () => {
        const l1 = expectSplit(splitLeaf(leaf(1), 1, 'right', genPaneId));
        const l2 = expectSplit(splitLeaf(l1, (l1.second as PaneLeaf).id, 'down', genPaneId));
        const layout = layoutTree(l2);
        expect(layout.panes).toHaveLength(3);
        const total = layout.panes.reduce((acc, p) => acc + p.rect.width * p.rect.height, 0);
        expect(total).toBeCloseTo(100 * 100);
        expect(layout.splits).toHaveLength(2);
        // 每个分裂节点都有自己的矩形（用于渲染分隔条）
        layout.splits.forEach((sp) => {
            expect(sp.rect.width).toBeGreaterThan(0);
            expect(sp.rect.height).toBeGreaterThan(0);
        });
    });
});

describe('getDropPosition', () => {
    const rect = (x = 0, y = 0, width = 100, height = 100): DOMRect =>
        ({ x, y, left: x, top: y, width, height, right: x + width, bottom: y + height, toJSON: () => ({}) }) as DOMRect;

    const at = (clientX: number, clientY: number, base = 0) => getDropPosition({ clientX, clientY } as DragEvent, rect(base, base));

    it('四象限落点对应四方向', () => {
        expect(at(10, 50)).toBe('left');
        expect(at(90, 50)).toBe('right');
        expect(at(50, 10)).toBe('up');
        expect(at(50, 90)).toBe('down');
    });

    it('中心点取最近边（稳定且不抛错）', () => {
        const pos = at(50, 50);
        expect(['left', 'right', 'up', 'down']).toContain(pos);
    });
});
