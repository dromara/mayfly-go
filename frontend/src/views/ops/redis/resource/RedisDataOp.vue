<template>
    <div class="redis-data-op h-full">
        <el-splitter>
            <el-splitter-panel size="340px" min="330px" max="60%">
                <!-- p-2 必须带 !：.card 的 padding: 20px 是未分层样式，优先级高于 @layer utilities 里的工具类 -->
                <div class="key-list card flex h-full min-h-0 flex-col gap-2 p-2!">
                    <!-- 检索行：搜索框独占整行宽度，主操作收成右侧方形按钮；选择态下结构不变 -->
                    <div class="flex items-center gap-2">
                        <el-input
                            v-model="scanState.scanParam.match"
                            class="search-input min-w-0 flex-1"
                            :placeholder="$t('redis.keyMatchTips')"
                            clearable
                            @clear="onClearMatch"
                            @keyup.enter="onSearch"
                        >
                            <template #prefix>
                                <SvgIcon name="Search" :size="15" />
                            </template>
                        </el-input>
                        <el-tooltip :content="$t('redis.addKey')" placement="top">
                            <el-button
                                v-auth="PERM_DATA_SAVE"
                                class="add-btn shrink-0"
                                type="primary"
                                :aria-label="$t('redis.addKey')"
                                :disabled="!canOperate || treeState.batchSelect"
                                @click="onShowNewKey"
                            >
                                <SvgIcon name="Plus" :size="16" />
                            </el-button>
                        </el-tooltip>
                    </div>

                    <!-- 视图行：筛选 / 终端 / 批量选择 / 刷新 从左到右成组排列；选择态只切换按钮态，行内元素不增删 -->
                    <div class="flex min-w-0 items-center gap-2">
                        <el-select
                            v-model="typeFilter"
                            class="type-filter shrink-0"
                            size="small"
                            clearable
                            :placeholder="$t('redis.filterType')"
                            @change="applyTypeFilter"
                        >
                            <template #prefix>
                                <SvgIcon name="Filter" :size="13" />
                            </template>
                            <el-option v-for="item in typeOptions" :key="item.view" :label="$t(item.label)" :value="item.view">
                                <span class="type-badge-mini" :style="{ '--badge-color': viewAppearance(item.view).color }">{{
                                    viewAppearance(item.view).badge
                                }}</span>
                                <span class="ml-2">{{ $t(item.label) }}</span>
                            </el-option>
                        </el-select>

                        <el-button class="shrink-0" size="small" :disabled="!canOperate" @click="onOpenConsole">
                            <SvgIcon name="Monitor" :size="13" />
                            <span class="ml-1.5">{{ $t('redis.terminal') }}</span>
                        </el-button>

                        <el-button
                            v-auth="PERM_DATA_DEL"
                            class="shrink-0"
                            size="small"
                            :type="treeState.batchSelect ? 'primary' : 'default'"
                            :disabled="!canOperate"
                            @click="onToggleBatch"
                        >
                            {{ $t('redis.batchSelect') }}
                        </el-button>

                        <div class="tool-group flex shrink-0 items-center">
                            <el-tooltip :content="$t('common.refresh')" placement="top">
                                <el-button
                                    class="tool-btn"
                                    text
                                    :aria-label="$t('common.refresh')"
                                    :disabled="!canOperate"
                                    :loading="scanState.scanning"
                                    @click="onRefreshKeys"
                                >
                                    <SvgIcon name="refresh" :size="15" />
                                </el-button>
                            </el-tooltip>
                        </div>
                    </div>

                    <el-scrollbar class="min-h-0 flex-1">
                        <!-- 首屏加载用骨架，不给已有内容盖一层转圈遮罩 -->
                        <div v-if="scanState.scanning && !scanState.keys.length" class="tree-skeleton">
                            <span v-for="row in SKELETON_WIDTHS" :key="row" class="skeleton-row" :style="{ width: `${row}%` }"></span>
                        </div>
                        <template v-else>
                            <el-tree
                                ref="keyTreeRef"
                                :data="treeState.keyTreeData"
                                :props="treeProps"
                                :indent="8"
                                node-key="key"
                                :show-checkbox="treeState.batchSelect"
                                :check-on-click-leaf="true"
                                :highlight-current="true"
                                :auto-expand-parent="false"
                                :default-expanded-keys="Array.from(treeState.keyTreeExpanded)"
                                @check="syncCheckedKeys"
                                @node-click="onTreeNodeClick"
                                @node-expand="onTreeNodeExpand"
                                @node-collapse="onTreeNodeCollapse"
                                @node-contextmenu="onRightClickNode"
                            >
                                <template #default="{ node, data }">
                                    <span class="key-node" :title="nodeTitle(data, node.label)">
                                        <SvgIcon v-if="data.type == 1" :size="15" :name="node.expanded ? 'FolderOpened' : 'Folder'" />
                                        <span v-else class="type-badge-mini" :style="{ '--badge-color': appearanceOf(data.key).color }">{{
                                            appearanceOf(data.key).badge
                                        }}</span>
                                        <span :class="['ml-1.5', data.type == 1 ? 'folder-label' : 'key-label']">{{ node.label }}</span>
                                        <span v-if="!node.isLeaf" class="node-count">{{ data.keyCount }}</span>
                                        <span v-if="ttlOf(data.key) > 0" class="node-ttl">{{ ttlText(data.key) }}</span>
                                    </span>
                                </template>
                                <template #empty>
                                    <div class="tree-empty">
                                        <SvgIcon name="Box" :size="26" />
                                        <span class="mt-1.5">{{ scanState.scanParam.match ? $t('redis.noMatchedKeys') : $t('redis.noKeys') }}</span>
                                    </div>
                                </template>
                            </el-tree>

                            <!-- 尾部：还有未扫完的数据就在列表末尾给虚线加载块 -->
                            <el-button v-if="keysHasMore" class="load-more mt-1.5" text :loading="scanState.scanning" @click="runScan(true)">
                                <SvgIcon name="ArrowDown" :size="14" />
                                <span class="ml-1">{{ $t('redis.loadMore') }}</span>
                            </el-button>
                        </template>
                    </el-scrollbar>

                    <!-- 底部状态条：常态放「分组」设置与读数，选择态原位换成选择操作，面板高度不变 -->
                    <div class="list-bar">
                        <template v-if="!treeState.batchSelect">
                            <el-dropdown trigger="click" @command="onSeparatorChange">
                                <el-button class="group-btn" text size="small" :aria-label="$t('redis.keyGroupTips')">
                                    {{ $t('redis.keyGroup') }}: {{ treeState.keySeparator || $t('redis.noGroup') }}
                                    <SvgIcon name="ArrowDown" :size="11" class="ml-1 opacity-70" />
                                </el-button>
                                <template #dropdown>
                                    <el-dropdown-menu>
                                        <el-dropdown-item command="">
                                            <span :class="{ 'font-bold': !treeState.keySeparator }">{{ $t('redis.noGroup') }}</span>
                                        </el-dropdown-item>
                                        <el-dropdown-item v-for="sep in SEPARATORS" :key="sep" :command="sep">
                                            <span class="font-mono" :class="{ 'font-bold': treeState.keySeparator === sep }">{{ sep }}</span>
                                        </el-dropdown-item>
                                    </el-dropdown-menu>
                                </template>
                            </el-dropdown>
                            <el-button class="flush-btn" text size="small" :disabled="!canOperate" @click="onFlushDb">
                                <SvgIcon name="Warning" :size="13" />
                                <span class="ml-1">{{ $t('redis.flushDb') }}</span>
                            </el-button>
                            <span class="list-status ml-auto">
                                {{ listStatusText }}
                            </span>
                        </template>
                        <template v-else>
                            <span class="text-xs text-muted-foreground">{{ $t('redis.selectedKeys', { count: selectedKeys.length }) }}</span>
                            <el-button class="shrink-0" text size="small" @click="onToggleSelectAll">
                                {{ allVisibleSelected ? $t('redis.clearSelection') : $t('common.selectAll') }}
                            </el-button>
                            <el-button
                                v-auth="PERM_DATA_DEL"
                                class="ml-auto shrink-0"
                                type="danger"
                                size="small"
                                icon="delete"
                                :disabled="!selectedKeys.length"
                                @click="onDeleteSelected"
                            >
                                {{ $t('redis.batchDeleteSelected', { count: selectedKeys.length }) }}
                            </el-button>
                        </template>
                    </div>

                    <contextmenu ref="contextmenuRef" :items="contextmenuItems" :dropdown="treeState.menuPosition" />
                </div>
            </el-splitter-panel>

            <el-splitter-panel min="380px">
                <div class="key-deatil card h-full p-2!">
                    <el-tabs v-if="hasTabs" v-model="state.activeName" class="h-full" @tab-remove="onRemoveTab">
                        <el-tab-pane v-for="tab in state.dataTabs" :key="tab.key" :name="tab.key" :label="tab.label" closable class="h-full">
                            <KeyDetail
                                :redis="redisInst"
                                :key-name="tab.key"
                                :dbs="configuredDbs"
                                :create-view="tab.createView"
                                :ttl="tab.ttl"
                                @del="onDeleteKey"
                                @renamed="onRenamed(tab, $event)"
                                @created="onCreated"
                                @refresh-tree="onRefreshKeys"
                                @open-console="onOpenConsole"
                            />
                        </el-tab-pane>

                        <!-- 命令控制台：实例 + 库级的独立 tab，由工具条「终端」打开，不挂在每个 key 的详情里 -->
                        <el-tab-pane v-if="state.consoleOpen" :name="CONSOLE_TAB" :label="$t('redis.tabConsole')" closable class="h-full">
                            <KeyConsole :redis="redisInst" :keys="scanState.keys" :hints="consoleHints" />
                        </el-tab-pane>
                    </el-tabs>
                    <div v-else class="flex h-full items-center justify-center">
                        <el-empty :description="$t('redis.selectKeyTip')" />
                    </div>
                </div>
            </el-splitter-panel>
        </el-splitter>

        <AutoFormDialog
            v-model:visible="state.newKeyDialog.visible"
            :title="$t('redis.addKey')"
            :items="newKeyItems"
            :data="state.newKeyDialog.data"
            :confirm-api="onCreateKey"
            width="520px"
        />
    </div>
