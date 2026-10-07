/**
 * Redis 数据操作页的 key 树域：按分隔符分组渲染、类型筛选、展开/勾选状态与右键菜单定位。
 *
 * 从 RedisDataOp 拆出的原因：树的渲染与交互是自成一套的状态机（展开集合、勾选清单、
 * 筛选词三者联动），收敛后「切库要重置哪些状态」只需看 reset；点击叶子打开详情这类
 * 跨域副作用通过 openKeyDetail 回调交还宿主，本文件不感知 tab 的存在。
 */
import { nextTick, reactive, ref, computed, type Ref } from 'vue';
import { useI18n } from 'vue-i18n';
import type { VirtualTreeInstance } from '@/components/virtual-tree';
import { viewAppearance } from '../../keyview/appearance';
import { coversAll } from '../../keyview/descriptor';
import { insertKeysToTree, keysToList, keysToTree, type TreeNode } from '../../utils';
import type { useKeyScan } from './useKeyScan';

/** 右键菜单实例（仅声明用到的成员） */
export interface ContextmenuRef {
    openContextmenu: (node: unknown) => void;
    closeContextmenu: () => void;
}

/**
 * 虚拟树字段映射：唯一键在 key、节点名在 name、子级在 children。
 * 叶子判定交给引擎（children 为空即叶子）；redis 树是全量物化的，不存在「可展开但未加载」的节点，无需占位 children
 */
export const keyTreeFieldNames = { value: 'key', label: 'name', children: 'children' };

