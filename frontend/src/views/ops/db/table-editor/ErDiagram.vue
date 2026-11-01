<template>
    <div class="er-diagram" ref="containerRef">
        <div class="er-diagram__toolbar">
            <el-button-group>
                <el-button size="small" title="放大（也可 Ctrl/⌘ + 滚轮）" @click="zoomIn">
                    <el-icon><ZoomIn /></el-icon>
                </el-button>
                <el-button size="small" title="缩小（也可 Ctrl/⌘ + 滚轮）" @click="zoomOut">
                    <el-icon><ZoomOut /></el-icon>
                </el-button>
                <el-button size="small" title="重置视图" @click="resetView">
                    <el-icon><Refresh /></el-icon>
                </el-button>
            </el-button-group>
            <el-button size="small" @click="fitView">{{ $t('db.teErAutoLayout') }}</el-button>
            <!-- 一个关键字同时匹配表名/表备注/字段名/字段备注：命中项高亮、其余淡出，回车逐个定位 -->
            <el-input
                v-model="keyword"
                class="er-diagram__search"
                size="small"
                clearable
                :placeholder="$t('db.erSearchPlaceholder')"
                @keyup.enter="jumpToNext"
            >
                <template #prefix>
                    <el-icon><Search /></el-icon>
                </template>
            </el-input>
            <span v-if="normalizedKeyword" class="er-diagram__count">
                {{ $t('db.erMatchCount', { tables: matchedTables.length, columns: matchedColumns }) }}
            </span>
            <el-checkbox v-if="normalizedKeyword" v-model="onlyMatched" size="small">{{ $t('db.erOnlyMatched') }}</el-checkbox>
            <span class="er-diagram__hint">{{ $t('db.erHint') }}</span>
        </div>
        <div
            class="er-diagram__canvas"
            ref="canvasRef"
            @wheel="handleWheel"
            @pointerdown="startPan"
            @pointermove="onPan"
            @pointerup="endPan"
            @pointercancel="endPan"
        >
            <svg :width="viewWidth" :height="viewHeight" :viewBox="viewBox">
                <g class="er-diagram__relations">
                    <line
                        v-for="(rel, idx) in relations"
                        :key="idx"
                        :x1="rel.x1"
                        :y1="rel.y1"
                        :x2="rel.x2"
                        :y2="rel.y2"
                        :stroke="rel.color"
                        stroke-width="2"
                        :stroke-dasharray="rel.dashed ? '5,5' : 'none'"
                    />
                </g>
                <g class="er-diagram__tables">
                    <g
                        v-for="table in layoutTables"
                        :key="table.name"
                        :transform="`translate(${table.x}, ${table.y})`"
                        class="er-diagram__table"
                        :class="{
                            'is-selected': selectedTable === table.name,
                            'is-matched': isTableMatched(table.name),
                            'is-dimmed': isDimmed(table.name),
                        }"
                        @click="handleTableClick(table.name)"
                        @pointerenter="onCardEnter($event, table)"
                        @pointerleave="scheduleHide"
                    >
                        <!-- 卡片高度固定：字段多的表折叠成「还有 N 个字段」，避免高矮参差 -->
                        <rect :width="CARD_WIDTH" :height="CARD_HEIGHT" rx="4" fill="white" stroke="#dcdfe6" stroke-width="1" />
                        <!-- 表头：表名 + 表备注（没有备注时不显示副标题）；表名/备注命中时表头变橙 -->
                        <rect
                            :width="CARD_WIDTH"
                            :height="HEADER_HEIGHT"
                            rx="4"
                            :fill="isTableMatched(table.name) ? '#f59e0b' : '#409eff'"
                        />
                        <text
                            :x="CARD_WIDTH / 2"
                            :y="hasTableComment(table) ? 16 : 21"
                            text-anchor="middle"
                            fill="white"
                            font-size="13"
                            font-weight="600"
                        >
                            {{ table.name }}
                        </text>
                        <!-- 有备注才显示副标题：没备注时留空，避免满屏「未设置」的噪音 -->
                        <text
                            v-if="hasTableComment(table)"
                            :x="CARD_WIDTH / 2"
                            y="30"
                            text-anchor="middle"
                            fill="#e6f2ff"
                            font-size="10"
                        >
                            <title>{{ table.comment }}</title>
                            {{ fitText(table.comment, 196, 10) }}
                        </text>
                        <!-- 字段行：每行固定 ROW_SLOT 高度；第一行「列名 + 类型」，有备注时第二行显示备注 -->
                        <g
                            v-for="(row, rowIdx) in displayRows(table)"
                            :key="rowIdx"
                            :transform="`translate(0, ${row.y})`"
                            class="er-diagram__row"
                        >
                            <rect
                                :width="CARD_WIDTH"
                                :height="ROW_SLOT"
                                :fill="row.col && isColumnMatched(table.name, row.col.index) ? '#fff3d6' : rowIdx % 2 === 0 ? '#fafafa' : 'white'"
                            />
                            <template v-if="row.col">
                                <text x="8" y="17" font-size="12" :fill="row.col.isPrimaryKey ? '#e6a23c' : '#303133'">
                                    <title>{{ row.col.name }}</title>
                                    {{ row.col.isPrimaryKey ? '🔑 ' : '' }}{{ fitText(row.col.name, PREFIX_WIDTH, 12) }}
                                </text>
                                <text :x="CARD_WIDTH - 8" y="17" text-anchor="end" font-size="10" fill="#c0c4cc">
                                    <title>{{ row.col.type }}</title>
                                    {{ fitText(row.col.type, TYPE_WIDTH, 10) }}
                                </text>
                                <text v-if="hasColumnComment(row.col)" x="8" y="33" font-size="10" fill="#909399">
                                    <title>{{ row.col.comment }}</title>
                                    {{ fitText(row.col.comment, COMMENT_WIDTH, 10) }}
                                </text>
                            </template>
                            <!-- 折叠行：剩余字段数，悬浮查看完整字段列表 -->
                            <text v-else x="8" y="24" font-size="11" fill="#909399">
                                {{ $t('db.erMoreFields', { count: row.more }) }}
                            </text>
                        </g>
                    </g>
                </g>
            </svg>
            <!-- 悬浮只补「卡片里没显示的那几个字段」，不重复卡片上已有的内容；
                 表备注在标题下完整显示（卡片里会按宽度截断） -->
            <div
                v-if="hoverCard"
                class="er-diagram__popover"
                :style="hoverCard.style"
                @pointerenter="cancelHide"
                @pointerleave="scheduleHide"
                @wheel.stop
                @pointerdown.stop
            >
                <div class="er-diagram__popover-title">
                    <span class="er-diagram__popover-name">{{ hoverCard.name }}</span>
                    <span class="er-diagram__popover-more">{{ $t('db.erMoreFieldsShort', { count: hoverCard.moreCount }) }}</span>
                </div>
                <div v-if="hoverCard.comment" class="er-diagram__popover-tablecomment">{{ hoverCard.comment }}</div>
                <div class="er-diagram__popover-body">
                    <div v-for="column in hoverCard.hiddenColumns" :key="column.name" class="er-diagram__popover-row">
                        <span class="er-diagram__popover-name">{{ column.isPrimaryKey ? '🔑 ' : '' }}{{ column.name }}</span>
                        <span class="er-diagram__popover-type">{{ column.type }}</span>
                        <span class="er-diagram__popover-desc" :title="column.comment">{{ column.comment }}</span>
                    </div>
                </div>
            </div>
            <div v-if="normalizedKeyword && !matchedTables.length" class="er-diagram__empty">
                {{ $t('db.erNoMatch') }}
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue';
import { ZoomIn, ZoomOut, Refresh, Search } from '@element-plus/icons-vue';
import type { TableDefinition } from '../types/schema';

