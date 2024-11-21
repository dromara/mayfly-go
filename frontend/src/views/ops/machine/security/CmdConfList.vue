<template>
    <div>
        <el-table :data="cmdConfs" stripe>
            <el-table-column prop="name" :label="$t('common.name')" show-overflow-tooltip min-width="100px"> </el-table-column>
            <el-table-column prop="cmds" :label="$t('machine.filterCmds')" min-width="320px" show-overflow-tooltip>
                <template #default="scope">
                    <el-tag class="ml2 mt2" v-for="cmd in scope.row.cmds" :key="cmd" type="danger">
                        {{ cmd }}
                    </el-tag>
                </template>
            </el-table-column>
            <el-table-column prop="codePaths" :label="$t('machine.relateMachine')" min-width="250px" show-overflow-tooltip>
                <template #default="scope">
                    <TagCodePath :path="scope.row.tags?.map((tag: any) => tag.codePath)" />
                </template>
            </el-table-column>
            <el-table-column prop="remark" :label="$t('common.remark')" show-overflow-tooltip width="120px"> </el-table-column>
            <el-table-column prop="creator" :label="$t('common.creator')" show-overflow-tooltip width="100px"> </el-table-column>

            <el-table-column :label="$t('common.operation')" min-width="120px">
                <template #header>
                    <el-text tag="b">{{ $t('common.operation') }}</el-text>
                    <el-button v-auth="'cmdconf:save'" class="ml5" type="primary" circle size="small" icon="Plus" @click="openFormDialog(false)"> </el-button>
                </template>
                <template #default="scope">
                    <el-button v-auth="'cmdconf:save'" @click="openFormDialog(scope.row)" type="primary" link>{{ $t('common.edit') }}</el-button>
                    <el-button v-auth="'cmdconf:del'" @click="deleteCmdConf(scope.row)" type="danger" link>{{ $t('common.delete') }}</el-button>
                </template>
            </el-table-column>
        </el-table>

        <el-drawer
            :title="$t('machine.cmdConfig')"
            v-model="dialogVisible"
            :show-close="false"
            width="600px"
            :destroy-on-close="true"
            :close-on-click-modal="false"
        >
            <template #header>
                <DrawerHeader :header="$t('machine.cmdConfig')" :back="cancelEdit" />
            </template>

            <el-form ref="formRef" :model="state.form" :rules="rules" label-width="auto">
                <el-form-item prop="name" :label="$t('common.name')" required>
                    <el-input v-model="form.name"></el-input>
                </el-form-item>

                <el-form-item prop="cmds" :label="$t('machine.filterCmds')" required>
                    <el-row>
                        <el-tag
                            class="ml2 mt2"
                            v-for="tag in form.cmds"
                            :key="tag"
                            closable
                            :disable-transitions="false"
                            @close="handleCmdClose(tag)"
                            type="danger"
                        >
                            {{ tag }}
                        </el-tag>
                        <el-input
                            v-if="state.inputCmdVisible"
                            ref="cmdInputRef"
                            v-model="state.cmdInputValue"
                            class="mt3"
                            size="small"
                            @keyup.enter="handleCmdInputConfirm"
                            @blur="handleCmdInputConfirm"
                            :placeholder="$t('machine.cmdPlaceholder')"
                        />
                        <el-button v-else class="ml2 mt2" size="small" @click="showCmdInput"> + {{ $t('machine.newCmd') }} </el-button>
                    </el-row>
                </el-form-item>

                <el-form-item :label="$t('common.remark')">
                    <el-input v-model="form.remark" type="textarea" :rows="2"></el-input>
                </el-form-item>

                <el-form-item ref="tagSelectRef" prop="codePaths" :label="$t('machine.relateMachine')">
                    <tag-tree-check height="calc(100vh - 430px)" :tag-type="TagResourceTypeEnum.MachineAuthCert.value" v-model="form.codePaths" />
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button :loading="submiting" @click="cancelEdit">{{ $t('common.cancel') }}</el-button>
                    <el-button v-auth="'cmdconf:save'" type="primary" :loading="submiting" @click="submitForm">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted, nextTick } from 'vue';
import TagTreeCheck from '../../component/TagTreeCheck.vue';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { cmdConfApi } from '../api';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import TagCodePath from '../../component/TagCodePath.vue';
import _ from 'lodash';
import { useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nFormValidate, useI18nPleaseInput, useI18nSaveSuccessMsg } from '@/hooks/useI18n';

const rules = {
    tags: [
        {
            required: true,
            message: useI18nPleaseInput('machine.relateMachine'),
            trigger: ['change'],
        },
    ],
    cmds: [
        {
            required: true,
            message: useI18nPleaseInput('machine.cmd'),
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
};

const tagSelectRef: any = ref(null);
const formRef: any = ref(null);
const cmdInputRef: any = ref(null);

const DefaultForm = {
    id: 0,
    name: '',
    codePaths: [],
    cmds: [] as any,
    remark: '',
};

const state = reactive({
    cmdConfs: [],
    dialogVisible: false,
    form: DefaultForm,
    submiting: false,
    inputCmdVisible: false,
    cmdInputValue: '',
});

const { cmdConfs, dialogVisible, form, submiting } = toRefs(state);

onMounted(async () => {
    getCmdConfs();
});

const getCmdConfs = async () => {
    state.cmdConfs = await cmdConfApi.list.request();
};

const handleCmdClose = (tag: string) => {
    state.form.cmds.splice(state.form.cmds.indexOf(tag), 1);
};

const showCmdInput = () => {
    state.inputCmdVisible = true;
    nextTick(() => {
        cmdInputRef.value!.input!.focus();
    });
};

const handleCmdInputConfirm = () => {
    if (state.cmdInputValue) {
        state.form.cmds.push(state.cmdInputValue);
    }
    state.inputCmdVisible = false;
    state.cmdInputValue = '';
};

const openFormDialog = (data: any) => {
    if (!data) {
        state.form = { ...DefaultForm };
    } else {
        state.form = _.cloneDeep(data);
        state.form.codePaths = data.tags?.map((tag: any) => tag.codePath);
    }
    state.dialogVisible = true;
};

const deleteCmdConf = async (data: any) => {
    await useI18nDeleteConfirm(data.name);
    await cmdConfApi.delete.request({ id: data.id });
    useI18nDeleteSuccessMsg();
    getCmdConfs();
};

const cancelEdit = () => {
    state.dialogVisible = false;
    // 取消表单的校验
    setTimeout(() => {
        state.form = { ...DefaultForm };
        formRef.value.resetFields();
    }, 200);
};

const submitForm = async () => {
    try {
        await useI18nFormValidate(formRef);
        state.submiting = true;
        await cmdConfApi.save.request(state.form);
        useI18nSaveSuccessMsg();

        cancelEdit();
        getCmdConfs();
    } finally {
        state.submiting = false;
    }
};
</script>
<style></style>
