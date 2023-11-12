<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :before-close="cancel" :close-on-click-modal="false" :destroy-on-close="true" width="38%">
            <el-form :model="form" ref="dbForm" :rules="rules" label-width="auto">
                <el-tabs v-model="tabActiveName">
                    <el-tab-pane label="基础信息" name="basic">
                        <el-form-item prop="name" label="别名" required>
                            <el-input v-model.trim="form.name" placeholder="请输入数据库别名" auto-complete="off"></el-input>
                        </el-form-item>
                        <el-form-item prop="type" label="类型" required>
                            <el-select style="width: 100%" v-model="form.type" placeholder="请选择数据库类型">
                                <el-option key="item.id" label="mysql" value="mysql"> </el-option>
                                <el-option key="item.id" label="postgres" value="postgres"> </el-option>
                            </el-select>
                        </el-form-item>
                        <el-form-item prop="host" label="host" required>
                            <el-col :span="18">
                                <el-input :disabled="form.id !== undefined" v-model.trim="form.host" placeholder="请输入主机ip" auto-complete="off"></el-input>
                            </el-col>
                            <el-col style="text-align: center" :span="1">:</el-col>
                            <el-col :span="5">
                                <el-input type="number" v-model.number="form.port" placeholder="请输入端口"></el-input>
                            </el-col>
                        </el-form-item>
                        <el-form-item prop="username" label="用户名" required>
                            <el-input v-model.trim="form.username" placeholder="请输入用户名"></el-input>
                        </el-form-item>
                        <el-form-item prop="password" label="密码">
                            <el-input type="password" show-password v-model.trim="form.password" placeholder="请输入密码" autocomplete="new-password">
                                <template v-if="form.id && form.id != 0" #suffix>
                                    <el-popover @hide="pwd = ''" placement="right" title="原密码" :width="200" trigger="click" :content="pwd">
                                        <template #reference>
                                            <el-link v-auth="'db:instance:save'" @click="getDbPwd" :underline="false" type="primary" class="mr5"
                                                >原密码
                                            </el-link>
                                        </template>
                                    </el-popover>
                                </template>
                            </el-input>
                        </el-form-item>

                        <el-form-item prop="remark" label="备注">
                            <el-input v-model="form.remark" auto-complete="off" type="textarea"></el-input>
                        </el-form-item>
                    </el-tab-pane>

                    <el-tab-pane label="其他配置" name="other">
                        <el-form-item prop="params" label="连接参数">
                            <el-input v-model.trim="form.params" placeholder="其他连接参数，形如: key1=value1&key2=value2">
                                <template #suffix>
                                    <el-link
                                        target="_blank"
                                        href="https://github.com/go-sql-driver/mysql#parameters"
                                        :underline="false"
                                        type="primary"
                                        class="mr5"
                                        >参数参考</el-link
                                    >
                                </template>
                            </el-input>
                        </el-form-item>

                        <el-form-item prop="sshTunnelMachineId" label="SSH隧道">
                            <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
                        </el-form-item>
                    </el-tab-pane>
                </el-tabs>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="testConn" :loading="state.testConnBtnLoading" type="success">测试连接</el-button>
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="btnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, ref } from 'vue';
import { dbApi } from './api';
import { ElMessage } from 'element-plus';
import { notBlank } from '@/common/assert';
import { RsaEncrypt } from '@/common/rsa';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';

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
            trigger: ['change', 'blur'],
        },
    ],
    username: [
        {
            required: true,
            message: '请输入用户名',
            trigger: ['change', 'blur'],
        },
    ],
};

const dbForm: any = ref(null);

const state = reactive({
    dialogVisible: false,
    tabActiveName: 'basic',
    form: {
        id: null,
        type: null,
        name: null,
        host: '',
        port: 3306,
        username: null,
        password: null,
        params: null,
        remark: '',
        sshTunnelMachineId: null as any,
    },
    // 原密码
    pwd: '',
    // 原用户名
    oldUserName: null,
    btnLoading: false,
    testConnBtnLoading: false,
});

const { dialogVisible, tabActiveName, form, pwd, btnLoading } = toRefs(state);

watch(props, (newValue: any) => {
    state.dialogVisible = newValue.visible;
    if (!state.dialogVisible) {
        return;
    }
    state.tabActiveName = 'basic';
    if (newValue.data) {
        state.form = { ...newValue.data };
        state.oldUserName = state.form.username;
    } else {
        state.form = { port: 3306 } as any;
        state.oldUserName = null;
    }
});

const getDbPwd = async () => {
    state.pwd = await dbApi.getInstancePwd.request({ id: state.form.id });
};

const getReqForm = async () => {
    const reqForm = { ...state.form };
    reqForm.password = await RsaEncrypt(reqForm.password);
    if (!state.form.sshTunnelMachineId) {
        reqForm.sshTunnelMachineId = -1;
    }
    return reqForm;
};

const testConn = async () => {
    dbForm.value.validate(async (valid: boolean) => {
        if (valid) {
            state.testConnBtnLoading = true;
            try {
                await dbApi.testConn.request(await getReqForm());
                ElMessage.success('连接成功');
            } finally {
                state.testConnBtnLoading = false;
            }
        } else {
            ElMessage.error('请正确填写信息');
            return false;
        }
    });
};

const btnOk = async () => {
    if (!state.form.id) {
        notBlank(state.form.password, '新增操作，密码不可为空');
    } else if (state.form.username != state.oldUserName) {
        notBlank(state.form.password, '已修改用户名，请输入密码');
    }

    dbForm.value.validate(async (valid: boolean) => {
        if (valid) {
            dbApi.saveInstance.request(await getReqForm()).then(() => {
                ElMessage.success('保存成功');
                emit('val-change', state.form);
                state.btnLoading = true;
                setTimeout(() => {
                    state.btnLoading = false;
                }, 1000);

                cancel();
            });
        } else {
            ElMessage.error('请正确填写信息');
            return false;
        }
    });
};

const cancel = () => {
    emit('update:visible', false);
    emit('cancel');
};
</script>
<style lang="scss"></style>
