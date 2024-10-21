<template>
    <el-tooltip :content="formatByteSize(fileDetail.size)" placement="left">
        <el-link v-if="props.canDownload" target="_blank" rel="noopener noreferrer" icon="Download" type="primary" :href="getFileUrl(props.fileKey)"></el-link>
    </el-tooltip>

    {{ fileDetail.filename }}
</template>

<script lang="ts" setup>
import { onMounted, ref, watch } from 'vue';
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
});

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

const fileDetail: any = ref({});

const setFileInfo = async () => {
    if (!props.fileKey) {
        return;
    }
    if (props.files && props.files.length > 0) {
        const file: any = props.files.find((file: any) => {
            return file.fileKey === props.fileKey;
        });
        fileDetail.value = file;
        return;
    }

    const files = await openApi.getFileDetail([props.fileKey]);
    fileDetail.value = files?.[0];
};
</script>

<style lang="scss"></style>
