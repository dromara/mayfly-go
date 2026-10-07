<template>
    <el-dialog v-model="visible" width="640px" :title="$t('mongo.exportTitle', { ns })" destroy-on-close :close-on-click-modal="false" @closed="onClosed">
        <div class="flex flex-col gap-3">
            <!-- 范围必须说清楚导的是「这一批」还是「这一个条件」：两者条数差几个数量级 -->
            <div>
                <div class="field-label">{{ $t('mongo.exportScope') }}</div>
                <el-radio-group v-model="scope">
                    <el-radio-button value="condition">{{ $t('mongo.exportByCondition') }}</el-radio-button>
                    <el-radio-button value="selected" :disabled="!selectedDocs.length">
                        {{ $t('mongo.exportSelected', { count: selectedDocs.length }) }}
                    </el-radio-button>
                </el-radio-group>
                <div v-if="selectedDocs.length && !selectedFilter" class="mt-1 text-[12px] text-yellow-700">
                    {{ $t('mongo.exportSelectedNoId') }}
                </div>
            </div>

            <div>
                <div class="field-label">{{ $t('mongo.exportFilter') }}</div>
                <pre class="filter-preview">{{ effectiveFilterText }}</pre>
                <span class="text-[12px] text-gray-500">{{ $t('mongo.exportFilterTip') }}</span>
            </div>

            <div>
                <div class="field-label">{{ $t('mongo.exportFormat') }}</div>
                <el-radio-group v-model="ioState.format">
                    <el-radio v-for="item in FORMATS" :key="item.value" :value="item.value">
                        {{ $t(item.labelKey) }}
                    </el-radio>
                </el-radio-group>
                <div class="text-[12px] text-gray-500">{{ $t(formatTipKey) }}</div>
            </div>

            <el-alert v-if="ioState.error" :title="ioState.error" type="error" show-icon :closable="false" />
            <el-alert
                v-else-if="ioState.exported !== null"
                :title="$t('mongo.exportDone', { count: ioState.exported })"
                type="success"
                show-icon
                :closable="false"
            />
        </div>

        <template #footer>
            <el-button @click="visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button v-auth="perms.dataExport" type="primary" :loading="ioState.exporting" :disabled="!canExport" @click="onExport">
                {{ $t('mongo.exportDo') }}
            </el-button>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue';

import { perms } from '../perms';
import { idInFilter } from '../docview/conditions';
import { useDocIO } from '../resource/composables/useDocIO';
import type { CollectionParam, ExportFormat, ExportScope, MongoDoc } from '../types';

const props = defineProps<{
    target: CollectionParam | null;
    /** 打开时带入的查询条件文本 */
    filterText: string;
    /** 已勾选的文档：非空时才提供「仅导出选中」 */
    selectedDocs: MongoDoc[];
    /** 打开时的默认范围：从底部动作条的「导出选中」进来就直接选中，不用再点一次单选 */
    defaultScope?: ExportScope;
}>();

const visible = defineModel<boolean>('visible', { default: false });

const { state: ioState, exportDocs } = useDocIO();

/** 取值与后端 mongoexport.Formats 同源，label/tip 走语言包 */
const FORMATS: { value: ExportFormat; labelKey: string; tipKey: string }[] = [
    { value: 'json', labelKey: 'mongo.exportFormatJson', tipKey: 'mongo.exportFormatJsonTip' },
    { value: 'csv', labelKey: 'mongo.exportFormatCsv', tipKey: 'mongo.exportFormatCsvTip' },
];

const scope = ref<ExportScope>('condition');

watch(visible, (opened) => {
    if (opened) {
        scope.value = props.selectedDocs.length ? (props.defaultScope ?? 'condition') : 'condition';
    }
});

const ns = computed(() => (props.target ? `${props.target.database}.${props.target.collection}` : ''));

/** 选中范围用的主键过滤器；缺 _id（被投影排除）时为 null，此时不允许按选中导出 */
const selectedFilter = computed(() => idInFilter(props.selectedDocs ?? []));

const effectiveFilter = computed(() => {
    if (scope.value === 'selected') {
        return selectedFilter.value;
    }
    const text = (props.filterText ?? '').trim();
    if (!text || text === '{}') {
        return {};
    }
    // 条件文本原样交给后端解析；这里只在非法时退回 `{}` 并在预览里显示原文，避免「点导出什么都没带回来」
    try {
        return JSON.parse(text);
    } catch {
        return {};
    }
});

const effectiveFilterText = computed(() => {
    if (scope.value === 'selected') {
        return selectedFilter.value ? JSON.stringify(selectedFilter.value) : '{}';
    }
    return (props.filterText ?? '').trim() || '{}';
});

const formatTipKey = computed(() => FORMATS.find((item) => item.value === ioState.format)?.tipKey ?? FORMATS[0].tipKey);

const canExport = computed(() => Boolean(props.target) && (scope.value === 'condition' || selectedFilter.value !== null));

async function onExport() {
    if (!props.target || !effectiveFilter.value) {
        return;
    }
    await exportDocs(props.target, effectiveFilter.value);
}

function onClosed() {
    scope.value = 'condition';
    ioState.exported = null;
    ioState.error = '';
}
</script>

<style lang="scss" scoped>
.field-label {
    margin-bottom: 4px;
    font-size: 13px;
    color: var(--el-text-color-secondary);
}

.filter-preview {
    max-height: 96px;
    margin: 0 0 4px;
    padding: 8px 10px;
    overflow: auto;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 4px;
    background: var(--el-fill-color-light);
    font-family: var(--el-font-family-mono, monospace);
    font-size: 12px;
    line-height: 1.6;
    word-break: break-all;
    white-space: pre-wrap;
}
</style>
