<template>
    <div>
        <el-drawer :title="title" v-model="dialogVisible" :before-close="cancel" :destroy-on-close="true" :close-on-click-modal="false" size="40%">
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <el-form :model="form" ref="dbForm" :rules="rules" label-width="auto">
                <el-divider content-position="left">基本</el-divider>

                <el-form-item ref="tagSelectRef" prop="tagCodePaths" label="标签">
                    <tag-tree-select
                        multiple
                        @change-tag="
                            (paths: any) => {
                                form.tagCodePaths = paths;
                                tagSelectRef.validate();
                            }
                        "
                        :select-tags="form.tagCodePaths"
                        style="width: 100%"
                    />
                </el-form-item>

                <el-form-item prop="name" label="名称" required>
                    <el-input v-model.trim="form.name" placeholder="请输入数据库别名" auto-complete="off"></el-input>
                </el-form-item>

                <el-form-item prop="type" label="类型" required>
                    <el-select @change="changeDbType" style="width: 100%" v-model="form.type" placeholder="请选择数据库类型">
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

                <el-form-item v-if="form.type !== DbType.sqlite" prop="host" label="host" required>
                    <el-col :span="18">
                        <el-input v-model.trim="form.host" placeholder="请输入ip" auto-complete="off"></el-input>
                    </el-col>
                    <el-col style="text-align: center" :span="1">:</el-col>
                    <el-col :span="5">
                        <el-input type="number" v-model.number="form.port" placeholder="端口"></el-input>
                    </el-col>
                </el-form-item>

                <el-form-item v-if="form.type === DbType.sqlite" prop="host" label="sqlite地址">
                    <el-input v-model.trim="form.host" placeholder="请输入sqlite文件在服务器的绝对地址"></el-input>
                </el-form-item>

                <el-form-item v-if="form.type === DbType.oracle" label="SID|服务名">
                    <el-col :span="5">
                        <el-select
                            @change="
                                () => {
                                    state.extra.serviceName = '';
                                    state.extra.sid = '';
                                }
                            "
                            v-model="state.extra.stype"
                            placeholder="请选择"
                        >
                            <el-option label="服务名" :value="1" />
                            <el-option label="SID" :value="2" />
                        </el-select>
                    </el-col>
                    <el-col style="text-align: center" :span="1">:</el-col>
                    <el-col :span="18">
                        <el-input v-if="state.extra.stype == 1" v-model="state.extra.serviceName" placeholder="请输入服务名"> </el-input>
                        <el-input v-else v-model="state.extra.sid" placeholder="请输入SID"> </el-input>
                    </el-col>
                </el-form-item>

                <el-form-item prop="remark" label="备注">
                    <el-input v-model="form.remark" auto-complete="off" type="textarea"></el-input>
                </el-form-item>

                <el-divider content-position="left">账号</el-divider>
                <div>
                    <ResourceAuthCertTableEdit
                        v-model="form.authCerts"
                        :resource-code="form.code"
                        :resource-type="TagResourceTypeEnum.Db.value"
                        :test-conn-btn-loading="testConnBtnLoading"
                        @test-conn="testConn"
                        :disable-ciphertext-type="[AuthCertCiphertextTypeEnum.PrivateKey.value]"
                    />
                </div>

                <el-divider content-position="left">其他</el-divider>
                <el-form-item prop="params" label="连接参数">
                    <el-input v-model.trim="form.params" placeholder="其他连接参数，形如: key1=value1&key2=value2"> </el-input>
                </el-form-item>

                <el-form-item prop="sshTunnelMachineId" label="SSH隧道">
                    <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
                </el-form-item>
            </el-form>

            <template #footer>
                <el-button @click="cancel()">取 消</el-button>
                <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">确 定</el-button>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { reactive, ref, toRefs, watchEffect } from 'vue';
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

const props = defineProps({
    visible: {
        type: Boolean,
    },
    data: {
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
            message: '请选择标签',
            trigger: ['change'],
        },
    ],
    name: [
        {
            required: true,
            message: '请输入别名',
            trigger: ['change', 'blur'],
        },
    ],
    type: [
        {
            required: true,
            message: '请选择数据库类型',
            trigger: ['change', 'blur'],
        },
    ],
    host: [
        {
            required: true,
            message: '请输入主机ip和port',
            trigger: ['blur'],
        },
    ],
    sid: [
        {
            required: true,
            message: '请输入SID',
            trigger: ['change', 'blur'],
        },
    ],
};

const dbForm: any = ref(null);
const tagSelectRef: any = ref(null);

const DefaultForm = {
    id: null,
    type: DbType.mysql,
    code: '',
    name: null,
    host: '',
    port: getDbDialect(DbType.mysql).getInfo().defaultPort,
    extra: '', // 连接需要的额外参数（json字符串）
    params: null,
    remark: '',
    sshTunnelMachineId: null as any,
    authCerts: [],
    tagCodePaths: [],
};

const state = reactive({
    dialogVisible: false,
    extra: {} as any, // 连接需要的额外参数（json）
    form: DefaultForm,
    submitForm: {} as any,
});

const { dialogVisible, form, submitForm } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveInstanceExec, data: saveInstanceRes } = dbApi.saveInstance.useApi(submitForm);
const { isFetching: testConnBtnLoading, execute: testConnExec } = dbApi.testConn.useApi(submitForm);

watchEffect(() => {
    state.dialogVisible = props.visible;
    if (!state.dialogVisible) {
        return;
    }
    const dbInst: any = props.data;
    if (dbInst) {
        state.form = { ...dbInst };
        state.form.tagCodePaths = dbInst.tags.map((t: any) => t.codePath) || [];
        try {
            state.extra = JSON.parse(state.form.extra);
        } catch (e) {
            state.extra = {};
        }
    } else {
        state.form = { ...DefaultForm };
        state.form.authCerts = [];
    }
});

const getReqForm = async () => {
    const reqForm: any = { ...state.form };
    reqForm.selectAuthCert = null;
    reqForm.tags = null;
    if (!state.form.sshTunnelMachineId) {
        reqForm.sshTunnelMachineId = -1;
    }
    if (Object.keys(state.extra).length > 0) {
        reqForm.extra = JSON.stringify(state.extra);
    }
    return reqForm;
};

const testConn = async (authCert: any) => {
    try {
        await dbForm.value.validate();
    } catch (e: any) {
        ElMessage.error('请正确填写信息');
        return false;
    }

    state.submitForm = await getReqForm();
    state.submitForm.authCerts = [authCert];
    await testConnExec();
    ElMessage.success('连接成功');
};

const btnOk = async () => {
    try {
        await dbForm.value.validate();
    } catch (e: any) {
        ElMessage.error('请正确填写信息');
        return false;
    }

    state.submitForm = await getReqForm();
    await saveInstanceExec();
    ElMessage.success('保存成功');
    state.form.id = saveInstanceRes as any;
    emit('val-change', state.form);
    cancel();
};

const cancel = () => {
    emit('update:visible', false);
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
