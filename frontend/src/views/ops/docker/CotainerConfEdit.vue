<template>
    <div>
        <el-drawer :title="title" v-model="dialogVisible" :before-close="onCancel" :destroy-on-close="true" :close-on-click-modal="false" size="40%">
            <template #header>
                <DrawerHeader :header="title" :back="onCancel" />
            </template>

            <el-form :model="form" ref="formRef" :rules="rules" label-width="auto">
                <el-form-item prop="tagCodePaths" :label="$t('tag.relateTag')" required>
                    <tag-tree-select multiple v-model="form.tagCodePaths" />
                </el-form-item>
                <el-form-item prop="name" :label="$t('common.name')" required>
                    <el-input v-model.trim="form.name" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="addr" :label="$t('docker.addr')" required>
                    <el-input v-model.trim="form.addr" :placeholder="$t('docker.addrTips')" auto-complete="off" type="textarea"></el-input>
                </el-form-item>
                <el-form-item prop="remark" :label="$t('common.remark')">
                    <el-input v-model.trim="form.remark" auto-complete="off" type="textarea"></el-input>
                </el-form-item>
            </el-form>

            <template #footer>
                <!-- <el-button @click="onTestConn" :loading="testConnBtnLoading" type="success">{{ $t('ac.testConn') }}</el-button> -->
                <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="saveBtnLoading" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, useTemplateRef } from 'vue';
import { dockerApi } from './api';
import { ElMessage } from 'element-plus';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { useI18nFormValidate, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';
import { Rules } from '@/common/rule';

const { t } = useI18n();

const props = defineProps({
    container: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits(['val-change', 'cancel']);

const rules = {
    tagCodePaths: [Rules.requiredSelect('tag.relateTag')],
    name: [Rules.requiredInput('common.name')],
    addr: [Rules.requiredInput('addr')],
};

const formRef: any = useTemplateRef('formRef');

const state = reactive({
    form: {
        id: null,
        code: '',
        tagCodePaths: [],
        name: null,
        addr: '',
        remark: '',
    },
    dbList: [0],
    pwd: '',
});

const { form } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveConfExec } = dockerApi.saveConf.useApi(form);

watch(dialogVisible, () => {
    if (!dialogVisible.value) {
        return;
    }

    const container: any = props.container;
    if (container) {
        state.form = { ...container };
        state.form.tagCodePaths = container.tags.map((t: any) => t.codePath);
    } else {
        state.form = {} as any;
    }
});

const onTestConn = async () => {
    await useI18nFormValidate(formRef);
    // await testConnExec();
    ElMessage.success(t('ac.connSuccess'));
};

const onConfirm = async () => {
    await useI18nFormValidate(formRef);
    await saveConfExec();
    useI18nSaveSuccessMsg();
    emit('val-change', state.form);
    onCancel();
};

const onCancel = () => {
    dialogVisible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
