<template>
    <div class="doc-table flex flex-col min-h-0">
        <!-- 量宽探针：与单元格同一份字号/字族（见下方 --doc-cell-font-* 变量），常驻但不可见 -->
        <span ref="probeRef" class="cell-probe font-mono" aria-hidden="true">M</span>
        <!-- 列选择：低覆盖率与超上限的字段不默认显示，但必须能被取回，否则用户以为数据丢了 -->
        <div v-if="!readonly && pickerColumns.length" class="flex items-center justify-end pb-1">
            <el-popover placement="bottom-end" trigger="click" width="260">
                <template #reference>
                    <el-button link type="primary" icon="Operation">
                        {{ $t('mongo.moreColumns', { hidden: optional.length }) }}
                    </el-button>
                </template>
                <el-checkbox-group :model-value="pinned" class="column-picker" @change="onPinnedChange">
                    <el-checkbox v-for="column in pickerColumns" :key="column.key" :value="column.key">
                        <span class="font-mono">{{ column.key }}</span>
                        <span class="ml-1 text-[12px] text-gray-400">{{ column.kinds.join('/') }} · {{ Math.round(column.coverage * 100) }}%</span>
                    </el-checkbox>
                </el-checkbox-group>
            </el-popover>
        </div>

        <el-table v-loading="loading" :data="docs" size="small" class="flex-1 min-h-0" :row-key="rowKey" @selection-change="onSelectionChange">
            <el-table-column v-if="selectable" type="selection" width="42" :reserve-selection="false" />
            <el-table-column
                v-for="column in columns"
                :key="column.key"
                :min-width="widthOf(column)"
                :align="isNumericColumn(column) ? 'right' : 'left'"
                show-overflow-tooltip
            >
                <template #header>
                    <span class="inline-flex items-center gap-1">
                        <!-- 只读结果（聚合输出等）没有可重查的条件区，列头就不该挂一个点了没反应的排序入口 -->
                        <button v-if="!readonly" type="button" class="th-sort" :title="$t('mongo.sortByColumn')" @click="emit('sort', column.key)">
                            <span class="font-mono">{{ column.key }}</span>
                            <span class="th-mark">{{ sortMark(column.key) }}</span>
                        </button>
                        <span v-else class="font-mono">{{ column.key }}</span>
                        <!-- 异构字段必须在列头看出来：同一列既有 String 又有 Number 时，等值过滤很可能漏行 -->
                        <el-tooltip v-if="column.kinds.length > 1" :content="column.kinds.join(' / ')" placement="top">
                            <span class="th-kind">{{ column.kinds.length }}</span>
                        </el-tooltip>
                    </span>
                </template>
                <template #default="{ row }">
                    <span v-if="column.key === ID_FIELD" class="inline-flex items-center gap-1 min-w-0">
                        <!-- plain 与 extjson 的单元格显示形态相同，不标出来就无法知道写回时要小心类型包装 -->
                        <el-tooltip v-if="row.mode === DOC_MODE_EXT_JSON" :content="$t('mongo.bsonRowTip')" placement="top">
                            <span class="bson-dot">BSON</span>
                        </el-tooltip>
                        <el-link v-if="!readonly" type="primary" underline="never" class="font-mono truncate" :title="idText(row)" @click="emit('open', row)">
                            {{ idText(row) || $t('mongo.noId') }}
                        </el-link>
                        <span v-else class="font-mono truncate">{{ idText(row) || $t('mongo.noId') }}</span>
                    </span>
                    <span v-else-if="isMissing(row, column.key)" class="text-gray-300">—</span>
                    <span v-else class="font-mono">{{ displayCell(columnValue(row, column.key)) }}</span>
                </template>
            </el-table-column>

            <el-table-column v-if="!readonly" :label="$t('common.operation')" width="168" fixed="right" align="center">
                <template #default="{ row }">
                    <el-button link type="primary" @click="emit('open', row)">{{ $t('common.detail') }}</el-button>
                    <el-button v-auth="perms.dataSave" link type="primary" :disabled="!row.idToken" @click="emit('edit', row)">
                        {{ $t('common.edit') }}
                    </el-button>
                    <el-button v-auth="perms.dataDel" link type="danger" :disabled="!row.idToken" @click="emit('remove', row)">
                        {{ $t('common.delete') }}
                    </el-button>
                </template>
            </el-table-column>

            <template #empty>
                <!-- 查询无结果与聚合无结果不是同一件事，前者要提示改条件 -->
                <span class="text-[13px] text-gray-400">{{ $t(readonly ? 'mongo.noResult' : 'mongo.noMatchedDoc') }}</span>
            </template>
        </el-table>
    </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue';
