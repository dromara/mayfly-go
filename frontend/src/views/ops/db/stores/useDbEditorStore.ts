import { defineStore } from 'pinia';
import { computed, reactive } from 'vue';
import type { TabInfo } from '../models/TabInfo';

/**
 * DB 编辑器模块级 Store
 *
 * 管理跨组件共享的数据库编辑器状态：
 * - 当前激活的数据库实例/库名（供子组件获取上下文）
 * - 打开的 tab 列表（供 tab 栏、快捷键等组件访问）
 *
 * 使用动态 id 支持多资源标签页隔离。
 */
export const useDbEditorStore = (id: string = 'dbEditorStore') =>
    defineStore(id, () => {
        /** 当前操作的数据库实例 id */
        const activeDbInstId = reactive({ value: 0 });
        /** 当前操作的数据库名 */
        const activeDbName = reactive({ value: '' });
        /** 当前激活的 tab key */
        const activeTabKey = reactive({ value: '' });
        /** 打开的 tabs (key -> TabInfo) */
        const tabs = reactive(new Map<string, TabInfo>());

        /** 是否存在打开的 tab */
        const hasOpenTabs = computed(() => tabs.size > 0);
        /** 当前激活的 TabInfo */
        const activeTab = computed(() => tabs.get(activeTabKey.value));
        /** 所有 tab 列表 */
        const tabList = computed(() => [...tabs.values()]);

        /** 设置当前激活的数据库上下文 */
        const setActiveDb = (instId: number, dbName: string) => {
            activeDbInstId.value = instId;
            activeDbName.value = dbName;
        };

        /** 添加或更新 tab */
        const setTab = (key: string, tab: TabInfo) => {
            tabs.set(key, tab);
            activeTabKey.value = key;
        };

        /** 移除 tab，返回是否移除成功 */
        const removeTab = (key: string): boolean => {
            const deleted = tabs.delete(key);
            if (deleted && activeTabKey.value === key) {
                const keys = [...tabs.keys()];
                activeTabKey.value = keys[keys.length - 1] ?? '';
            }
            return deleted;
        };

        /** 设置当前激活 tab */
        const setActiveTab = (key: string) => {
            activeTabKey.value = key;
        };

        /** 关闭所有 tab */
        const clearTabs = () => {
            tabs.clear();
            activeTabKey.value = '';
            activeDbInstId.value = 0;
            activeDbName.value = '';
        };

        return {
            activeDbInstId,
            activeDbName,
            activeTabKey,
            tabs,
            hasOpenTabs,
            activeTab,
            tabList,
            setActiveDb,
            setTab,
            removeTab,
            setActiveTab,
            clearTabs,
        };
    })();