interface Props {
    tables: TableDefinition[];
}

const props = defineProps<Props>();
const emit = defineEmits<{ (e: 'table-click', tableName: string): void }>();

const containerRef = ref<HTMLElement>();
const canvasRef = ref<HTMLElement>();
const selectedTable = ref<string>('');
const scale = ref(1);
const offsetX = ref(0);
const offsetY = ref(0);
// 画布像素尺寸：viewBox 与容器同宽高比，滚轮与拖拽的位移才能按真实像素换算
const viewWidth = ref(1200);
const viewHeight = ref(700);

const MIN_SCALE = 0.2;
const MAX_SCALE = 3;
const CARD_WIDTH = 220;
const HEADER_HEIGHT = 34;
// 每行固定占位（留出备注行的高度），行高不随有无备注变化，卡片才能等高
const ROW_SLOT = 40;
// 卡片最多显示 8 个字段，超出折叠成一行「还有 N 个字段」
const MAX_VISIBLE_ROWS = 8;
// 等高的卡片高度 = 表头 + 8 行字段 + 1 行折叠提示 + 底部留白
const CARD_HEIGHT = HEADER_HEIGHT + (MAX_VISIBLE_ROWS + 1) * ROW_SLOT + 4;
// 各文本列的可用像素宽度（卡片 220 减去左右内边距与间距）
const PREFIX_WIDTH = 118;
const TYPE_WIDTH = 62;
const COMMENT_WIDTH = 204;