import { columnValue, COLUMN_MIN_WIDTH, columnWidthOf, ID_FIELD, isNumericColumn, type InferredColumn } from '../docview/schema';
import { displayCell, DOC_MODE_EXT_JSON } from '../docview/extjson';
import { docKey, idText } from '../docview/fields';
import type { MongoDoc } from '../types';
import { perms } from '../perms';

interface Props {
    docs: MongoDoc[];
    /** 默认展示的列（已含 _id 与用户固定的列） */
    columns: InferredColumn[];
    /** 被覆盖率或列数上限筛掉的列 */
    optional?: InferredColumn[];
    /** 用户固定的列名，勾选框直接受控于它 */
    pinned?: string[];
    /** 当前生效的排序文档（键序即优先级），用于列头的方向标记 */
    sort?: Record<string, unknown>;
    loading?: boolean;
    /** 显示勾选列（批量动作的前提） */
    selectable?: boolean;
    /** 只读结果（如聚合输出）：没有主键令牌，写操作入口一律不出现 */
    readonly?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
    optional: () => [],
    pinned: () => [],
    sort: () => ({}),
    loading: false,
    selectable: false,
    readonly: false,
});

const emit = defineEmits<{
    /** 打开文档详情（点主键或点「详情」） */
    open: [doc: MongoDoc];
    /** 点列头切换排序：升 → 降 → 取消，循环由父级条件区负责 */
    sort: [field: string];
    /** 固定列变化（父级据此重算列集合并决定是否重查） */
    pin: [keys: string[]];
    /** 勾选集合变化 */
    selectionChange: [docs: MongoDoc[]];
    edit: [doc: MongoDoc];
    remove: [doc: MongoDoc];
}>();

const probeRef = ref<HTMLElement>();
/** 量宽可用性计数：挂载与字体就绪各 +1，作为列宽 computed 的依赖 */
const measureSeq = ref(0);

/**
 * 量宽：canvas 按单元格那一份字体度量，不依赖布局。
 *
 * 为什么不用 DOM 探针量 offsetWidth：tab 用 v-show 常驻，隐藏期间 offsetWidth 恒为 0，
 * 切回来就会把整表算成一排最小宽（全是假省略号）。canvas 只依赖字体度量，
 * 隐藏与否结果一致；字号字族取自探针的计算样式，与单元格共用同一份 CSS 变量（见样式末尾）。
 */
const FONT_WEIGHT_CELL = '400';
const FONT_WEIGHT_HEADER = '700';

/** BSON 徽标占位：实测徽标 36 + 与主键文本的间距，留一点余量避免把值挤成省略号 */
const ID_BADGE_WIDTH = 44;

let measureCtx: CanvasRenderingContext2D | null | undefined;

function cellFont(weight: string): string {
    const probe = probeRef.value;
    if (!probe) {
        return '';
    }
    const style = window.getComputedStyle(probe);
    return `${weight} ${style.fontSize} ${style.fontFamily}`;
}

function measureWith(text: string, font: string): number {
    if (!font || measureCtx === null) {
        // measureCtx 为 null 表示这台浏览器没有 2d context，交给调用方按「量不了」处理
        return 0;
    }
    if (measureCtx === undefined) {
        measureCtx = document.createElement('canvas').getContext('2d');
        if (!measureCtx) {
            return 0;
        }
    }
    measureCtx.font = font;
    return measureCtx.measureText(text).width;
}

/**
 * 列宽表：随「文档批次 / 列集合 / 量宽可用性」重算。
 *
 * 做成 computed 而不是手写缓存，是因为量宽依赖两个只有渲染期才知道的事实：
 * 探针节点要挂载后才能读到计算样式，webfont 要就绪后度量结果才准。
 * 这两件事各推进一次 `measureSeq`，列宽就整体重算一遍；
 * 若只清缓存而不参与依赖，隐藏 tab 切回来时会一直沿用量成 0 的那批结果，整表塌成下限。
 */
