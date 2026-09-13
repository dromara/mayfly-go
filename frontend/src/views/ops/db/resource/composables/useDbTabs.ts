/**
 * DB 数据操作页的标签页状态
 *
 * DbDataOp 同时承载「表数据 / 查询 / 表管理」三类标签页。标签的增删切换是纯状态逻辑，
 * 与组件里的树节点回调、建表弹窗、DDL 预览等视图逻辑无关，故抽到此处：
 * 组件只保留副作用（注册 SQL 补全、激活子组件、定位树节点），状态一律问本组合式函数。
 *
 * 为什么是组合式函数而不是全局 store：每个 DB 资源标签页都有独立的 DbDataOp 实例
 * （见 helpers.ts 的 getDbOpTab，按 实例/库 维度开页），标签页状态属实例私有，
 * 放进全局 store 会让多个实例的标签互相串台。
 */
import { computed, reactive, ref } from 'vue';
import { TabInfo } from '../TabInfo';

export function useDbTabs() {
    /** 标签页集合，key 见各 addXxxTab 的构造规则 */
    const tabs = reactive(new Map<string, TabInfo>());

    /** 当前激活标签页 key，直接作为 el-tabs 的 v-model */
    const activeTabKey = ref('');

    /** 按插入顺序展开的标签页列表 */
    const tabList = computed(() => [...tabs.values()]);

    /** 是否存在标签页（无则整个标签区不渲染） */
    const hasOpenTabs = computed(() => tabs.size > 0);

    /** 当前激活的标签页 */
    const activeTab = computed(() => tabs.get(activeTabKey.value));

    /**
     * 激活指定标签页
     * @returns 该标签页是否已存在——已存在时调用方无需重复创建
     */
    const activateTab = (key: string): boolean => {
        activeTabKey.value = key;
        return tabs.has(key);
    };

    /** 新增标签页（调用前一般已 activateTab，故此处不改激活项） */
    const addTab = (tab: TabInfo) => {
        tabs.set(tab.key, tab);
    };

    /**
     * 关闭标签页。关闭的是当前激活项时，激活相邻的后一个（无则前一个）。
     * @returns 激活项是否发生变化——调用方据此决定是否执行切换副作用
     */
    const closeTab = (key: string): boolean => {
        const keys = [...tabs.keys()];
        const idx = keys.indexOf(key);
        if (idx < 0) {
            return false;
        }
        tabs.delete(key);
        if (activeTabKey.value !== key) {
            return false;
        }
        activeTabKey.value = keys[idx + 1] ?? keys[idx - 1] ?? '';
        return true;
    };

    /** 关闭全部标签页 */
    const clearTabs = () => {
        tabs.clear();
    };

    return { tabs, activeTabKey, tabList, hasOpenTabs, activeTab, activateTab, addTab, closeTab, clearTabs };
}
