import { onBeforeUnmount, onMounted } from 'vue';

/**
 * 让 Escape 在「下拉已收起」时仍能关掉抽屉。
 *
 * Element Plus 的 select 在自身 keydown 处理里对 Escape 一律 preventDefault + stopPropagation
 * （源码里没有「下拉未展开则放行」的分支），而抽屉的关闭监听挂在 document 的冒泡阶段，
 * 于是焦点只要停在 select 的输入框上，事件就到不了抽屉：实测连按三次都不关，
 * 把焦点移到任意普通输入框后立刻恢复。
 *
 * 这里不自己改抽屉的可见状态，那样会绕过宿主写在 before-close 里的未保存确认与表单校验；
 * 只把事件以抽屉为目标补发一次，让 EP 原有的关闭路径照常走完。
 * 下拉正处于展开态时不补发：那次 Escape 的正确用途是收起下拉。
 */
export function useEscapeClosesDrawer() {
    const onKeydown = (event: KeyboardEvent) => {
        if (event.key !== 'Escape' || event.target === document.body) return;

        const target = event.target as HTMLElement | null;
        const combobox = target?.closest?.<HTMLElement>('input[role="combobox"], [role="combobox"][aria-expanded]');
        if (combobox?.getAttribute('aria-expanded') !== 'false') return;

        const drawer = combobox.closest<HTMLElement>('.el-drawer, .el-dialog');
        if (!drawer) return;

        // 冒泡回 document 交给 EP 自己的处理；目标是抽屉而非 select，不会再被 select 掐掉
        drawer.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }));
    };

    onMounted(() => document.addEventListener('keydown', onKeydown, true));
    onBeforeUnmount(() => document.removeEventListener('keydown', onKeydown, true));
}
