<template>
    <div>
        <el-drawer :title="title" v-model="dialogVisible" :before-close="cancel" :destroy-on-close="true" :close-on-click-modal="false" size="40%">
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <el-form :model="form" ref="dbFormRef" :rules="rules" label-width="auto">
                <el-divider content-position="left">{{ $t('common.basic') }}</el-divider>

                <el-form-item prop="tagCodePaths" :label="$t('tag.relateTag')">
                    <tag-tree-select multiple v-model="form.tagCodePaths" />
                </el-form-item>

                <el-form-item prop="name" :label="$t('common.name')" required>
                    <el-input v-model.trim="form.name" auto-complete="off"></el-input>
                </el-form-item>

                <el-form-item prop="type" :label="$t('common.type')" required>
                    <el-select @change="changeDbType" style="width: 100%" v-model="form.type">
                        <el-option
                            v-for="(dbTypeAndDialect, key) in getDbDialectMap()"
                            :key="key"
                            :value="dbTypeAndDialect[0]"
                            :label="dbTypeAndDialect[1].getInfo().name"
                        >
                            <SvgIcon :name="dbTypeAndDialect[1].getInfo().icon" :size="20" />
                            {{ dbTypeAndDialect[1].getInfo().name }}
                        </el-option>

                        <template #prefix>
                            <SvgIcon :name="getDbDialect(form.type).getInfo().icon" :size="20" />
                        </template>
                    </el-select>
                </el-form-item>

                <el-form-item v-if="form.type !== DbType.sqlite" prop="host" label="Host" required>
                    <el-col :span="18">
                        <el-input v-model.trim="form.host" auto-complete="off"></el-input>
                    </el-col>
                    <el-col style="text-align: center" :span="1">:</el-col>
                    <el-col :span="5">
                        <el-input type="number" v-model.number="form.port" :placeholder="$t('db.port')"></el-input>
                    </el-col>
                </el-form-item>

                <el-form-item v-if="form.type === DbType.sqlite" prop="host" label="Path">
                    <el-input v-model.trim="form.host" :placeholder="$t('db.sqlitePathPlaceholder')"></el-input>
                </el-form-item>

                <el-form-item v-if="form.type === DbType.oracle" label="SID|Service">
                    <el-col :span="5">
                        <el-select
                            @change="
                                () => {
                                    state.extra.serviceName = '';
                                    state.extra.sid = '';
                                }
                            "
                            v-model="state.extra.stype"
                        >
                            <el-option label="Service" :value="1" />
                            <el-option label="SID" :value="2" />
                        </el-select>
                    </el-col>
                    <el-col class="text-center" :span="1">:</el-col>
                    <el-col :span="18">
                        <el-input v-if="state.extra.stype == 1" v-model="state.extra.serviceName" placeholder="Service Name"> </el-input>
                        <el-input v-else v-model="state.extra.sid" placeholder="SID"> </el-input>
                    </el-col>
                </el-form-item>

                <el-form-item prop="remark" :label="$t('common.remark')">
                    <el-input v-model="form.remark" auto-complete="off" type="textarea"></el-input>
                </el-form-item>

                <el-divider content-position="left">{{ $t('common.account') }}</el-divider>
                <div>
                    <ResourceAuthCertTableEdit
                        v-model="form.authCerts"
                        :resource-code="form.code"
                        :resource-type="TagResourceTypeEnum.DbInstance.value"
                        :test-conn-btn-loading="testConnBtnLoading"
                        @test-conn="testConn"
                        :disable-ciphertext-type="[AuthCertCiphertextTypeEnum.PrivateKey.value]"
                    />
                </div>

                <el-divider content-position="left">{{ $t('common.other') }}</el-divider>
                <el-form-item prop="params" :label="$t('db.connParam')">
                    <el-input v-model.trim="form.params" :placeholder="$t('db.connParamPlaceholder')"> </el-input>
                </el-form-item>

                <el-form-item prop="sshTunnelMachineId" :label="$t('machine.sshTunnel')">
                    <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
                </el-form-item>
            </el-form>

            <template #footer>
                <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { computed, reactive, toRefs, useTemplateRef, watchEffect } from 'vue';
import { dbApi } from './api';
import { ElMessage } from 'element-plus';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import { DbType, getDbDialect, getDbDialectMap } from './dialect';
import SvgIcon from '@/components/svgIcon/index.vue';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import ResourceAuthCertTableEdit from '../component/ResourceAuthCertTableEdit.vue';
import { AuthCertCiphertextTypeEnum } from '../tag/enums';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import { useI18nFormValidate, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';
import { notBlankI18n } from '@/common/assert';
import { Rules } from '@/common/rule';

const { t } = useI18n();

const props = defineProps({
    data: {
        type: [Boolean, Object],
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
    type: [Rules.requiredSelect('common.type')],
    host: [Rules.requiredInput('Host:Port')],
};

const dbFormRef: any = useTemplateRef('dbFormRef');

const DefaultForm = {
    id: null,
    type: DbType.mysql,
    code: '',
    name: null,
    host: '',
    port: getDbDialect(DbType.mysql).getInfo().defaultPort,
    extra: null, // 连接需要的额外参数（json字符串）
    params: null,
    remark: '',
    sshTunnelMachineId: null as any,
    authCerts: [],
    tagCodePaths: [],
};

const state = reactive({
    extra: {} as any, // 连接需要的额外参数（json）
    form: DefaultForm,
});

const { form } = toRefs(state);

const submitForm = computed(() => {
    const reqForm: any = { ...state.form };
    reqForm.selectAuthCert = null;
    reqForm.tags = null;
    if (!state.form.sshTunnelMachineId) {
        reqForm.sshTunnelMachineId = -1;
    }
    if (Object.keys(state.extra).length > 0) {
        reqForm.extra = state.extra;
    }
    return reqForm;
});

const { isFetching: saveBtnLoading, execute: saveInstanceExec, data: saveInstanceRes } = dbApi.saveInstance.useApi(submitForm);
const { isFetching: testConnBtnLoading, execute: testConnExec } = dbApi.testConn.useApi(submitForm);

watchEffect(() => {
    if (!dialogVisible.value) {
        return;
    }
    const dbInst: any = props.data;
    if (dbInst) {
        state.form = { ...dbInst };
        state.form.tagCodePaths = dbInst.tags.map((t: any) => t.codePath) || [];
        state.extra = dbInst.extra || {};
    } else {
        state.form = { ...DefaultForm };
        state.form.authCerts = [];
    }
});

const testConn = async (authCert: any) => {
    await useI18nFormValidate(dbFormRef);
    submitForm.value.authCerts = [authCert];
    await testConnExec();
    ElMessage.success(t('db.connSuccess'));
};

const btnOk = async () => {
    await useI18nFormValidate(dbFormRef);
    notBlankI18n(submitForm.value.authCerts, 'db.acName');
    await saveInstanceExec();
    useI18nSaveSuccessMsg();
    state.form.id = saveInstanceRes as any;
    emit('val-change', state.form);
    cancel();
};

const cancel = () => {
    dialogVisible.value = false;
    emit('cancel');
    state.extra = {};
};

const changeDbType = (val: string) => {
    if (!state.form.id) {
        state.form.port = getDbDialect(val).getInfo().defaultPort as any;
    }
    state.extra = {};
};
</script>
<style lang="scss"></style>
