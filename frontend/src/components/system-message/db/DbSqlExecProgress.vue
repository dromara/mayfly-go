<template>
    <div class="w-full py-1">
        <el-row> <TagCodePath :code="progress.dbCode" /> / {{ progress.dbName }} </el-row>

        <!-- 文件名 -->
        <div class="flex items-center gap-2 mb-2 mt-2">
            <SvgIcon name="Document" :size="16" class="text-primary flex-shrink-0" />
            <span class="flex-1 text-sm font-semibold text-gray-700 dark:text-gray-200 truncate" :title="progress.title">
                {{ progress.title }}
            </span>
            <!-- 取消按钮 -->
            <el-button v-if="!progress.terminated && progress.status !== 'cancelled'" type="danger" size="small" text :loading="cancelLoading" @click="handleCancel">
                <SvgIcon name="Close" :size="14" />
                {{ $t('common.cancel') }}
            </el-button>
        </div>

        <!-- 详细信息 -->
        <el-descriptions border size="small">
            <el-descriptions-item :label="$t('db.executedStatements')">{{ progress.executedStatements }}</el-descriptions-item>
            <el-descriptions-item :label="$t('db.elapsedTime')">{{ state.elapsedTime }}</el-descriptions-item>
        </el-descriptions>
    </div>
</template>
<script lang="ts" setup>
import { onMounted, onUnmounted, reactive, ref } from 'vue';
import { formatTime } from 'element-plus/es/components/countdown/src/utils';
import TagCodePath from '@/views/ops/component/TagCodePath.vue';

const cancelLoading = ref(false);

interface Progress {
    dbCode: string;
    dbName: string;
    title: string;
    executedStatements: number;
    terminated: boolean;
    status?: string;
}

interface Props {
    progress?: Progress;
    onCancel?: () => void;
}

const props = withDefaults(defineProps<Props>(), {
    progress: () => ({
        dbCode: '',
        dbName: '',
        title: '',
        executedStatements: 0,
        terminated: false,
        status: '',
    }),
    onCancel: undefined,
});

const state = reactive({
    elapsedTime: '00:00:00',
});

let timer: ReturnType<typeof setInterval> | undefined = undefined;
const startTime = Date.now();

onMounted(async () => {
    timer = setInterval(() => {
        const elapsed = Date.now() - startTime;
        state.elapsedTime = formatTime(elapsed, 'HH:mm:ss');
    }, 1000);
});

onUnmounted(async () => {
    if (timer != undefined) {
        clearInterval(timer); // 在Vue实例销毁前，清除我们的定时器
        timer = undefined;
    }
});

// 处理取消执行
const handleCancel = () => {
    if (props.onCancel) {
        cancelLoading.value = true;
        props.onCancel();
    }
};
</script>