export function useKeyTree(deps: {
    /** 扫描域：树渲染的数据来源（key 列表、类型摘要、描述符） */
    scan: ReturnType<typeof useKeyScan>;
    keyTreeRef: Ref<VirtualTreeInstance | null>;
    contextmenuRef: Ref<ContextmenuRef | null>;
    /** 点击叶子打开 key 详情（tab 域副作用，由宿主注入） */
    openKeyDetail: (key: string) => void;
}) {
    const { t } = useI18n();

    const state = reactive({
        keySeparator: ':',
        keyTreeData: [] as TreeNode[],
        keyTreeExpanded: new Set<string>(),
        checkedKeys: [] as string[],
        batchSelect: false,
        // 右键菜单的定位坐标：菜单层挂 body 下，用视口坐标
        menuPosition: { x: 0, y: 0 },
    });

    const typeFilter = ref('');

    /** 单个 key 是否通过当前类型筛选（无筛选即全通过）：全量渲染、增量插入与批量「全选」共用同一判据 */
    const matchesFilter = (key: string) => !typeFilter.value || deps.scan.state.summaries[key]?.type === typeFilter.value;

    /** 当前类型筛选下可见的 key：树渲染与批量「全选」共用同一份判据，避免选到列表里看不见的 key */
    const visibleKeys = computed(() => {
        const { keys } = deps.scan.state;
        return typeFilter.value ? keys.filter(matchesFilter) : keys;
    });

    /** 当前筛选下的 key 是否已全被勾选：决定底部按钮是「全选」还是「取消全选」，两者必须同一个判据 */
    const allVisibleSelected = computed(() => coversAll(visibleKeys.value, state.checkedKeys));

    /**
     * key 的类型皮肤（主色 + 速记字母）：皮肤表在前端 keyview/appearance.ts，
     * 后端描述符只说「这是哪个视角」，改配色不必重编后端
     */
    const appearanceOf = (key: string) => viewAppearance(deps.scan.descriptorOf(key)?.view);

    /** 悬浮标题补上类型全称：徽章是缩写，鼠标停上去就能看到完整词 */
    const nodeTitle = (data: Record<string, unknown>, label: string) => {
        const labelKey = data.type == 1 ? '' : deps.scan.descriptorOf(data.key as string)?.label;
        return labelKey ? `${label} · ${t(labelKey)}` : label;
    };

    const ttlOf = (key: string) => deps.scan.summaryOf(key)?.ttl ?? -1;

    const ttlText = (key: string) => {
        const ttl = ttlOf(key);
        if (ttl <= 0) {
            return '';
        }
        // 按量级选单位：不足 1 分钟必须给秒，否则 30 秒会被算成「0m」，看起来像已过期而不是快到期
        if (ttl >= 86400) {
            return `${Math.floor(ttl / 86400)}d`;
        }
        if (ttl >= 3600) {
            return `${Math.floor(ttl / 3600)}h`;
        }
        if (ttl >= 60) {
            return `${Math.floor(ttl / 60)}m`;
        }
        return `${ttl}s`;
    };

    /**
     * 渲染分组树。mode='full' 全量重建（重搜/切库/类型筛选变化）；mode='append' 只把新增 key
     * 增量插入已有树（加载更多），百万级下避免每批都重建整棵树。appendedKeys 是扫描域本批新增的
     * 原始 key，按当前类型筛选后再插入，与可见集口径一致
     */
    function renderKeyTree(mode: 'full' | 'append' = 'full', appendedKeys: string[] = []) {
        if (!state.keySeparator) {
            // 不分组：平铺一层，无层级无排序，重建成本低，增量无意义
            state.keyTreeData = keysToList(visibleKeys.value);
        } else if (mode === 'append' && state.keyTreeData.length) {
            insertKeysToTree(state.keyTreeData, appendedKeys.filter(matchesFilter), state.keySeparator, state.keyTreeExpanded);
            // el-tree-v2 按引用 watch data：就地增量修改后要换一个新顶层引用，引擎才会重建内部节点
            state.keyTreeData = [...state.keyTreeData];
        } else {
            state.keyTreeData = keysToTree(visibleKeys.value, state.keySeparator, state.keyTreeExpanded);
        }
        nextTick(() => {
            if (visibleKeys.value.length <= 20) {
                expandAllKeyNode(state.keyTreeData);
            }
        });
    }

    const applyTypeFilter = () => {
        renderKeyTree();
        if (!state.batchSelect) {
            return;
        }
        // 被筛掉的 key 必须同时退出勾选，否则「删除选中」会打到用户看不见的 key 上
        const visible = new Set(visibleKeys.value);
        state.checkedKeys = state.checkedKeys.filter((key) => visible.has(key));
        nextTick(() => deps.keyTreeRef.value?.setCheckedKeys(state.checkedKeys));
    };

    const expandAllKeyNode = (nodes: TreeNode[]) => {
        nodes.forEach((node) => {
            if (!node.children) {
                return;
            }
            state.keyTreeExpanded.add(node.key as string);
            expandAllKeyNode(node.children);
        });
    };

    const onTreeNodeClick = (data: Record<string, unknown>) => {
        deps.contextmenuRef.value?.closeContextmenu();
        // 目录交给 el-tree 自身展开/收起，不打开详情
        if (data.type == 1) {
            return;
        }
        // 选择态下点整行即勾选/取消（由 el-tree 的 check-on-click-leaf 承担），
        // 勾选变化经 @check 同步到已选清单，这里不再重复处理
        if (state.batchSelect) {
            return;
        }
        deps.openKeyDetail(data.key as string);
    };

    const onTreeNodeExpand = (data: Record<string, unknown>) => {
        state.keyTreeExpanded.add(data.key as string);
    };

    const onTreeNodeCollapse = (data: Record<string, unknown>) => {
        state.keyTreeExpanded.delete(data.key as string);
    };

    const onRightClickNode = (event: MouseEvent, node: Record<string, unknown>) => {
        state.menuPosition.x = event.clientX;
        state.menuPosition.y = event.clientY;
        deps.contextmenuRef.value?.openContextmenu(node);
        deps.keyTreeRef.value?.setCurrentKey(node.key as string);
    };

    /** 虚拟树 check 事件已归一为叶子 key 列表：勾选目录只为展开看子项，批量删除目标必须是真实 key */
    const syncCheckedKeys = (checkedLeafKeys: string[]) => {
        state.checkedKeys = checkedLeafKeys;
    };

    /** 全选 / 取消全选：setCheckedKeys 不触发 @check，已选清单要自己同步 */
    const onToggleSelectAll = () => {
        const keys = allVisibleSelected.value ? [] : [...visibleKeys.value];
        deps.keyTreeRef.value?.setCheckedKeys(keys);
        state.checkedKeys = [...keys];
    };

    /** 退出选择态时清空勾选，避免下次进入还带着上次的选择 */
    const exitBatch = () => {
        deps.keyTreeRef.value?.setCheckedKeys([]);
        state.checkedKeys = [];
        state.batchSelect = false;
    };

    /**
     * 选择态开关：再点一次即退出，因此不需要「取消」按钮，
     * 也不会为了塞动作按钮而把搜索框挤窄（面板结构在选择态下完全不变）
     */
    const onToggleBatch = () => {
        if (state.batchSelect) {
            exitBatch();
            return;
        }
        state.batchSelect = true;
    };

    /** 分隔符即树的分组方式，选「不分组」时平铺成一层 */
    const onSeparatorChange = (separator: string) => {
        state.keySeparator = separator;
        renderKeyTree();
    };

    /** 切库后清空本域状态；分组分隔符保留——分组偏好是用户习惯，跨库沿用 */
    function reset() {
        state.keyTreeData = [];
        state.keyTreeExpanded.clear();
        state.checkedKeys = [];
        state.batchSelect = false;
        typeFilter.value = '';
    }

    return {
        state,
        typeFilter,
        visibleKeys,
        allVisibleSelected,
        appearanceOf,
        nodeTitle,
        ttlOf,
        ttlText,
        renderKeyTree,
        applyTypeFilter,
        onTreeNodeClick,
        onTreeNodeExpand,
        onTreeNodeCollapse,
        onRightClickNode,
        syncCheckedKeys,
        onToggleSelectAll,
        exitBatch,
        onToggleBatch,
        onSeparatorChange,
        reset,
    };
}
