<template>
    <div>
        <el-dialog
            destroy-on-close
            :before-close="handleClose"
            :title="title || path"
            v-model="dialogVisible"
            :close-on-click-modal="false"
            top="5vh"
            width="65%"
        >
            <div v-loading="loadingContent">
                <monaco-editor :can-change-mode="true" v-model="fileContent" :language="fileType" />
            </div>

            <template #footer>
                <el-button @click="handleClose">{{ $t('common.cancel') }}</el-button>
                <el-button v-loading="saveing" v-auth="'machine:file:write'" type="primary" @click="updateContent">{{ $t('common.save') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { computed, reactive, Ref, ref, toRefs, watch } from 'vue';
import { machineApi } from '../api';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { useI18nSaveSuccessMsg } from '@/hooks/useI18n';

const props = defineProps({
    protocol: { type: Number, default: 1 },
    title: { type: String, default: '' },
    machineId: { type: Number },
    authCertName: { type: String },
    fileId: { type: Number, default: 0 },
    path: { type: String, default: '' },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits(['cancel', 'update:machineId']);

const updateFileContent = machineApi.updateFileContent;

const saveing: Ref<any> = ref(false);

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

const getFileContent = async () => {
    fileContent.value = '';
    state.fileType = getFileType(props.path);
    await getFileContentExec();
};

const handleClose = () => {
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
        useI18nSaveSuccessMsg();
        handleClose();
        fileContent.value = '';
    } finally {
        saveing.value = false;
    }
};

const getFileType = (path: string) => {
    if (path.endsWith('.sh')) {
        return 'shell';
    }
    if (path.endsWith('js')) {
        return 'javascript';
    }
    if (path.endsWith('json')) {
        return 'json';
    }
    if (path.endsWith('Dockerfile')) {
        return 'dockerfile';
    }
    if (path.endsWith('nginx.conf')) {
        return 'shell';
    }
    if (path.endsWith('sql')) {
        return 'sql';
    }
    if (path.endsWith('yaml') || path.endsWith('yml')) {
        return 'yaml';
    }
    if (path.endsWith('xml') || path.endsWith('html')) {
        return 'html';
    }
    if (path.endsWith('py')) {
        return 'python';
    }
    return 'text';
};
</script>
<style lang="scss"></style>
