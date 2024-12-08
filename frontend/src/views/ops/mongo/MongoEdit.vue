<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :before-close="cancel" :close-on-click-modal="false" width="38%" :destroy-on-close="true">
            <el-form :model="form" ref="mongoForm" :rules="rules" label-width="auto">
                <el-tabs v-model="tabActiveName">
                    <el-tab-pane :label="$t('common.basic')" name="basic">
                        <el-form-item ref="tagSelectRef" prop="tagCodePaths" :label="$t('tag.relateTag')" required>
                            <tag-tree-select
                                @change-tag="
                                    (tagCodePaths) => {
                                        form.tagCodePaths = tagCodePaths;
                                        tagSelectRef.validate();
                                    }
                                "
                                multiple
                                :select-tags="form.tagCodePaths"
                                style="width: 100%"
                            />
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
                    <el-button @click="testConn" :loading="testConnBtnLoading" type="success">{{ $t('ac.testConn') }}</el-button>
                    <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, ref, watchEffect } from 'vue';
import { mongoApi } from './api';
import { ElMessage } from 'element-plus';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import { useI18nFormValidate, useI18nPleaseInput, useI18nPleaseSelect, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
    visible: {
        type: Boolean,
    },
    mongo: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const rules = {
    tagCodePaths: [
        {
            required: true,
            message: useI18nPleaseSelect('tag.relateTag'),
            trigger: ['change', 'blur'],
        },
    ],
    name: [
        {
            required: true,
            message: useI18nPleaseInput('common.name'),
            trigger: ['change', 'blur'],
        },
    ],
    uri: [
        {
            required: true,
            message: useI18nPleaseInput('mongo.connUrl'),
            trigger: ['change', 'blur'],
        },
    ],
};

const mongoForm: any = ref(null);
const tagSelectRef: any = ref(null);

const state = reactive({
    dialogVisible: false,
    tabActiveName: 'basic',
    form: {
        id: null,
        code: '',
        name: null,
        uri: null,
        sshTunnelMachineId: null as any,
        tagCodePaths: [],
    },
    submitForm: {},
});

const { dialogVisible, tabActiveName, form, submitForm } = toRefs(state);

const { isFetching: testConnBtnLoading, execute: testConnExec } = mongoApi.testConn.useApi(submitForm);
const { isFetching: saveBtnLoading, execute: saveMongoExec } = mongoApi.saveMongo.useApi(submitForm);

watchEffect(() => {
    state.dialogVisible = props.visible;
    if (!state.dialogVisible) {
        return;
    }
    state.tabActiveName = 'basic';
    const mongo: any = props.mongo;
    if (mongo) {
        state.form = { ...mongo };
        state.form.tagCodePaths = mongo.tags.map((t: any) => t.codePath);
    } else {
        state.form = { db: 0 } as any;
    }
});

const getReqForm = () => {
    const reqForm = { ...state.form };
    if (!state.form.sshTunnelMachineId || state.form.sshTunnelMachineId <= 0) {
        reqForm.sshTunnelMachineId = -1;
    }
    return reqForm;
};

const testConn = async () => {
    await useI18nFormValidate(mongoForm);
    state.submitForm = getReqForm();
    await testConnExec();
    ElMessage.success(t('ac.connSuccess'));
};

const btnOk = async () => {
    await useI18nFormValidate(mongoForm);
    state.submitForm = getReqForm();
    await saveMongoExec();
    useI18nSaveSuccessMsg();
    emit('val-change', state.form);
    cancel();
};

const cancel = () => {
    emit('update:visible', false);
    emit('cancel');
};
</script>
<style lang="scss"></style>
