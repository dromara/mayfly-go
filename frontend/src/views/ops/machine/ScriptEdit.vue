<template>
    <div>
        <el-drawer
            :append-to-body="false"
            :title="title"
            v-model="dialogVisible"
            :close-on-click-modal="false"
            :before-close="onCancel"
            :show-close="true"
            :destroy-on-close="true"
            size="1000px"
            header-class="mb-1!"
        >
            <template #header>
                <DrawerHeader :header="title" :back="onCancel" />
            </template>

            <el-form :model="form" :rules="rules" ref="scriptForm" label-position="top">
                <el-form-item prop="name" :label="$t('common.name')" required>
                    <el-input v-model="form.name"></el-input>
                </el-form-item>

                <el-form-item prop="description" :label="$t('common.remark')" required>
                    <el-input v-model="form.description"></el-input>
                </el-form-item>

                <el-form-item prop="type" :label="$t('common.type')" required>
                    <EnumSelect :enums="ScriptResultEnum" v-model="form.type" default-first-option />
                </el-form-item>

                <el-form-item prop="category" :label="$t('machine.category')">
                    <el-select v-model="form.category" filterable allow-create :placeholder="$t('machine.categoryTips')">
                        <el-option v-for="item in categorys" :key="item" :label="item" :value="item" />
                    </el-select>
                </el-form-item>

                <el-form-item class="w-full!">
                    <template #label>
                        <el-tooltip placement="top">
                            <template #content>
                                <span>{{ $t('machine.scriptParamTips1') }}</span>
                                <br />{{ $t('machine.scriptParamTips2') }}
                            </template>
                            <span> {{ $t('machine.scriptParam') }}<SvgIcon name="question-filled" /> </span>
                        </el-tooltip>
                    </template>
                    <dynamic-form-edit v-model="params" />
                </el-form-item>

                <el-form-item required prop="script">
                    <div class="w-full">
                        <monaco-editor v-model="form.script" language="shell" height="300px" />
                    </div>
                </el-form-item>
            </el-form>

            <template #footer>
                <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                <el-button v-auth="'machine:script:save'" type="primary" :loading="btnLoading" @click="onConfirm">
                    {{ $t('common.save') }}
                </el-button>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { Rules } from '@/common/rule';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { DynamicFormEdit } from '@/components/dynamic-form';
import EnumSelect from '@/components/enum-select/EnumSelect.vue';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import SvgIcon from '@/components/svg-icon/index.vue';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { reactive, ref, toRefs, useTemplateRef, watch } from 'vue';
import type { FormInstance } from 'element-plus';
import { machineApi } from './api';
import { ScriptResultEnum } from './enums';
import type { MachineScriptForm, MachineScriptVO } from './types';

const props = defineProps({
    data: {
        type: Object as () => MachineScriptVO | null,
    },
    title: {
        type: String,
    },
    machineId: {
        type: Number,
    },
    isCommon: {
        type: Boolean,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits(['cancel', 'submitSuccess']);

const rules = {
    name: [Rules.requiredInput('common.name')],
    description: [Rules.requiredInput('common.remark')],
    type: [Rules.requiredSelect('common.type')],
    script: [Rules.requiredInput('machine.script')],
};

const { isCommon, machineId } = toRefs(props);
const scriptForm = useTemplateRef<FormInstance>('scriptForm');
const categorys = ref([] as string[]);

const state = reactive({
    params: [] as Record<string, unknown>[],
    form: {
        id: null,
        name: '',
        machineId: 0,
        description: '',
        script: '',
        params: '',
        type: null,
        category: '',
    } as MachineScriptForm,
    btnLoading: false,
});

const { params, form, btnLoading } = toRefs(state);

watch(props, (newValue: Record<string, unknown>) => {
    if (!dialogVisible.value) {
        return;
    }
    machineApi.scriptCategorys.request().then((res: string[]) => {
        categorys.value = res;
    });
    if (newValue.data) {
        state.form = { ...(newValue.data as MachineScriptForm) };
        if (state.form.params) {
            state.params = JSON.parse(state.form.params);
        }
    } else {
        state.form = {} as MachineScriptForm;
        state.form.script = '';
    }
});

const onConfirm = async () => {
    state.form.machineId = isCommon.value ? 9999999 : (machineId?.value as number);
    await useI18nFormValidate(scriptForm);
    if (state.params) {
        state.form.params = JSON.stringify(state.params);
    }
    machineApi.saveScript.request(state.form).then(() => {
        Msg.saveSuccess();
        emit('submitSuccess');
        onCancel();
    });
};

const onCancel = () => {
    dialogVisible.value = false;
    emit('cancel');
    state.params = [];
};
</script>
<style lang="scss"></style>
