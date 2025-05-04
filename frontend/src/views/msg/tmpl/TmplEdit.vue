<template>
    <div>
        <el-drawer :title="title" v-model="visible" :before-close="cancel" :destroy-on-close="true" :close-on-click-modal="false" size="40%">
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <el-form :model="formData" ref="formRef" :rules="rules" label-position="top" label-width="auto">
                <el-form-item prop="name" :label="$t('msg.name')">
                    <el-input v-model.trim="formData.name" auto-complete="off" clearable></el-input>
                </el-form-item>

                <el-form-item prop="status" :label="$t('common.status')">
                    <EnumSelect :enums="ChannelStatusEnum" v-model="formData.status" />
                </el-form-item>

                <el-form-item prop="remark" :label="$t('common.remark')">
                    <el-input v-model.trim="formData.remark" auto-complete="off" type="textarea" clearable></el-input>
                </el-form-item>

                <el-form-item prop="channelIds" :label="$t('msg.msgChannel')">
                    <el-select v-model="formData.channelIds" multiple clearable filterable>
                        <el-option v-for="item in state.channels" :key="item.id" :label="item.name" :value="item.id">
                            {{ $t(EnumValue.getLabelByValue(ChannelTypeEnum, item.type)) }}
                            <el-divider direction="vertical" />
                            {{ item.code }}
                            <el-divider direction="vertical" />
                            {{ item.name }}
                        </el-option>
                    </el-select>
                </el-form-item>

                <el-form-item prop="msgType" :label="$t('common.type')">
                    <EnumSelect :enums="TmplTypeEnum" v-model="formData.msgType" />
                </el-form-item>

                <el-form-item prop="title" :label="$t('msg.title')">
                    <el-input v-model.trim="formData.title" auto-complete="off" clearable></el-input>
                </el-form-item>

                <FormItemTooltip prop="tmpl" :label="$t('msg.tmpl')" :tooltip="$t('msg.msgTmplTooltip')">
                    <MonacoEditor
                        class="!w-full"
                        height="200px"
                        v-model="formData.tmpl"
                        :language="EnumValue.getLabelByValue(TmplTypeEnum, formData.msgType)"
                    ></MonacoEditor>
                </FormItemTooltip>
            </el-form>

            <template #footer>
                <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { reactive, watchEffect, useTemplateRef, toRefs } from 'vue';
import { channelApi, tmplApi } from '../api';
import { useI18nFormValidate, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { Rules } from '@/common/rule';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import EnumSelect from '@/components/enumselect/EnumSelect.vue';
import { ChannelStatusEnum, TmplStatusEnum, TmplTypeEnum, ChannelTypeEnum } from '../enums';
import EnumValue from '@/common/Enum';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import FormItemTooltip from '@/components/form/FormItemTooltip.vue';

const props = defineProps({
    form: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['cancel', 'success']);

const visible = defineModel<boolean>('visible', { default: false });

const formRef: any = useTemplateRef('formRef');

const rules = {
    name: [Rules.requiredInput('msg.name')],
    type: [Rules.requiredSelect('common.type')],
    tmpl: [Rules.requiredInput('msg.tmpl')],
};

const defaultForm = () => {
    return {
        id: null,
        name: null,
        msgType: TmplTypeEnum.Text.value,
        title: '',
        tmpl: '',
        status: TmplStatusEnum.Enable.value,
        remark: '',
        channelIds: [],
        extra: {},
    };
};

const state = reactive({
    edit: false,
    form: defaultForm(),
    channels: [] as any,
});

const { form: formData } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveFormExec } = tmplApi.save.useApi(formData);

watchEffect(() => {
    if (visible.value) {
        channelApi.list.request({ pageNum: 1, pageSize: 200 }).then((res) => {
            state.channels = res?.list;
        });
    }

    const form: any = props.form;
    if (form) {
        state.form = { ...form };
        tmplApi.relateChannels.request({ id: form.id }).then((res) => {
            state.form.channelIds = res.map((item: any) => item.id);
        });
        state.edit = true;
    } else {
        state.edit = false;
        state.form = defaultForm();
    }
});

const btnOk = async () => {
    await useI18nFormValidate(formRef);
    await saveFormExec();
    useI18nSaveSuccessMsg();
    emit('success', state.form);
    //重置表单域
    formRef.value.resetFields();
    cancel();
};

const cancel = () => {
    visible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
