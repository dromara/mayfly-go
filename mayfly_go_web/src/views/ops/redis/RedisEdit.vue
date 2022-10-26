<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :before-close="cancel" :close-on-click-modal="false" :destroy-on-close="true" width="38%">
            <el-form :model="form" ref="redisForm" :rules="rules" label-width="85px">
                <el-form-item prop="tagId" label="标签:" required>
                    <tag-select v-model:tag-id="form.tagId" v-model:tag-path="form.tagPath" style="width: 100%" />
                </el-form-item>
                <el-form-item prop="name" label="名称:" required>
                    <el-input v-model.trim="form.name" placeholder="请输入redis名称" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="mode" label="mode:" required>
                    <el-select style="width: 100%" v-model="form.mode" placeholder="请选择模式">
                        <el-option label="standalone" value="standalone"> </el-option>
                        <el-option label="cluster" value="cluster"> </el-option>
                        <el-option label="sentinel" value="sentinel"> </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item prop="host" label="host:" required>
                    <el-input
                        v-model.trim="form.host"
                        placeholder="请输入host:port；sentinel模式为: mastername=sentinelhost:port，若集群或哨兵需设多个节点可使用','分割"
                        auto-complete="off"
                        type="textarea"
                    ></el-input>
                </el-form-item>
                <el-form-item prop="password" label="密码:">
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
                <el-form-item prop="db" label="库号:" required>
                    <el-select @change="changeDb" v-model="dbList" multiple placeholder="请选择可操作库号" style="width: 100%">
                        <el-option v-for="db in [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15]" :key="db" :label="db" :value="db" />
                    </el-select>
                </el-form-item>
                <el-form-item prop="remark" label="备注:">
                    <el-input v-model.trim="form.remark" auto-complete="off" type="textarea"></el-input>
                </el-form-item>
                <el-form-item prop="enableSshTunnel" label="SSH隧道:">
                    <el-col :span="3">
                        <el-checkbox @change="getSshTunnelMachines" v-model="form.enableSshTunnel" :true-label="1" :false-label="-1"></el-checkbox>
                    </el-col>
                    <el-col :span="2" v-if="form.enableSshTunnel == 1"> 机器: </el-col>
                    <el-col :span="19" v-if="form.enableSshTunnel == 1">
                        <el-select style="width: 100%" v-model="form.sshTunnelMachineId" placeholder="请选择SSH隧道机器">
                            <el-option
                                v-for="item in sshTunnelMachineList"
                                :key="item.id"
                                :label="`${item.ip}:${item.port} [${item.name}]`"
                                :value="item.id"
                            >
                            </el-option>
                        </el-select>
                    </el-col>
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="btnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, watch, defineComponent, ref } from 'vue';
import { redisApi } from './api';
import { projectApi } from '../project/api.ts';
import { machineApi } from '../machine/api.ts';
import { ElMessage } from 'element-plus';
import { RsaEncrypt } from '@/common/rsa';
import TagSelect from '../component/TagSelect.vue';

export default defineComponent({
    name: 'RedisEdit',
    components: {
        TagSelect,
    },
    props: {
        visible: {
            type: Boolean,
        },
        projects: {
            type: Array,
        },
        redis: {
            type: [Boolean, Object],
        },
        title: {
            type: String,
        },
    },
    setup(props: any, { emit }) {
        const redisForm: any = ref(null);
        const state = reactive({
            dialogVisible: false,
            projects: [],
            envs: [],
            sshTunnelMachineList: [],
            form: {
                id: null,
                tagId: null as any,
                tatPath: null as any,
                name: null,
                mode: 'standalone',
                host: '',
                password: null,
                db: '',
                project: null,
                projectId: null,
                envId: null,
                env: null,
                remark: '',
                enableSshTunnel: null,
                sshTunnelMachineId: null,
            },
            dbList: [0],
            pwd: '',
            btnLoading: false,
            rules: {
                projectId: [
                    {
                        required: true,
                        message: '请选择项目',
                        trigger: ['change', 'blur'],
                    },
                ],
                envId: [
                    {
                        required: true,
                        message: '请选择环境',
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
            },
        });

        watch(props, async (newValue) => {
            state.dialogVisible = newValue.visible;
            if (!state.dialogVisible) {
                return;
            }
            state.projects = newValue.projects;
            if (newValue.redis) {
                state.form = { ...newValue.redis };
                convertDb(state.form.db);
            } else {
                state.envs = [];
                state.form = { db: '0', enableSshTunnel: -1 } as any;
                state.dbList = [];
            }
            getSshTunnelMachines();
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

        const getSshTunnelMachines = async () => {
            if (state.form.enableSshTunnel == 1 && state.sshTunnelMachineList.length == 0) {
                const res = await machineApi.list.request({ pageNum: 1, pageSize: 100 });
                state.sshTunnelMachineList = res.list;
            }
        };

        const getPwd = async () => {
            state.pwd = await redisApi.getRedisPwd.request({ id: state.form.id });
        };

        const btnOk = async () => {
            redisForm.value.validate(async (valid: boolean) => {
                if (valid) {
                    const reqForm = { ...state.form };
                    if (reqForm.mode == 'sentinel' && reqForm.host.split('=').length != 2) {
                        ElMessage.error('sentinel模式host需为: mastername=sentinelhost:sentinelport模式');
                        return;
                    }
                    reqForm.password = await RsaEncrypt(reqForm.password);
                    redisApi.saveRedis.request(reqForm).then(() => {
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

        return {
            ...toRefs(state),
            redisForm,
            changeDb,
            getSshTunnelMachines,
            getPwd,
            btnOk,
            cancel,
        };
    },
});
</script>
<style lang="scss">
</style>
