<template>
    <div ref="panesRef" class="terminal-panes">
        <!-- 窗格：扁平渲染（key=窗格id），拆分/移动时终端实例不会被销毁重建，SSH 会话始终保持 -->
        <div
            v-for="pane of layout.panes"
            :key="pane.id"
            class="pane-leaf"
            :style="rectStyle(pane.rect)"
            @dragover="onDragover($event, pane.id)"
            @dragleave="onDragleave(pane.id)"
            @drop="onDrop($event, pane.id)"
        >
            <!-- 窗格标题栏：多窗格时显示，作为拖拽把手与关闭入口（窗格层职责，不侵入终端组件） -->
            <div v-if="paneCount > 1" class="pane-header" draggable="true" :title="t('components.terminal.paneMoveTip')" @dragstart="onPaneDragStart($event, pane.id)">
                <span class="pane-header-status" :class="paneStatusMeta(pane.id).className" :title="t(paneStatusMeta(pane.id).textKey)" />
                <span class="pane-header-title">{{ t('components.terminal.paneTitle', { id: pane.id }) }}</span>
                <span class="pane-header-actions">
                    <!-- 拆分入口：标题栏下拉，对任意类型窗格（SSH/RDP/...）通用 -->
                    <el-dropdown v-if="paneCount < maxPanes" trigger="click" placement="bottom-end" @command="(cmd: string | number | object) => onSplitCommand(pane.id, cmd)">
                        <SvgIcon
                            name="Plus"
                            :size="14"
                            class="pane-header-split pointer-icon"
                            :title="t('components.terminal.split')"
                            @click.stop
                            @dragstart.stop
                        />
                        <template #dropdown>
                            <el-dropdown-menu>
                                <el-dropdown-item v-for="direction in splitDirections" :key="direction" :command="direction">
                                    {{ t(splitTextKey(direction)) }}
                                </el-dropdown-item>
                            </el-dropdown-menu>
                        </template>
                    </el-dropdown>

                    <SvgIcon
                        name="Refresh"
                        :size="14"
                        class="pane-header-reconn pointer-icon"
                        :title="t('components.terminal.reConnTips')"
                        @click="reconnectPane(pane.id)"
                    />

                    <SvgIcon
                        name="CloseBold"
                        :size="14"
                        class="pane-header-close pointer-icon"
                        :title="t('components.terminal.closePane')"
                        @click="closePane(pane.id)"
                    />
                </span>
            </div>

            <div class="pane-body">
                <!--
                    窗格内容插槽：默认渲染 SSH 终端（TerminalBody）；
                    其他类型终端（如 RDP）通过作用域插槽接入，需实现 TerminalExpose 约定的
                    init/close/focus/fitTerminal/getStatus（getStatus 可选，缺省时不参与状态聚合）
                -->
                <slot name="pane" :pane="pane" :register="(el: unknown) => registerBody(pane.id, el)" :menu-items="buildPaneMenuItems(pane.id)" :set-status="(status: TerminalStatus) => onPaneStatusChange(pane.id, status)">
                    <TerminalBody
                        :ref="(el) => registerBody(pane.id, el)"
                        :mount-init="mountInit"
                        :cmd="cmd"
                        :socket-url="socketUrl"
                        :machine-id="machineId"
                        :auth-cert-name="authCertName"
                        :file-id="fileId"
                        :protocol="protocol"
                        :extra-menu-items="buildPaneMenuItems(pane.id)"
                        @status-change="(status) => onPaneStatusChange(pane.id, status)"
                    />
                </slot>
            </div>

            <!-- 四方向放置遮罩 -->
            <div v-if="dropTarget.id === pane.id && dropTarget.pos" class="pane-drop-mask" :class="`drop-${dropTarget.pos}`" />
        </div>

        <!-- 可拖拽分隔条 -->
        <div
            v-for="sp of layout.splits"
            :key="sp.node.id"
            class="pane-divider"
            :class="sp.node.dir === 'row' ? 'divider-row' : 'divider-col'"
            :style="dividerStyle(sp)"
            @mousedown="dividerMousedown($event, sp.node)"
        />
    </div>
</template>

<script lang="ts" setup>
import { useDebounceFn, useEventListener } from '@vueuse/core';
import {
    type PaneNode,
    type PaneSplit,
    type SplitDirection,
    countLeaves,
    getDropPosition,
    insertLeaf,
    layoutTree,
    rectStyle,
    removeLeaf,
    splitLeaf,
} from './paneTree';
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import SvgIcon from '@/components/svg-icon/index.vue';
import { ContextmenuItem } from '@/components/contextmenu';
import TerminalBody from './TerminalBody.vue';
import { TerminalStatus } from './common';

