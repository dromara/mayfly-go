<template>
    <div class="db-schema-diff flex h-full flex-col">
        <el-form inline class="mb-1">
            <el-form-item :label="$t('db.diffSource')">
                <!-- 源库固定为打开本面板的那个库；显示其所属资源树路径，多真实库名时仍可下拉选具体库 -->
                <div class="flex flex-wrap items-center gap-2">
                    <TagCodePath v-if="srcCode" :code="srcCode" />
                    <el-select v-if="srcDbs.length > 1" v-model="state.srcDb" style="width: 200px">
                        <el-option v-for="item in srcDbs" :key="item" :label="item" :value="item" />
                    </el-select>
                    <el-tag v-else type="info" effect="plain">{{ state.srcDb }}</el-tag>
                </div>
            </el-form-item>
            <el-form-item :label="$t('db.diffTarget')">
                <!-- 目标库走统一资源树选择器：深路径由组件自行折叠到末两段（悬浮可见折叠段），
                     宽度与表清单保持一致，不再靠拉宽选择器硬撑 -->
                <DbSelectTree
                    v-model:db-id="state.dstDbId"
                    v-model:db-name="state.dstDb"
                    :placeholder="$t('db.selectDbPlaceholder')"
                    style="width: 320px"
                    @select-db="state.errorText = ''"
                />
            </el-form-item>
            <el-form-item :label="$t('db.diffTables')">
                <!-- 留空即比对两侧表名并集（有上限），上限内不选也能看出缺表 -->
                <el-select
                    v-model="state.tables"
                    multiple
                    filterable
                    collapse-tags
                    clearable
                    style="width: 320px"
                    :placeholder="$t('db.diffTablesPlaceholder')"
                >
                    <el-option v-for="table in state.sourceTables" :key="table" :label="table" :value="table" />
                </el-select>
            </el-form-item>
            <el-form-item>
                <el-button type="primary" :loading="state.loading" @click="run">{{ $t('db.diffRun') }}</el-button>
            </el-form-item>
        </el-form>

        <!-- 失败必须留在界面上：只把表格清空会被读成「比过了，没有差异」 -->
        <el-alert v-if="state.errorText" :title="state.errorText" type="error" :closable="false" show-icon class="mb-2" />

        <template v-else-if="state.result">
            <div class="mb-2 text-[12px] text-gray-500">
                {{ $t('db.diffCompared', { compared: state.result.compared, diff: state.result.diffTables }) }}
            </div>
            <el-alert v-if="state.result.note" :title="state.result.note" type="warning" :closable="false" show-icon class="mb-2" />

            <el-empty v-if="!state.result.tables?.length" :description="$t('db.diffClean')" :image-size="70" />
            <el-table v-else :data="state.result.tables" size="small" max-height="460">
                <el-table-column type="expand">
                    <template #default="{ row }">
                        <div class="px-8 py-2">
                            <div v-for="column in row.columns ?? []" :key="column.column + column.kind" class="text-[12px]">
                                <span class="text-gray-500">{{ kindText(column.kind) }}</span>
                                <span class="ml-2">{{ column.column }}</span>
                                <!-- 单侧存在的列只有一侧有值：拼上「→」会渲染出悬空箭头，读起来像截断 -->
                                <span v-if="column.source && column.target" class="ml-2 text-gray-400">{{ column.source }} → {{ column.target }}</span>
                                <span v-else class="ml-2 text-gray-400">{{ column.source || column.target }}</span>
                            </div>
                            <div v-for="(index, idx) in row.indexes ?? []" :key="'idx-' + idx" class="text-[12px]">
                                <span class="text-gray-500">{{ kindText(index.kind) }}</span>
                                <span class="ml-2">({{ index.columns }})</span>
                                <span v-if="index.source && index.target" class="ml-2 text-gray-400">{{ index.source }} → {{ index.target }}</span>
                                <span v-else class="ml-2 text-gray-400">{{ index.source || index.target }}</span>
                            </div>
                            <div v-if="row.primaryKey" class="text-[12px]">
                                <span class="text-gray-500">{{ kindText(row.primaryKey.kind) }}</span>
                                <span v-if="row.primaryKey.source && row.primaryKey.target" class="ml-2 text-gray-400"
                                    >{{ row.primaryKey.source }} → {{ row.primaryKey.target }}</span
                                >
                                <span v-else class="ml-2 text-gray-400">{{ row.primaryKey.source || row.primaryKey.target }}</span>
                            </div>
                            <!-- 「比不了」不是差异也不是一致：单独呈现，不混进差异计数 -->
                            <div v-for="(item, unIdx) in row.unavailable ?? []" :key="'un-' + unIdx" class="text-[12px] text-amber-600">
                                <span>{{ $t('db.diffUnavailable') }}</span>
                                <span class="ml-2">{{ aspectText(item.aspect) }}：{{ item.reason }}</span>
                            </div>
                        </div>
                    </template>
                </el-table-column>
                <el-table-column prop="table" :label="$t('db.table')" min-width="180" show-overflow-tooltip />
                <el-table-column :label="$t('db.diffKind')" width="160">
                    <template #default="{ row }">{{ kindText(row.kind) }}</template>
                </el-table-column>
                <el-table-column :label="$t('db.diffSummary')" min-width="240">
                    <template #default="{ row }">{{ summary(row) }}</template>
                </el-table-column>
            </el-table>
        </template>

        <el-divider content-position="left">{{ $t('db.baselineSection') }}</el-divider>

        <!-- 基线主体是源库：面板就是从这条库记录打开的，被持续跟踪的应当是它本身而不是一次性的对比对侧 -->
        <div class="mb-2 flex flex-wrap items-center gap-2">
            <el-button size="small" :loading="state.baselineSaving" @click="saveBaseline">
                {{ state.baselineRows.length ? $t('db.baselineRefresh') : $t('db.baselineSave') }}
            </el-button>
            <el-button size="small" :loading="state.driftLoading" :disabled="!state.baselineRows.length" @click="checkDrift">
                {{ $t('db.baselineCheck') }}
            </el-button>
            <el-button size="small" text type="danger" :disabled="!state.baselineRows.length" @click="clearBaseline">
                {{ $t('db.baselineClear') }}
            </el-button>
            <span class="text-[12px] text-gray-500">{{ baselineSummary }}</span>
        </div>

        <!-- 覆盖范围被截断时必须常驻：不然下一次「无漂移」会被读成整个库都没变 -->
        <el-alert v-if="state.baselineNote" :title="state.baselineNote" type="warning" :closable="false" show-icon class="mb-2" />

        <el-alert
            v-if="state.drift && !state.drift.driftedTables?.length"
            :title="$t('db.baselineNoDrift')"
            type="success"
            :closable="false"
            show-icon
            class="mb-2"
        />

        <el-table v-if="state.drift?.driftedTables?.length" :data="state.drift.driftedTables" size="small" max-height="260">
            <el-table-column prop="table" :label="$t('db.table')" min-width="180" show-overflow-tooltip />
            <el-table-column prop="detail" :label="$t('db.baselineDriftDetail')" min-width="320" show-overflow-tooltip />
        </el-table>
    </div>
