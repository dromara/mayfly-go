<template>
    <div>
        <el-drawer :title="title" v-model="dialogVisible" :before-close="onCancel" :destroy-on-close="true" :close-on-click-modal="false" size="40%">
            <template #header>
                <DrawerHeader :header="title" :back="onCancel" />
            </template>

            <el-form :model="form" ref="redisFormRef" :rules="rules" label-width="auto">
                <el-form-item prop="tagCodePaths" :label="$t('tag.relateTag')" required>
                    <tag-tree-select multiple v-model="form.tagCodePaths" />
                </el-form-item>
                <el-form-item prop="name" :label="$t('common.name')" required>
                    <el-input v-model.trim="form.name" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="mode" label="mode" required>
                    <el-select v-model="form.mode">
                        <el-option label="standalone" value="standalone"> </el-option>
                        <el-option label="cluster" value="cluster"> </el-option>
                        <el-option label="sentinel" value="sentinel"> </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item prop="host" label="host" required>
                    <el-input v-model.trim="form.host" :placeholder="$t('redis.hostTips')" auto-complete="off" type="textarea"></el-input>
                </el-form-item>
                <el-form-item prop="username" :label="$t('common.username')">
                    <el-input v-model.trim="form.username"></el-input>
                </el-form-item>
                <el-form-item prop="password" :label="$t('common.password')">
                    <el-input type="password" show-password v-model.trim="form.password" autocomplete="new-password"> </el-input>
                </el-form-item>
                <el-form-item v-if="form.mode == 'sentinel'" prop="redisNodePassword" :label="$t('redis.nodePassword')">
                    <el-input type="password" show-password v-model.trim="form.redisNodePassword" autocomplete="new-password"> </el-input>
                </el-form-item>
                <el-form-item prop="db" label="DB" required>
                    <el-select @change="changeDb" :disabled="form.mode == 'cluster'" v-model="dbList" multiple allow-create filterable style="width: 100%">
                        <el-option v-for="db in [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15]" :key="db" :label="db" :value="db" />
                    </el-select>
                </el-form-item>
                <el-form-item prop="remark" :label="$t('common.remark')">
                    <el-input v-model.trim="form.remark" auto-complete="off" type="textarea"></el-input>
                </el-form-item>
                <el-form-item prop="sshTunnelMachineId" :label="$t('machine.sshTunnel')">
                    <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="onTestConn" :loading="testConnBtnLoading" type="success">{{ $t('ac.testConn') }}</el-button>
                    <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, useTemplateRef } from 'vue';
import { redisApi } from './api';
import { ElMessage } from 'element-plus';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { useI18nFormValidate, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';
import { Rules } from '@/common/rule';

const { t } = useI18n();

const props = defineProps({
    redis: {
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
    host: [Rules.requiredInput('ip:port')],
    db: [Rules.requiredSelect('DB')],
    mode: [Rules.requiredSelect('mode')],
};

const redisFormRef: any = useTemplateRef('redisFormRef');

const state = reactive({
    form: {
        id: null,
        code: '',
        tagCodePaths: [],
        name: null,
        mode: 'standalone',
        host: '',
        username: null,
        password: null,
        redisNodePassword: null,
        db: '',
        remark: '',
        sshTunnelMachineId: -1,
    },
    submitForm: {} as any,
    dbList: [0],
    pwd: '',
});

const { form, submitForm, dbList } = toRefs(state);

const { isFetching: testConnBtnLoading, execute: testConnExec } = redisApi.testConn.useApi(submitForm);
const { isFetching: saveBtnLoading, execute: saveRedisExec } = redisApi.saveRedis.useApi(submitForm);

watch(dialogVisible, () => {
    if (!dialogVisible.value) {
        return;
    }

    const redis: any = props.redis;
    if (redis) {
        state.form = { ...redis };
        state.form.tagCodePaths = redis.tags.map((t: any) => t.codePath);
        convertDb(state.form.db);
    } else {
        state.form = { db: '0', tagCodePaths: [] } as any;
        state.dbList = [0];
    }
});

const convertDb = (db: string) => {
    state.dbList = db.split(',').map((x) => Number.parseInt(x));
};

/**
 * 改变表单中的数据库字段，方便表单错误提示。如全部删光，可提示请添加库号
 */
const changeDb = () => {
    state.form.db = state.dbList.length == 0 ? '' : state.dbList.join(',');
};

const getReqForm = async () => {
    const reqForm = { ...state.form };
    if (reqForm.mode == 'sentinel' && reqForm.host.split('=').length != 2) {
        ElMessage.error(t('redis.sentinelHostErr'));
        return;
    }
    if (!state.form.sshTunnelMachineId || state.form.sshTunnelMachineId <= 0) {
        reqForm.sshTunnelMachineId = -1;
    }
    return reqForm;
};

const onTestConn = async () => {
    await useI18nFormValidate(redisFormRef);
    state.submitForm = await getReqForm();
    await testConnExec();
    ElMessage.success(t('ac.connSuccess'));
};

const onConfirm = async () => {
    await useI18nFormValidate(redisFormRef);
    state.submitForm = await getReqForm();
    await saveRedisExec();
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
