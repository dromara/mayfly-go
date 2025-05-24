<template>
    <div>
        <el-table :data="cmdConfs" stripe>
            <el-table-column prop="name" :label="$t('common.name')" show-overflow-tooltip min-width="100px"> </el-table-column>
            <el-table-column prop="cmds" :label="$t('machine.filterCmds')" min-width="320px" show-overflow-tooltip>
                <template #default="scope">
                    <el-tag class="ml-0.5 mt-0.5" v-for="cmd in scope.row.cmds" :key="cmd" type="danger">
                        {{ cmd }}
                    </el-tag>
                </template>
            </el-table-column>
            <el-table-column prop="codePaths" :label="$t('machine.relateMachine')" min-width="250px" show-overflow-tooltip>
                <template #default="scope">
                    <TagCodePath :path="scope.row.tags" />
                </template>
            </el-table-column>
            <el-table-column prop="remark" :label="$t('common.remark')" show-overflow-tooltip width="120px"> </el-table-column>
            <el-table-column prop="creator" :label="$t('common.creator')" show-overflow-tooltip width="100px"> </el-table-column>

            <el-table-column :label="$t('common.operation')" min-width="120px">
                <template #header>
                    <el-text tag="b">{{ $t('common.operation') }}</el-text>
                    <el-button v-auth="'cmdconf:save'" class="ml-1" type="primary" circle size="small" icon="Plus" @click="onOpenFormDialog(false)">
                    </el-button>
                </template>
                <template #default="scope">
                    <el-button v-auth="'cmdconf:save'" @click="onOpenFormDialog(scope.row)" type="primary" link>{{ $t('common.edit') }}</el-button>
                    <el-button v-auth="'cmdconf:del'" @click="onDeleteCmdConf(scope.row)" type="danger" link>{{ $t('common.delete') }}</el-button>
                </template>
            </el-table-column>
        </el-table>

        <el-drawer
            :title="$t('machine.cmdConfig')"
            v-model="dialogVisible"
            :show-close="false"
            size="40%"
            :destroy-on-close="true"
            :close-on-click-modal="false"
        >
            <template #header>
                <DrawerHeader :header="$t('machine.cmdConfig')" :back="onCancelEdit" />
            </template>

            <el-form ref="formRef" :model="state.form" :rules="rules" label-width="auto">
                <el-form-item prop="name" :label="$t('common.name')" required>
                    <el-input v-model="form.name"></el-input>
                </el-form-item>

                <el-form-item prop="cmds" :label="$t('machine.filterCmds')">
                    <el-row>
                        <el-tag
                            class="ml-0.5 mt-0.5"
                            v-for="tag in form.cmds"
                            :key="tag"
                            closable
                            :disable-transitions="false"
                            @close="onCmdClose(tag)"
                            type="danger"
                        >
                            {{ tag }}
                        </el-tag>
                        <el-input
                            v-if="state.inputCmdVisible"
                            ref="cmdInputRef"
                            v-model="state.cmdInputValue"
                            class="mt-0.5"
                            size="small"
                            @keyup.enter="onCmdInputConfirm"
                            @blur="onCmdInputConfirm"
                            :placeholder="$t('machine.cmdPlaceholder')"
                        />
                        <el-button v-else class="ml-0.5 mt-0.5" size="small" @click="onShowCmdInput"> + {{ $t('machine.newCmd') }} </el-button>
                    </el-row>
                </el-form-item>

                <el-form-item :label="$t('common.remark')">
                    <el-input v-model="form.remark" type="textarea" :rows="2"></el-input>
                </el-form-item>

                <el-form-item ref="tagSelectRef" prop="codePaths" :label="$t('machine.relateMachine')">
                    <tag-tree-check
                        height="calc(100vh - 430px)"
                        :tag-type="`${TagResourceTypeEnum.Machine.value}/${TagResourceTypeEnum.AuthCert.value}`"
                        v-model="form.codePaths"
                    />
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button :loading="submiting" @click="onCancelEdit">{{ $t('common.cancel') }}</el-button>
                    <el-button v-auth="'cmdconf:save'" type="primary" :loading="submiting" @click="onSubmitForm">{{ $t('common.confirm') }}</el-button>
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
import { useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nFormValidate, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { Rules } from '@/common/rule';
import { deepClone } from '@/common/utils/object';

const rules = {
    tags: [Rules.requiredInput('machine.relateMachine')],
    cmds: [Rules.requiredInput('machine.cmd')],
    name: [Rules.requiredInput('common.name')],
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

const onCmdClose = (tag: string) => {
    state.form.cmds.splice(state.form.cmds.indexOf(tag), 1);
};

const onShowCmdInput = () => {
    state.inputCmdVisible = true;
    nextTick(() => {
        cmdInputRef.value!.input!.focus();
    });
};

const onCmdInputConfirm = () => {
    if (state.cmdInputValue) {
        state.form.cmds.push(state.cmdInputValue);
    }
    state.inputCmdVisible = false;
    state.cmdInputValue = '';
};

const onOpenFormDialog = (data: any) => {
    if (!data) {
        state.form = { ...DefaultForm };
    } else {
        state.form = deepClone(data);
        state.form.codePaths = data.tags?.map((tag: any) => tag.codePath);
        state.form.cmds = data.cmds || [];
    }
    state.dialogVisible = true;
};

const onDeleteCmdConf = async (data: any) => {
    await useI18nDeleteConfirm(data.name);
    await cmdConfApi.delete.request({ id: data.id });
    useI18nDeleteSuccessMsg();
    getCmdConfs();
};

const onCancelEdit = () => {
    state.dialogVisible = false;
    // 取消表单的校验
    setTimeout(() => {
        state.form = { ...DefaultForm };
        formRef.value.resetFields();
    }, 200);
};

const onSubmitForm = async () => {
    try {
        await useI18nFormValidate(formRef);
        state.submiting = true;
        await cmdConfApi.save.request(state.form);
        useI18nSaveSuccessMsg();

        onCancelEdit();
        getCmdConfs();
    } finally {
        state.submiting = false;
    }
};
</script>
<style></style>
