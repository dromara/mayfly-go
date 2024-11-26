<template>
    <div>
        <el-drawer :title="title" v-model="dialogVisible" :before-close="cancel" :destroy-on-close="true" :close-on-click-modal="false" size="40%">
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <el-form :model="form" ref="redisForm" :rules="rules" label-width="auto">
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
                            <el-select
                                @change="changeDb"
                                :disabled="form.mode == 'cluster'"
                                v-model="dbList"
                                multiple
                                allow-create
                                filterable
                                style="width: 100%"
                            >
                                <el-option v-for="db in [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15]" :key="db" :label="db" :value="db" />
                            </el-select>
                        </el-form-item>
                        <el-form-item prop="remark" :label="$t('common.remark')">
                            <el-input v-model.trim="form.remark" auto-complete="off" type="textarea"></el-input>
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
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, ref, watch } from 'vue';
import { redisApi } from './api';
import { ElMessage } from 'element-plus';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { useI18nFormValidate, useI18nPleaseInput, useI18nPleaseSelect, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
    visible: {
        type: Boolean,
    },
    redis: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

const emit = defineEmits(['update:visible', 'val-change', 'cancel']);

const rules = {
    tagCodePaths: [
        {
            required: true,
            message: useI18nPleaseSelect('tag.relateTag'),
            trigger: ['blur', 'change'],
        },
    ],
    name: [
        {
            required: true,
            message: useI18nPleaseInput('common.name'),
            trigger: ['change', 'blur'],
        },
    ],
    host: [
        {
            required: true,
            message: useI18nPleaseInput('ip:port'),
            trigger: ['change', 'blur'],
        },
    ],
    db: [
        {
            required: true,
            message: useI18nPleaseSelect('DB'),
            trigger: ['change', 'blur'],
        },
    ],
    mode: [
        {
            required: true,
            message: useI18nPleaseSelect('mode'),
            trigger: ['change', 'blur'],
        },
    ],
};

const redisForm: any = ref(null);
const tagSelectRef: any = ref(null);

const state = reactive({
    dialogVisible: false,
    tabActiveName: 'basic',
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

const { dialogVisible, tabActiveName, form, submitForm, dbList } = toRefs(state);

const { isFetching: testConnBtnLoading, execute: testConnExec } = redisApi.testConn.useApi(submitForm);
const { isFetching: saveBtnLoading, execute: saveRedisExec } = redisApi.saveRedis.useApi(submitForm);

watch(
    () => props.visible,
    () => {
        state.dialogVisible = props.visible;
        if (!state.dialogVisible) {
            return;
        }
        state.tabActiveName = 'basic';
        const redis: any = props.redis;
        if (redis) {
            state.form = { ...redis };
            state.form.tagCodePaths = redis.tags.map((t: any) => t.codePath);
            convertDb(state.form.db);
        } else {
            state.form = { db: '0', tagCodePaths: [] } as any;
            state.dbList = [0];
        }
    }
);

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

const testConn = async () => {
    await useI18nFormValidate(redisForm);
    state.submitForm = await getReqForm();
    await testConnExec();
    ElMessage.success(t('ac.connSuccess'));
};

const btnOk = async () => {
    await useI18nFormValidate(redisForm);
    state.submitForm = await getReqForm();
    await saveRedisExec();
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