</template>

<script lang="ts" setup>
/**
 * 库间结构对比（漂移检测）面板。
 *
 * 结论全部来自服务端：这里不重算任何差异，只做呈现与文案映射。
 * 面板刻意不提供「生成修复 DDL」——跨方言生成 ALTER 的每一个猜测都可能把生产库改坏，
 * 差异给人看，人再决定走哪条变更通道（那条通道上有预检与审批）。
 */
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { dbApi } from '../api';
import type { SchemaBaselineRow, SchemaDiffResult, SchemaDriftResult, SchemaTableDiff } from '../types';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { formatDate } from '@/common/utils/format';
import DbSelectTree from '../widgets/DbSelectTree.vue';
import TagCodePath from '../../component/TagCodePath.vue';

const props = defineProps<{
    srcDbId: number;
    /** 源库记录对应的真实库名清单（可能多个） */
    srcDbs: string[];
}>();

const { t } = useI18n();

/** 源库所属资源标签路径（按 srcDbId 反查 code 后渲染，让「这个库挂在哪个资源树下」可见） */
const srcCode = ref('');

const KIND_KEYS: Record<string, string> = {
    'only-in-source': 'db.diffOnlyInSource',
    'only-in-target': 'db.diffOnlyInTarget',
    changed: 'db.diffChanged',
    'column-only-in-source': 'db.diffColumnOnlyInSource',
    'column-only-in-target': 'db.diffColumnOnlyInTarget',
    'column-type-changed': 'db.diffColumnType',
    'column-size-changed': 'db.diffColumnSize',
    'column-nullable-diff': 'db.diffColumnNullable',
    'column-default-diff': 'db.diffColumnDefault',
    'column-auto-increment-diff': 'db.diffColumnAutoInc',
    'primary-key-changed': 'db.diffPrimaryKey',
    'index-only-in-source': 'db.diffIndexOnlyInSource',
    'index-only-in-target': 'db.diffIndexOnlyInTarget',
    'index-unique-diff': 'db.diffIndexUnique',
};