const viewBox = computed(() => `${offsetX.value} ${offsetY.value} ${viewWidth.value / scale.value} ${viewHeight.value / scale.value}`);

function hasTableComment(table: TableDefinition): boolean {
    return !!table.comment?.trim();
}

function hasColumnComment(column: { comment?: string }): boolean {
    return !!column.comment?.trim();
}

/**
 * 按**像素宽度**截断文本（不是按字符数）。
 *
 * SVG 的 text 既不会自动换行也不会省略，超出部分直接画到卡片外面；而中文是全角、宽度约为 ASCII 的两倍，
 * 按「16 个字符」截断时中文实际占 160px，会盖住右侧类型并溢出 220px 的卡片。
 */
function fitText(text: string | undefined, maxWidth: number, fontSize: number): string {
    const value = (text ?? '').trim();
    if (!value) {
        return '';
    }
    let width = 0;
    let result = '';
    for (const char of value) {
        const charWidth = /[\u2e80-\u9fff\u3000-\u303f\uff00-\uffef]/.test(char) ? fontSize : fontSize * 0.55;
        if (width + charWidth > maxWidth) {
            return `${result}…`;
        }
        width += charWidth;
        result += char;
    }
    return result;
}

interface DisplayRow {
    /** 普通字段行 */
    col?: TableDefinition['columns'][number] & { index: number };
    /** 折叠行：剩余字段数 */
    more?: number;
    y: number;
}

/**
 * 卡片内要画的行：前 MAX_VISIBLE_ROWS 个字段 + 一行折叠提示。
 *
 * 固定行位（而不是按内容撑高）是等高卡片的前提；超出的字段不画在卡片里，
 * 由悬浮浮层展示完整列表，避免 15 个字段的表把整张图撑得高矮参差。
 */
function displayRows(table: TableDefinition): DisplayRow[] {
    const rows: DisplayRow[] = [];
    const shown = Math.min(table.columns.length, MAX_VISIBLE_ROWS);
    for (let idx = 0; idx < shown; idx += 1) {
        rows.push({ col: { ...table.columns[idx], index: idx }, y: HEADER_HEIGHT + idx * ROW_SLOT });
    }
    if (table.columns.length > MAX_VISIBLE_ROWS) {
        rows.push({ more: table.columns.length - MAX_VISIBLE_ROWS, y: HEADER_HEIGHT + shown * ROW_SLOT });
    }
    return rows;
}

// ---------------------------- 搜索：一个关键字命中表名/表备注/字段名/字段备注 ----------------------------

// 自动布局锁定的列数（0 = 按缩放自适应）；任何手动缩放都会解除锁定
const fitColumns = ref(0);

const keyword = ref('');
const onlyMatched = ref(false);
const normalizedKeyword = computed(() => keyword.value.trim().toLowerCase());

interface TableMatch {
    /** 命中的列下标；表名或表备注命中时为空集，但表本身仍算命中 */
    columns: Set<number>;
}

