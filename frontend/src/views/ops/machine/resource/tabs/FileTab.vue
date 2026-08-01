<template>
    <div class="machine-file-tab h-full">
        <!-- 如果已选择文件配置，显示文件管理器 -->
        <MachineFile
            v-if="selectedFileConf"
            :machine-id="machineId"
            :auth-cert-name="authCertName"
            :protocol="protocol"
            :file-id="selectedFileConf.fileId"
            :path="selectedFileConf.path"
        />
        <!-- 否则显示文件配置选择列表 -->
        <FileConfList
            v-else
            :machine-id="machineId"
            :auth-cert-name="authCertName"
            :protocol="protocol"
            :visible="true"
            :open-file-manager="false"
            @select="handleSelect"
        />
    </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue';
import MachineFile from '@/views/ops/machine/file/MachineFile.vue';
import FileConfList from '@/views/ops/machine/file/FileConfList.vue';

const props = defineProps<{
    tabKey?: string;
    machineId: number;
    authCertName: string;
    protocol: number;
    fileId?: number; // 可选：直接指定文件配置 ID
    initialPath?: string; // 可选：初始路径
}>();

// 已选择的文件配置
const selectedFileConf = ref<{ fileId: number; path: string; name: string } | null>(
    // 如果传入了 fileId 和 initialPath，直接初始化
    props.fileId && props.initialPath
        ? { fileId: props.fileId, path: props.initialPath, name: '' }
        : null
);

// 处理文件配置选择
const handleSelect = (fileConf: { fileId: number; path: string; name: string }) => {
    selectedFileConf.value = {
        fileId: fileConf.fileId,
        path: fileConf.path,
        name: fileConf.name,
    };
};
</script>

<style lang="scss">
.machine-file-tab {
    height: 100%;
    overflow: hidden;
}
</style>