</template>

<script lang="ts" setup>
import { AutoFormDialog, defineFormItems } from '@/components/auto-form';
import { copyToClipboard } from '@/common/utils/string';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import { Msg, useI18nConfirm, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { treeEvents } from '@/views/ops/resource/tree';
import { computed, onMounted, reactive, ref, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import { redisApi } from '../api';
import KeyConsole from '../keyview/KeyConsole.vue';
import KeyDetail from '../KeyDetail.vue';
import { CONSOLE_KEY_PLACEHOLDER } from '../keyview/descriptor';
import { viewAppearance } from '../keyview/appearance';
import { PERM_DATA_DEL, PERM_DATA_SAVE } from '../keyview/permission';
import { RedisInst } from '../redis';
import type { RedisOpTabApi } from './index';
import { useKeyScan } from './composables/useKeyScan';
import { treeProps, useKeyTree, type ContextmenuRef, type KeyTreeRef } from './composables/useKeyTree';

/** key 详情 tab，createView 非空表示这是「新增 key」的编辑态 */
interface RedisDataTab {
    key: string;
    label: string;
    createView?: string;
    ttl?: number;
}

const props = defineProps<{
    tabKey?: string;
    redisInfo: Record<string, unknown>;
}>();

/** 常见的 key 分隔符，走「更多」下拉切换，避免在工具条常驻一个输入框占宽 */
const SEPARATORS = [':', '/', '.', '-', '_'] as const;

/** 命令控制台 tab 的标识：不可能与真实 key 名相同，避免与 key tab 撞名 */
const CONSOLE_TAB = '__console__';

/** 首屏骨架的行宽（%），长短错落才像一列 key，避免整块等宽的死板感 */
const SKELETON_WIDTHS = [62, 78, 54, 86, 70, 46, 80];

const contextmenuRef = useTemplateRef<ContextmenuRef>('contextmenuRef');
const keyTreeRef = useTemplateRef<KeyTreeRef>('keyTreeRef');

const redisInst: RedisInst = new RedisInst();
const { t } = useI18n();

// 扫描域管数据获取、树域管渲染交互，两域不互相引用；组件保留 tab 与动作，
// 并在 runScan 里编排「扫完重渲染树」这个唯一的会合点。
// 树点击要打开详情（tab 域副作用）通过回调交还本组件，树域不感知 tab
const scan = useKeyScan();
const { state: scanState, canOperate, keysHasMore, descriptorOf } = scan;

const tree = useKeyTree({ scan, keyTreeRef, contextmenuRef, openKeyDetail: showKeyDetail });
const { state: treeState, typeFilter, allVisibleSelected, appearanceOf, nodeTitle, ttlOf, ttlText, exitBatch } = tree;
// 树的交互事件与筛选/勾选动作也由树域提供，模板直接绑定同名方法
const {
    applyTypeFilter,
    onTreeNodeClick,
    onTreeNodeExpand,
    onTreeNodeCollapse,
    onRightClickNode,
    syncCheckedKeys,
    onToggleSelectAll,
    onToggleBatch,
    onSeparatorChange,
} = tree;

// 组件私有状态只剩 tab 与弹层：扫描与树的状态分别由两个 composable 持有
const state = reactive({
    consoleOpen: false,
    activeName: '',
    dataTabs: {} as Record<string, RedisDataTab>,
    newKeyDialog: {
        visible: false,
        data: { key: '', type: 'string', ttl: -1 } as Record<string, unknown>,
    },
});

/** 控制台快捷命令的目标 key：最近一次打开的 key tab，关掉即失效（不对已消失的 key 给提示） */
const consoleKey = ref('');

/**
 * 实例配置里开放的库号：复制 key 的目标库只能从这些库里选。
 * 资源树也只渲染这些库，写成其它库的 key 会在界面上看不见也删不掉
 */
const configuredDbs = computed(() => {
    const dbs = String(props.redisInfo.db ?? '')
        .split(',')
        .map((item) => Number.parseInt(item))
        .filter((item) => !Number.isNaN(item));
    // 配置缺失时至少给出当前库，避免下拉无选项可选
    return dbs.length ? dbs : [scanState.scanParam.db ?? 0];
});

const tabKeys = computed(() => Object.keys(state.dataTabs));

/** tab 栏顺序：key tab 在前、控制台在后，关闭时取相邻 tab 也按这个顺序 */
const tabNames = computed(() => [...tabKeys.value, ...(state.consoleOpen ? [CONSOLE_TAB] : [])]);
const hasTabs = computed(() => tabNames.value.length > 0);
const selectedKeys = computed(() => treeState.checkedKeys);

/**
 * 底部读数的三态语义：浏览态「共」是整库数量，加载完两边相等；
 * 搜索时 SCAN 协议不返回匹配总数，游标未归零前只能对照全库数量防误读，
 * 归零后已加载数量就是精确匹配数
 */
const listStatusText = computed(() => {
    if (!scanState.scanParam.match?.trim()) {
        return t('redis.keysLoaded', { loaded: scanState.keys.length, total: scanState.dbsize });
    }
    if (keysHasMore.value) {
        return t('redis.keysMatchedInDb', { loaded: scanState.keys.length, total: scanState.dbsize });
    }
    return t('redis.keysMatchedAll', { total: scanState.keys.length });
});

/**
 * 命令控制台的快捷命令：模板由该 key 默认视角的描述符声明（哪些命令对这个类型有意义，
 * 只有后端处理器知道），前端只做占位符替换，不内置任何命令
 */
const consoleHints = computed(() => {
    const key = consoleKey.value;
    if (!key || !scanState.summaries[key]?.type) {
        return [];
    }
    return (descriptorOf(key)?.consoleHints ?? []).map((hint) => hint.split(CONSOLE_KEY_PLACEHOLDER).join(key));
});

/** 类型筛选项取各原生类型的默认视角，与后端注册中心同源，不额外维护枚举 */
const typeOptions = computed(() => scanState.descriptors.filter((item) => item.default));

const newKeyItems = defineFormItems<{ key: string; type: string; ttl: number }>([
    { prop: 'key', label: 'Key', required: true },
    {
        prop: 'type',
        label: 'common.type',
        type: 'select',
        options: async () => typeOptions.value.map((item) => ({ value: item.view, label: item.label })),
        props: { 'default-first-option': true },
    },
    { prop: 'ttl', label: 'redis.ttl', type: 'number', description: 'redis.ttlTips' },
]);

const contextmenuItems = [
    new ContextmenuItem('newTabOpenKey', 'redis.newTabOpen')
        .withIcon('plus')
        .withHideFunc((data: unknown) => !(data as Record<string, unknown>).isLeaf)
        .withOnClick((data: unknown) => showKeyDetail((data as Record<string, unknown>).key as string, true)),
    new ContextmenuItem('copyKey', 'common.copy')
        .withIcon('CopyDocument')
        .withHideFunc((data: unknown) => !(data as Record<string, unknown>).isLeaf)
        .withOnClick(async (data: unknown) => {
            // 复制成功提示由 copyToClipboard 统一发出，叠两条会同时弹两个 toast
            await copyToClipboard((data as Record<string, unknown>).key as string);
        }),
    new ContextmenuItem('delKey', 'common.delete')
        .withIcon('delete')
        .withPermission(PERM_DATA_DEL)
        .withHideFunc((data: unknown) => !(data as Record<string, unknown>).isLeaf)
        .withOnClick((data: unknown) => onDeleteKey((data as Record<string, unknown>).key as string)),
];

/** 扫描完成后必须重渲染树：两域不互相引用，这个唯一的会合点收在本组件 */
const runScan = async (appendKey = true) => {
    await scan.scan(appendKey);
    tree.renderKeyTree();
};

/**
 * key 数量变化后失效左侧资源树的实例节点，让 db 角标重新按 keyspace 取值。
 * 角标数据属于实例节点的子层，因此失效的是实例节点 key（= 实例 code），与 db 模块同一事件约定
 */
const invalidateTreeCount = () => {
    const code = props.redisInfo.code;
    if (code) {
        treeEvents.emit('node:invalidate', { key: String(code) });
    }
};

const onRefreshKeys = async () => {
    scanState.summaries = {};
    await runScan(false);
    invalidateTreeCount();
};

const onSearch = () => runScan(false);

const onClearMatch = () => runScan(false);

function tabLabel(key: string): string {
    return key.length > 40 ? `${key.slice(0, 40)}...` : key;
}

function showKeyDetail(key: string, newTab = false) {
    if (state.dataTabs[key]) {
        state.activeName = key;
        return;
    }
    if (!newTab) {
        // 非新 tab 的点击视为「就地切换」，只保留当前这一个详情，避免误开一堆
        delete state.dataTabs[state.activeName];
    }
    consoleKey.value = key;
    state.dataTabs[key] = { key, label: tabLabel(key) };
    state.activeName = key;
}

const onRemoveTab = (targetName: string) => {
    const index = tabNames.value.indexOf(targetName);
    if (targetName === consoleKey.value) {
        consoleKey.value = '';
    }
    if (targetName === CONSOLE_TAB) {
        state.consoleOpen = false;
    } else {
        delete state.dataTabs[targetName];
    }
    state.activeName = tabNames.value[index + 1] ?? tabNames.value[index - 1] ?? '';
};

/** 打开（或切到）命令控制台：已打开时只激活，不重复创建 */
const onOpenConsole = () => {
    state.consoleOpen = true;
    state.activeName = CONSOLE_TAB;
};

async function onShowNewKey() {
    await scan.loadDescriptors();
    state.newKeyDialog.data = { key: '', type: typeOptions.value[0]?.view ?? 'string', ttl: -1 };
    state.newKeyDialog.visible = true;
}

/** 新增 key 只登记 tab，真正的写入发生在详情里填完首个成员之后 */
async function onCreateKey(form: Record<string, unknown>) {
    const key = String(form.key ?? '').trim();
    const view = String(form.type ?? 'string');
    const ttl = Number(form.ttl ?? -1);

    delete state.dataTabs[key];
    state.dataTabs[key] = { key, label: tabLabel(key), createView: view, ttl: ttl > 0 ? ttl : undefined };
    state.activeName = key;
    state.newKeyDialog.visible = false;
    // 保存动作的提示由弹层宿主负责，这里不重复弹消息
}

/** 首个成员写入即 key 创建成功，刷新树让节点带上类型角标 */
const onCreated = async () => {
    await runScan(false);
    invalidateTreeCount();
};

const onDeleteKey = async (key: string) => {
    await useI18nDeleteConfirm(key);
    await redisApi.delKeys.request({ id: scanState.scanParam.id as number, db: scanState.scanParam.db as number, keys: [key] });
    Msg.deleteSuccess();
    onRemoveTab(key);
    await onRefreshKeys();
};

const onDeleteSelected = async () => {
    const keys = [...selectedKeys.value];
    if (!keys.length) {
        return;
    }
    await useI18nConfirm('redis.batchDeleteKeysConfirm', { count: keys.length });
    await redisApi.delKeys.request({ id: scanState.scanParam.id as number, db: scanState.scanParam.db as number, keys });
    Msg.deleteSuccess();
    keys.forEach((key) => onRemoveTab(key));
    await onRefreshKeys();
    exitBatch();
};

const onRenamed = async (tab: RedisDataTab, newKey: string) => {
    delete state.dataTabs[tab.key];
    state.dataTabs[newKey] = { key: newKey, label: tabLabel(newKey) };
    state.activeName = newKey;
    await onRefreshKeys();
};

const onFlushDb = async () => {
    // 清空整库的确认文案自己说完整（带上真实的库与 key 总数）：列表处于搜索/过滤态时只剩几十行，
    // 再套一层通用删除文案会让用户误判危险范围
    await useI18nConfirm('redis.flushDbConfirm', { db: scanState.scanParam.db, total: scanState.dbsize });
    await redisInst.runCmd(['FLUSHDB']);
    Msg.operateSuccess();
    await onRefreshKeys();
};

/**
 * 切库（或首次挂载）：扫描域与树域各自负责自己的重置（resetData/reset），
 * 组件只清 tab 与控制台——往某个域里新增状态时，不会再漏改这里的重置清单
 */
const onDbClick = async (dbInfo: Record<string, unknown>) => {
    if (!scan.applyDatabase(dbInfo)) {
        return;
    }
    scan.resetData();
    tree.reset();
    state.dataTabs = {};
    state.activeName = '';
    // 切库后控制台的目标库已经变了，直接关掉让历史随组件销毁，避免看着旧库的结果操作新库
    state.consoleOpen = false;
    consoleKey.value = '';

    redisInst.id = dbInfo.id as number;
    redisInst.db = Number.parseInt(String(dbInfo.db));

    await scan.loadDescriptors();
    await runScan(false);
};

const onRefresh = () => {
    if (canOperate.value) {
        onRefreshKeys();
    }
};

onMounted(() => {
    onDbClick(props.redisInfo);
});

defineExpose({
    onDbClick,
    onRefresh,
} satisfies RedisOpTabApi);
</script>

<style lang="scss" scoped>
@use '../keyview/toolbar.scss' as *;

.key-list {
    gap: 8px;

    // 滚动条浮在内容右缘之上，不给它留出空间就会压住节点的计数与 TTL 角标
    :deep(.el-scrollbar__view) {
        padding-right: 8px;
    }

    .search-input :deep(.el-input__wrapper) {
        border-radius: $redis-radius;
    }

    // 新增 key 收成方形图标按钮，把整行宽度让给搜索框
    .add-btn {
        width: 32px;
        padding: 0;
        border-radius: $redis-radius;
    }

    .type-filter {
        // 占位「按类型」+ 前缀图标 + 选中值都要放得下；再窄会让右侧工具条换行
        width: 120px;

        :deep(.el-select__wrapper) {
            border-radius: $redis-radius;
        }
    }

    :deep(.el-tree-node__content) {
        height: 28px;
        border-radius: 6px;
    }

    :deep(.el-tree-node__content:hover) {
        background-color: var(--el-fill-color-light);
    }

    :deep(.el-tree-node.is-current > .el-tree-node__content) {
        background-color: var(--el-color-primary-light-9);
    }

    .key-node {
        display: flex;
        width: 100%;
        min-width: 0;
        height: 28px;
        align-items: center;
        overflow: hidden;
    }

    .folder-label {
        font-weight: 600;
        color: var(--el-text-color-primary);
    }

    .key-label {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-family: var(--el-font-family-mono, ui-monospace, monospace);
        font-size: 12.5px;
        // 类型已经由彩色圆点表达，key 名用正文色保证可读性与深浅主题一致
        color: var(--el-text-color-primary);
    }

    .node-count {
        margin-left: auto;
        padding-left: 6px;
        color: var(--el-text-color-placeholder);
        font-size: 11px;
    }

    .node-ttl {
        margin-left: auto;
        padding: 0 6px;
        border-radius: 999px;
        background-color: var(--el-color-warning-light-9);
        color: var(--el-color-warning);
        font-size: 11px;
    }

    // 首屏骨架：微光扫过表示在读，比转圈遮罩更贴近列表本身的形状
    .tree-skeleton {
        display: flex;
        flex-direction: column;
        gap: 10px;
        padding: 6px 4px;

        .skeleton-row {
            height: 12px;
            border-radius: 6px;
            background: linear-gradient(90deg, var(--el-fill-color-light) 25%, var(--el-fill-color) 37%, var(--el-fill-color-light) 63%);
            background-size: 400% 100%;
            animation: key-list-shimmer 1.4s ease-in-out infinite;
        }

        // 降级要嵌在同一层选择器里，否则特异性低于上面的动画声明而盖不住
        @media (prefers-reduced-motion: reduce) {
            .skeleton-row {
                animation: none;
            }
        }
    }

    // 清空当前库：库级一次性危险动作，放在状态条并用危险色，不再收进图标菜单
    .flush-btn {
        height: 22px;
        padding: 0 4px;
        color: var(--el-color-danger);
        font-size: 12px;
    }

    .flush-btn:hover {
        color: var(--el-color-danger-dark-2);
    }

    // 分组设置放在底部状态条：文字 + 当前值，比又一个图标好懂，也不占工具条宽度
    .group-btn {
        height: 22px;
        padding: 0 4px;
        color: var(--el-text-color-secondary);
        font-size: 12px;
    }

    .tree-empty {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 2px;
        padding: 28px 0;
        color: var(--el-text-color-secondary);
        font-size: 12px;
    }
}

@keyframes key-list-shimmer {
    0% {
        background-position: 100% 50%;
    }
    100% {
        background-position: 0 50%;
    }
}

// 类型徽章：字母是主编码，颜色只用于加速扫读。
// 底色/字色/描边都由主色与主题变量混合派生，不写死白字——主色上的白字在小字号下对比度不足
.type-badge-mini {
    display: inline-flex;
    width: 20px;
    height: 15px;
    flex-shrink: 0;
    align-items: center;
    justify-content: center;
    padding: 0;
    border: 1px solid color-mix(in srgb, var(--badge-color) 40%, transparent);
    border-radius: 4px;
    background-color: color-mix(in srgb, var(--badge-color) 14%, var(--el-bg-color));
    color: color-mix(in srgb, var(--badge-color) 65%, var(--el-text-color-primary));
    font-family: var(--el-font-family-mono, ui-monospace, monospace);
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.2px;
    line-height: 13px;
}

.key-deatil {
    :deep(.el-tabs__header) {
        margin-bottom: 0;
    }

    // 页签栏与内容之间的呼吸空间加在内容侧：上面的 header margin 已被归零，
    // 不加就会与页签下划线黏在一起（key 详情与命令控制台共用这一层）
    :deep(.el-tabs__content) {
        padding-top: 10px;
    }

    :deep(.el-tabs__item) {
        height: 32px;
        padding: 0 12px;
        line-height: 32px;
    }

    :deep(.el-tabs__nav-next),
    :deep(.el-tabs__nav-prev) {
        line-height: 32px;
    }
}
</style>
