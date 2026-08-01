<template>
    <div class="w-full py-1">
        <el-row>
            <TagCodePath :code="progress.authCertName" />
        </el-row>

        <!-- 文件路径 -->
        <div v-if="progress.path" class="mb-3 px-1">
            <span class="text-xs text-gray-500 dark:text-gray-400 truncate block" :title="progress.path">
                {{ progress.path }}
            </span>
        </div>

        <!-- 文件名 -->
        <div class="flex items-center gap-2 mb-2">
            <SvgIcon name="Document" :size="16" class="text-primary flex-shrink-0" />
            <span class="flex-1 text-sm font-semibold text-gray-700 dark:text-gray-200 truncate" :title="progress.filename">
                {{ progress.filename }}
            </span>
            <!-- 取消按钮 -->
            <el-button v-if="progress.status === '' || progress.status === 'uploading'" type="danger" size="small" text :loading="cancelLoading" @click="handleCancel">
                <SvgIcon name="Close" :size="14" />
                {{ $t('common.cancel') }}
            </el-button>
        </div>

        <!-- 进度条 -->
        <div class="flex items-center gap-2 mb-3">
            <div class="flex-1">
                <el-progress :percentage="percent" :status="progressStatus" :stroke-width="10" :show-text="false" />
            </div>
            <span class="text-sm font-bold text-primary min-w-[45px] text-right"> {{ percent }}% </span>
        </div>

        <!-- 详细信息 -->
        <div class="bg-gray-50 dark:bg-gray-800 rounded-md px-3 py-2">
            <div class="flex items-center justify-between text-xs gap-4">
                <span class="flex items-center gap-1.5 text-gray-600 dark:text-gray-400 font-medium">
                    <SvgIcon name="Files" :size="14" />
                    {{ $t('components.terminal.machineFileUpload.totalSize') }}
                    <span class="font-mono font-semibold text-gray-800 dark:text-gray-200">
                        {{ formatByteSize(progress.totalSize) }}
                    </span>
                </span>

                <span class="flex items-center gap-1.5 text-gray-600 dark:text-gray-400 font-medium">
                    <SvgIcon name="Upload" :size="14" />
                    {{ $t('components.terminal.machineFileUpload.uploaded') }}
                    <span class="font-mono font-semibold text-gray-800 dark:text-gray-200">
                        {{ formatByteSize(progress.uploadedSize) }}
                    </span>
                </span>

                <span class="flex items-center gap-1.5 text-gray-600 dark:text-gray-400 font-medium">
                    <SvgIcon name="Odometer" :size="14" />
                    {{ $t('components.terminal.machineFileUpload.speed') }}
                    <span class="font-mono font-semibold text-primary">
                        {{ speed }}
                    </span>
                </span>
            </div>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { formatByteSize } from '@/common/utils/format';
import TagCodePath from '@/views/ops/component/TagCodePath.vue';
import { computed, ref } from 'vue';

const cancelLoading = ref(false);

interface Progress {
    authCertName: string; // 授权凭证名
    path: string; // 文件路径
    filename: string;
    percent: number;
    uploadedSize: number;
    totalSize: number;
    timestamp?: number; // 时间戳，用于计算速度
    status: '' | 'complete' | 'error' | 'uploading';
}

interface Props {
    progress?: Progress;
    onCancel?: () => void;
}

const props = withDefaults(defineProps<Props>(), {
    progress: () => ({
        authCertName: '',
        path: '',
        filename: '',
        percent: 0,
        uploadedSize: 0,
        totalSize: 0,
        timestamp: 0,
        status: '',
    }),
    onCancel: undefined,
});

const progressStatus = computed(() => {
    if (props.progress.status === 'complete') {
        return 'success';
    } else if (props.progress.status === 'error') {
        return 'danger';
    } else if (props.progress.status === 'uploading') {
        return 'primary';
    } else {
        return '';
    }
});

// 计算百分比
const percent = computed(() => {
    if (!props.progress.totalSize || !props.progress.uploadedSize) {
        return 0;
    }
    return Math.min(100, Math.floor((props.progress.uploadedSize / props.progress.totalSize) * 100));
});

// 计算速度
const lastTimestamp = ref(0);
const lastUploadedSize = ref(0);

const speed = computed(() => {
    if (!props.progress.timestamp || !props.progress.uploadedSize) {
        return '0 B/s';
    }

    // 首次更新，记录初始值
    if (lastTimestamp.value === 0) {
        lastTimestamp.value = props.progress.timestamp;
        lastUploadedSize.value = props.progress.uploadedSize;
        return '0 B/s';
    }

    // 计算时间差和大小差
    const timeDiff = (props.progress.timestamp - lastTimestamp.value) / 1000; // 转换为秒
    const sizeDiff = props.progress.uploadedSize - lastUploadedSize.value;

    // 更新时间戳和大小
    lastTimestamp.value = props.progress.timestamp;
    lastUploadedSize.value = props.progress.uploadedSize;

    // 计算速度
    if (timeDiff <= 0) return '0 B/s';
    const speedBytes = sizeDiff / timeDiff;

    // 格式化速度
    if (speedBytes < 1024) {
        return `${speedBytes.toFixed(0)} B/s`;
    } else if (speedBytes < 1024 * 1024) {
        return `${(speedBytes / 1024).toFixed(1)} KB/s`;
    } else {
        return `${(speedBytes / (1024 * 1024)).toFixed(1)} MB/s`;
    }
});

// 处理取消上传
const handleCancel = () => {
    if (props.onCancel) {
        cancelLoading.value = true;
        props.onCancel();
    }
};
</script>
