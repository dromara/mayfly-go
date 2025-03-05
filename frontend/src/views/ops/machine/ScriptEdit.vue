<template>
    <div class="mock-data-dialog">
        <el-dialog
            :title="title"
            v-model="dialogVisible"
            :close-on-click-modal="false"
            :before-close="cancel"
            :show-close="true"
            :destroy-on-close="true"
            width="1000px"
        >
            <el-form :model="form" :rules="rules" ref="scriptForm" label-width="auto">
                <el-form-item prop="name" :label="$t('common.name')" required>
                    <el-input v-model="form.name"></el-input>
                </el-form-item>

                <el-form-item prop="description" :label="$t('common.remark')" required>
                    <el-input v-model="form.description"></el-input>
                </el-form-item>

                <el-form-item prop="type" :label="$t('common.type')" required>
                    <EnumSelect :enums="ScriptResultEnum" v-model="form.type" default-first-option />
                </el-form-item>

                <el-form-item class="w100">
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

                <el-form-item required prop="script" class="100w">
                    <div style="width: 100%">
                        <monaco-editor v-model="form.script" language="shell" height="300px" />
                    </div>
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button v-auth="'machine:script:save'" type="primary" :loading="btnLoading" @click="btnOk">
                        {{ $t('common.save') }}
                    </el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, watch } from 'vue';
import { machineApi } from './api';
import { ScriptResultEnum } from './enums';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { DynamicFormEdit } from '@/components/dynamic-form';
import SvgIcon from '@/components/svgIcon/index.vue';
import EnumSelect from '@/components/enumselect/EnumSelect.vue';
import { useI18nFormValidate, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { Rules } from '@/common/rule';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    data: {
        type: Object,
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

const emit = defineEmits(['update:visible', 'cancel', 'submitSuccess']);

const rules = {
    name: [Rules.requiredInput('common.name')],
    description: [Rules.requiredInput('common.remark')],
    type: [Rules.requiredSelect('common.type')],
    script: [Rules.requiredInput('machine.script')],
};

const { isCommon, machineId } = toRefs(props);
const scriptForm: any = ref(null);

const state = reactive({
    dialogVisible: false,
    params: [] as any,
    form: {
        id: null,
        name: '',
        machineId: 0,
        description: '',
        script: '',
        params: '',
        type: null,
    },
    btnLoading: false,
});

const { dialogVisible, params, form, btnLoading } = toRefs(state);

watch(props, (newValue: any) => {
    state.dialogVisible = newValue.visible;
    if (!newValue.visible) {
        return;
    }
    if (newValue.data) {
        state.form = { ...newValue.data };
        if (state.form.params) {
            state.params = JSON.parse(state.form.params);
        }
    } else {
        state.form = {} as any;
        state.form.script = '';
    }
});

const btnOk = async () => {
    state.form.machineId = isCommon.value ? 9999999 : (machineId?.value as any);
    await useI18nFormValidate(scriptForm);
    if (state.params) {
        state.form.params = JSON.stringify(state.params);
    }
    machineApi.saveScript.request(state.form).then(() => {
        useI18nSaveSuccessMsg();
        emit('submitSuccess');
        cancel();
    });
};

const cancel = () => {
    emit('update:visible', false);
    emit('cancel');
    state.params = [];
};
</script>
<style lang="scss"></style>
