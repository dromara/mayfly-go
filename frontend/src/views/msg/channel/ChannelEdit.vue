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

                <el-form-item prop="type" :label="$t('common.type')">
                    <EnumSelect
                        :enums="ChannelTypeEnum"
                        v-model="formData.type"
                        @change="
                            () => {
                                formData.extra = {};
                            }
                        "
                    />
                </el-form-item>

                <el-form-item prop="url" label="URL">
                    <el-input v-model.trim="formData.url" auto-complete="off" clearable></el-input>
                </el-form-item>

                <component v-if="channelTypeComp" :is="channelTypeComp" v-model:extra="formData.extra" />
            </el-form>

            <template #footer>
                <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watchEffect, useTemplateRef, shallowReactive, computed } from 'vue';
import { channelApi } from '../api';
import { useI18nFormValidate, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { Rules } from '@/common/rule';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import EnumSelect from '@/components/enumselect/EnumSelect.vue';
import { ChannelStatusEnum, ChannelTypeEnum } from '../enums';
import EnumValue from '@/common/Enum';
import ChannelEmail from './ChannelEmail.vue';
import ChannelDing from './ChannelDing.vue';

const props = defineProps({
    form: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

const channels: any = shallowReactive({
    ChannelEmail,
    ChannelDing,
});

const channelTypeComp = computed(() => {
    return channels[EnumValue.getEnumByValue(ChannelTypeEnum, state.form.type)?.extra?.component];
});

//定义事件
const emit = defineEmits(['cancel', 'success']);

const visible = defineModel<boolean>('visible', { default: false });

const formRef: any = useTemplateRef('formRef');

const rules = {
    name: [Rules.requiredInput('msg.name')],
    type: [Rules.requiredSelect('common.type')],
    url: [Rules.requiredInput('URL')],
};

const defaultForm = () => {
    return {
        id: null,
        name: null,
        type: null,
        url: '',
        status: ChannelStatusEnum.Enable.value,
        remark: '',
        extra: {},
    };
};

const state = reactive({
    edit: false,
    form: defaultForm(),
});

const { form: formData } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveFormExec } = channelApi.save.useApi(formData);

watchEffect(() => {
    const form: any = props.form;
    if (form) {
        state.form = { ...form };
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
