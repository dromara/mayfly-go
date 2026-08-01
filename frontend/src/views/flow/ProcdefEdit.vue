<template>
    <div>
        <el-drawer :append-to-body="false" :title="title" v-model="visible" :before-close="onCancel" :destroy-on-close="true" :close-on-click-modal="false" size="40%">
            <template #header>
                <DrawerHeader :header="title" :back="onCancel" />
            </template>

            <el-form :model="form" ref="formRef" :rules="rules" label-width="auto">
                <el-form-item prop="name" :label="$t('common.name')">
                    <el-input v-model.trim="form.name" auto-complete="off" clearable></el-input>
                </el-form-item>
                <el-form-item prop="defKey" label="Key">
                    <el-input :disabled="form.id" v-model.trim="form.defKey" auto-complete="off" clearable></el-input>
                </el-form-item>
                <el-form-item prop="status" :label="$t('common.status')">
                    <EnumSelect :enums="ProcdefStatus" v-model="form.status" />
                </el-form-item>

                <FormItemTooltip prop="condition" :label="$t('flow.triggeringCondition')" :tooltip="$t('flow.triggeringConditionTips')">
                    <el-input
                        v-model="form.condition"
                        :rows="10"
                        type="textarea"
                        :placeholder="$t('flow.conditionPlaceholder')"
                        auto-complete="off"
                        clearable
                    ></el-input>
                </FormItemTooltip>

                <el-form-item prop="remark" :label="$t('common.remark')">
                    <el-input v-model.trim="form.remark" auto-complete="off" clearable></el-input>
                </el-form-item>

                <el-form-item prop="msgTmplId" :label="$t('flow.notify')">
                    <MsgTmplSelect v-model="form.msgTmplId" clearable />
                </el-form-item>

                <el-form-item ref="tagSelectRef" prop="codePaths" :label="$t('tag.relateTag')">
                    <tag-tree-check height="300px" v-model="form.codePaths" :tag-type="[TagResourceTypePath.Db, TagResourceTypeEnum.Redis.value]" />
                </el-form-item>
            </el-form>

            <template #footer>
                <div>
                    <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="onSave">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { TagResourceTypeEnum, TagResourceTypePath } from '@/common/commonEnum';
import { Rules } from '@/common/rule';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import EnumSelect from '@/components/enum-select/EnumSelect.vue';
import FormItemTooltip from '@/components/form/FormItemTooltip.vue';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { reactive, ref, toRefs, watch, type PropType } from 'vue';
import type { FormInstance } from 'element-plus';
import { useI18n } from 'vue-i18n';
import MsgTmplSelect from '../msg/components/MsgTmplSelect.vue';
import TagTreeCheck from '../ops/component/TagTreeCheck.vue';
import { procdefApi } from './api';
import { ProcdefStatus } from './enums';
import type { Procdef } from './types';

const { t } = useI18n();

const props = defineProps({
    data: {
        type: Object as PropType<Procdef | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

const formRef = ref<FormInstance | null>(null);

interface ProcdefForm extends Partial<Procdef> {
    msgTmplId?: number | null;
    codePaths?: string[];
}

const rules = {
    name: [Rules.requiredInput('common.name')],
    defKey: [Rules.requiredInput('key')],
};

const state = reactive<{
    tasks: unknown[];
    form: ProcdefForm;
}>({
    tasks: [],
    form: {
        id: null,
        name: null,
        defKey: null,
        status: null,
        condition: '',
        remark: null,
        msgTmplId: null,
        codePaths: [],
    } as unknown as ProcdefForm,
});

const { form } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveFlowDefExec } = procdefApi.save.useApi(form);

watch(props, async (newValue: Record<string, unknown>) => {
    if (newValue.data) {
        const data = newValue.data as Record<string, unknown>;
        state.form = await procdefApi.detail.request({ id: (data as { id: number }).id });
        state.form.codePaths = (data.tags as Array<{ codePath: string }>)?.map((tag) => tag.codePath);
    } else {
        state.form = { status: ProcdefStatus.Enable.value } as Procdef;
        state.form.condition = t('flow.conditionDefault');
        state.tasks = [];
    }
});

const onSave = async () => {
    await useI18nFormValidate(formRef);
    await saveFlowDefExec();
    Msg.saveSuccess();
    emit('val-change', state.form);
    //重置表单域
    formRef.value?.resetFields();
    state.form = {} as Procdef;
};

const onCancel = () => {
    visible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
