<template>
    <div class="w-full py-1">
        <el-row>
            <TagCodePath :code="progress.dbCode" /> / {{ progress.dbName }}<span v-if="progress.table"> . {{ progress.table }}</span>
        </el-row>

        <!-- 文件名 -->
        <div class="flex items-center gap-2 mb-2 mt-2">
            <SvgIcon name="Document" :size="16" class="text-primary flex-shrink-0" />
            <span class="flex-1 text-sm font-semibold text-gray-700 dark:text-gray-200 truncate" :title="progress.title">
                {{ progress.title }}
            </span>
        </div>

        <el-progress :percentage="percentage" :stroke-width="8" :status="['error', 'failed', 'cancelled'].includes(progress.status || '') ? 'exception' : progress.terminated ? 'success' : ''"" />

        <!-- 详细信息 -->
        <el-descriptions border size="small" class="mt-2">
            <el-descriptions-item :label="$t('db.importedRows')">{{ progress.imported }} / {{ progress.total }}</el-descriptions-item>
            <el-descriptions-item :label="$t('db.elapsedTime')">{{ state.elapsedTime }}</el-descriptions-item>
        </el-descriptions>
    </div>
</template>
<script lang="ts" setup>
import { computed, onMounted, onUnmounted, reactive } from 'vue';
import { formatTime } from 'element-plus/es/components/countdown/src/utils';
import TagCodePath from '@/views/ops/component/TagCodePath.vue';
import SvgIcon from '@/components/svg-icon/index.vue';

interface Progress {
    dbCode: string;
    dbName: string;
    table: string;
    title: string;
    imported: number;
    total: number;
    terminated: boolean;
    status?: string;
}

const props = withDefaults(defineProps<{ progress?: Progress }>(), {
    progress: () => ({
        dbCode: '',
        dbName: '',
        table: '',
        title: '',
        imported: 0,
        total: 0,
        terminated: false,
        status: '',
    }),
});

const percentage = computed(() => {
    const { imported, total } = props.progress;
    if (!total || total <= 0) return imported > 0 ? 100 : 0;
    return Math.min(100, Math.round((imported / total) * 100));
});

const state = reactive({ elapsedTime: '00:00:00' });

let timer: ReturnType<typeof setInterval> | undefined;
const startTime = Date.now();

onMounted(() => {
    timer = setInterval(() => {
        state.elapsedTime = formatTime(Date.now() - startTime, 'HH:mm:ss');
    }, 1000);
});

onUnmounted(() => {
    if (timer != undefined) {
        clearInterval(timer);
        timer = undefined;
    }
});
</script>