const ASPECT_KEYS: Record<string, string> = {
    columns: 'db.diffAspectColumns',
    'column-types': 'db.diffAspectColumnTypes',
    indexes: 'db.diffAspectIndexes',
    'primary-key': 'db.diffAspectPrimaryKey',
};

const state = reactive({
    loading: false,
    /** 对比失败的原因：留在界面上，不清空表格区就看不见 */
    errorText: '',
    result: null as SchemaDiffResult | null,
    sourceTables: [] as string[],
    srcDb: props.srcDbs[0] ?? '',
    // 目标库经资源树选择器直接落到具体库（dbId + 真实库名），不再需要候选清单与二级下拉
    dstDbId: undefined as number | undefined,
    dstDb: '',
    tables: [] as string[],
    /** 源库已有的基线行（一张表一行） */
    baselineRows: [] as SchemaBaselineRow[],
    baselineSaving: false,
    baselineNote: '',
    drift: null as SchemaDriftResult | null,
    driftLoading: false,
});

// 表清单候选属于源库：切了源库不重拉，用户选到的会是上一个库的表名
watch(
    () => state.srcDb,
    async () => {
        state.tables = [];
        state.sourceTables = [];
        state.errorText = '';
        try {
            const tables = await dbApi.tableInfos.request({ id: props.srcDbId, db: state.srcDb, limit: 500, offset: 0 });
            state.sourceTables = (tables ?? []).map((table) => table.tableName);
        } catch {
            state.sourceTables = [];
        }
    }
);

const run = async () => {
    if (!state.dstDbId || !state.dstDb) {
        state.errorText = state.dstDbId ? t('db.diffTargetUnresolved') : t('db.diffNeedTarget');
        return;
    }
    if (state.dstDbId === props.srcDbId && state.dstDb === state.srcDb) {
        state.errorText = t('db.diffSameDb');
        return;
    }

    state.loading = true;
    state.errorText = '';
    try {
        state.result = await dbApi.schemaDiff.request({
            srcDbId: props.srcDbId,
            srcDb: state.srcDb,
            dstDbId: state.dstDbId,
            dstDb: state.dstDb,
            tables: state.tables.length ? state.tables : undefined,
        });
    } catch (e: unknown) {
        // 中心层已 toast 过一次，但一次性弹窗的结果必须留在面板上，否则关掉 toast 就只剩一张空表
        state.errorText = e instanceof Error ? e.message : String(e);
        state.result = null;
    } finally {
        state.loading = false;
    }
};

function kindText(kind?: string): string {
    if (!kind) {
        return '-';
    }
    // 未知 kind 直接透出服务端原文：新增判据时这里不会静默漏翻译，看得到也能查
    return KIND_KEYS[kind] ? t(KIND_KEYS[kind]) : kind;
}

function aspectText(aspect: string): string {
    return ASPECT_KEYS[aspect] ? t(ASPECT_KEYS[aspect]) : aspect;
}

/** 概览列：差异数与「比不了」数必须分开说，合并成一个数字就等于把观测缺口算进结论里 */
function summary(row: SchemaTableDiff): string {
    const parts: string[] = [];
    const columns = row.columns?.length ?? 0;
    const indexes = row.indexes?.length ?? 0;
    if (columns) {
        parts.push(t('db.diffColumnCount', { n: columns }));
    }
    if (indexes) {
        parts.push(t('db.diffIndexCount', { n: indexes }));
    }
    if (row.primaryKey) {
        parts.push(t('db.diffPrimaryKey'));
    }
    const unavailable = row.unavailable?.length ?? 0;
    if (unavailable) {
        parts.push(t('db.diffUnavailableCount', { n: unavailable }));
    }
    return parts.join(' · ') || '-';
}

