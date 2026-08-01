<template>
    <div class="mock-data-dialog">
        <el-drawer
            :append-to-body="false"
            :title="title"
            v-model="visible"
            :close-on-click-modal="false"
            :before-close="cancel"
            :show-close="true"
            :destroy-on-close="true"
            size="50%"
        >
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <el-form :model="form" ref="formRef" :rules="rules" label-width="auto">
                <el-form-item prop="name" :label="$t('common.name')">
                    <el-input v-model="form.name"></el-input>
                </el-form-item>

                <el-form-item prop="cron" :label="$t('machine.cronExpression')">
                    <CrontabInput v-model="form.cron" />
                </el-form-item>

                <el-form-item prop="status" :label="$t('common.status')">
                    <EnumSelect :enums="CronJobStatusEnum" v-model="form.status" default-first-option />
                </el-form-item>

                <el-form-item prop="saveExecResType" :label="$t('machine.execResRecordType')">
                    <EnumSelect :enums="CronJobSaveExecResTypeEnum" v-model="form.saveExecResType" default-first-option />
                </el-form-item>

                <el-form-item prop="remark" :label="$t('common.remark')">
                    <el-input v-model="form.remark"></el-input>
                </el-form-item>

                <el-form-item prop="script" :label="$t('machine.script')" required>
                    <monaco-editor style="width: 100%" v-model="form.script" language="shell" height="200px"
                /></el-form-item>

                <el-form-item ref="tagSelectRef" prop="codePaths" :label="$t('machine.relateMachine')">
                    <tag-tree-check height="200px" :tag-type="`${TagResourceTypeEnum.Machine.value}`" v-model="form.codePaths" />
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel()" :disabled="submitDisabled">{{ $t('common.cancel') }}</el-button>
                    <el-button v-auth="'machine:script:save'" type="primary" :loading="btnLoading" @click="btnOk" :disabled="submitDisabled">
                        {{ $t('common.confirm') }}
                    </el-button>
                </div>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { Rules } from '@/common/rule';
import CrontabInput from '@/components/crontab/CrontabInput.vue';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import EnumSelect from '@/components/enum-select/EnumSelect.vue';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { onMounted, reactive, ref, toRefs, watch } from 'vue';
import type { FormInstance } from 'element-plus';
import TagTreeCheck from '../../component/TagTreeCheck.vue';
import { cronJobApi, machineApi } from '../api';
import { CronJobSaveExecResTypeEnum, CronJobStatusEnum } from '../enums';
import type { MachineCronJob, MachineCronJobForm } from '../types';
import type { SimpleMachineVO } from '../types';
import type { ResourceTag } from '@/types/common';

const props = defineProps({
    data: {
        type: Object as () => MachineCronJob | null,
    },
    title: {
        type: String,
    },
});

const emit = defineEmits(['cancel', 'submitSuccess']);

const visible = defineModel<boolean>('visible', { default: false });

const formRef = ref<FormInstance | null>(null);

const rules = {
    name: [Rules.requiredInput('common.name')],
    cron: [Rules.requiredInput('machine.cronExpression')],
    status: [Rules.requiredSelect('common.status')],
    saveExecResType: [Rules.requiredSelect('machine.execResRecordType')],
    script: [Rules.requiredInput('machine.script')],
};

const state = reactive({
    submitDisabled: false,
    chooseMachines: [],
    form: {
        id: null,
        name: '',
        cron: '',
        remark: '',
        script: '',
        status: 1,
        saveExecResType: -1,
        codePaths: [] as string[],
    } as MachineCronJobForm,
    machines: [] as SimpleMachineVO[],
    btnLoading: false,
});

const { submitDisabled, form, btnLoading } = toRefs(state);

onMounted(async () => {
    const res = await machineApi.list.request({ pageNum: 1, pageSize: 100 });
    state.machines = res.list;
});

watch(visible, async (val) => {
    if (!val) {
        return;
    }
    if (props.data) {
        state.form = { ...props.data };
        state.form.codePaths = props.data.tags?.map((tag: ResourceTag) => tag.codePath);
    } else {
        state.form = { script: '', status: 1 } as MachineCronJobForm;
        state.chooseMachines = [];
    }
});

const btnOk = async () => {
    try {
        await useI18nFormValidate(formRef);
        state.submitDisabled = true;
        await cronJobApi.save.request(state.form);
        Msg.saveSuccess();
        emit('submitSuccess');
        cancel();
    } finally {
        state.submitDisabled = false;
    }
};

const cancel = () => {
    visible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
