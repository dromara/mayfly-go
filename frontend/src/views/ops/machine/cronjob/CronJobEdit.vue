<template>
    <div class="mock-data-dialog">
        <el-drawer
            :title="title"
            v-model="dialogVisible"
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
                    <tag-tree-check height="200px" :tag-type="TagResourceTypeEnum.Machine.value" v-model="form.codePaths" />
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
import { ref, toRefs, reactive, watch, onMounted } from 'vue';
import { ElMessage } from 'element-plus';
import { cronJobApi, machineApi } from '../api';
import { CronJobStatusEnum, CronJobSaveExecResTypeEnum } from '../enums';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import CrontabInput from '@/components/crontab/CrontabInput.vue';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import TagTreeCheck from '../../component/TagTreeCheck.vue';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import EnumSelect from '@/components/enumselect/EnumSelect.vue';
import { useI18n } from 'vue-i18n';
import { useI18nFormValidate, useI18nPleaseInput, useI18nPleaseSelect, useI18nSaveSuccessMsg } from '@/hooks/useI18n';

const { t } = useI18n();

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
});

const emit = defineEmits(['update:visible', 'cancel', 'submitSuccess']);

const formRef: any = ref(null);

const rules = {
    name: [
        {
            required: true,
            message: useI18nPleaseInput('common.name'),
            trigger: ['change', 'blur'],
        },
    ],
    cron: [
        {
            required: true,
            message: useI18nPleaseInput('machine.cronExpression'),
            trigger: ['change', 'blur'],
        },
    ],
    status: [
        {
            required: true,
            message: useI18nPleaseSelect('common.status'),
            trigger: ['change', 'blur'],
        },
    ],
    saveExecResType: [
        {
            required: true,
            message: useI18nPleaseSelect('machine.execResRecordType'),
            trigger: ['change', 'blur'],
        },
    ],
    script: [
        {
            required: true,
            message: useI18nPleaseInput('machine.script'),
            trigger: ['change', 'blur'],
        },
    ],
};

const state = reactive({
    dialogVisible: false,
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
        codePaths: [],
    },
    machines: [] as any,
    btnLoading: false,
});

const { dialogVisible, submitDisabled, form, btnLoading } = toRefs(state);

onMounted(async () => {
    const res = await machineApi.list.request({ pageNum: 1, pageSize: 100 });
    state.machines = res.list;
});

watch(props, async (newValue: any) => {
    state.dialogVisible = newValue.visible;
    if (!newValue.visible) {
        return;
    }
    if (newValue.data) {
        state.form = { ...newValue.data };
        state.form.codePaths = newValue.data.tags?.map((tag: any) => tag.codePath);
    } else {
        state.form = { script: '', status: 1 } as any;
        state.chooseMachines = [];
    }
});

const btnOk = async () => {
    try {
        await useI18nFormValidate(formRef);
        state.submitDisabled = true;
        await cronJobApi.save.request(state.form);
        useI18nSaveSuccessMsg();
        emit('submitSuccess');
        cancel();
    } finally {
        state.submitDisabled = false;
    }
};

const cancel = () => {
    emit('update:visible', false);
    emit('cancel');
};
</script>
<style lang="scss"></style>