/** 基线状态一句话：没建过基线就说没建过，不拿「0 张表漂移」冒充「结构稳定」 */
const baselineSummary = computed(() => {
    if (!state.baselineRows.length) {
        return t('db.baselineNone');
    }
    const checkedAt = state.baselineRows
        .map((row) => row.checkedAt)
        .filter(Boolean)
        .sort()
        .pop();
    const drifted = state.baselineRows.filter((row) => row.drifted).length;
    const parts = [t('db.baselineRowCount', { n: state.baselineRows.length })];
    if (drifted) {
        parts.push(t('db.baselineDriftCount', { n: drifted }));
    }
    parts.push(checkedAt ? t('db.baselineCheckedAt', { time: formatDate(checkedAt) }) : t('db.baselineNeverChecked'));
    return parts.join(' · ');
});

/** 拉取已有基线行；读不到就当没建过（不弹错：面板首次加载不该因为一个可选信息报错） */
async function loadBaseline() {
    try {
        state.baselineRows = (await dbApi.listSchemaBaseline.request({ dbId: props.srcDbId, db: state.srcDb })) ?? [];
    } catch {
        state.baselineRows = [];
    }
}

async function saveBaseline() {
    state.baselineSaving = true;
    state.errorText = '';
    try {
        // 基线范围跟当前表清单选择走：选了几个表就只跟踪几个表，不隐式扩大到整库
        const result = await dbApi.saveSchemaBaseline.request({
            id: props.srcDbId,
            db: state.srcDb,
            tables: state.tables.length ? state.tables : undefined,
        });
        state.baselineNote = result?.note ?? '';
        state.drift = null;
        Msg.success(t('db.baselineSaved', { n: result?.tables ?? 0 }));
        await loadBaseline();
    } catch (e: unknown) {
        state.errorText = e instanceof Error ? e.message : String(e);
    } finally {
        state.baselineSaving = false;
    }
}

async function checkDrift() {
    state.driftLoading = true;
    state.errorText = '';
    try {
        state.drift = await dbApi.checkSchemaBaseline.request({ id: props.srcDbId, db: state.srcDb });
        state.baselineNote = state.drift?.note ?? '';
        await loadBaseline();
    } catch (e: unknown) {
        state.errorText = e instanceof Error ? e.message : String(e);
        state.drift = null;
    } finally {
        state.driftLoading = false;
    }
}

async function clearBaseline() {
    if (!(await useI18nConfirm(t('db.baselineClearConfirm'), { type: 'warning' }))) {
        return;
    }
    state.errorText = '';
    try {
        await dbApi.deleteSchemaBaseline.request({ dbId: props.srcDbId, db: state.srcDb });
        state.drift = null;
        state.baselineNote = '';
        Msg.success(t('db.baselineCleared'));
        await loadBaseline();
    } catch (e: unknown) {
        state.errorText = e instanceof Error ? e.message : String(e);
    }
}

onMounted(async () => {
    // 反查源库 code 以渲染其资源树路径；拿不到不影响面板（只是少一行来源标签）
    try {
        const dbRes = await dbApi.dbs.request({ id: props.srcDbId });
        srcCode.value = dbRes?.list?.[0]?.code ?? '';
    } catch {
        srcCode.value = '';
    }
    try {
        const tables = await dbApi.tableInfos.request({ id: props.srcDbId, db: state.srcDb, limit: 500, offset: 0 });
        state.sourceTables = (tables ?? []).map((table) => table.tableName);
    } catch {
        // 表清单读不到不影响已经填好的源库信息：留空让用户看到「没有可选项」而不是一个错误弹窗
    }
    await loadBaseline();
});

// 切了源库就要重拉基线：不重拉时下面显示的还是上一个库的表数与漂移数
watch(
    () => state.srcDb,
    async () => {
        state.drift = null;
        state.baselineNote = '';
        await loadBaseline();
    }
);
</script>
