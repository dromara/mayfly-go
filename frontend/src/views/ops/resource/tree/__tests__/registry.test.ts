import { describe, expect, it, vi, beforeEach } from 'vitest';

import { getContributor, getRootContributor, isNodeSelectable, registerContributor, resolveHasChildren, toNodeMatcher } from '../registry';
import type { TreeNode } from '../types';

const makeNode = (kind: string, key = kind): TreeNode => ({ key, kind, label: key, hasChildren: false, params: {} });

describe('tree/registry 贡献者注册表', () => {
    beforeEach(() => {
        vi.restoreAllMocks();
    });

    it('注册后可按 kind 获取', () => {
        const c = { kind: 'test-a' };
        registerContributor(c);
        expect(getContributor('test-a')).toBe(c);
        expect(getContributor('not-exist')).toBeUndefined();
    });

    it('DEV 下重复注册告警（首次注册不告警，第二次起告警防静默覆盖）', () => {
        const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
        registerContributor({ kind: 'test-dup' });
        expect(warn).not.toHaveBeenCalled();
        registerContributor({ kind: 'test-dup' });
        expect(warn).toHaveBeenCalledTimes(1);
        expect(warn.mock.calls[0][0]).toContain('test-dup');
    });

    it('hasChildren 支持布尔与函数形态', () => {
        registerContributor({ kind: 'test-bool', hasChildren: true });
        registerContributor({ kind: 'test-fn', hasChildren: (n: TreeNode) => n.key === 'yes' });
        expect(resolveHasChildren(getContributor('test-bool')!, makeNode('test-bool'))).toBe(true);
        expect(resolveHasChildren(getContributor('test-fn')!, makeNode('test-fn', 'yes'))).toBe(true);
        expect(resolveHasChildren(getContributor('test-fn')!, makeNode('test-fn', 'no'))).toBe(false);
        // 未声明默认 false
        registerContributor({ kind: 'test-none' });
        expect(resolveHasChildren(getContributor('test-none')!, makeNode('test-none'))).toBe(false);
    });

    it('贡献者声明 resourceType 自动登记「后端资源类型 → 根节点 kind」映射', () => {
        registerContributor({ kind: 'test-root', resourceType: 101 });
        expect(getRootContributor(101)?.kind).toBe('test-root');
        expect(getRootContributor('101')?.kind).toBe('test-root');
        expect(getRootContributor(999)).toBeUndefined();
    });

    it('isNodeSelectable：贡献者 selectable 布尔/函数/缺省三形态 + 禁用节点不可选', () => {
        registerContributor({ kind: 'sel-bool', selectable: true });
        registerContributor({ kind: 'sel-fn', selectable: (n: TreeNode) => n.key === 'yes' });
        registerContributor({ kind: 'sel-none' });

        expect(isNodeSelectable(makeNode('sel-bool'))).toBe(true);
        expect(isNodeSelectable(makeNode('sel-fn', 'yes'))).toBe(true);
        expect(isNodeSelectable(makeNode('sel-fn', 'no'))).toBe(false);
        expect(isNodeSelectable(makeNode('sel-none'))).toBe(false);
        // 未注册贡献者的 kind（如 tag 骨架/占位行）不可选
        expect(isNodeSelectable(makeNode('sel-unknown'))).toBe(false);
        // 禁用节点即使声明可选也不可取
        expect(isNodeSelectable({ ...makeNode('sel-bool'), disabled: true })).toBe(false);
    });

    it('toNodeMatcher：kind 清单与谓词函数两形态统一为判定函数', () => {
        const byKinds = toNodeMatcher(['a', 'b']);
        expect(byKinds(makeNode('a'))).toBe(true);
        expect(byKinds(makeNode('c'))).toBe(false);
        const byFn = toNodeMatcher((n) => n.key.startsWith('k'));
        expect(byFn(makeNode('x', 'key1'))).toBe(true);
        expect(byFn(makeNode('x', 'no'))).toBe(false);
    });
});
