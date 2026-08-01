import { nextTick, reactive, ref, type Component, type ComponentPublicInstance } from 'vue';

export const ResourceOpCtxKey = 'ResourceOpCtx';

/** 资源操作组件实例（包含可选生命周期回调） */
export interface ResourceCompInstance extends ComponentPublicInstance {
    onActivate?: () => void;
    onRefresh?: () => void;
    onClose?: () => void;
    onResize?: () => void;
}

export interface ResourceOpCtx {
    /**
     * 获取树节点
     * @param nodeKey 节点key
     */
    getTreeNode(nodeKey: string): Record<string, unknown>;

    setCurrentTreeKey(nodeKey: string): void;

    reloadTreeNode(nodeKey: string): void;
}

/**
 * 资源操作标签页
 */
export interface ResourceOpTab {
    // 标签页唯一标识
    key: string;
    name: string; // 名称

    component: Component; // 组件
    // 组件 props（可选）
    componentProps?: Record<string, unknown>;
    // 组件实例
    componentInstance?: any;
    // component key（包含时间戳，用于 keep-alive 缓存控制）
    componentKey?: string;

    // 自定义 tab 标签组件（可选），如果提供则使用自定义组件渲染 tab 标签，否则使用默认的 icon + name 显示
    tabComponent?: Component;
    // tab 标签组件的 props（包含 icon、tabProps 等）
    tabComponentProps?: { icon?: { name: string; color?: string }; [key: string]: unknown };
}

/**
 * 非 tab 组件（如弹窗、抽屉等）
 * 用于注册不显示在 tab 栏的独立组件
 */
export interface ResourceOpOverlay {
    key: string; // 组件唯一标识
    component: Component; // 组件
    props?: Record<string, unknown>; // 组件 props
    visible: boolean; // 是否显示
}

export const allResourceOpTabs = reactive<Map<string, ResourceOpTab>>(new Map());

// 当前激活的 tab key
export const activeResourceOpTabKey = ref<string>('');

// 非 tab 组件注册表
export const allResourceOpOverlays = reactive<Map<string, ResourceOpOverlay>>(new Map());

/**
 * 注册组件实例到 tab
 * @param key tab key
 * @param instance 组件实例
 */
export function registerComponentInstance(key: string, instance: ComponentPublicInstance) {
    const tab = allResourceOpTabs.get(key);
    if (tab && !tab.componentInstance) {
        tab.componentInstance = instance;
    }
}

/**
 * 获取组件实例
 * @param key tab key
 * @returns 组件实例
 */
export function getComponentInstance<T = ResourceCompInstance>(key: string): T | undefined {
    const tab = allResourceOpTabs.get(key);
    return tab?.componentInstance as T;
}

/**
 * 创建资源操作 tab
 * @param tab tab 配置
 */
export function createResourceOpTab(tab: ResourceOpTab): Promise<ResourceOpTab> {
    const resourceOpTab = getResourceOpTab(tab.key);
    if (resourceOpTab) {
        // 已存在，直接激活
        activeResourceOpTabKey.value = tab.key;
        tab = resourceOpTab;
    } else {
        // 创建新 tab，直接生成 componentKey
        tab.componentKey = `${tab.key}-${Date.now()}`;
        allResourceOpTabs.set(tab.key, tab);
        activeResourceOpTabKey.value = tab.key;
    }

    // 已有实例直接返回，避免无谓轮询
    if (tab.componentInstance) {
        return Promise.resolve(tab);
    }

    // 等待组件实例就绪后返回 tab 配置
    return new Promise((resolve) => {
        let attempts = 0;
        const maxAttempts = 100; // 最多重试100次，防止无限轮询
        const checkInstance = () => {
            if (tab.componentInstance) {
                resolve(tab);
            } else if (++attempts < maxAttempts) {
                setTimeout(checkInstance, 50);
            } else {
                // 超时仍返回 tab，避免 Promise 永远 pending
                resolve(tab);
            }
        };
        nextTick().then(() => checkInstance());
    });
}

/**
 * 移除资源操作 tab
 * @param key tab key
 */
export function removeResourceOpTab(key: string) {
    const tab = allResourceOpTabs.get(key);
    if (tab) {
        // 清除实例引用，辅助 GC 回收组件实例
        tab.componentInstance = undefined;
    }
    allResourceOpTabs.delete(key);
}

/**
 * 获取资源操作 tab
 * @param key tab key
 * @returns tab 配置
 */
export function getResourceOpTab(key: string) {
    return allResourceOpTabs.get(key);
}

/**
 * 激活指定 tab
 * @param key tab key
 */
export function activateResourceOpTab(key: string) {
    activeResourceOpTabKey.value = key;
}

/**
 * 获取当前激活的 tab
 * @returns tab 配置
 */
export function getActiveResourceOpTab() {
    return allResourceOpTabs.get(activeResourceOpTabKey.value);
}

/**
 * 更新 tab 的自定义组件 props
 * @param key tab key
 * @param tabComponentProps 自定义 tab 组件的 props
 */
export function updateTabComponentProps(key: string, tabComponentProps: Record<string, unknown>) {
    const tab = allResourceOpTabs.get(key);
    if (tab) {
        tab.tabComponentProps = { ...tab.tabComponentProps, ...tabComponentProps };
    }
}

// ==================== 非 tab 组件（Overlay）管理 ====================

/**
 * 显示或注册非 tab 组件（Overlay）
 * - 已存在：更新 props 并显示
 * - 不存在：注册并显示
 *
 * @param key 组件 key
 * @param component 组件（首次注册时必需）
 * @param props 组件 props
 */
export function showResourceOpOverlay(key: string, component?: Component, props?: Record<string, unknown>) {
    const existing = allResourceOpOverlays.get(key);
    if (existing) {
        // 已存在，更新 props 并显示
        existing.visible = true;
        if (props) {
            // 确保 visible 属性也设置为 true
            existing.props = { ...existing.props, ...props, visible: true };
        } else {
            existing.props = { ...existing.props, visible: true };
        }
    } else if (component) {
        // 不存在且有组件，注册并显示
        allResourceOpOverlays.set(key, {
            key,
            component,
            props: { ...props, visible: true },
            visible: true,
        });
    }
}

/**
 * 隐藏非 tab 组件
 * @param key 组件 key
 */
export function hideResourceOpOverlay(key: string) {
    const overlay = allResourceOpOverlays.get(key);
    if (overlay) {
        overlay.visible = false;
    }
}
