<template>
    <div>
        <el-drawer :append-to-body="false" :title="title" v-model="dialogVisible" :before-close="onCancel" :destroy-on-close="true" :close-on-click-modal="false" size="40%">
            <template #header>
                <DrawerHeader :header="title" :back="onCancel" />
            </template>

            <el-form :model="form" ref="formRef" :rules="rules" label-width="auto">
                <el-form-item prop="tagCodePaths" :label="$t('tag.relateTag')" required>
                    <TagTreeSelect multiple :code="form.code" v-model="form.tagCodePaths" />
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
import { Rules } from '@/common/rule';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { reactive, toRefs, useTemplateRef, watch, type ComponentPublicInstance, type PropType } from 'vue';
import type { FormInstance } from 'element-plus';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import { dockerApi } from './api';
import type { Container } from './types';

const props = defineProps({
    container: {
        type: Object as PropType<Container | null>,
        default: null,
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

const formRef = useTemplateRef<FormInstance>('formRef');

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

    const container = props.container as Record<string, unknown> | null;
    if (container) {
        state.form = { ...container } as typeof state.form;
    } else {
        state.form = { id: null, code: '', tagCodePaths: [], name: null, addr: '', remark: '' };
    }
});

const onTestConn = async () => {
    await useI18nFormValidate(formRef);
    // await testConnExec();
    Msg.success('ac.connSuccess');
};

const onConfirm = async () => {
    await useI18nFormValidate(formRef);
    await saveConfExec();
    Msg.saveSuccess();
    emit('val-change', state.form);
    onCancel();
};

const onCancel = () => {
    dialogVisible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
