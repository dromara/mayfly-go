<template>
    <div>
        <el-drawer :title="title" v-model="visible" :before-close="onCancel" :destroy-on-close="true" :close-on-click-modal="false" size="40%">
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
import { toRefs, reactive, watch, ref } from 'vue';
import { procdefApi } from './api';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { ProcdefStatus } from './enums';
import TagTreeCheck from '../ops/component/TagTreeCheck.vue';
import { TagResourceTypeEnum, TagResourceTypePath } from '@/common/commonEnum';
import EnumSelect from '@/components/enumselect/EnumSelect.vue';
import { useI18nFormValidate, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';
import FormItemTooltip from '@/components/form/FormItemTooltip.vue';
import { Rules } from '@/common/rule';
import MsgTmplSelect from '../msg/components/MsgTmplSelect.vue';

const { t } = useI18n();

const props = defineProps({
    data: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

const formRef: any = ref(null);

const rules = {
    name: [Rules.requiredInput('common.name')],
    defKey: [Rules.requiredInput('key')],
};

const state = reactive({
    tasks: [] as any,
    form: {
        id: null,
        name: null,
        defKey: null,
        status: null,
        condition: '',
        remark: null,
        msgTmplId: null,
        codePaths: [],
    },
});

const { form } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveFlowDefExec } = procdefApi.save.useApi(form);

watch(props, async (newValue: any) => {
    if (newValue.data) {
        state.form = await procdefApi.detail.request({ id: newValue.data.id });
        state.form.codePaths = newValue.data.tags?.map((tag: any) => tag.codePath);
    } else {
        state.form = { status: ProcdefStatus.Enable.value } as any;
        state.form.condition = t('flow.conditionDefault');
        state.tasks = [];
    }
});

const onSave = async () => {
    await useI18nFormValidate(formRef);
    await saveFlowDefExec();
    useI18nSaveSuccessMsg();
    emit('val-change', state.form);
    //重置表单域
    formRef.value.resetFields();
    state.form = {} as any;
};

const onCancel = () => {
    visible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
