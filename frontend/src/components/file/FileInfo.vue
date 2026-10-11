<template>
    <el-button v-if="loading" :loading="loading" name="loading" link type="primary" />

    <template v-else>
        <!-- v-if 必须放在 el-tooltip 上：插槽内无有效子节点会触发 [ElOnlyChild] 警告 -->
        <el-tooltip v-if="props.canDownload && fileSize" :content="fileSize" placement="left">
            <!--
              业务模块提供的鉴权下载地址：走全站统一的 iframe 入口。
              不用 <a target="_blank">：该地址携带查询参数凭据，开新标签再因 Content-Disposition 关闭会页面闪动；
              而私有文件也不能作为无凭据的 href 直链暴露给用户复制
            -->
            <el-link v-if="props.downloadUrl" @click="downloadByServerUrl" icon="Download" type="primary" underline="never"></el-link>
            <el-link v-else target="_blank" rel="noopener noreferrer" icon="Download" type="primary" :href="getFileUrl(props.fileKey)"></el-link>
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
import { downloadFile } from '@/common/utils/file';
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
        /**
         * 业务模块的服务端下载地址。
         *
         * 缺省表示该文件是可匿名读取的展示类资产，直接拼通用文件地址；
         * 对只能经权限校验获取的数据类产物（如迁移导出文件），必须由调用方传入本模块的下载入口，
         * 否则通用文件路由会拒读
         */
        downloadUrl?: string;
    }>(),
    { canDownload: true, showFileSize: false }
);

const downloadByServerUrl = () => {
    if (props.downloadUrl) {
        downloadFile(props.downloadUrl);
    }
};

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