const widths = computed<Record<string, number>>(() => {
    if (!measureSeq.value) {
        // 探针还没挂载，读不到单元格那份字体：先给下限，挂载后会自动重算
        return {};
    }
    const res: Record<string, number> = {};
    for (const column of props.columns) {
        res[column.key] = columnWidthOf(column, props.docs, {
            measureCell: (text) => measureWith(text, cellFont(FONT_WEIGHT_CELL)),
            // 表头是粗体，按正文字重量会偏窄，列宽就会差几像素而画省略号
            measureHeader: (text) => measureWith(text, cellFont(FONT_WEIGHT_HEADER)),
            idBadgeWidth: ID_BADGE_WIDTH,
        });
    }
    return res;
});

function widthOf(column: InferredColumn): number {
    return widths.value[column.key] ?? COLUMN_MIN_WIDTH;
}

onMounted(() => {
    measureSeq.value += 1;
    // webfont 就绪前后的度量结果不同（回退字体与真字体的字符宽度差几十像素），就绪后再算一次
    document.fonts?.ready
        .then(() => {
            measureSeq.value += 1;
        })
        .catch(() => {
            // 字体加载失败不影响布局，继续用回退字体的度量结果
        });
});

/** 可选列 = 被筛掉的列 + 用户已固定的列（后者要能被取消，否则固定容易撤回难） */
const pickerColumns = computed(() => {
    const pinnedCols = props.columns.filter((column) => props.pinned.includes(column.key));
    return [...props.optional, ...pinnedCols];
});

/** 行 key 用主键令牌：同一文档在翻页/截断后仍是同一行，DOM 可复用 */
function rowKey(row: MongoDoc): string {
    return docKey(row);
}

function isMissing(row: MongoDoc, key: string): boolean {
    return columnValue(row, key) === undefined;
}

/** 列头方向标记：未参与排序时给一个可点的空位，让人知道这里能排序 */
function sortMark(field: string): string {
    const direction = props.sort?.[field];
    if (direction === 1 || direction === 'asc' || direction === 'ascending') {
        return '↑';
    }
    if (direction === -1 || direction === 'desc' || direction === 'descending') {
        return '↓';
    }
    return direction === undefined ? '' : String(direction);
}

function onPinnedChange(keys: unknown) {
    emit('pin', (keys as string[]) ?? []);
}

function onSelectionChange(docs: MongoDoc[]) {
    emit('selectionChange', docs);
}
</script>

<style lang="scss" scoped>
.doc-table {
    .th-sort {
        display: inline-flex;
        align-items: center;
        gap: 3px;
        padding: 0;
        border: 0;
        background: none;
        color: inherit;
        font: inherit;
        cursor: pointer;

        &:hover .th-mark {
            color: var(--el-color-primary);
        }
    }

    .th-mark {
        min-width: 10px;
        color: var(--el-color-primary);
        font-weight: 700;
    }

    /*
      单元格与量宽探针共用的字体契约：字号与字族只在这一处定义，
      量宽从探针的计算样式读取，因此「量出来的」与「渲染出来的」不可能分叉。
    */
    --doc-cell-font-size: 12px;
    --doc-cell-font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;

    :deep(.el-table) {
        font-size: var(--doc-cell-font-size);
    }

    :deep(.font-mono) {
        font-family: var(--doc-cell-font-family);
    }

    /*
      主键列渲染成 el-link，Element Plus 给链接自带 14px 字号：不压回基准字号，
      这一列会比其他列大一号（且量宽口径与渲染不一致）。
    */
    :deep(.el-table .el-link) {
        font-size: var(--doc-cell-font-size);
    }

    .cell-probe {
        font-size: var(--doc-cell-font-size);
        font-family: var(--doc-cell-font-family);
        position: absolute;
        top: 0;
        left: -9999px;
        visibility: hidden;
        white-space: pre;
        pointer-events: none;
    }

    .bson-dot {
        flex-shrink: 0;
        padding: 0 4px;
        border-radius: 3px;
        background: var(--el-color-warning-light-9);
        color: var(--el-color-warning-dark-2);
        font-size: 10px;
        line-height: 14px;
    }

    .th-kind {
        padding: 0 5px;
        border-radius: 999px;
        background: var(--el-color-warning-light-9);
        color: var(--el-color-warning-dark-2);
        font-size: 11px;
        line-height: 15px;
    }

    .column-picker {
        display: flex;
        flex-direction: column;
        gap: 2px;
        max-height: 280px;
        overflow: auto;
    }

    :deep(.el-table__cell) {
        padding-top: 5px;
        padding-bottom: 5px;
    }

    :deep(.cell) {
        line-height: 18px;
    }
}
</style>