/** 分隔条拖拽时 first 节点占比上下限 */
const MIN_RATIO = 0.1;
const MAX_RATIO = 0.9;
/** 分隔条拖动中 fit 自适应防抖（ms） */
const FIT_DEBOUNCE = 100;
/** 布局变化后尺寸稳定兜底 fit 延迟（ms） */
const FIT_SETTLE_DELAY = 300;

/** 拆分方向对应的菜单图标 */
const SPLIT_MENU_ICONS: Record<SplitDirection, string> = { left: 'Back', right: 'Right', up: 'Top', down: 'Bottom' };

/** 全部拆分方向（标题栏下拉与右键菜单共用） */
const splitDirections = Object.keys(SPLIT_MENU_ICONS) as SplitDirection[];

/** 拆分菜单项文案 i18n key */
const splitTextKey = (direction: SplitDirection) => `components.terminal.split${direction[0].toUpperCase()}${direction.slice(1)}`;

/** 窗格状态严重度（越大越严重），多窗格状态聚合取最严重 */
const STATUS_SEVERITY: Record<TerminalStatus, number> = {
    [TerminalStatus.Connected]: 0,
    [TerminalStatus.NoConnected]: 1,
    [TerminalStatus.Disconnected]: 2,
    [TerminalStatus.Error]: 3,
};

// 各窗格连接状态（标题栏状态点展示用）
const paneStatuses = reactive<Record<number, TerminalStatus>>({});

const onPaneStatusChange = (paneId: number, status: TerminalStatus) => {
    paneStatuses[paneId] = status;
    emit('statusChange', status);
};

/** 窗格状态元信息（状态点颜色类 + 文案 key） */
const paneStatusMeta = (paneId: number): { className: string; textKey: string } => {
    const status = paneStatuses[paneId] ?? TerminalStatus.NoConnected;
    const meta: Record<TerminalStatus, { className: string; textKey: string }> = {
        [TerminalStatus.Connected]: { className: 'is-connected', textKey: 'components.terminal.connSuccess' },
        [TerminalStatus.NoConnected]: { className: 'is-noconnected', textKey: 'components.terminal.notConn' },
        [TerminalStatus.Disconnected]: { className: 'is-disconnected', textKey: 'components.terminal.connFail' },
        [TerminalStatus.Error]: { className: 'is-error', textKey: 'components.terminal.connError' },
    };
    return meta[status];
};

/**
 * 终端多窗格容器：递归二叉树布局（同 VS Code），渲染采用扁平化方案
 * - 右键终端可向左/右/上/下拆分出新终端窗格（每个窗格独立 WebSocket -> 独立 SSH 会话）
 * - 分隔条可拖拽调整比例；拖动窗格标题栏可移动窗格到其他窗格的四方向
 * - 布局树仅用于计算各窗格的百分比矩形，终端实例始终不销毁重建
 */
const props = withDefaults(
    defineProps<{
        /** mounted时，是否执行init方法 */
        mountInit?: boolean;
        /** 初始化执行命令 */
        cmd?: string;
        /** 连接url */
        socketUrl?: string;
        /** 机器ID（用于文件传输） */
        machineId?: number;
        /** 授权凭证名（用于文件传输） */
        authCertName?: string;
        /** 文件ID（用于文件传输） */
        fileId?: number;
        /** 协议类型（用于文件传输） */
        protocol?: number;
        /** 最大窗格数 */
        maxPanes?: number;
    }>(),
    { mountInit: true, cmd: '', socketUrl: '', machineId: 0, authCertName: '', fileId: 0, protocol: 1, maxPanes: 6 }
);

const emit = defineEmits(['statusChange']);

const { t } = useI18n();

let paneSeq = 0;
const genPaneId = () => ++paneSeq;

// 布局树根节点，初始为单个窗格
const root = ref<PaneNode>({ id: genPaneId(), type: 'leaf' });

// 布局计算：窗格矩形 + 分隔条位置
const layout = computed(() => layoutTree(root.value));

const paneCount = computed(() => layout.value.panes.length);

const panesRef = ref<HTMLElement | null>(null);

// 每个窗格的 TerminalBody 实例
const bodyRefs = new Map<number, InstanceType<typeof TerminalBody>>();