const matches = computed(() => {
    const result = new Map<string, TableMatch>();
    const kw = normalizedKeyword.value;
    if (!kw) {
        return result;
    }
    for (const table of props.tables) {
        const columns = new Set<number>();
        table.columns.forEach((column, idx) => {
            if (column.name?.toLowerCase().includes(kw) || column.comment?.toLowerCase().includes(kw)) {
                columns.add(idx);
            }
        });
        const tableHit = table.name.toLowerCase().includes(kw) || !!table.comment?.toLowerCase().includes(kw) || columns.size > 0;
        if (tableHit) {
            result.set(table.name, { columns });
        }
    }
    return result;
});

const matchedTables = computed(() => props.tables.filter((table) => matches.value.has(table.name)));
const matchedColumns = computed(() => Array.from(matches.value.values()).reduce((sum, item) => sum + item.columns.size, 0));

function isTableMatched(name: string): boolean {
    return !!normalizedKeyword.value && matches.value.has(name);
}

function isDimmed(name: string): boolean {
    return !!normalizedKeyword.value && !matches.value.has(name);
}

function isColumnMatched(name: string, columnIdx: number): boolean {
    return !!normalizedKeyword.value && !!matches.value.get(name)?.columns.has(columnIdx);
}

/** 「只看匹配」时只排匹配到的表；默认保留全量并淡出其他卡片，避免卡片凭空消失失去空间感 */
const visibleTables = computed(() => (onlyMatched.value && normalizedKeyword.value ? matchedTables.value : props.tables));

/**
 * 每行容纳的卡片数由「视口覆盖的世界宽度」决定：
 * 缩小 ⇒ 世界宽度变大 ⇒ 每行放更多；放大 ⇒ 每行放更少。
 * 若改用画布 CSS 宽度（旧实现）列数就与缩放无关，缩小时两侧会留大片空白。
 */
const columnsPerRow = computed(() => {
    // 「自动布局」会临时锁一个可读列数，避免列数与缩放互相放大到不可读
    if (fitColumns.value) {
        return fitColumns.value;
    }
    const worldWidth = viewWidth.value / scale.value;
    return Math.max(1, Math.floor((worldWidth - 80) / 280));
});

/** 网格布局：卡片等高，行距 = 卡片高 + 40 */
const layoutTables = computed(() => {
    const perRow = columnsPerRow.value;
    const source = visibleTables.value;
    return source.map((table, idx) => ({
        ...table,
        x: 40 + (idx % perRow) * 280,
        y: 40 + Math.floor(idx / perRow) * (CARD_HEIGHT + 40),
        height: CARD_HEIGHT,
    }));
});

// 计算关系线
const relations = computed(() => {
    const rels: Array<{ x1: number; y1: number; x2: number; y2: number; color: string; dashed: boolean }> = [];
    const tableMap = new Map(layoutTables.value.map((t) => [t.name, t]));

    props.tables.forEach((table) => {
        const source = tableMap.get(table.name);
        if (!source) return;

        table.constraints
            ?.filter((c) => c.type === 'FOREIGN KEY' && c.referencedTable)
            .forEach((fk) => {
                const target = tableMap.get(fk.referencedTable!);
                if (!target) return;
                rels.push({
                    x1: source.x + CARD_WIDTH / 2,
                    y1: source.y + CARD_HEIGHT,
                    x2: target.x + CARD_WIDTH / 2,
                    y2: target.y,
                    color: '#409eff',
                    dashed: false,
                });
            });
    });

    return rels;
});

function handleTableClick(name: string) {
    selectedTable.value = name;
    emit('table-click', name);
}

function zoomIn() {
    fitColumns.value = 0;
    clearTimers();
    hoverCard.value = null;
    scale.value = Math.min(scale.value * 1.2, MAX_SCALE);
}
function zoomOut() {
    fitColumns.value = 0;
    clearTimers();
    hoverCard.value = null;
    scale.value = Math.max(scale.value / 1.2, MIN_SCALE);
}
function resetView() {
    fitColumns.value = 0;
    clearTimers();
    hoverCard.value = null;
    scale.value = 1;
    offsetX.value = 0;
    offsetY.value = 0;
}

