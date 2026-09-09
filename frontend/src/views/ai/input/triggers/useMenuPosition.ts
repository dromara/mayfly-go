/**
 * 触发器浮层定位 composable（TriggerMenu 的 floating-ui 定位）
 *
 * 以编辑器光标坐标构造 virtual element，floating-ui 计算浮层位置；
 * autoUpdate 监听浮层尺寸变化重算（资源树懒加载展开后底边仍贴光标上方向上生长）。
 */

import { onBeforeUnmount, ref } from 'vue';
import { autoUpdate, computePosition, flip, offset, shift as shiftMiddleware } from '@floating-ui/dom';

export function useMenuPosition() {
    /** 浮层定位样式（透传给浮层组件的 menu-style prop） */
    const style = ref<Record<string, string>>({});

    /** autoUpdate 停止函数（浮层关闭/组件卸载时清理） */
    let stopAutoUpdate: () => void = () => {};

    /**
     * 计算并持续跟随浮层位置
     * @param getCoords 光标坐标（view.coordsAtPos 结果）
     * @param menuEl 浮层自身 DOM（误传编辑器容器会定位到错误位置）
     */
    const update = (
        getCoords: () => { left: number; top: number; bottom: number },
        menuEl: HTMLElement | null | undefined,
    ) => {
        if (!menuEl) return;

        const buildRect = (coords: { left: number; top: number; bottom: number }) => ({
            x: coords.left,
            y: coords.top,
            width: 0,
            height: coords.bottom - coords.top,
            top: coords.top,
            right: coords.left,
            bottom: coords.bottom,
            left: coords.left,
        });

        const applyPosition = () =>
            computePosition({ getBoundingClientRect: () => buildRect(getCoords()) } as HTMLElement, menuEl, {
                placement: 'top-start',
                middleware: [offset(8), flip(), shiftMiddleware({ padding: 8 })],
            }).then(({ x, y }) => {
                style.value = {
                    left: `${x}px`,
                    top: `${y}px`,
                };
            });

        applyPosition();
        stopAutoUpdate();
        stopAutoUpdate = autoUpdate({ getBoundingClientRect: () => buildRect(getCoords()) } as HTMLElement, menuEl, applyPosition);
    };

    const stop = () => {
        stopAutoUpdate();
        stopAutoUpdate = () => {};
    };

    onBeforeUnmount(stop);

    return { style, update, stop };
}