const registerBody = (id: number, el: unknown) => {
    if (el) {
        bodyRefs.set(id, el as InstanceType<typeof TerminalBody>);
    } else {
        bodyRefs.delete(id);
    }
};

const getBody = (id: number) => bodyRefs.get(id);

// 当前拖拽中的窗格（用于抑制拖到自己身上的放置遮罩）
const draggingId = ref(0);

// 当前拖拽悬停的窗格及放置方向（用于渲染放置遮罩）
const dropTarget = reactive<{ id: number; pos: SplitDirection | '' }>({ id: 0, pos: '' });

// ============ 对外操作 ============

const fitAll = () => {
    bodyRefs.forEach((body) => body.fitTerminal());
};

// 分隔条拖动过程中的防抖自适应
const debouncedFitAll = useDebounceFn(fitAll, FIT_DEBOUNCE);

const nextTickFitAll = () => {
    nextTick(() => {
        fitAll();
        // 布局稳定后再补一次，避免过渡期间尺寸不准
        setTimeout(fitAll, FIT_SETTLE_DELAY);
    });
};

/** 拆分指定窗格 */
const split = (paneId: number, direction: SplitDirection) => {
    if (paneCount.value >= props.maxPanes) {
        return;
    }
    root.value = splitLeaf(root.value, paneId, direction, genPaneId);
    nextTickFitAll();
};

/** 关闭指定窗格（最后一个窗格不可关） */
const closePane = (paneId: number) => {
    if (paneCount.value <= 1) {
        return;
    }
    const newRoot = removeLeaf(root.value, paneId);
    if (!newRoot) {
        return;
    }
    root.value = newRoot;
    // 关闭被移除窗格的连接，并释放其实例与状态
    getBody(paneId)?.close();
    bodyRefs.delete(paneId);
    delete paneStatuses[paneId];
    nextTickFitAll();
};

/** 将源窗格移动到目标窗格的指定方向 */
const movePane = (srcId: number, targetId: number, position: SplitDirection) => {
    if (srcId === targetId) {
        return;
    }
    // 先摘除源窗格，再插入到目标位置
    const rest = removeLeaf(root.value, srcId);
    if (!rest) {
        return;
    }
    root.value = insertLeaf(rest, targetId, { id: srcId, type: 'leaf' }, position, genPaneId);
    nextTickFitAll();
};

// ============ 分隔条拖拽 ============

// 当前拖拽中的分隔条监听控制器（组件卸载时中止，避免 document 监听泄漏）
let activeDividerDrag: AbortController | null = null;

/** 分隔条定位样式：居中于分裂节点的分割线上 */
const dividerStyle = (sp: { node: PaneSplit; rect: { left: number; top: number; width: number; height: number } }): Record<string, string> => {
    const { node, rect } = sp;
    if (node.dir === 'row') {
        return {
            left: `calc(${rect.left + rect.width * node.ratio}% - 3px)`,
            top: `${rect.top}%`,
            width: '6px',
            height: `${rect.height}%`,
        };
    }
    return {
        left: `${rect.left}%`,
        top: `calc(${rect.top + rect.height * node.ratio}% - 3px)`,
        width: `${rect.width}%`,
        height: '6px',
    };
};

const dividerMousedown = (event: MouseEvent, node: PaneSplit) => {
    event.preventDefault();
    const panesRect = panesRef.value?.getBoundingClientRect();
    const splitView = layout.value.splits.find((sp) => sp.node.id === node.id);
    if (!panesRect || !splitView) {
        return;
    }
    // 分裂节点区域的像素范围
    const splitPx = {
        left: panesRect.left + (splitView.rect.left / 100) * panesRect.width,
        top: panesRect.top + (splitView.rect.top / 100) * panesRect.height,
        width: (splitView.rect.width / 100) * panesRect.width,
        height: (splitView.rect.height / 100) * panesRect.height,
    };
    const isRow = node.dir === 'row';

    activeDividerDrag?.abort();
    activeDividerDrag = new AbortController();
    const { signal } = activeDividerDrag;

    const onMove = (e: MouseEvent) => {
        const ratio = isRow ? (e.clientX - splitPx.left) / splitPx.width : (e.clientY - splitPx.top) / splitPx.height;
        node.ratio = Math.min(MAX_RATIO, Math.max(MIN_RATIO, ratio));
        debouncedFitAll();
    };
    const onUp = () => {
        activeDividerDrag?.abort();
        activeDividerDrag = null;
        fitAll();
    };
    document.addEventListener('mousemove', onMove, { signal });
    document.addEventListener('mouseup', onUp, { signal });
};

