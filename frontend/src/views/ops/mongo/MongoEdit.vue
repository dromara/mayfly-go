<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :before-close="onCancel" :close-on-click-modal="false" width="38%" :destroy-on-close="true">
            <el-form :model="form" ref="mongoFormRef" :rules="rules" label-width="auto">
                <el-tabs v-model="tabActiveName">
                    <el-tab-pane :label="$t('common.basic')" name="basic">
                        <el-form-item prop="tagCodePaths" :label="$t('tag.relateTag')" required>
                            <TagTreeSelect multiple :code="form.code" v-model="form.tagCodePaths" />
                        </el-form-item>

                        <el-form-item prop="name" :label="$t('common.name')" required>
                            <el-input v-model.trim="form.name" auto-complete="off"></el-input>
                        </el-form-item>
                        <el-form-item prop="uri" label="uri" required>
                            <el-input
                                type="textarea"
                                :rows="2"
                                v-model.trim="form.uri"
                                placeholder="mongodb://username:password@host1:port1"
                                auto-complete="off"
                            ></el-input>
                        </el-form-item>
                    </el-tab-pane>

                    <el-tab-pane :label="$t('common.other')" name="other">
                        <el-form-item prop="sshTunnelMachineId" :label="$t('machine.sshTunnel')">
                            <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
                        </el-form-item>
                    </el-tab-pane>
                </el-tabs>
            </el-form>

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
import { Rules } from '@/common/rule';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { reactive, toRefs, useTemplateRef, watchEffect, type PropType } from 'vue';
import type { FormInstance } from 'element-plus';
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

const rules = {
    tagCodePaths: [Rules.requiredSelect('tag.relateTag')],
    name: [Rules.requiredInput('common.name')],
    uri: [Rules.requiredInput('mongo.connUrl')],
};

const mongoFormRef = useTemplateRef<FormInstance>('mongoFormRef');

const state = reactive({
    tabActiveName: 'basic',
    form: {
        id: null,
        code: '',
        name: null,
        uri: null,
        sshTunnelMachineId: null as number | null,
        tagCodePaths: [],
    } as MongoForm,
});

const { tabActiveName, form } = toRefs(state);

const { isFetching: testConnBtnLoading, execute: testConnExec } = mongoApi.testConn.useApi();
const { isFetching: saveBtnLoading, execute: saveMongoExec } = mongoApi.saveMongo.useApi();

watchEffect(() => {
    if (!dialogVisible.value) {
        return;
    }
    state.tabActiveName = 'basic';
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