/**
 * 自动布局：解除列数锁定、按当前缩放重排，并把视口移回左上角。
 *
 * 这里刻意不改缩放：83 张表要「全部可见」需要缩到 0.1 倍以下，字完全看不清；
 * 而缩小本身已经会让每行排更多列（columnsPerRow 跟着缩放走），所以用户缩一下就是全览效果。
 */
function fitView() {
    fitColumns.value = 0;
    clearTimers();
    hoverCard.value = null;
    offsetX.value = 0;
    offsetY.value = 0;
}

// ---------------------------- 悬浮浮层：只补折叠掉的字段 ----------------------------

interface HoverCard {
    name: string;
    /** 表备注（卡片上会被截断，这里完整显示） */
    comment?: string;
    /** 卡片里没有显示的那部分字段 */
    hiddenColumns: TableDefinition['columns'];
    moreCount: number;
    style: Record<string, string>;
}

const hoverCard = ref<HoverCard | null>(null);

// 卡片与浮层之间有几像素间隙：鼠标穿过去时卡片先触发 leave，若立刻隐藏就永远移不进去。
// 所以显示/隐藏都加延迟，鼠标进入浮层时撤销隐藏。
const HOVER_SHOW_DELAY = 150;
const HOVER_HIDE_DELAY = 260;
let showTimer: ReturnType<typeof setTimeout> | null = null;
let hideTimer: ReturnType<typeof setTimeout> | null = null;

function clearTimers() {
    if (showTimer) {
        clearTimeout(showTimer);
        showTimer = null;
    }
    if (hideTimer) {
        clearTimeout(hideTimer);
        hideTimer = null;
    }
}

/** 鼠标离开卡片/浮层：延迟关闭，给「从卡片移到浮层」留出时间 */
function scheduleHide() {
    if (hideTimer) {
        clearTimeout(hideTimer);
    }
    hideTimer = setTimeout(() => {
        hoverCard.value = null;
        hideTimer = null;
    }, HOVER_HIDE_DELAY);
}

/** 鼠标进入浮层：撤销即将发生的关闭 */
function cancelHide() {
    if (hideTimer) {
        clearTimeout(hideTimer);
        hideTimer = null;
    }
}

function onCardEnter(event: PointerEvent, table: TableDefinition) {
    // 拖拽过程中不弹浮层，否则一路拖着卡片会糊满屏
    if (panning) {
        return;
    }
    clearTimers();
    const hiddenCount = table.columns.length - MAX_VISIBLE_ROWS;
    // 没有折叠字段时卡片已经显示全了，再弹一层重复内容没有意义
    if (hiddenCount <= 0) {
        hoverCard.value = null;
        return;
    }
    const canvas = canvasRef.value;
    const target = event.currentTarget as SVGGElement | null;
    if (!canvas || !target) {
        return;
    }
    showTimer = setTimeout(() => {
        showTimer = null;
        const cardRect = target.getBoundingClientRect();
        const canvasRect = canvas.getBoundingClientRect();
        const width = 340;
        // 紧贴卡片右侧（2px），右侧放不下翻到左侧
        let left = cardRect.right - canvasRect.left + 2;
        if (left + width > canvasRect.width - 4) {
            left = cardRect.left - canvasRect.left - width - 2;
        }
        // 高度取「可用空间」与上限的较小值：内容超出就在浮层内滚动，绝不超出画布被裁掉
        let top = cardRect.top - canvasRect.top;
        const available = canvasRect.height - top - 8;
        const maxHeight = Math.max(180, Math.min(420, available));
        if (top + maxHeight > canvasRect.height - 8) {
            top = canvasRect.height - maxHeight - 8;
        }
        hoverCard.value = {
            name: table.name,
            comment: table.comment,
            hiddenColumns: table.columns.slice(MAX_VISIBLE_ROWS),
            moreCount: hiddenCount,
            style: {
                left: `${Math.max(4, left)}px`,
                top: `${Math.max(4, top)}px`,
                width: `${width}px`,
                maxHeight: `${maxHeight}px`,
            },
        };
    }, HOVER_SHOW_DELAY);
}

