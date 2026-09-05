<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :before-close="onCancel" :close-on-click-modal="false" width="38%" :destroy-on-close="true">
            <auto-form ref="mongoFormRef" v-model="form" :tabs="tabs" label-width="auto">
                <!-- 关联标签 -->
                <template #tagCodePaths>
                    <TagTreeSelect multiple :code="form.code" v-model="form.tagCodePaths" />
                </template>

                <!-- SSH 隧道 -->
                <template #sshTunnelMachineId>
                    <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
                </template>
            </auto-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="onTestConn" :loading="testConnBtnLoading" type="success">{{ $t('ac.testConn') }}</el-button>
                    <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { reactive, toRefs, useTemplateRef, watchEffect, type PropType } from 'vue';
import { AutoForm, type AutoFormItem, type AutoFormTab } from '@/components/auto-form';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import { mongoApi } from './api';
import type { Mongo } from './types';

/** Mongo 编辑表单类型（允许 null 的字段重定义） */
interface MongoForm extends Omit<Partial<Mongo>, 'id' | 'name' | 'uri' | 'sshTunnelMachineId'> {
    id?: number | null;
    name?: string | null;
    uri?: string | null;
    sshTunnelMachineId?: number | null;
    tagCodePaths?: string[];
    db?: number;
}

const props = defineProps({
    mongo: {
        type: Object as PropType<Mongo | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

/** 表单声明（AutoFormTab[]，Tab 布局共享表单数据与校验；tagCodePaths/sshTunnel 走插槽） */
const tabs: AutoFormTab[] = [
    {
        name: 'basic',
        label: 'common.basic',
        items: [
            { prop: 'tagCodePaths', label: 'tag.relateTag', required: true },
            { prop: 'name', label: 'common.name', required: true },
            { prop: 'uri', label: 'uri', type: 'textarea', rows: 2, required: true, placeholder: 'mongodb://username:password@host1:port1' },
        ],
    },
    {
        name: 'other',
        label: 'common.other',
        items: [{ prop: 'sshTunnelMachineId', label: 'machine.sshTunnel' }],
    },
];

const mongoFormRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('mongoFormRef');

const state = reactive({
    form: {
        id: null,
        code: '',
        name: null,
        uri: null,
        sshTunnelMachineId: null as number | null,
        tagCodePaths: [],
    } as MongoForm,
});

const { form } = toRefs(state);

const { isFetching: testConnBtnLoading, execute: testConnExec } = mongoApi.testConn.useApi();
const { isFetching: saveBtnLoading, execute: saveMongoExec } = mongoApi.saveMongo.useApi();

watchEffect(() => {
    if (!dialogVisible.value) {
        return;
    }
    const mongo = props.mongo as MongoForm | false | null;
    if (mongo) {
        state.form = { ...mongo };
    } else {
        state.form = { db: 0, tagCodePaths: [] } as MongoForm;
    }
});

const getReqForm = () => {
    const reqForm = { ...state.form };
    if (!state.form.sshTunnelMachineId || state.form.sshTunnelMachineId <= 0) {
        reqForm.sshTunnelMachineId = -1;
    }
    return reqForm;
};

const onTestConn = async () => {
    await useI18nFormValidate(mongoFormRef);
    await testConnExec(getReqForm());
    Msg.success('ac.connSuccess');
};

const onConfirm = async () => {
    await useI18nFormValidate(mongoFormRef);
    await saveMongoExec(getReqForm());
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
