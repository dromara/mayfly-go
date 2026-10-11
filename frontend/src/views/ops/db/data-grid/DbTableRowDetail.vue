<template>
    <el-dialog v-model="visible" :title="title" width="760px">
        <div class="row-detail">
            <div v-for="item in items" :key="item.name" class="row-detail__row">
                <div class="row-detail__head">
                    <!-- 类型角标置于字段名之前，由方言把 columnType 归一为简称（12/ab/时钟图标），与数据网格表头同一真源；完整列类型悬浮才给 -->
                    <el-tooltip v-if="item.subscript" :content="item.type" placement="top">
                        <span class="row-detail__subscript">
                            <SvgIcon v-if="item.subscript === 'icon-clock'" :size="11" name="Clock" />
                            <span v-else>{{ item.subscript }}</span>
                        </span>
                    </el-tooltip>
                    <el-tag v-else-if="item.type && item.type !== '-'" size="small" type="info">{{ item.type }}</el-tag>
                    <span class="row-detail__name" :title="item.name">{{ item.name }}</span>
                    <el-tag v-if="item.isPrimaryKey" size="small">{{ $t('db.primaryKey') }}</el-tag>
                    <!-- 脱敏列的值是掩码文本，不标出来就会被当成真值复制走 -->
                    <el-tag v-if="item.masked" size="small" type="warning">{{ $t('db.maskedTag') }}</el-tag>
                    <!-- 列备注默认收起，由底部「显示列备注」开关展开 -->
                    <span v-if="showComment && item.comment" class="row-detail__comment" :title="item.comment">{{ item.comment }}</span>
                </div>

                <div class="row-detail__value">
                    <span v-if="item.nil" class="row-detail__null">NULL</span>
                    <pre v-else-if="item.multiline" class="row-detail__pre">{{ item.text }}</pre>
                    <span v-else class="row-detail__text">{{ item.text }}</span>
                </div>

                <div class="row-detail__ops">
                    <el-button v-if="item.binary && !item.nil" link type="primary" @click="openPreview(item)">
                        {{ $t('db.binaryPreviewEntry') }}
                    </el-button>
                    <el-button link type="primary" :disabled="item.nil" @click="onCopy(item)">{{ $t('common.copy') }}</el-button>
                </div>
            </div>
        </div>

        <template #footer>
            <div class="row-detail__footer">
                <el-button link type="primary" @click="showComment = !showComment">
                    {{ showComment ? $t('db.hideColumnComment') : $t('db.showColumnComment') }}
                </el-button>
                <el-button type="primary" @click="visible = false">{{ $t('common.close') }}</el-button>
            </div>
        </template>

        <BinaryValuePreview v-model:visible="previewVisible" :hex="previewHex" :column-name="previewName" />
    </el-dialog>
</template>

<script lang="ts" setup>
/**
 * 网格右键「行详情」：把一行按列纵向摊开看。
 *
 * 存在的理由（与「表单视图」不重复）：表单视图是可编辑的，且在只读库上直接隐去；
 * 这里面向的是「一列几十列、单元格被截断」时看清某一行的原始值，因此只读、不受编辑权限限制，
 * 并显式标出 NULL / 脱敏 / 主键 / 二进制——这四种情况在横向网格里都容易被看成别的东西。
 */
import { computed, ref } from 'vue';
import { copyToClipboard } from '@/common/utils/string';
import type { TableColumnDef } from '../types';
import { ColumnTypeSubscript, getDbDialect } from '../dialect/index';
import { isBinaryColumnType } from '../widgets/binaryValue';
import BinaryValuePreview from '../widgets/BinaryValuePreview.vue';
import SvgIcon from '@/components/svg-icon/index.vue';

const props = defineProps<{ columns: TableColumnDef[]; row: Record<string, unknown>; dbType?: string; title?: string }>();

const visible = defineModel<boolean>('visible', { default: false });

/** 列备注默认收起：几十列时备注会把面板撑得很长，需要时由底部开关展开 */
const showComment = ref(false);

/** 超过该长度或多行的值改用等宽块级展示，挤在一行里读不了 */
const MULTILINE_LENGTH = 80;

interface DetailItem {
    name: string;
    type: string;
    subscript: string;
    comment?: string;
    text: string;
    nil: boolean;
    masked: boolean;
    binary: boolean;
    multiline: boolean;
    isPrimaryKey: boolean;
}

const items = computed<DetailItem[]>(() => {
    const dialect = props.dbType ? getDbDialect(props.dbType) : undefined;
    return (props.columns ?? [])
        .filter((column) => column.columnName)
        .map((column) => {
            const raw = props.row?.[column.columnName];
            const nil = raw === null || raw === undefined;
            const text = nil ? '' : String(raw);
            const typeText = column.columnType || column.dataType || '';
            return {
                name: column.columnName,
                type: typeText || '-',
                // 角标由方言把 columnType 归一为 DataType 再查 ColumnTypeSubscript，与数据网格表头同一真源（不依赖调用方是否预先富化）
                subscript: dialect ? ColumnTypeSubscript[dialect.getDataType(typeText)] ?? '' : '',
                comment: column.columnComment,
                text,
                nil,
                masked: !!column.masked,
                binary: isBinaryColumnType(typeText),
                multiline: text.length > MULTILINE_LENGTH || text.includes('\n'),
                isPrimaryKey: !!column.isPrimaryKey,
            };
        });
});

const previewVisible = ref(false);
const previewHex = ref('');
const previewName = ref('');

function openPreview(item: DetailItem) {
    previewHex.value = item.text;
    previewName.value = item.name;
    previewVisible.value = true;
}

// 复制成功提示由 copyToClipboard 统一发出，这里不再叠一条
function onCopy(item: DetailItem) {
    void copyToClipboard(item.text);
}
</script>

<style lang="scss" scoped>
.row-detail {
    max-height: 60vh;
    overflow: auto;
}

.row-detail__row {
    display: grid;
    grid-template-columns: 220px 1fr auto;
    gap: 8px;
    align-items: start;
    padding: 6px 4px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    font-size: 12px;
}

.row-detail__head {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 4px;
}

.row-detail__name {
    font-weight: 600;
    word-break: break-all;
}

.row-detail__subscript {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 18px;
    height: 16px;
    padding: 0 3px;
    border-radius: 3px;
    background-color: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
    font-size: 10px;
    line-height: 1;
    cursor: default;
}

.row-detail__comment {
    color: var(--el-text-color-secondary);
    font-weight: normal;
    word-break: break-all;
}

.row-detail__footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.row-detail__value {
    min-width: 0;
}

.row-detail__null {
    color: var(--el-text-color-placeholder);
}

.row-detail__text {
    word-break: break-all;
}

.row-detail__pre {
    margin: 0;
    padding: 4px 6px;
    border-radius: 3px;
    background-color: var(--el-fill-color-lighter);
    font-family: var(--el-font-family-mono, ui-monospace, monospace);
    white-space: pre-wrap;
    word-break: break-all;
}

.row-detail__ops {
    display: flex;
    align-items: center;
    gap: 4px;
    white-space: nowrap;
}
</style>
