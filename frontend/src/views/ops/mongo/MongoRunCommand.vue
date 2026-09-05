<template>
    <div>
        <el-dialog width="750px" title="runCommand" v-model="runCmdDialog.visible" :before-close="close" :destroy-on-close="true">
            <auto-form v-model="runCmdDialog" :items="runCmdItems" label-width="auto">
                <template #cmdName>
                    <el-select
                        class="w-full!"
                        @change="changeCmd"
                        filterable
                        v-model="runCmdDialog.cmdName"
                        :placeholder="$t('mongo.cmdTemplatePlaceholder')"
                    >
                        <el-option v-for="item in mongoCmds" :key="item.name" :label="`${item.name} | ${item.description}`" :value="item.name" />
                    </el-select>
                </template>
                <template #db>
                    <el-select v-model="runCmdDialog.db" filterable>
                        <el-option v-for="item in dbs" :key="item.Name" :label="item.Name" :value="item.Name" />
                    </el-select>
                </template>
                <template #runBtn>
                    <el-button @click="onRunCommand" type="primary">Run</el-button>
                    <el-tooltip effect="dark" placement="top">
                        <template #content> {{ $t('mongo.moreCmdTips') }}-> https://www.mongodb.com/docs/manual/reference/command/ </template>
                        <span class="ml-2">
                            <el-icon><InfoFilled /></el-icon>
                        </span>
                    </el-tooltip>
                </template>
                <template #cmd>
                    <monaco-editor style="width: 100%" height="235px" v-model="runCmdDialog.cmd" language="json" />
                </template>
                <template #res>
                    <monaco-editor style="width: 100%" height="235px" v-model="runCmdDialog.cmdRes" language="json" />
                </template>
            </auto-form>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { Msg } from '@/hooks/useI18n';
import type { AutoFormItem } from '@/components/auto-form';
import { defineAsyncComponent, reactive, toRefs, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { mongoApi } from './api';
import type { MongoDatabase } from './types';

const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

const { t } = useI18n();

const props = defineProps({
    id: {
        type: [Number],
        required: true,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

const mongoCmds: Record<string, Record<string, unknown>> = {
    usersInfo: {
        name: 'usersInfo',
        description: t('mongo.usersInfoDesc'),
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
        description: t('mongo.createUserDesc'),
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
        description: t('mongo.grantRolesToUserDesc'),
        cmd: {
            grantRolesToUser: '<user>',
            roles: [''],
        },
    },
    dropUser: {
        name: 'dropUser',
        description: t('mongo.dropUserDesc'),
        cmd: {
            dropUser: '<user>',
        },
    },
    roleInfo: {
        name: 'roleInfo',
        description: t('mongo.roleInfoDesc'),
        cmd: {
            rolesInfo: 1,
            showAuthenticationRestrictions: false,
            showBuiltinRoles: true,
            showPrivileges: false,
        },
    },
    createRole: {
        name: 'createRole',
        description: t('mongo.createRoleDesc'),
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
    dbs: [] as MongoDatabase[],
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

/** runCommand 表单声明（模板/库选择、Run 按钮、cmd/res 编辑器为 custom 插槽） */
const runCmdItems: AutoFormItem[] = [
    { prop: 'cmdName', label: 'mongo.template', type: 'custom', span: 12 },
    { prop: 'db', label: 'mongo.db', type: 'custom', span: 8 },
    { prop: 'runBtn', type: 'custom', span: 4 },
    { prop: 'cmd', label: 'cmd', type: 'custom' },
    { prop: 'res', label: 'res', type: 'custom' },
];

watch(visible, async (val) => {
    if (!val) {
        state.runCmdDialog.visible = false;
        return;
    }
    state.runCmdDialog.visible = val;
    state.dbs = (await mongoApi.databases.request({ id: props.id })).Databases;
});

const close = () => {
    visible.value = false;
    state.runCmdDialog.cmd = '';
    state.runCmdDialog.cmdRes = '';
    state.runCmdDialog.cmdName = '';
    state.runCmdDialog.db = '';
    state.dbs = [];
};

const changeCmd = (val: string) => {
    const mongoCmd = mongoCmds[val];
    state.runCmdDialog.cmd = JSON.stringify(mongoCmd.cmd, null, 4);
    state.runCmdDialog.db = state?.dbs[0]?.Name ?? '';
    state.runCmdDialog.cmdRes = '';
};

const onRunCommand = async () => {
    const orderCmds: Record<string, unknown>[] = [];
    const cmdObj = JSON.parse(state.runCmdDialog.cmd);

    for (let item of Object.keys(cmdObj)) {
        let obj: Record<string, unknown> = {};
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
    Msg.success('mongo.runSuccess');
};
</script>

<style></style>