onBeforeUnmount(() => activeDividerDrag?.abort());

// ============ 窗格拖拽放置 ============

/** 窗格标题栏拖拽开始：携带窗格 id 供目标窗格放置时识别 */
const onPaneDragStart = (e: DragEvent, paneId: number) => {
    e.dataTransfer?.setData('text/pane-id', String(paneId));
    e.dataTransfer && (e.dataTransfer.effectAllowed = 'move');
    draggingId.value = paneId;
};

const onDragover = (e: DragEvent, paneId: number) => {
    if (!draggingId.value || draggingId.value === paneId) {
        return;
    }
    e.preventDefault();
    e.dataTransfer && (e.dataTransfer.dropEffect = 'move');
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    dropTarget.id = paneId;
    dropTarget.pos = getDropPosition(e, rect);
};

const onDragleave = (paneId: number) => {
    if (dropTarget.id === paneId) {
        dropTarget.id = 0;
        dropTarget.pos = '';
    }
};

const onDrop = (e: DragEvent, paneId: number) => {
    e.preventDefault();
    const srcId = Number(e.dataTransfer?.getData('text/pane-id') || 0);
    const pos = dropTarget.pos;
    resetDropTarget();
    if (!srcId || !pos) {
        return;
    }
    movePane(srcId, paneId, pos);
};

// 拖拽在窗外松开/取消时，清除可能残留的放置遮罩
const resetDropTarget = () => {
    dropTarget.id = 0;
    dropTarget.pos = '';
    draggingId.value = 0;
};
useEventListener(window, 'dragend', resetDropTarget);
useEventListener(window, 'drop', resetDropTarget);

// ============ 窗格右键菜单扩展项（注入 TerminalBody，终端组件不感知窗格业务） ============

/** 组装指定窗格的右键菜单扩展项：拆分终端（二级菜单）+ 关闭窗格（多窗格时） */
const buildPaneMenuItems = (paneId: number): ContextmenuItem[] => {
    const items: ContextmenuItem[] = [
        // 四方向拆分收进二级菜单，避免一级菜单持续膨胀
        new ContextmenuItem('split', 'components.terminal.split')
            .withIcon('Grid')
            .withHideFunc(() => paneCount.value >= props.maxPanes)
            .withChildren(
                splitDirections.map((direction) =>
                    new ContextmenuItem(`split-${direction}`, splitTextKey(direction))
                        .withIcon(SPLIT_MENU_ICONS[direction])
                        .withOnClick(() => split(paneId, direction))
                )
            ),
    ];
    if (paneCount.value > 1) {
        items.push(new ContextmenuItem('closePane', 'components.terminal.closePane').withIcon('Close').withOnClick(() => closePane(paneId)));
    }
    return items;
};

/** 标题栏拆分下拉的 command 处理 */
const onSplitCommand = (paneId: number, command: string | number | object) => {
    split(paneId, command as SplitDirection);
};

/** 重连指定窗格（init 会关闭旧连接并重新建连） */
const reconnectPane = (paneId: number) => {
    getBody(paneId)?.init();
};

// 新增窗格（拆分/移动产生的新叶子）后主动建连：外层的 mountInit 仅控制首次挂载是否自动连接（如资源树终端入口由父组件控制首连），
// 不能约束后续拆分出来的新窗格，否则新窗格永远不会初始化连接
watch(
    () => layout.value.panes.map((pane) => pane.id),
    (ids, oldIds) => {
        const added = oldIds ? ids.filter((id) => !oldIds.includes(id)) : [];
        // mountInit 为 true 时新窗格挂载后会自行建连，无需重复触发（否则会关闭刚建的连接再重连一次）
        if (added.length && !props.mountInit) {
            // flush: post 保证此时新窗格实例已挂载并注册
            nextTick(() => added.forEach((id) => getBody(id)?.init()));
        }
    },
    { flush: 'post' }
);

// ============ 对外暴露（兼容原 TerminalBody 的用法） ============

const init = () => {
    bodyRefs.forEach((body) => body.init());
};

const focus = () => {
    const first = bodyRefs.values().next().value;
    first?.focus();
};

const close = () => {
    bodyRefs.forEach((body) => body.close());
    bodyRefs.clear();
};

const getStatus = (): TerminalStatus => {
    // 多窗格聚合：任一窗格异常即上报最严重状态（如任一断开则标红）
    let worst = TerminalStatus.Connected;
    bodyRefs.forEach((body) => {
        const status = body.getStatus();
        if ((STATUS_SEVERITY[status] ?? 0) > (STATUS_SEVERITY[worst] ?? 0)) {
            worst = status;
        }
    });
    return worst;
};

