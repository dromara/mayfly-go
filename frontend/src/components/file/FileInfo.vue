<template>
    <el-button v-if="loading" :loading="loading" name="loading" link type="primary" />

    <template v-else>
        <el-tooltip :content="fileSize" placement="left">
            <el-link
                v-if="props.canDownload"
                target="_blank"
                rel="noopener noreferrer"
                icon="Download"
                type="primary"
                :href="getFileUrl(props.fileKey)"
            ></el-link>
        </el-tooltip>

        {{ fileDetail?.filename }}
        <!-- 文件大小显示 -->
        <span v-if="props.showFileSize && fileDetail?.size" class="file-size">({{ fileSize }})</span>
    </template>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue';
import openApi from '@/common/openApi';
import { getFileUrl } from '@/common/request';
import { formatByteSize } from '@/common/utils/format';

interface FileDetail {
    fileKey?: string;
    filename?: string;
    size?: number;
}

const props = withDefaults(
    defineProps<{
        fileKey: string;
        files?: FileDetail[];
        canDownload?: boolean;
        showFileSize?: boolean;
    }>(),
    { canDownload: true, showFileSize: false }
);

const loading = ref(false);

const fileSize = computed(() => {
    return fileDetail.value?.size ? formatByteSize(fileDetail.value.size) : '';
});

const fileDetail = ref<FileDetail | null>(null);

const setFileInfo = async () => {
    try {
        if (!props.fileKey) {
            return;
        }
        loading.value = true;
        if (props.files && props.files.length > 0) {
            const file = props.files.find((f) => f.fileKey === props.fileKey);
            fileDetail.value = file ?? null;
            return;
        }

        const files = await openApi.getFileDetail([props.fileKey]);
        fileDetail.value = files?.[0] ?? null;
    } finally {
        loading.value = false;
    }
};

// 监听 fileKey 变化，立即执行一次
watch(
    () => props.fileKey,
    (val) => {
        if (val) {
            setFileInfo();
        }
    },
    { immediate: true }
);
</script>

<style lang="scss" scoped>
.file-size {
    margin-left: 1px;
    color: #909399;
    font-size: 8px;
}
</style>
