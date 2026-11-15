<template>
    <el-dialog v-model="visible" :title="$t('db.aiTerms.title')" width="820px" append-to-body destroy-on-close>
        <div class="flex flex-col gap-2">
            <div class="flex items-start justify-between gap-3">
                <div class="text-xs text-gray-400">{{ $t('db.aiTerms.tip') }}</div>
                <el-button type="primary" plain size="small" icon="Plus" @click="openCreate">{{ $t('db.aiTerms.add') }}</el-button>
            </div>

            <el-table v-loading="loading" :data="terms" size="small" border max-height="360" :empty-text="$t('db.aiTerms.empty')">
                <el-table-column prop="term" :label="$t('db.aiTerms.term')" min-width="130" show-overflow-tooltip />
                <el-table-column :label="$t('db.aiTerms.tables')" min-width="200">
                    <template #default="scope">
                        <span v-if="!scope.row.tables?.length" class="text-gray-400">-</span>
                        <span v-else class="font-mono text-xs">{{ scope.row.tables.join(', ') }}</span>
                    </template>
                </el-table-column>
                <el-table-column :label="$t('db.aiTerms.schema')" width="150" show-overflow-tooltip>
                    <template #default="scope">
                        <span v-if="!scope.row.schema" class="text-gray-400">{{ $t('db.aiTerms.schemaShared') }}</span>
                        <span v-else>{{ scope.row.schema }}</span>
                    </template>
                </el-table-column>
                <el-table-column prop="details" :label="$t('db.aiTerms.details')" min-width="220" show-overflow-tooltip />
                <el-table-column :label="$t('common.operation')" width="120" fixed="right">
                    <template #default="scope">
                        <el-button link type="primary" @click="edit(scope.row)">{{ $t('common.edit') }}</el-button>
                        <el-button link type="danger" @click="remove(scope.row)">{{ $t('common.delete') }}</el-button>
                    </template>
                </el-table-column>
            </el-table>

            <div class="text-xs text-gray-400">{{ $t('db.aiTerms.limitTip', { max: MAX_AI_TERMS }) }}</div>
        </div>

        <el-dialog v-model="formVisible" :title="form.id ? $t('common.edit') : $t('db.aiTerms.add')" width="560px" append-to-body>
            <el-form :model="form" label-width="96px">
                <el-form-item :label="$t('db.aiTerms.term')" required>
                    <el-input v-model="form.term" :placeholder="$t('db.aiTerms.termPlaceholder')" clearable />
                </el-form-item>
                <el-form-item :label="$t('db.aiTerms.schema')">
                    <!-- 可创建：词典要能指向本资源下的任意库，下拉只是把「当前库 / 已登记过的库 / 全库共用」放到手边 -->
                    <el-select v-model="form.schema" filterable allow-create default-first-option class="w-full" :placeholder="$t('db.aiTerms.schemaShared')">
                        <el-option :value="''" :label="$t('db.aiTerms.schemaShared')" />
                        <el-option v-for="schema in schemaOptions" :key="schema" :value="schema" :label="schema" />
                    </el-select>
                </el-form-item>
                <el-form-item :label="$t('db.aiTerms.tables')">
                    <el-select
                        v-model="form.tables"
                        multiple
                        filterable
                        allow-create
                        default-first-option
                        class="w-full"
                        :placeholder="$t('db.aiTerms.tablesTip')"
                    >
                        <el-option v-for="table in tableOptions" :key="table.tableName" :value="table.tableName" :label="tableLabel(table)" />
                    </el-select>
                    <div class="text-xs text-gray-400 mt-1">{{ $t('db.aiTerms.tablesTip') }}</div>
                </el-form-item>
                <el-form-item :label="$t('db.aiTerms.details')">
                    <el-input
                        v-model="form.details"
                        type="textarea"
                        :rows="3"
                        maxlength="1000"
                        show-word-limit
                        :placeholder="$t('db.aiTerms.detailsPlaceholder')"
                    />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="formVisible = false">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="saving" @click="save">{{ $t('common.save') }}</el-button>
            </template>
        </el-dialog>
    </el-dialog>
</template>

<script lang="ts">
/** 与后端 sqldraft.maxTermsPerDb 同值：界面上提示上限，超限仍由后端权威拒绝 */
export const MAX_AI_TERMS = 50;
</script>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from 'vue';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { aiTermApi } from './api';
import type { AiTerm } from './types';
import { DbInst } from '../db';
import type { DbTableInfo } from '../types';

const props = defineProps<{ dbId: number; dbName: string }>();
const visible = defineModel<boolean>('visible', { default: false });

const terms = ref<AiTerm[]>([]);
const loading = ref(false);
const saving = ref(false);
const formVisible = ref(false);
const tables = ref<DbTableInfo[]>([]);

const form = reactive<{ id?: number; schema: string; term: string; tables: string[]; details: string }>({
    schema: '',
    term: '',
    tables: [],
    details: '',
});

const tableOptions = computed(() => tables.value);

// 库候选：当前库 + 已登记过的其它库（同名术语在不同库口径不同时，要能一眼看到）
const schemaOptions = computed(() => {
    const names = new Set<string>();
    if (props.dbName) {
        names.add(props.dbName);
    }
    for (const item of terms.value) {
        if (item.schema) {
            names.add(item.schema);
        }
    }
    return [...names].filter((name) => name !== form.schema);
});

const tableLabel = (table: DbTableInfo) => (table.tableComment ? `${table.tableName} | ${table.tableComment}` : table.tableName);

const reset = () => {
    form.id = undefined;
    form.schema = props.dbName;
    form.term = '';
    form.tables = [];
    form.details = '';
};

const openCreate = () => {
    reset();
    formVisible.value = true;
};

const load = async () => {
    loading.value = true;
    try {
        terms.value = await aiTermApi.list.request({ dbId: props.dbId });
    } finally {
        loading.value = false;
    }
};

const edit = (row: AiTerm) => {
    form.id = row.id;
    form.schema = row.schema ?? '';
    form.term = row.term;
    form.tables = [...(row.tables ?? [])];
    form.details = row.details ?? '';
    formVisible.value = true;
};

const remove = async (row: AiTerm) => {
    if (!row.id) {
        return;
    }
    // 删掉一条口径等于让模型回到「按表名猜」，必须确认并说清删的是哪个术语
    if (!(await useI18nConfirm('db.aiTerms.deleteConfirm', { term: row.term }))) {
        return;
    }
    await aiTermApi.remove.request({ id: row.id });
    Msg.deleteSuccess();
    await load();
};

const save = async () => {
    if (!form.term.trim()) {
        Msg.warning('db.aiTerms.termRequired');
        return;
    }
    saving.value = true;
    try {
        const payload: AiTerm & { id?: number } = {
            id: form.id,
            dbId: props.dbId,
            schema: form.schema.trim(),
            term: form.term.trim(),
            tables: form.tables.map((name) => name.trim()).filter(Boolean),
            details: form.details.trim(),
        };
        if (form.id) {
            await aiTermApi.update.request(payload);
        } else {
            await aiTermApi.save.request(payload);
        }
        Msg.saveSuccess();
        formVisible.value = false;
        await load();
    } finally {
        saving.value = false;
    }
};

onMounted(async () => {
    await load();
    // 表候选来自当前库的结构缓存（与补全同源）：手输表名容易打错，而词典错了会直接带偏生成
    try {
        tables.value = (await DbInst.getInst(props.dbId).loadTables(props.dbName)) ?? [];
    } catch {
        tables.value = [];
    }
});

defineExpose({ load });
</script>
