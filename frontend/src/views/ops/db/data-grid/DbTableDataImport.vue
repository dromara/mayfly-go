<template>
    <el-dialog
        v-model="visible"
        :title="$t('db.importDataTitle', { tableName })"
        width="820px"
        top="6vh"
        destroy-on-close
        :close-on-click-modal="false"
        @closed="onClosed"
    >
        <!-- 文件选择：始终提示支持的格式，避免只看到「分隔符」误以为仅支持 CSV -->
        <div class="flex flex-wrap items-center gap-3 mb-3">
            <el-upload :show-file-list="false" :auto-upload="false" :on-change="onFileChange" accept=".csv,.tsv,.txt,.xlsx,.xlsm">
                <el-button type="primary" plain size="small" icon="Upload">{{ $t('db.importSelectFile') }}</el-button>
            </el-upload>
            <el-text size="small" type="info">{{ $t('db.importFormatsHint') }}</el-text>
            <el-text v-if="state.filename" size="small" type="success">
                {{ state.filename }}（{{ $t('db.importParsedRows', { rows: state.preview?.totalRows ?? 0 }) }}）
            </el-text>
        </div>

        <!-- 解析选项：选中文件后再出现，按文件类型显示 CSV 分隔符 或 Excel 工作表 -->
        <div v-if="state.file" class="flex flex-wrap items-center gap-3 mb-3">
            <el-checkbox v-model="state.hasHeader" size="small" @change="reloadPreview">{{ $t('db.importHasHeader') }}</el-checkbox>
            <div v-if="isExcel && (state.preview?.sheets?.length ?? 0) > 1" class="flex items-center gap-1">
                <span class="text-xs text-g-6">{{ $t('db.importSheet') }}</span>
                <el-select v-model="state.sheet" size="small" style="width: 160px" @change="reloadPreview">
                    <el-option v-for="sh in state.preview?.sheets ?? []" :key="sh" :label="sh" :value="sh" />
                </el-select>
            </div>
            <div v-else-if="!isExcel" class="flex items-center gap-1">
                <span class="text-xs text-g-6">{{ $t('db.importSeparator') }}</span>
                <el-select v-model="state.separator" size="small" style="width: 90px" @change="reloadPreview">
                    <el-option label="," value="," />
                    <el-option label=";" value=";" />
                    <el-option label="\t" value="\t" />
                    <el-option label="|" value="|" />
                </el-select>
            </div>
        </div>

        <!-- 列映射 -->
        <el-table :data="mappingRows" size="small" max-height="260" border>
            <el-table-column prop="sourceLabel" :label="$t('db.importFileColumn')" width="180" show-overflow-tooltip />
            <el-table-column :label="$t('db.importSample')" min-width="160" show-overflow-tooltip>
                <template #default="{ row }">{{ row.sample }}</template>
            </el-table-column>
            <el-table-column :label="$t('db.importTargetColumn')">
                <template #default="{ row }">
                    <el-select v-model="row.target" size="small" clearable filterable :placeholder="$t('db.importSkip')">
                        <el-option v-for="col in columns" :key="col.columnName" :label="col.columnName" :value="col.columnName">
                            <span>{{ col.columnName }}</span>
                            <span class="ml-2 text-g-6 text-xs">{{ col.columnType || col.dataType }}</span>
                        </el-option>
                    </el-select>
                </template>
            </el-table-column>
        </el-table>

        <!-- 导入选项 -->
        <div class="flex flex-wrap items-center gap-4 mt-3">
            <el-checkbox v-model="state.emptyAsNull" size="small">{{ $t('db.importEmptyAsNull') }}</el-checkbox>
            <div class="flex items-center gap-1">
                <span class="text-xs text-g-6">{{ $t('db.importConflict') }}</span>
                <el-select v-model="state.duplicateStrategy" size="small" style="width: 120px">
                    <el-option :label="$t('db.importConflictInsert')" :value="-1" />
                    <el-option :label="$t('db.importConflictIgnore')" :value="1" />
                    <el-option :label="$t('db.importConflictUpdate')" :value="2" />
                </el-select>
            </div>
            <div class="flex items-center gap-1">
                <span class="text-xs text-g-6">{{ $t('db.importBatchSize') }}</span>
                <el-input-number v-model="state.batchSize" :min="1" :max="5000" size="small" controls-position="right" style="width: 120px" />
            </div>
        </div>

        <template #footer>
            <el-button @click="visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" :loading="state.importing" :disabled="!state.file" @click="onImport">
                {{ $t('db.importStart') }}
            </el-button>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
import { computed, reactive } from 'vue';
import { Msg } from '@/hooks/useI18n';
import { previewImportFile, importTableData } from '../api';
import type { ImportColumn, ImportPreviewResult } from '../types';

/** 目标表列的展示子集（兼容 TableColumnDef 与 ColumnMetadata） */
interface TargetColumn {
    columnName: string;
    dataType?: string;
    columnType?: string;
}