// ---------------------------- 搜索定位 ----------------------------

let matchCursor = 0;

/** 把某张表移到视口中心（保持当前缩放） */
function centerOn(table: TableDefinition) {
    const positioned = layoutTables.value.find((item) => item.name === table.name);
    if (!positioned) {
        return;
    }
    offsetX.value = positioned.x + CARD_WIDTH / 2 - viewWidth.value / (2 * scale.value);
    offsetY.value = positioned.y + CARD_HEIGHT / 2 - viewHeight.value / (2 * scale.value);
}

/** 回车在匹配结果间循环定位，配合工具栏的匹配计数使用 */
function jumpToNext() {
    const list = matchedTables.value;
    if (!list.length) {
        return;
    }
    const target = list[matchCursor % list.length];
    matchCursor = (matchCursor + 1) % list.length;
    centerOn(target);
}

// ---------------------------- 交互：拖拽平移 + 滚轮 ----------------------------

let panning = false;
let panStartX = 0;
let panStartY = 0;
let panOriginX = 0;
let panOriginY = 0;

/**
 * 在画布空白处按下才平移；按在表节点上是选中，不该挪动画布。
 *
 * 用 pointer + setPointerCapture：拖拽过程中鼠标常常会移出弹窗范围，
 * 捕获后 move/up 仍回到画布，既不会中断平移，也不会在遮罩上合成一次点击把弹窗关掉。
 */
function startPan(e: PointerEvent) {
    if (e.button !== 0) {
        return;
    }
    if ((e.target as Element)?.closest?.('.er-diagram__table')) {
        return;
    }
    clearTimers();
    hoverCard.value = null;
    panning = true;
    panStartX = e.clientX;
    panStartY = e.clientY;
    panOriginX = offsetX.value;
    panOriginY = offsetY.value;
    canvasRef.value?.setPointerCapture?.(e.pointerId);
    e.preventDefault();
}

function onPan(e: PointerEvent) {
    if (!panning) {
        return;
    }
    // 屏幕像素 → SVG 世界坐标（除以缩放）
    offsetX.value = panOriginX - (e.clientX - panStartX) / scale.value;
    offsetY.value = panOriginY - (e.clientY - panStartY) / scale.value;
}

function endPan(e?: PointerEvent) {
    if (!panning) {
        return;
    }
    panning = false;
    if (e) {
        canvasRef.value?.releasePointerCapture?.(e.pointerId);
    }
}

/**
 * 滚轮语义与业界一致：
 * - 普通滚轮 → 上下平移
 * - Shift + 滚轮 → 左右平移
 * - Ctrl/⌘ + 滚轮 → 以鼠标位置为中心缩放
 */
function handleWheel(e: WheelEvent) {
    e.preventDefault();
    clearTimers();
    hoverCard.value = null;
    if (e.ctrlKey || e.metaKey) {
        const canvas = canvasRef.value;
        if (!canvas) {
            return;
        }
        const rect = canvas.getBoundingClientRect();
        const ratioX = (e.clientX - rect.left) / rect.width;
        const ratioY = (e.clientY - rect.top) / rect.height;
        const beforeW = viewWidth.value / scale.value;
        const beforeH = viewHeight.value / scale.value;
        const worldX = offsetX.value + ratioX * beforeW;
        const worldY = offsetY.value + ratioY * beforeH;
        const factor = e.deltaY < 0 ? 1.1 : 1 / 1.1;
        fitColumns.value = 0;
        scale.value = Math.min(Math.max(scale.value * factor, MIN_SCALE), MAX_SCALE);
        const afterW = viewWidth.value / scale.value;
        const afterH = viewHeight.value / scale.value;
        offsetX.value = worldX - ratioX * afterW;
        offsetY.value = worldY - ratioY * afterH;
        return;
    }
    const step = e.deltaY / scale.value;
    if (e.shiftKey) {
        offsetX.value += step;
    } else {
        offsetY.value += step;
    }
}

// ---------------------------- 尺寸同步 ----------------------------

