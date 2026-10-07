<template>
    <el-dialog v-model="visible" :title="title" width="820px" :close-on-click-modal="false">
        <div class="mb-3 flex items-center gap-2">
            <el-input v-model="path" :placeholder="$t('machine.diskPathPlaceholder')" class="w-80" @keyup.enter="analyze" />
            <el-select v-model="depth" class="w-32">
                <el-option :label="$t('machine.diskDepth', { n: 1 })" :value="1" />
                <el-option :label="$t('machine.diskDepth', { n: 2 })" :value="2" />
                <el-option :label="$t('machine.diskDepth', { n: 3 })" :value="3" />
            </el-select>
            <el-button type="primary" icon="search" :loading="loading" @click="analyze">{{ $t('machine.diskAnalyze') }}</el-button>
        </div>

        <el-table v-loading="loading" :data="nodes" size="small" stripe max-height="420">
            <el-table-column prop="path" :label="$t('machine.directory')" min-width="300" show-overflow-tooltip />
            <el-table-column :label="$t('machine.size')" width="120">
                <template #default="{ row }">{{ formatByteSize(row.size) }}</template>
            </el-table-column>
            <el-table-column :label="$t('machine.proportion')" min-width="180">
                <template #default="{ row }">
                    <el-progress :percentage="pct(row.size)" :stroke-width="12" :show-text="false" />
                </template>
            </el-table-column>
        </el-table>
    </el-dialog>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue';
import { formatByteSize } from '@/common/utils/format';
import { useI18n } from 'vue-i18n';
import { diskApi } from '../api';
import type { DiskNode } from '../types';

const { t } = useI18n();

const props = defineProps<{ machineId?: number | null; machineName?: string }>();

const visible = defineModel<boolean>('visible', { default: false });

const path = ref('/');
const depth = ref(2);
const loading = ref(false);
const nodes = ref<DiskNode[]>([]);

const title = computed(() => (props.machineName ? `${t('machine.diskAnalyze')} - ${props.machineName}` : t('machine.diskAnalyze')));

// 占比以最大目录为基准（相对占比条，非磁盘使用率）。
// 字节数经 json-bigint 可能以字符串回传，而字符串的 > 是字典序比较（位数不同时结果错），必须先收敛为 number
const sizeOf = (v: unknown) => {
    const n = Number(v);
    return Number.isFinite(n) ? n : 0;
};

const maxSize = computed(() => nodes.value.reduce((m, n) => (sizeOf(n.size) > m ? sizeOf(n.size) : m), 0));
const pct = (size: number) => (maxSize.value > 0 ? Math.round((sizeOf(size) / maxSize.value) * 100) : 0);

const analyze = async () => {
    if (!props.machineId) return;
    loading.value = true;
    try {
        const res = await diskApi.analyze.request({ machineId: props.machineId, path: path.value, depth: depth.value });
        nodes.value = res?.nodes || [];
    } finally {
        loading.value = false;
    }
};

watch(visible, (val) => {
    if (val) {
        path.value = '/';
        depth.value = 2;
        nodes.value = [];
    }
});
</script>
