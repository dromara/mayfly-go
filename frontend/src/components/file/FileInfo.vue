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
import { computed, onMounted, Ref, ref, watch } from 'vue';
import openApi from '@/common/openApi';
import { getFileUrl } from '@/common/request';
import { formatByteSize } from '@/common/utils/format';

const props = defineProps({
    fileKey: {
        type: String,
        required: true,
    },
    files: {
        type: [Array],
    },
    canDownload: {
        type: Boolean,
        default: true,
    },
    showFileSize: {
        type: Boolean,
        default: false,
    },
});

const loading: Ref<boolean> = ref(false);

onMounted(async () => {
    setFileInfo();
});

watch(
    () => props.fileKey,
    async (val) => {
        if (val) {
            setFileInfo();
        }
    }
);

const fileSize = computed(() => {
    return fileDetail.value.size ? formatByteSize(fileDetail.value.size) : '';
});

const fileDetail: any = ref({});

const setFileInfo = async () => {
    try {
        if (!props.fileKey) {
            return;
        }
        loading.value = true;
        if (props.files && props.files.length > 0) {
            const file: any = props.files.find((file: any) => {
                return file.fileKey === props.fileKey;
            });
            fileDetail.value = file;
            return;
        }

        const files = await openApi.getFileDetail([props.fileKey]);
        fileDetail.value = files?.[0];
    } finally {
        loading.value = false;
    }
};
</script>

<style lang="scss" scoped>
.file-size {
    margin-left: 1px;
    color: #909399;
    font-size: 8px;
}
</style>