defineExpose({ init, close, focus, fitAll, fitTerminal: fitAll, getStatus, split, closePane, movePane });
</script>

<style lang="scss" scoped>
.terminal-panes {
    position: relative;
    display: flex;
    width: 100%;
    height: 100%;
    overflow: hidden;

    // 窗格（绝对定位，尺寸由布局树计算）
    .pane-leaf {
        position: absolute;
        display: flex;
        flex-direction: column;
        min-width: 0;
        min-height: 0;
        overflow: hidden;
    }

    // 窗格标题栏（拖拽把手），颜色全部走主题 token，明暗主题同源适配
    .pane-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 8px;
        height: 24px;
        padding: 0 8px;
        font-size: 12px;
        color: var(--el-text-color-regular);
        background: var(--el-fill-color-light);
        border-bottom: 1px solid var(--el-border-color-lighter);
        user-select: none;
        cursor: move;
        flex-shrink: 0;

        .pane-header-status {
            flex-shrink: 0;
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: var(--el-color-primary);
            cursor: help;

            &.is-connected {
                background: var(--el-color-success);
            }

            &.is-disconnected,
            &.is-error {
                background: var(--el-color-danger);
            }
        }

        .pane-header-title {
            overflow: hidden;
            white-space: nowrap;
            text-overflow: ellipsis;
        }

        .pane-header-actions {
            display: flex;
            align-items: center;
            gap: 6px;
            flex-shrink: 0;
        }

        .pane-header-split,
        .pane-header-reconn,
        .pane-header-close {
            padding: 2px;
            border-radius: var(--el-border-radius-small);
            color: var(--el-text-color-secondary);
            transition: color 150ms ease-out, background-color 150ms ease-out;
        }

        .pane-header-split:hover,
        .pane-header-reconn:hover {
            color: var(--el-color-primary);
            background: var(--el-fill-color);
        }

        .pane-header-close {
            &:hover {
                color: var(--el-color-danger);
                background: var(--el-fill-color);
            }
        }
    }

    @media (prefers-reduced-motion: reduce) {
        .pane-header .pane-header-close {
            transition: none;
        }
    }

    // 终端主体（撑满标题栏以下区域）
    .pane-body {
        display: flex;
        flex: 1;
        min-height: 0;
    }

    // 可拖拽分隔条：视觉 1px 细线（跟随主题 border token），元素本体仅作拖拽命中区
    .pane-divider {
        position: absolute;
        z-index: 5;

        // 视觉细线：居中 1px，hover/active 时高亮为主色
        &::before {
            content: '';
            position: absolute;
            background: var(--el-border-color);
            transition: background-color 150ms ease-out;
        }

        // 命中区向外扩 3px，好拖不碍眼
        &::after {
            content: '';
            position: absolute;
        }

        &.divider-row {
            cursor: col-resize;

            &::before {
                top: 0;
                bottom: 0;
                left: 50%;
                width: 1px;
                transform: translateX(-50%);
            }

            &::after {
                top: 0;
                bottom: 0;
                left: -3px;
                right: -3px;
            }
        }

        &.divider-col {
            cursor: row-resize;

            &::before {
                left: 0;
                right: 0;
                top: 50%;
                height: 1px;
                transform: translateY(-50%);
            }

            &::after {
                left: 0;
                right: 0;
                top: -3px;
                bottom: -3px;
            }
        }

        &:hover::before,
        &:active::before {
            background: var(--el-color-primary);
        }
    }

    @media (prefers-reduced-motion: reduce) {
        .pane-divider::before {
            transition: none;
        }
    }

    // 拖拽放置遮罩（四方向半区高亮，跟随主题主色）
    .pane-drop-mask {
        position: absolute;
        z-index: 10;
        pointer-events: none;
        background: color-mix(in srgb, var(--el-color-primary) 18%, transparent);
        border: 1px dashed var(--el-color-primary);

        &.drop-left {
            top: 0;
            bottom: 0;
            left: 0;
            width: 50%;
        }

        &.drop-right {
            top: 0;
            bottom: 0;
            right: 0;
            width: 50%;
        }

        &.drop-up {
            top: 0;
            right: 0;
            left: 0;
            height: 50%;
        }

        &.drop-down {
            right: 0;
            bottom: 0;
            left: 0;
            height: 50%;
        }
    }
}
</style>
