<template>
    <div>
        <!-- 文件配置选择统一用弹窗承载（tab 与机器列表入口同一形态） -->
        <el-dialog v-if="dialogVisible" :title="title" v-model="dialogVisible" :show-close="true" :before-close="handleClose" width="60%">
            <FileConfTable :machine-id="machineId" @open="onOpenConf" />
        </el-dialog>

        <!-- 目录配置：在抽屉里打开文件管理器 -->
        <el-drawer
            :append-to-body="false"
            resizable
            destroy-on-close
            :title="fileDialog.title"
            v-model="fileDialog.visible"
            :close-on-click-modal="false"
            size="70%"
            header-class="mb-0!"
        >
            <machine-file
                :machine-id="machineId ?? undefined"
                :auth-cert-name="props.authCertName"
                :file-id="fileDialog.fileId"
                :path="fileDialog.path"
                :protocol="protocol"
            />
        </el-drawer>

        <machine-file-content
            :title="fileContent.title"
            v-model:visible="fileContent.contentVisible"
            :machine-id="machineId ?? undefined"
            :auth-cert-name="props.authCertName"
            :file-id="fileContent.fileId"
            :path="fileContent.path"
        />
    </div>
</template>

<script lang="ts" setup>
import { defineAsyncComponent, reactive } from 'vue';
import { FileTypeEnum } from '../enums';
import type { MachineFileVO } from '../types';
import FileConfTable from './FileConfTable.vue';

const MachineFile = defineAsyncComponent(() => import('./MachineFile.vue'));
const MachineFileContent = defineAsyncComponent(() => import('./MachineFileContent.vue'));

const props = defineProps({
    protocol: { type: Number, default: 1 },
    authCertName: { type: String },
    title: { type: String },
    openFileManager: { type: Boolean, default: true }, // true: 点击目录配置在本组件抽屉里打开文件管理器；false: 抛给父组件（tab 场景）
});

const dialogVisible = defineModel<boolean>('visible', { default: false });
const machineId = defineModel<number | null>('machineId');

const emit = defineEmits(['cancel', 'select']);

const state = reactive({
    fileDialog: {
        visible: false,
        title: '',
        fileId: 0,
        path: '',
    },
    fileContent: {
        title: '',
        fileId: 0,
        contentVisible: false,
        path: '',
    },
});

const { fileDialog, fileContent } = state;

/**
 * 点击配置项：文件类型看内容，目录类型进文件管理器。
 * 判据走 FileTypeEnum，不再和后端存储值（1/2）纠缠
 */
function onOpenConf(conf: MachineFileVO) {
    if (conf.type === FileTypeEnum.File.value) {
        fileContent.fileId = conf.id;
        fileContent.path = conf.path;
        fileContent.title = `${conf.name} => ${conf.path}`;
        fileContent.contentVisible = true;
        return;
    }

    if (props.openFileManager) {
        fileDialog.fileId = conf.id;
        fileDialog.path = conf.path;
        fileDialog.title = `${conf.name} => ${conf.path}`;
        fileDialog.visible = true;
        return;
    }

    // 内联场景交给父组件在 tab 中打开文件管理器
    emit('select', { fileId: conf.id, path: conf.path, name: conf.name, type: conf.type });
}

/** 弹窗关闭：通知调用方取消，并交还机器上下文 */
function handleClose() {
    dialogVisible.value = false;
    machineId.value = null;
    emit('cancel');
}
</script>
