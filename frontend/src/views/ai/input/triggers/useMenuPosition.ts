/**
 * 触发器浮层定位 composable（floating-ui 计算，重算订阅显式管理）
 *
 * 与「教科书用法」的三点差异，都是拖拽实测出来的：
 * 1. 参照物是光标虚拟元素（没有真实 DOM）：floating-ui 的 autoUpdate 感知不到它所在
 *    的滚动容器；且浮层每次开合都是新 DOM，自动监听可能绑在已卸载的旧节点上——曾实测
 *    「抽屉内滚动时浮层脱离光标、滞留在视口底外被裁切整行」。因此改显式订阅：
 *    window 捕获阶段 scroll（scroll 不冒泡，捕获才能收到抽屉 body / 编辑器内容区等
 *    内层滚动）+ window resize + ResizeObserver（资源树懒加载撑高面板后立即重排）。
 * 2. 每次重算都实时重取浮层 DOM（getMenuEl）并丢弃已卸载节点：陈旧节点的 rect 全是 0，
 *    会让「贴光标上方」少减浮层高度，面板挂到光标下方溢出视口（实测踩过）。
 * 3. strategy: 'fixed'：浮层 CSS 本就是 position: fixed，按视口基准算坐标才与滚动一致。
 */

import { onBeforeUnmount, ref } from 'vue';
import { computePosition, flip, offset, shift as shiftMiddleware, size as sizeMiddleware } from '@floating-ui/dom';

/** 光标坐标（ProseMirror view.coordsAtPos 的口径：视口坐标系） */
export interface CaretCoords {
    left: number;
    top: number;
    bottom: number;
}

export function useMenuPosition() {
    /** 浮层定位样式（透传给浮层组件的 menu-style prop） */
    const style = ref<Record<string, string>>({});

    /** 取数回调：重算时实时调用，保证参与测量的永远是当前在场的那块浮层 DOM */
    let getCoords: (() => CaretCoords) | null = null;
    let getMenuEl: (() => HTMLElement | null | undefined) | null = null;
    let resizeObserver: ResizeObserver | null = null;
    /** 当前被 ResizeObserver 观测的节点（菜单↔面板中途切换时需换绑） */
    let observedEl: HTMLElement | null = null;

    const buildRect = (coords: CaretCoords) => ({
        x: coords.left,
        y: coords.top,
        width: 0,
        height: coords.bottom - coords.top,
        top: coords.top,
        right: coords.left,
        bottom: coords.bottom,
        left: coords.left,
    });

    /** 重算一次；浮层已卸载（开合切换的瞬间）直接跳过 */
    const reposition = () => {
        const menuEl = getMenuEl?.();
        if (!menuEl?.isConnected || !getCoords) return;

        // 同一场开合内菜单↔面板可能中途切换组件：换绑观测目标，
        // observe() 会立刻投递一次初始尺寸，等价于立即补一次重排
        if (resizeObserver && observedEl !== menuEl) {
            resizeObserver.disconnect();
            resizeObserver.observe(menuEl);
            observedEl = menuEl;
        }

        const reference = { getBoundingClientRect: () => buildRect(getCoords!()) };
        void computePosition(reference, menuEl, {
            strategy: 'fixed',
            placement: 'top-start',
            middleware: [
                offset(8),
                flip({ padding: 8 }),
                shiftMiddleware({ padding: 8 }),
                // 两侧空间都不够时先收缩高度而不是溢出：以浮层自己声明的 --trigger-layer-max
                // 为上限（不突破原有设计高度），保证贴边时底边不被视口裁掉（面板内部自行滚动）
                sizeMiddleware({
                    padding: 8,
                    apply({ availableHeight, elements }) {
                        const cap = Number.parseFloat(getComputedStyle(elements.floating).getPropertyValue('--trigger-layer-max')) || 320;
                        elements.floating.style.maxHeight = `${Math.floor(Math.min(cap, availableHeight))}px`;
                    },
                }),
            ],
        }).then(({ x, y }) => {
            if (!menuEl.isConnected) return; // 等待期间被关闭/换新，不再回写旧节点
            style.value = { left: `${x}px`, top: `${y}px` };
        });
    };

    const stopTracking = () => {
        window.removeEventListener('scroll', reposition, true);
        window.removeEventListener('resize', reposition);
        resizeObserver?.disconnect();
        resizeObserver = null;
        observedEl = null;
    };

    /**
     * 启动定位跟踪（每次开合调一次）：订阅滚动/缩放/浮层尺寸变化并立即计算一次。
     * 调用时机必须晚于浮层 DOM 挂载（nextTick 之后）
     */
    const start = (coordsGetter: () => CaretCoords, menuGetter: () => HTMLElement | null | undefined) => {
        getCoords = coordsGetter;
        getMenuEl = menuGetter;
        const menuEl = getMenuEl();
        if (!menuEl) return;

        reposition();
        stopTracking();
        window.addEventListener('scroll', reposition, true);
        window.addEventListener('resize', reposition);
        resizeObserver = new ResizeObserver(reposition);
        resizeObserver.observe(menuEl);
        observedEl = menuEl;
    };

    const stop = () => {
        stopTracking();
        getCoords = null;
        getMenuEl = null;
    };

    onBeforeUnmount(stop);

    return { style, reposition, start, stop };
}