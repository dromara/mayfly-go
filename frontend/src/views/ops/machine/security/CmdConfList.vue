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
                    <el-button v-auth="'cmdconf:save'" class="ml-1" type="primary" circle size="small" icon="Plus" @click="onOpenFormDialog(null)">
                    </el-button>
                </template>
                <template #default="scope">
                    <el-button v-auth="'cmdconf:save'" @click="onOpenFormDialog(scope.row)" type="primary" link>{{ $t('common.edit') }}</el-button>
                    <el-button v-auth="'cmdconf:del'" @click="onDeleteCmdConf(scope.row)" type="danger" link>{{ $t('common.delete') }}</el-button>
                </template>
            </el-table-column>
        </el-table>

        <auto-form-drawer ref="drawerRef" v-model:visible="dialogVisible" :title="$t('machine.cmdConfig')" :items="items" :data="editForm" size="40%" @confirm="onSubmitForm">
            <!-- 过滤命令（动态标签输入） -->
            <template #cmds="{ form }">
                <el-row>
                    <el-tag
                        class="ml-0.5 mt-0.5"
                        v-for="tag in form.cmds"
                        :key="tag"
                        closable
                        :disable-transitions="false"
                        @close="onCmdClose(form, tag)"
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
                        @keyup.enter="onCmdInputConfirm(form)"
                        @blur="onCmdInputConfirm(form)"
                        :placeholder="$t('machine.cmdPlaceholder')"
                    />
                    <el-button v-else class="ml-0.5 mt-0.5" size="small" @click="onShowCmdInput"> + {{ $t('machine.newCmd') }} </el-button>
                </el-row>
            </template>

            <!-- 关联机器 -->
            <template #codePaths="{ form }">
                <tag-tree-check
                    height="calc(100vh - 430px)"
                    :tag-type="`${TagResourceTypeEnum.Machine.value}/${TagResourceTypeEnum.AuthCert.value}`"
                    v-model="form.codePaths"
                />
            </template>

            <template #footer="{ form }">
                <el-button :loading="submiting" @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
                <el-button v-auth="'cmdconf:save'" type="primary" :loading="submiting" @click="onSubmitForm(form)">{{ $t('common.confirm') }}</el-button>
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { Rules } from '@/common/rule';
import { deepClone } from '@/common/utils/object';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import { Msg, useI18nDeleteConfirm, useI18nFormValidate } from '@/hooks/useI18n';
import { nextTick, onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import type { InputInstance } from 'element-plus';
import TagCodePath from '../../component/TagCodePath.vue';
import TagTreeCheck from '../../component/TagTreeCheck.vue';
import { cmdConfApi } from '../api';
import type { MachineCmdConfVO } from '../types';
import type { ResourceTag } from '@/types/common';

/** 表单声明（AutoFormItem[]；命令标签输入与关联机器走 custom 插槽） */
const items: AutoFormItem[] = [
    { prop: 'name', label: 'common.name', required: true },
    { prop: 'cmds', label: 'machine.filterCmds', type: 'custom', rules: [Rules.requiredInput('machine.cmd')] },
    { prop: 'remark', label: 'common.remark', type: 'textarea', rows: 2 },
    { prop: 'codePaths', label: 'machine.relateMachine', type: 'custom' },
];

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('drawerRef');
const cmdInputRef = useTemplateRef<InputInstance>('cmdInputRef');

const DefaultForm = {
    id: 0,
    name: '',
    codePaths: [] as string[],
    cmds: [] as string[],
    remark: '',
};

const state = reactive({
    cmdConfs: [] as MachineCmdConfVO[],
    dialogVisible: false,
    submiting: false,
    inputCmdVisible: false,
    cmdInputValue: '',
});

const { cmdConfs, dialogVisible, submiting } = toRefs(state);

/** 传给 AutoFormDrawer 的回填数据（onOpenFormDialog 时设置；深拷贝由组件内部完成） */
const editForm = ref<AutoFormData | null>(null);

onMounted(async () => {
    getCmdConfs();
});

const getCmdConfs = async () => {
    state.cmdConfs = await cmdConfApi.list.request();
};

const onCmdClose = (rawForm: AutoFormData, tag: string) => {
    const form = rawForm as unknown as MachineCmdConfVO;
    form.cmds?.splice(form.cmds.indexOf(tag), 1);
};

const onShowCmdInput = () => {
    state.inputCmdVisible = true;
    nextTick(() => {
        cmdInputRef.value?.focus();
    });
};

const onCmdInputConfirm = (rawForm: AutoFormData) => {
    const form = rawForm as unknown as MachineCmdConfVO;
    if (state.cmdInputValue) {
        form.cmds?.push(state.cmdInputValue);
    }
    state.inputCmdVisible = false;
    state.cmdInputValue = '';
};

const onOpenFormDialog = (data: MachineCmdConfVO | null) => {
    if (!data) {
        editForm.value = { ...DefaultForm } as unknown as AutoFormData;
    } else {
        editForm.value = {
            ...DefaultForm,
            ...deepClone(data),
            codePaths: data.tags?.map((tag: ResourceTag) => tag.codePath) || [],
            cmds: data.cmds || [],
        } as unknown as AutoFormData;
    }
    state.dialogVisible = true;
};

const onDeleteCmdConf = async (data: MachineCmdConfVO) => {
    await useI18nDeleteConfirm(data.name);
    await cmdConfApi.delete.request({ id: data.id });
    Msg.deleteSuccess();
    getCmdConfs();
};

const onSubmitForm = async (rawForm: AutoFormData) => {
    const form = rawForm as unknown as MachineCmdConfVO;
    try {
        await useI18nFormValidate(drawerRef);
        state.submiting = true;
        await cmdConfApi.save.request(form);
        Msg.saveSuccess();

        state.dialogVisible = false;
        getCmdConfs();
    } finally {
        state.submiting = false;
    }
};
</script>
<style></style>
