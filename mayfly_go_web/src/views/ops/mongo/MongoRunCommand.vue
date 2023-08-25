<template>
    <div>
        <el-dialog width="700px" title="runCommand" v-model="runCmdDialog.visible" :before-close="close" :destroy-on-close="true">
            <el-form label-width="auto">
                <el-row class="mb10">
                    <el-col :span="12">
                        <el-form-item label="模板">
                            <el-select class="w100" @change="changeCmd" filterable v-model="runCmdDialog.cmdName" placeholder="选择命令模板">
                                <el-option v-for="item in mongoCmds" :key="item.name" :label="`${item.name} | ${item.description}`" :value="item.name" />
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :span="8">
                        <el-form-item label="库">
                            <el-select v-model="runCmdDialog.db" filterable placeholder="选择库">
                                <el-option v-for="item in dbs" :key="item.Name" :label="item.Name" :value="item.Name" />
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :span="4">
                        <el-form-item class="ml10">
                            <el-button @click="onRunCommand" type="primary">Run</el-button>
                            <el-tooltip effect="dark" placement="top">
                                <template #content> 更多命令查看-> https://www.mongodb.com/docs/manual/reference/command/ </template>
                                <span class="ml10">
                                    <el-icon><InfoFilled /></el-icon>
                                </span>
                            </el-tooltip>
                        </el-form-item>
                    </el-col>
                </el-row>

                <el-form-item label="cmd">
                    <monaco-editor style="width: 100%" height="235px" v-model="runCmdDialog.cmd" language="json" />
                </el-form-item>

                <el-form-item label="res">
                    <monaco-editor style="width: 100%" height="235px" v-model="runCmdDialog.cmdRes" language="json" />
                </el-form-item>
            </el-form>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { mongoApi } from './api';
import { watch, defineAsyncComponent, toRefs, reactive } from 'vue';
import { ElMessage } from 'element-plus';

const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

const props = defineProps({
    visible: {
        type: Boolean,
    },
    id: {
        type: [Number],
        required: true,
    },
});

//定义事件
const emit = defineEmits(['update:visible']);

const mongoCmds = {
    usersInfo: {
        name: 'usersInfo',
        description: '获取用户信息',
        cmd: {
            usersInfo: 1,
            showCredentials: false,
            showCustomData: false,
            showPrivileges: false,
            showAuthenticationRestrictions: false,
            filter: {},
        },
    },
    createUser: {
        name: 'createUser',
        description: '创建新用户',
        cmd: {
            createUser: '<username>',
            pwd: '<cleartext password>',
            roles: [
                {
                    role: '<role>',
                    db: '<database>',
                },
            ],
        },
    },
    grantRolesToUser: {
        name: 'grantRolesToUser',
        description: '授予对用户的额外角色',
        cmd: {
            grantRolesToUser: '<user>',
            roles: [''],
        },
    },
    dropUser: {
        name: 'dropUser',
        description: '删除用户',
        cmd: {
            dropUser: '<user>',
        },
    },
    roleInfo: {
        name: 'roleInfo',
        description: '获取角色信息',
        cmd: {
            rolesInfo: 1,
            showAuthenticationRestrictions: false,
            showBuiltinRoles: true,
            showPrivileges: false,
        },
    },
    createRole: {
        name: 'createRole',
        description: '创建角色',
        cmd: {
            createRole: '<new role>',
            privileges: [{ resource: {}, actions: ['<action>'] }],
            roles: [{ role: '<role>', db: '<database>' }],
            authenticationRestrictions: [
                {
                    clientSource: ['<IP> | <CIDR range>'],
                    serverAddress: ['<IP> |<CIDR range>'],
                },
            ],
            writeConcern: '<write concern document>',
            comment: '<any>',
        },
    },
};

const state = reactive({
    dbs: [] as any,
    selectDbDisabled: false,
    runCmdDialog: {
        visible: false,
        cmdName: '',
        db: '',
        cmd: '',
        cmdRes: '',
    },
});

const { dbs, runCmdDialog } = toRefs(state);

watch(props, async (newValue: any) => {
    if (!newValue.visible) {
        state.runCmdDialog.visible = false;
        return;
    }
    state.runCmdDialog.visible = newValue.visible;
    state.dbs = (await mongoApi.databases.request({ id: props.id })).Databases;
});

const close = () => {
    emit('update:visible', false);
    state.runCmdDialog.cmd = '';
    state.runCmdDialog.cmdRes = '';
    state.runCmdDialog.cmdName = '';
    state.runCmdDialog.db = '';
    state.dbs = [];
};

const changeCmd = (val: any) => {
    const mongoCmd = mongoCmds[val];
    state.runCmdDialog.cmd = JSON.stringify(mongoCmd.cmd, null, 4);
    state.runCmdDialog.db = state?.dbs[0]?.Name;
    state.runCmdDialog.cmdRes = '';
};

const onRunCommand = async () => {
    const orderCmds = [] as any;
    const cmdObj = JSON.parse(state.runCmdDialog.cmd);

    for (let item of Object.keys(cmdObj)) {
        let obj = {};
        obj[item] = cmdObj[item];
        orderCmds.push(obj);
    }

    state.runCmdDialog.cmdRes = '';
    const res = await mongoApi.runCommand.request({
        id: props.id,
        database: state.runCmdDialog.db,
        command: orderCmds,
    });
    state.runCmdDialog.cmdRes = JSON.stringify(res, null, 4);
    ElMessage.success('执行成功');
};
</script>

<style></style>
