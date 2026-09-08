import { afterEach, describe, expect, it, vi } from 'vitest';
import { mount } from '@vue/test-utils';

// SvgIcon 带样式块，会触发 postcss 配置加载，测试中 mock 掉
vi.mock('@/components/svg-icon/index.vue', () => ({ default: { name: 'SvgIcon', render: () => null } }));

import Contextmenu from '../index.vue';
import { ContextmenuItem } from '../item';

const buildItems = (onSelect: (id: string) => void): ContextmenuItem[] => [
    new ContextmenuItem('copy', 'txt.copy').withOnClick(() => onSelect('copy')),
    new ContextmenuItem('split', 'txt.split').withChildren([
        new ContextmenuItem('split-left', 'txt.splitLeft').withOnClick(() => onSelect('split-left')),
        new ContextmenuItem('split-right', 'txt.splitRight').withOnClick(() => onSelect('split-right')),
    ]),
];

const mountMenu = (items: ContextmenuItem[] = []) =>
    mount(Contextmenu, {
        props: { dropdown: { x: 30, y: 40 }, items },
        global: {
            mocks: { $t: (key: string) => key },
        },
        attachTo: document.body,
    });

const contentEl = () => document.body.querySelector('[data-slot="context-menu-content"]');

const flush = () => new Promise((r) => setTimeout(r, 10));

// reka-ui 会在 body 上残留焦点守卫等全局节点，影响后续测试的菜单打开，逐测清理
describe('Contextmenu 组件', () => {
    afterEach(() => {
        document.body.innerHTML = '';
    });

    it('首次 openContextmenu 即渲染菜单内容（先空 items 挂载再更新）', async () => {
        const wrapper = mountMenu();
        await wrapper.setProps({ items: buildItems(vi.fn()) });
        await (wrapper.vm as any).openContextmenu({});
        await flush();

        const menu = contentEl();
        expect(menu, '首次打开应渲染菜单内容').toBeTruthy();
        expect(document.body.textContent).toContain('txt.copy');
        // reka 在菜单打开状态下卸载会在 happy-dom 残留全局状态，影响后续测试，先关再卸
        (wrapper.vm as any).closeContextmenu();
        await flush();
        wrapper.unmount();
    });

    it('点击叶子项回调仅执行一次', async () => {
        const onSelect = vi.fn();
        const wrapper = mountMenu(buildItems(onSelect));
        await (wrapper.vm as any).openContextmenu({});
        await flush();

        const items = Array.from(document.body.querySelectorAll('[data-slot="context-menu-item"]'));
        const leaf = items.find((el) => el.textContent?.includes('txt.copy')) as HTMLElement;
        expect(leaf).toBeTruthy();

        leaf.click();
        await flush();
        expect(onSelect).toHaveBeenCalledTimes(1);
        expect(onSelect).toHaveBeenCalledWith('copy');
        (wrapper.vm as any).closeContextmenu();
        await flush();
        wrapper.unmount();
    });

    it('closeContextmenu 可关闭菜单', async () => {
        const wrapper = mountMenu(buildItems(vi.fn()));
        await (wrapper.vm as any).openContextmenu({});
        await flush();
        expect(contentEl()).toBeTruthy();

        (wrapper.vm as any).closeContextmenu();
        await flush();
        expect(contentEl()).toBeFalsy();
        wrapper.unmount();
    });
});
