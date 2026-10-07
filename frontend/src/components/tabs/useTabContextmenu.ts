import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import { ref } from 'vue';
import type { TabItem } from './types';

/**
 * 标签条内置右键菜单：关闭当前 / 关闭其它 / 关闭右侧 / 关闭全部。
 *
 * 作为共享能力内置于各标签条实现（UnifiedTabBar / ElTabsBar），业务页面无需自行接线即可获得一致行为。
 * 每个关闭动作都逐个回吐为契约的 close(key)（而非某种「批量关闭」专用事件），
 * 因此各页面已有的 onRemoveTab 清理逻辑对单个关闭与批量关闭一视同仁；
 * 仅当「当前激活标签被这批关闭波及」时，才额外回吐 update:modelValue(fallback) 落到目标标签。
 */
export function useTabContextmenu(options: {
    /** 读取当前全部标签的快照（批量关闭据此遍历 / 定位左右侧） */
    getTabs: () => TabItem[];
    /** 读取当前激活标签 key，用于判断激活项是否被这批关闭波及 */
    getActiveKey: () => string;
    /** 关闭指定标签，一般转发为 emit('close', key) */
    close: (key: string) => void;
    /** 激活指定标签，一般转发为 emit('update:modelValue', key) */
    activate: (key: string) => void;
}) {
    const { getTabs, getActiveKey, close, activate } = options;

    const contextmenuRef = ref<InstanceType<typeof Contextmenu>>();
    const dropdown = ref({ x: 0, y: 0 });

    /** 与关闭按钮同口径：closable === false 的标签不参与任何关闭 */
    const isClosable = (tab: TabItem) => tab.closable !== false;

    /**
     * 批量关闭一组标签：逐个回吐 close(key)，复用页面既有的单个关闭清理逻辑；
     * 若当前激活标签在这批里（关完无处停留），再激活 fallback 兜底（fallback 省略表示全关，交给页面空态）。
     */
    const closeTabs = (targets: TabItem[], fallback?: string) => {
        const activeKey = getActiveKey();
        const hitActive = targets.some((tab) => tab.key === activeKey);
        for (const tab of targets) {
            close(tab.key);
        }
        if (fallback !== undefined && hitActive) {
            activate(fallback);
        }
    };

    /** 目标标签右侧、可关闭的标签（关闭右侧用） */
    const closableRightOf = (targetKey: string) => {
        const tabs = getTabs();
        const idx = tabs.findIndex((tab) => tab.key === targetKey);
        return idx < 0 ? [] : tabs.slice(idx + 1).filter(isClosable);
    };

    const items = [
        // 关闭当前：与标签上的 ✕ 同口径，非可关闭标签不显示本项
        new ContextmenuItem('closeCurrent', 'layout.tagsView.close')
            .withIcon('Close')
            .withHideFunc((data: unknown) => {
                const key = (data as { key: string }).key;
                return !getTabs().some((tab) => tab.key === key && isClosable(tab));
            })
            .withOnClick((data: unknown) => close((data as { key: string }).key)),

        // 关闭其它：关掉除目标外全部可关闭标签，激活项被波及则落到目标
        new ContextmenuItem('closeOthers', 'layout.tagsView.closeOther')
            .withIcon('CircleClose')
            .withHideFunc((data: unknown) => {
                const key = (data as { key: string }).key;
                return !getTabs().some((tab) => tab.key !== key && isClosable(tab));
            })
            .withOnClick((data: unknown) => {
                const target = (data as { key: string }).key;
                closeTabs(
                    getTabs().filter((tab) => tab.key !== target && isClosable(tab)),
                    target
                );
            }),

        // 关闭右侧：关掉目标右边全部可关闭标签；右边无可关闭项时不显示
        new ContextmenuItem('closeRight', 'layout.tagsView.closeRight')
            .withIcon('Right')
            .withHideFunc((data: unknown) => closableRightOf((data as { key: string }).key).length === 0)
            .withOnClick((data: unknown) => {
                const target = (data as { key: string }).key;
                closeTabs(closableRightOf(target), target);
            }),

        // 关闭全部：关掉所有可关闭标签，关完无激活项（fallback 省略），页面回到空态
        new ContextmenuItem('closeAll', 'layout.tagsView.closeAll')
            .withIcon('FolderDelete')
            .withHideFunc(() => !getTabs().some(isClosable))
            .withOnClick(() => closeTabs(getTabs().filter(isClosable))),
    ];

    const openTabContextmenu = (event: MouseEvent, tab: TabItem) => {
        dropdown.value = { x: event.clientX, y: event.clientY };
        contextmenuRef.value?.openContextmenu({ key: tab.key });
    };

    return { contextmenuRef, dropdown, items, openTabContextmenu };
}
