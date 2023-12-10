<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :before-close="cancel" :close-on-click-modal="false" :destroy-on-close="true" width="38%">
            <el-form :model="form" ref="redisForm" :rules="rules" label-width="auto">
                <el-tabs v-model="tabActiveName">
                    <el-tab-pane label="基础信息" name="basic">
                        <el-form-item ref="tagSelectRef" prop="tagId" label="标签" required>
                            <tag-tree-select
                                @change-tag="
                                    (tagIds) => {
                                        form.tagId = tagIds;
                                        tagSelectRef.validate();
                                    }
                                "
                                multiple
                                :resource-code="form.code"
                                :resource-type="TagResourceTypeEnum.Redis.value"
                                style="width: 100%"
                            />
                        </el-form-item>
                        <el-form-item prop="name" label="名称" required>
                            <el-input v-model.trim="form.name" placeholder="请输入redis名称" auto-complete="off"></el-input>
                        </el-form-item>
                        <el-form-item prop="mode" label="mode" required>
                            <el-select style="width: 100%" v-model="form.mode" placeholder="请选择模式">
                                <el-option label="standalone" value="standalone"> </el-option>
                                <el-option label="cluster" value="cluster"> </el-option>
                                <el-option label="sentinel" value="sentinel"> </el-option>
                            </el-select>
                        </el-form-item>
                        <el-form-item prop="host" label="host" required>
                            <el-input
                                v-model.trim="form.host"
                                placeholder="请输入host:port；sentinel模式为: mastername=sentinelhost:port，若集群或哨兵需设多个节点可使用','分割"
                                auto-complete="off"
                                type="textarea"
                            ></el-input>
                        </el-form-item>
                        <el-form-item prop="username" label="用户名">
                            <el-input v-model.trim="form.username" placeholder="用户名"></el-input>
                        </el-form-item>
                        <el-form-item prop="password" label="密码">
                            <el-input
                                type="password"
                                show-password
                                v-model.trim="form.password"
                                placeholder="请输入密码, 修改操作可不填"
                                autocomplete="new-password"
                                ><template v-if="form.id && form.id != 0" #suffix>
                                    <el-popover @hide="pwd = ''" placement="right" title="原密码" :width="200" trigger="click" :content="pwd">
                                        <template #reference>
                                            <el-link @click="getPwd" :underline="false" type="primary" class="mr5">原密码</el-link>
                                        </template>
                                    </el-popover>
                                </template></el-input
                            >
                        </el-form-item>
                        <el-form-item prop="db" label="库号" required>
                            <el-select
                                @change="changeDb"
                                :disabled="form.mode == 'cluster'"
                                v-model="dbList"
                                multiple
                                allow-create
                                filterable
                                placeholder="请选择可操作库号"
                                style="width: 100%"
                            >
                                <el-option v-for="db in [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15]" :key="db" :label="db" :value="db" />
                            </el-select>
                        </el-form-item>
                        <el-form-item prop="remark" label="备注">
                            <el-input v-model.trim="form.remark" auto-complete="off" type="textarea"></el-input>
                        </el-form-item>
                    </el-tab-pane>

                    <el-tab-pane label="其他配置" name="other">
                        <el-form-item prop="sshTunnelMachineId" label="SSH隧道">
                            <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
                        </el-form-item>
                    </el-tab-pane>
                </el-tabs>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="testConn" :loading="testConnBtnLoading" type="success">测试连接</el-button>
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, ref } from 'vue';
import { redisApi } from './api';
import { ElMessage } from 'element-plus';
import { RsaEncrypt } from '@/common/rsa';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import { TagResourceTypeEnum } from '@/common/commonEnum';

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
    tagId: [
        {
            required: true,
            message: '请选择标签',
            trigger: ['blur', 'change'],
        },
    ],
    name: [
        {
            required: true,
            message: '请输入名称',
            trigger: ['change', 'blur'],
        },
    ],
    host: [
        {
            required: true,
            message: '请输入主机ip:port',
            trigger: ['change', 'blur'],
        },
    ],
    db: [
        {
            required: true,
            message: '请选择库号',
            trigger: ['change', 'blur'],
        },
    ],
    mode: [
        {
            required: true,
            message: '请选择模式',
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
        tagId: [],
        name: null,
        mode: 'standalone',
        host: '',
        username: null,
        password: null,
        db: '',
        remark: '',
        sshTunnelMachineId: -1,
    },
    submitForm: {} as any,
    dbList: [0],
    pwd: '',
});

const { dialogVisible, tabActiveName, form, submitForm, dbList, pwd } = toRefs(state);

const { isFetching: testConnBtnLoading, execute: testConnExec } = redisApi.testConn.useApi(submitForm);
const { isFetching: saveBtnLoading, execute: saveRedisExec } = redisApi.saveRedis.useApi(submitForm);

watch(props, async (newValue: any) => {
    state.dialogVisible = newValue.visible;
    if (!state.dialogVisible) {
        return;
    }
    state.tabActiveName = 'basic';
    if (newValue.redis) {
        state.form = { ...newValue.redis };
        convertDb(state.form.db);
    } else {
        state.form = { db: '0' } as any;
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

const getPwd = async () => {
    state.pwd = await redisApi.getRedisPwd.request({ id: state.form.id });
};

const getReqForm = async () => {
    const reqForm = { ...state.form };
    if (reqForm.mode == 'sentinel' && reqForm.host.split('=').length != 2) {
        ElMessage.error('sentinel模式host需为: mastername=sentinelhost:sentinelport模式');
        return;
    }
    if (!state.form.sshTunnelMachineId || state.form.sshTunnelMachineId <= 0) {
        reqForm.sshTunnelMachineId = -1;
    }
    reqForm.password = await RsaEncrypt(reqForm.password);
    return reqForm;
};

const testConn = async () => {
    redisForm.value.validate(async (valid: boolean) => {
        if (!valid) {
            ElMessage.error('请正确填写信息');
            return false;
        }

        state.submitForm = await getReqForm();
        await testConnExec();
        ElMessage.success('连接成功');
    });
};

const btnOk = async () => {
    redisForm.value.validate(async (valid: boolean) => {
        if (!valid) {
            ElMessage.error('请正确填写信息');
            return false;
        }
        state.submitForm = await getReqForm();
        await saveRedisExec();
        ElMessage.success('保存成功');
        emit('val-change', state.form);
        cancel();
    });
};

const cancel = () => {
    emit('update:visible', false);
    emit('cancel');
};
</script>
<style lang="scss"></style>
