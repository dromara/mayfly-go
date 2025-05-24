<template>
    <div>
        <el-dialog :title="title" v-model="visible" :show-close="false" :before-close="onCancel" width="600px" :destroy-on-close="true">
            <el-form ref="roleFormRef" :model="form" :rules="rules" label-width="auto">
                <el-form-item prop="name" :label="$t('system.role.roleName')" required>
                    <el-input v-model="form.name" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="code" :label="$t('system.role.roleCode')" required>
                    <el-input
                        :disabled="form.id != null"
                        v-model="form.code"
                        :placeholder="$t('system.role.roleCodePlaceholder')"
                        auto-complete="off"
                    ></el-input>
                </el-form-item>
                <el-form-item prop="status" :label="$t('common.status')" required>
                    <EnumSelect :enums="RoleStatusEnum" v-model="form.status" />
                </el-form-item>
                <el-form-item :label="$t('common.remark')">
                    <el-input v-model="form.remark" type="textarea" :rows="3"></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watchEffect, useTemplateRef } from 'vue';
import { roleApi } from '../api';
import { RoleStatusEnum } from '../enums';
import EnumSelect from '@/components/enumselect/EnumSelect.vue';
import { useI18nFormValidate } from '@/hooks/useI18n';
import { Rules } from '@/common/rule';

const rules = {
    name: [Rules.requiredInput('system.role.roleName')],
    code: [Rules.requiredInput('system.role.roleCode')],
    status: [Rules.requiredSelect('common.status')],
};

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

const roleFormRef: any = useTemplateRef('roleFormRef');

const state = reactive({
    form: {
        id: null,
        name: '',
        code: '',
        status: 1,
        remark: '',
    },
});

const { form } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveRoleExec } = roleApi.save.useApi(form);

watchEffect(() => {
    if (!visible.value) {
        return;
    }
    if (props.data) {
        state.form = { ...(props.data as any) };
    } else {
        state.form = {} as any;
    }
});

const onCancel = () => {
    visible.value = false;
    // 若父组件有取消事件，则调用
    emit('cancel');
};

const onConfirm = async () => {
    await useI18nFormValidate(roleFormRef);
    await saveRoleExec();
    emit('val-change', state.form);
    onCancel();
};
</script>
<style lang="scss"></style>