let resizeObserver: ResizeObserver | null = null;

function syncCanvasSize() {
    const canvas = canvasRef.value;
    if (!canvas) {
        return;
    }
    viewWidth.value = Math.max(canvas.clientWidth, 320);
    viewHeight.value = Math.max(canvas.clientHeight, 240);
}

onMounted(() => {
    syncCanvasSize();
    if (typeof ResizeObserver !== 'undefined' && canvasRef.value) {
        resizeObserver = new ResizeObserver(syncCanvasSize);
        resizeObserver.observe(canvasRef.value);
    }
});

onBeforeUnmount(() => {
    resizeObserver?.disconnect();
    clearTimers();
    panning = false;
});
</script>

<style scoped lang="scss">
.er-diagram {
    display: flex;
    flex-direction: column;
    height: 100%;
    border: 1px solid var(--el-border-color-light);
    border-radius: 4px;
    overflow: hidden;

    &__toolbar {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 12px;
        background: var(--el-fill-color-light);
        border-bottom: 1px solid var(--el-border-color-light);
    }

    &__search {
        width: 230px;
    }

    &__count {
        font-size: 12px;
        color: #d97706;
        white-space: nowrap;
    }

    &__hint {
        margin-left: auto;
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }

    &__canvas {
        position: relative;
        flex: 1;
        overflow: hidden;
        background: #f5f7fa;
        cursor: grab;
        user-select: none;
        touch-action: none;

        &:active {
            cursor: grabbing;
        }
    }

    &__popover {
        position: absolute;
        z-index: 10;
        max-height: 360px;
        overflow: auto;
        background: white;
        border: 1px solid var(--el-border-color-light);
        border-radius: 4px;
        box-shadow: 0 6px 18px rgb(0 0 0 / 12%);
        // 必须可交互：否则鼠标从卡片移到浮层上时浮层跟着消失，根本没法滚动或复制
        pointer-events: auto;
        // 滚到边界后不要把滚动继续传给画布（画布会把滚轮当成平移）
        overscroll-behavior: contain;
    }

    &__popover-title {
        display: flex;
        align-items: baseline;
        justify-content: space-between;
        gap: 8px;
        padding: 7px 10px;
        font-size: 12px;
        font-weight: 600;
        color: #303133;
        background: #f5f7fa;
        border-bottom: 1px solid var(--el-border-color-light);
    }

    &__popover-more {
        flex: none;
        font-size: 11px;
        font-weight: 400;
        color: #d97706;
    }

    &__popover-tablecomment {
        overflow: hidden;
        padding: 5px 10px;
        font-size: 11px;
        color: #909399;
        text-overflow: ellipsis;
        white-space: nowrap;
        border-bottom: 1px dashed var(--el-border-color-lighter);
    }

    &__popover-body {
        padding: 3px 0;
    }

    &__popover-row {
        display: grid;
        grid-template-columns: 128px 78px 1fr;
        gap: 8px;
        padding: 3px 10px;
        font-size: 11px;
        line-height: 16px;
    }

    &__popover-name {
        overflow: hidden;
        color: #303133;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    &__popover-type {
        overflow: hidden;
        color: #c0c4cc;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    &__popover-desc {
        overflow: hidden;
        color: #909399;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    &__empty {
        position: absolute;
        top: 16px;
        left: 50%;
        transform: translateX(-50%);
        padding: 6px 12px;
        font-size: 12px;
        color: var(--el-text-color-secondary);
        background: white;
        border: 1px solid var(--el-border-color-light);
        border-radius: 4px;
    }

    &__table {
        cursor: pointer;

        &:hover {
            filter: brightness(0.97);
        }
        &.is-selected rect:first-of-type {
            stroke: #409eff;
            stroke-width: 2;
        }
        // 命中的表描橙色边框；未命中的淡出但不隐藏（避免卡片凭空消失让人失去空间感）
        &.is-matched rect:first-of-type {
            stroke: #f59e0b;
            stroke-width: 2;
        }
        &.is-dimmed {
            opacity: 0.2;
        }
    }
}
</style>
