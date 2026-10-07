<template>
    <div>
        <el-dialog destroy-on-close :before-close="handleClose" v-model="dialogVisible" :close-on-click-modal="false" top="5vh" width="65%">
            <template #header>
                <!-- 机器上的绝对路径通常很长，截断显示但保留完整值可查 -->
                <div class="flex min-w-0 items-center gap-2">
                    <span class="truncate font-medium" :title="path">{{ title || path }}</span>
                    <el-tag v-if="isDirty" size="small" type="warning" effect="light">{{ $t('machine.unsavedBadge') }}</el-tag>
                </div>
            </template>

            <div v-loading="loadingContent">
                <monaco-editor :can-change-mode="true" v-model="fileContent" :language="fileType" />
            </div>

            <template #footer>
                <el-button @click="handleClose">{{ $t('common.cancel') }}</el-button>
                <el-button v-loading="saveing" :disabled="!isDirty" v-auth="FILE_PERM.write" type="primary" @click="updateContent">
                    {{ $t('common.save') }}
                </el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { computed, defineAsyncComponent, reactive, Ref, ref, toRefs, watch } from 'vue';
import { machineApi } from '../api';
import { FILE_PERM } from './constants';
import { getFileLanguage } from './utils/fileLang';

const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

const props = defineProps({
    protocol: { type: Number, default: 1 },
    title: { type: String, default: '' },
    machineId: { type: Number },
    authCertName: { type: String },
    fileId: { type: Number, default: 0 },
    path: { type: String, default: '' },
});

// 保存成功后通知宿主刷新当前行（mtime/size 不会停留在旧值）
const emit = defineEmits(['saved']);

const dialogVisible = defineModel<boolean>('visible', { default: false });

const updateFileContent = machineApi.updateFileContent;

const saveing: Ref<boolean> = ref(false);

// 本次打开时从服务端读到的内容，作为「有无未保存修改」的基线
const loadedContent = ref('');

const state = reactive({
    loadingContent: false,
    fileType: '',
});

const { fileType } = toRefs(state);

const {
    isFetching: loadingContent,
    execute: getFileContentExec,
    data: fileContent,
} = machineApi.fileContent.useApi(
    computed(() => {
        return {
            fileId: props.fileId,
            path: props.path,
            machineId: props.machineId,
            authCertName: props.authCertName,
            protocol: props.protocol,
        };
    })
);

watch(props, async (newValue) => {
    if (dialogVisible.value) {
        await getFileContent();
    }
});

const isDirty = computed(() => dialogVisible.value && (fileContent.value ?? '') !== loadedContent.value);

const getFileContent = async () => {
    fileContent.value = '';
    state.fileType = getFileLanguage(props.path);
    await getFileContentExec();
    // 取内容失败时 data 仍为空，基线同样为空，不会把旧文件误判成未保存修改
    loadedContent.value = fileContent.value ?? '';
};

/**
 * 关闭弹层：内容有未保存修改时先二次确认，避免误按 Esc 直接丢改动
 */
const handleClose = async () => {
    if (isDirty.value) {
        if (!(await useI18nConfirm('machine.unsavedCloseConfirm'))) {
            // 取消或关掉弹窗：不继续后续操作
            return;
        }
    }
    dialogVisible.value = false;
};

const updateContent = async () => {
    try {
        saveing.value = true;
        await updateFileContent.request({
            content: fileContent.value,
            id: props.fileId,
            path: props.path,
            machineId: props.machineId,
            authCertName: props.authCertName,
            protocol: props.protocol,
        });
        Msg.saveSuccess();
        // 先归基线再关闭，否则关闭会被脏检查二次拦住
        loadedContent.value = fileContent.value ?? '';
        emit('saved');
        dialogVisible.value = false;
    } finally {
        saveing.value = false;
    }
};
</script>
