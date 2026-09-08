import { describe, expect, it } from 'vitest';
import { ContextmenuItem, filterVisibleItems } from '../item';

/** 构造菜单项（仅填充过滤逻辑涉及的字段） */
const item = (clickId: string, opts: Partial<ContextmenuItem> = {}): ContextmenuItem => {
    const ci = new ContextmenuItem(clickId, `txt.${clickId}`);
    return Object.assign(ci, opts);
};

describe('filterVisibleItems', () => {
    it('剔除 affix 项与 hideFunc 命中的项', () => {
        const items = [
            item('a'),
            item('affixed', { affix: true }),
            item('hidden', { hideFunc: () => true }),
        ];
        const visible = filterVisibleItems(items, {});
        expect(visible.map((v) => v.clickId)).toEqual(['a']);
    });

    it('hideFunc 按业务数据判定', () => {
        const items = [item('a', { hideFunc: (data: any) => data.count > 3 })];
        expect(filterVisibleItems(items, { count: 5 })).toHaveLength(0);
        expect(filterVisibleItems(items, { count: 1 })).toHaveLength(1);
    });

    it('子菜单递归过滤', () => {
        const items = [
            item('parent', {
                children: [item('child-visible'), item('child-hidden', { hideFunc: () => true })],
            }),
        ];
        const visible = filterVisibleItems(items, {});
        expect(visible).toHaveLength(1);
        expect(visible[0].children!.map((c) => c.clickId)).toEqual(['child-visible']);
    });

    it('子项全部不可见时父项剔除（不渲染空壳子菜单）', () => {
        const items = [
            item('parent', {
                children: [item('child', { hideFunc: () => true })],
            }),
            item('standalone'),
        ];
        const visible = filterVisibleItems(items, {});
        expect(visible.map((v) => v.clickId)).toEqual(['standalone']);
    });

    it('多层级递归（三级菜单）', () => {
        const items = [
            item('lv1', {
                children: [
                    item('lv2', {
                        children: [item('lv3', { hideFunc: (data: any) => data.block })],
                    }),
                ],
            }),
        ];
        const visible = filterVisibleItems(items, { block: false });
        expect((visible[0].children![0].children ?? []).map((c) => c.clickId)).toEqual(['lv3']);

        const blocked = filterVisibleItems(items, { block: true });
        // lv3 被隐藏后 lv2 被剔除，进而 lv1 也被剔除
        expect(visible).not.toBe(blocked);
        expect(blocked).toHaveLength(0);
    });

    it('过滤产生副本，不污染调用方持有的原项', () => {
        const child = item('child');
        const parent = item('parent', { children: [child, item('hidden-child', { hideFunc: () => true })] });
        filterVisibleItems([parent], {});
        expect(parent.children).toHaveLength(2);
        expect(parent.children![0]).toBe(child);
    });

    it('副本保留链式方法（原型未丢失）', () => {
        const parent = item('parent', { children: [item('child')] });
        const [copy] = filterVisibleItems([parent], {});
        expect(typeof copy.withOnClick).toBe('function');
        expect(copy.withOnClick(() => {}).onClickFunc).toBeTypeOf('function');
    });
});
