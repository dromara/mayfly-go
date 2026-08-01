<template>
    <div class="w-full py-2 max-w-125">
        <!-- 机器和路径信息 -->
        <el-row>
            <TagCodePath :code="progress.authCertName" />
        </el-row>

        <div v-if="progress.path" class="mb-3 px-1">
            <span class="text-xs text-gray-500 dark:text-gray-400 truncate block" :title="progress.path">
                {{ progress.path }}
            </span>
        </div>

        <!-- 文件夹信息和取消按钮 -->
        <div class="flex justify-between items-center mb-2">
            <span class="font-semibold text-sm text-gray-700 dark:text-gray-200 truncate flex-1 mr-2" :title="progress.folderName">
                {{ progress.folderName }}
            </span>
            <span class="text-xs text-gray-500 dark:text-gray-400 mr-2">
                {{ progress.uploadedFiles }}/{{ progress.totalFiles }}
                <span class="ml-1">({{ formatSize(progress.uploadedSize) }}/{{ formatSize(progress.totalSize) }})</span>
            </span>
            <!-- 取消按钮 -->
            <el-button
                v-if="progress.status === '' || progress.status === 'uploading'"
                type="danger"
                size="small"
                text
                :loading="cancelLoading"
                @click="handleCancel"
            >
                {{ $t('common.cancel') }}
            </el-button>
        </div>

        <!-- 所有文件列表 -->
        <div v-if="fileList.length > 0" class="mt-3 pt-3 border-t border-gray-200 dark:border-gray-700">
            <div class="text-xs font-semibold text-primary mb-2">{{ $t('machine.fileList') }}:</div>

            <!-- 文件列表滚动区域 -->
            <el-scrollbar max-height="200px">
                <div
                    v-for="file in fileList"
                    :key="file.path"
                    class="flex items-center gap-2 py-1.5 px-2 text-xs hover:bg-gray-50 dark:hover:bg-gray-800 rounded"
                >
                    <!-- 文件状态图标 -->
                    <SvgIcon
                        v-if="file.status === 'uploading'"
                        :size="12"
                        name="Loading"
                        color="var(--el-color-primary)"
                        class="animate-[rotating_2s_linear_infinite] shrink-0"
                    />
                    <SvgIcon v-else-if="file.status === 'complete'" :size="12" name="CircleCheck" color="var(--el-color-success)" class="shrink-0" />
                    <SvgIcon v-else-if="file.status === 'error'" :size="12" name="CircleClose" color="var(--el-color-danger)" class="shrink-0" />
                    <div v-else class="w-4 h-4 rounded-full border-2 border-gray-300 dark:border-gray-600 shrink-0" />

                    <!-- 文件路径 -->
                    <span class="flex-1 truncate text-gray-700 dark:text-gray-300" :title="file.path">
                        {{ file.path }}
                    </span>

                    <!-- 上传进度或状态 -->
                    <span v-if="file.status === 'uploading'" class="shrink-0 flex items-center gap-1.5">
                        <!-- 文件大小 -->
                        <span class="text-[10px] text-gray-500 dark:text-gray-400">
                            {{ formatSize(file.currentSize) }} / {{ formatSize(file.totalSize) }}
                        </span>
                        <!-- 传输速率 -->
                        <span v-if="file.speed" class="text-[10px] text-primary font-semibold"> {{ file.speed }}/s </span>
                        <!-- 进度百分比 -->
                        <span class="text-[10px] text-primary font-semibold"> {{ file.progress }}% </span>
                    </span>
                    <span v-else-if="file.status === 'complete'" class="text-xs text-success shrink-0">
                        {{ $t('common.complete') }}
                    </span>
                    <span v-else-if="file.status === 'error'" class="text-xs text-danger shrink-0">
                        {{ $t('common.error') }}
                    </span>
                    <span v-else class="text-xs text-gray-400 shrink-0">
                        {{ $t('machine.waiting') }}
                    </span>
                </div>
            </el-scrollbar>
        </div>
    </div>
</template>

<script lang="ts" setup>
import TagCodePath from '@/views/ops/component/TagCodePath.vue';
import { ref, computed } from 'vue';

const cancelLoading = ref(false);

const props = defineProps({
    progress: {
        type: Object,
        required: true,
    },
    onCancel: {
        type: Function,
        default: undefined,
    },
});

// 将 Map 转换为数组以便遍历
const fileList = computed(() => {
    if (!props.progress.files || !(props.progress.files instanceof Map)) {
        return [];
    }
    return Array.from(props.progress.files.values());
});

// 格式化文件大小
const formatSize = (bytes: number): string => {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return (bytes / Math.pow(k, i)).toFixed(1) + ' ' + sizes[i];
};

// 处理取消上传
const handleCancel = () => {
    if (props.onCancel) {
        cancelLoading.value = true;
        props.onCancel();
    }
};
</script>