interface MappingRow {
    /** 传给后端的文件列标识：稳定的列位置下标字符串（避免同名表头错位） */
    source: string;
    /** 展示用列名/列序号 */
    sourceLabel: string;
    /** 首个非空样本值 */
    sample: string;
    /** 目标列名，空=跳过 */
    target: string;
}

const props = defineProps<{
    dbId: number;
    dbName: string;
    tableName: string;
    /** 目标表列元数据，用于构建目标列下拉 */
    columns: TargetColumn[];
}>();

const visible = defineModel<boolean>('visible', { default: false });
const emit = defineEmits<{ success: [] }>();

const state = reactive({
    file: null as File | null,
    filename: '',
    hasHeader: true,
    separator: ',',
    sheet: '',
    emptyAsNull: true,
    duplicateStrategy: -1,
    batchSize: 500,
    preview: null as ImportPreviewResult | null,
    importing: false,
});

const isExcel = computed(() => /\.(xlsx|xlsm)$/i.test(state.filename));

// 目标列名小写索引，供按表头名自动匹配（不区分大小写）
const targetNameIndex = computed(() => {
    const map = new Map<string, string>();
    for (const col of props.columns) map.set(col.columnName.toLowerCase(), col.columnName);
    return map;
});

/** 按表头名自动猜测目标列：文件表头名（忽略大小写）与表列名一致时预选 */
function guessTarget(headerName: string): string {
    if (!headerName) return '';
    return targetNameIndex.value.get(headerName.trim().toLowerCase()) ?? '';
}

/**
 * 映射行：以后端解析出的列位置为准（有/无表头一致），source 传下标、label/自动匹配用表头名。
 * headers 由后端归一为非空数组；此处仍对 null 做保护，避免版本错配崩溃。
 */
const mappingRows = computed<MappingRow[]>(() => {
    const preview = state.preview;
    if (!preview) return [];
    const headers = preview.headers ?? [];
    const hasHeader = headers.length > 0;
    const colCount = hasHeader ? headers.length : (preview.sampleRows[0]?.length ?? 0);
    return Array.from({ length: colCount }, (_, i) => {
        const headerName = hasHeader ? headers[i] : '';
        return {
            source: String(i),
            sourceLabel: hasHeader ? headerName || `#${i + 1}` : `#${i + 1}`,
            sample: preview.sampleRows.map((r) => r[i]).find((v) => v !== undefined && v !== '') ?? '',
            target: guessTarget(headerName),
        };
    });
});

function onFileChange(uploadFile: { raw?: File }) {
    const raw = uploadFile.raw;
    if (!raw) return;
    state.file = raw;
    state.filename = raw.name;
    reloadPreview();
}

async function reloadPreview() {
    if (!state.file) return;
    try {
        state.preview = await previewImportFile(state.file, {
            dbId: props.dbId,
            hasHeader: state.hasHeader,
            separator: isExcel.value ? undefined : state.separator,
            sheet: isExcel.value ? state.sheet || undefined : undefined,
        });
        // Excel 默认选中第一个工作表，使下拉有明确当前值（后端 sheet 为空时亦取首个，二者一致）
        if (isExcel.value && !state.sheet && state.preview?.sheets?.length) {
            state.sheet = state.preview.sheets[0];
        }
    } catch (e) {
        state.preview = null;
        Msg.error('db.importParseFail', { error: e instanceof Error ? e.message : String(e) });
    }
}

async function onImport() {
    if (!state.file) {
        Msg.warning('db.importNoFile');
        return;
    }
    const columns: ImportColumn[] = mappingRows.value.map((r) => ({ source: r.source, target: r.target }));
    if (!columns.some((c) => c.target)) {
        Msg.warning('db.importNeedMapping');
        return;
    }

    state.importing = true;
    try {
        const res = await importTableData(state.file, {
            dbId: props.dbId,
            dbName: props.dbName,
            table: props.tableName,
            columns,
            hasHeader: state.hasHeader,
            separator: isExcel.value ? undefined : state.separator,
            sheet: isExcel.value ? state.sheet || undefined : undefined,
            emptyAsNull: state.emptyAsNull,
            duplicateStrategy: state.duplicateStrategy,
            batchSize: state.batchSize,
        });
        Msg.success('db.importSuccess', { imported: res.imported, total: res.totalRows });
        // 提醒级别不阻断导入，但必须出现在操作者眼前：管理员配这一级别若只进服务端日志等于没配
        if (res.warnings?.length) {
            Msg.warning('db.importPolicyWarnings', { warnings: res.warnings.join('；') });
        }
        emit('success');
        visible.value = false;
    } catch (e) {
        // 失败详情已由请求中心层统一 toast，此处仅记录调试日志
        console.error('import table data failed', e);
    } finally {
        state.importing = false;
    }
}

function onClosed() {
    state.file = null;
    state.filename = '';
    state.preview = null;
    state.importing = false;
}
</script>

<style lang="scss" scoped>
.text-g-6 {
    color: var(--el-text-color-secondary);
}
</style>
