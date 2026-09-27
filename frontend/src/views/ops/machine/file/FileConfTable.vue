<template>
    <div class="file-conf-table flex min-h-0 flex-col">
        <!-- 工具条：入口带文字；总数不另放读数，分页条已经带 -->
        <div class="list-bar mb-2">
            <el-button v-auth="FILE_PERM.add" size="small" type="primary" icon="Plus" :disabled="hasDraft" @click="addDraft">
                {{ $t('machine.newFileConf') }}
            </el-button>
        </div>

        <el-table :data="rows" row-key="rowKey" size="small" stripe v-loading="loading" class="min-h-0 flex-1">
            <el-table-column prop="name" :label="$t('common.name')" min-width="130">
                <template #default="scope">
                    <el-input
                        v-if="scope.row.draft"
                        v-model="scope.row.data.name"
                        size="small"
                        :placeholder="$t('common.pleaseInput', { label: $t('common.name') })"
                    />
                    <span v-else class="block truncate font-medium" :title="scope.row.data.name">{{ scope.row.data.name }}</span>
                </template>
            </el-table-column>

            <el-table-column prop="type" :label="$t('common.type')" width="130">
                <template #default="scope">
                    <EnumSelect v-if="scope.row.draft" :enums="FileTypeEnum" size="small" v-model="scope.row.data.type" />
                    <span v-else>{{ typeLabel(scope.row.data.type) }}</span>
                </template>
            </el-table-column>

            <el-table-column prop="path" :label="$t('common.path')" min-width="220">
                <template #default="scope">
                    <el-input
                        v-if="scope.row.draft"
                        v-model="scope.row.data.path"
                        size="small"
                        :placeholder="$t('common.pleaseInput', { label: $t('common.path') })"
                    />
                    <span v-else class="block truncate font-mono text-xs" :title="scope.row.data.path">{{ scope.row.data.path }}</span>
                </template>
            </el-table-column>

            <el-table-column :label="$t('common.operation')" width="180">
                <template #default="scope">
                    <div class="flex items-center gap-1.5">
                        <template v-if="scope.row.draft">
                            <el-button size="small" type="primary" @click="saveDraft">{{ $t('common.save') }}</el-button>
                            <el-button size="small" text @click="discardDraft">{{ $t('common.cancel') }}</el-button>
                        </template>
                        <template v-else>
                            <el-button size="small" @click="emit('open', scope.row.data)">{{ openLabel(scope.row.data) }}</el-button>
                            <el-button v-auth="FILE_PERM.del" size="small" type="danger" text @click="deleteRow(scope.row.data)">
                                {{ $t('common.delete') }}
                            </el-button>
                        </template>
                    </div>
                </template>
            </el-table-column>

            <!-- 空态要教下一步，而不是一句「暂无数据」 -->
            <template #empty>
                <p class="py-6 text-sm text-muted-foreground">{{ $t('machine.fileConfEmpty') }}</p>
            </template>
        </el-table>

        <el-pagination
            class="mt-2 flex justify-end"
            layout="prev, pager, next, total"
            size="small"
            :total="total"
            :page-size="pageSize"
            v-model:current-page="pageNum"
            @current-change="getFiles"
        />
    </div>
</template>

<script lang="ts" setup>
import { computed, reactive, toRefs, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import EnumSelect from '@/components/enum-select/EnumSelect.vue';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { machineApi } from '../api';
import { FileTypeEnum } from '../enums';
import { FILE_PERM } from './constants';
import type { MachineFileVO } from '../types';

const props = defineProps<{
    // 调用方的机器 id 来自 defineModel，未选机器时为 null/undefined
    machineId?: number | null;
}>();

// 打开某个配置的去向（抽屉 / tab / 内容预览）由父组件决定，本组件只管列表与增删
const emit = defineEmits<{ open: [conf: MachineFileVO] }>();

const { t } = useI18n();

/** 列表行视图模型：草稿行（尚未保存，id 为 0）与已保存行共用同一张表，用 draft 标记区分可编辑态 */
type ConfRow = { rowKey: string; draft: boolean; data: MachineFileVO };

const state = reactive({
    loading: false,
    confs: [] as MachineFileVO[],
    total: 0,
    pageNum: 1,
    pageSize: 8,
});

const { loading, total, pageNum, pageSize } = toRefs(state);

const rows = computed<ConfRow[]>(() => state.confs.map((x) => ({ rowKey: x.id ? `conf-${x.id}` : 'conf-draft', draft: !x.id, data: x })));

const hasDraft = computed(() => rows.value.some((x) => x.draft));

watch(
    () => props.machineId,
    (id) => {
        state.pageNum = 1;
        if (id) {
            getFiles();
        } else {
            state.confs = [];
            state.total = 0;
        }
    },
    { immediate: true }
);

const typeLabel = (type: number) => (type === FileTypeEnum.Directory.value ? t('machine.directory') : t('machine.file'));

/** 目录配置进文件管理器，文件配置看内容，按钮文字直接说清去向 */
const openLabel = (conf: MachineFileVO) => (conf.type === FileTypeEnum.Directory.value ? t('machine.openFileManager') : t('common.preview'));

// 必须用函数声明（提升）：下面的 watch 是 immediate 的，setup 同步阶段就要调用它，
// 写成 const 箭头函数会命中 TDZ 抛「Cannot access 'getFiles' before initialization」
async function getFiles() {
    if (!props.machineId) {
        return;
    }
    try {
        state.loading = true;
        const res = await machineApi.files.request({ id: props.machineId, pageNum: state.pageNum, pageSize: state.pageSize });
        state.confs = res.list || [];
        state.total = res.total ?? 0;
    } finally {
        state.loading = false;
    }
};

/** 新增：在表头插入一行草稿，同一时刻只允许一行草稿，「新增」按钮随之禁用 */
function addDraft() {
    if (hasDraft.value) {
        return;
    }
    const draft = { id: 0, name: '', path: '', type: FileTypeEnum.Directory.value, machineId: props.machineId ?? 0 } as MachineFileVO;
    state.confs = [draft, ...state.confs];
}

async function saveDraft() {
    const draft = state.confs.find((x) => !x.id);
    if (!draft) {
        return;
    }
    if (!draft.name || !draft.path) {
        Msg.warning('machine.fileConfRequired');
        return;
    }
    await machineApi.addConf.request({ ...draft, machineId: props.machineId ?? 0 });
    Msg.saveSuccess();
    getFiles();
}

function discardDraft() {
    state.confs = state.confs.filter((x) => x.id);
}

async function deleteRow(conf: MachineFileVO) {
    await useI18nDeleteConfirm(conf.name);
    await machineApi.delConf.request({ machineId: props.machineId, id: conf.id });
    Msg.deleteSuccess();
    getFiles();
}

// 供父组件在别处新增/删除配置后主动刷新
defineExpose({ refresh: getFiles });
</script>

<style lang="scss" scoped>
@use '@/theme/common/ops-toolbar.scss' as *;
</style>
