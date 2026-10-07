import { afterEach, describe, expect, it, vi } from 'vitest';
import { mount, VueWrapper } from '@vue/test-utils';
import { defineComponent, h } from 'vue';

import { useEscapeClosesDrawer } from '../useEscapeClosesDrawer';

/**
 * Escape 能否关掉抽屉，取决于事件是否活着到达 document 的冒泡阶段。
 *
 * select 对 Escape 无条件 stopPropagation，EP 抽屉的关闭监听却挂在 document 冒泡阶段，
 * 于是焦点停在 select 上时抽屉怎么按都不关。这里按真实链路搭 DOM：
 * select 的掐事件用同样的 stopPropagation 复现，EP 的关闭动作用一个 document 冒泡监听代表
 */
const cleanups: (() => void)[] = [];

afterEach(() => {
    for (const fn of cleanups.splice(0)) fn();
    document.body.replaceChildren();
});

function setup(options: { expanded?: string; withCombobox?: boolean }) {
    const { expanded = 'false', withCombobox = true } = options;
    const drawer = document.createElement('div');
    drawer.className = 'el-drawer';
    const input = document.createElement('input');
    if (withCombobox) {
        input.setAttribute('role', 'combobox');
        input.setAttribute('aria-expanded', expanded);
    }
    drawer.appendChild(input);
    document.body.appendChild(drawer);

    const epClose = vi.fn();
    const onDocKeydown = (event: KeyboardEvent) => {
        if (event.key === 'Escape') epClose();
    };
    document.addEventListener('keydown', onDocKeydown);
    cleanups.push(() => document.removeEventListener('keydown', onDocKeydown));

    const host = withCombobox ? document.querySelector<HTMLInputElement>('input[role="combobox"]')! : document.querySelector('input')!;
    // EP select 的行为：Escape 到这就断了
    const onSelectKeydown = (event: KeyboardEvent) => {
        if (withCombobox && event.key === 'Escape') event.stopPropagation();
    };
    host.addEventListener('keydown', onSelectKeydown);
    cleanups.push(() => host.removeEventListener('keydown', onSelectKeydown));

    const wrapper: VueWrapper = mount(
        defineComponent({
            setup: () => {
                useEscapeClosesDrawer();
                return () => h('div');
            },
        }),
        { attachTo: document.body }
    );
    // 不卸载的话，其 document 捕获监听会活到下一个用例，一个按键被多个实例各自补发
    cleanups.push(() => wrapper.unmount());

    return { host, epClose };
}

const pressEscape = (el: HTMLElement) => el.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }));

describe('useEscapeClosesDrawer', () => {
    it('下拉已收起时 Escape 能活着到达抽屉', () => {
        const { host, epClose } = setup({ expanded: 'false' });
        pressEscape(host);
        expect(epClose, 'select 掐掉事件后没有补发，抽屉收不到 Escape').toHaveBeenCalledTimes(1);
    });

    it('下拉展开时不补发：那次 Escape 的正当用途是收起下拉', () => {
        const { host, epClose } = setup({ expanded: 'true' });
        pressEscape(host);
        expect(epClose, '补发会让一次 Escape 同时关掉下拉和抽屉').not.toHaveBeenCalled();
    });

    it('补发的事件不会再被自己接住（单次按键只关一次）', () => {
        const { host, epClose } = setup({ expanded: 'false' });
        pressEscape(host);
        expect(epClose).toHaveBeenCalledTimes(1);
        pressEscape(host);
        expect(epClose).toHaveBeenCalledTimes(2);
    });

    it('找不到宿主抽屉时不凭空补发', () => {
        const { host, epClose } = setup({ expanded: 'false' });
        // 把 combobox 挪出抽屉：本接缝只负责「补发给宿主」，没有宿主就不该造出一个 Escape
        host.closest('.el-drawer')!.replaceWith(host);
        epClose.mockClear();
        pressEscape(host);
        expect(epClose, '无宿主抽屉仍补发，等于凭空触发一次关闭').not.toHaveBeenCalled();
    });

    it('普通输入框由 EP 自己处理，本接缝不额外补发', () => {
        const { host, epClose } = setup({ withCombobox: false });
        pressEscape(host);
        expect(epClose, '不该出现双份关闭动作').toHaveBeenCalledTimes(1);
    });
});
