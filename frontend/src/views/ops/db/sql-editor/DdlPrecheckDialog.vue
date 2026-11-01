<template>
    <el-dialog v-model="visible" :title="t('db.ddlPrecheck')" width="720px" append-to-body destroy-on-close>
        <div v-loading="loading" class="min-h-32">
            <el-alert v-if="state.error" :title="state.error" type="error" :closable="false" show-icon />
            <template v-else-if="state.result">
                <div v-if="!state.result.findings.length" class="py-6 text-center text-sm text-gray-500">
                    <el-result icon="success" :title="t('db.ddlPrecheckClean')" />
                </div>
                <template v-else>
                    <div class="mb-2 text-xs text-gray-500">{{ t('db.ddlPrecheckSummary', { danger: dangerCount, warn: warnCount }) }}</div>
                    <div v-for="(finding, index) in state.result.findings" :key="index" class="mb-2 flex items-start gap-2">
                        <el-tag :type="tagType(finding.level)" size="small" disable-transitions>{{ t(`db.ddlRiskLevel_${finding.level}`) }}</el-tag>
                        <div class="text-sm">
                            <span>{{ findingText(finding) }}</span>
                            <span v-if="finding.table" class="ml-1 text-xs text-gray-400">({{ finding.table }})</span>
                        </div>
                    </div>
                    <el-divider />
                    <div class="text-xs text-gray-500 mb-1">{{ t('db.ddlPrecheckTables') }}</div>
                    <el-table :data="state.result.tables" size="small" border max-height="200">
                        <el-table-column prop="name" :label="t('db.tableName')" min-width="160" />
                        <el-table-column prop="rows" :label="t('db.ddlTableRows')" width="110" />
                        <el-table-column :label="t('db.ddlTableSize')" width="130">
                            <template #default="{ row }">{{ formatBytes(row.bytes) }}</template>
                        </el-table-column>
                        <el-table-column width="100">
                            <template #header>{{ t('db.ddlPrimaryKey') }}</template>
                            <template #default="{ row }">{{ row.hasPk ? t('common.yes') : t('db.ddlNoPk') }}</template>
                        </el-table-column>
                    </el-table>
                </template>
            </template>
        </div>
        <template #footer>
            <el-button @click="visible = false">{{ t('common.close') }}</el-button>
            <el-button type="primary" :loading="loading" @click="load">{{ t('common.refresh') }}</el-button>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
/**
 * DDL 预检结果。
 *
 * 后端只下发事实与 code（level/code/table/args），文案在这里拼：
 * 预检是提示性输出，不该把中文写进后端；同时 code 让后续支持「按规则关闭提示」成为可能。
 */
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { dbApi } from '../api';
import type { DbDdlFinding, DbDdlPrecheckResult } from '../types';

const props = defineProps<{
    modelValue: boolean;
    dbId: number;
    dbName: string;
    sql: string;
}>();

const emit = defineEmits<{ (e: 'update:modelValue', value: boolean): void }>();

const { t } = useI18n();

const visible = computed({
    get: () => props.modelValue,
    set: (value: boolean) => emit('update:modelValue', value),
});

const loading = ref(false);
// 直接用接口类型：手写一份结构会在后端加字段时静默漂移
const state = reactive<{ result: DbDdlPrecheckResult | null; error: string }>({
    result: null,
    error: '',
});

const dangerCount = computed(() => state.result?.findings.filter((item) => item.level === 'danger').length ?? 0);
const warnCount = computed(() => state.result?.findings.filter((item) => item.level === 'warn').length ?? 0);

function tagType(level: DbDdlFinding['level']) {
    if (level === 'danger') {
        return 'danger';
    }
    return level === 'warn' ? 'warning' : 'info';
}

/** 文案按 code 取，未知 code 原样显示，避免新增规则后界面空白 */
function findingText(finding: DbDdlFinding) {
    const key = `db.ddlRisk_${finding.code}`;
    return t(key, finding.args ?? {});
}

function formatBytes(bytes: number) {
    if (!bytes) {
        return '-';
    }
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let value = bytes;
    let unit = 0;
    while (value >= 1024 && unit < units.length - 1) {
        value /= 1024;
        unit++;
    }
    return `${value.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

async function load() {
    if (!props.sql.trim()) {
        state.result = null;
        state.error = t('db.noSqlToRunMsg');
        return;
    }
    loading.value = true;
    state.error = '';
    try {
        state.result = await dbApi.dbDdlPrecheck.request({ id: props.dbId, db: props.dbName, sql: props.sql });
    } catch (e: unknown) {
        state.result = null;
        state.error = e instanceof Error ? e.message : String(e);
    } finally {
        loading.value = false;
    }
}

// 打开即预检；关闭时清空，避免下次打开先闪一眼上次的结论
watch(
    () => props.modelValue,
    (open) => {
        if (open) {
            void load();
            return;
        }
        state.result = null;
        state.error = '';
    },
);
</script>
