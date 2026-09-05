<template>
    <div>
        <auto-form-drawer ref="drawerRef" v-model:visible="visible" :title="title" :items="items" :data="editData" size="1000px" :confirm-loading="saveBtnLoading" @confirm="onConfirm" @cancel="emit('cancel')">
            <!-- 权限账号（远程搜索，走插槽保留 remote 能力） -->
            <template #permissionAccount>
                <el-select
                    remote
                    :remote-method="getAccount"
                    v-model="permissionAccount"
                    filterable
                    multiple
                    :placeholder="$t('system.sysconf.permissionPlaceholder')"
                    class="w-full"
                >
                    <el-option v-for="item in accounts" :key="item.id" :label="`${item.username} [${item.name}]`" :value="item.username"> </el-option>
                </el-select>
            </template>

            <!-- 配置项表单定义（v1 JSON Schema 表格编辑器） -->
            <template #params>
                <auto-form-schema-edit v-model="params" />
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { computed, reactive, ref, useTemplateRef, watch } from 'vue';
import { configApi, accountApi } from '../api';
import { AutoFormDrawer, AutoFormSchemaEdit, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import { isJsonFormSchema, type AutoFormJsonSchema } from '@/components/auto-form/json';
import { useI18nFormValidate } from '@/hooks/useI18n';

/** 系统配置编辑表单类型 */
interface ConfigForm {
    id?: number | null;
    name?: string;
    key?: string;
    params?: string;
    value?: string;
    remark?: string;
    permission?: string;
}

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

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('drawerRef');

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；permissionAccount/params 走插槽承载复杂控件） */
const items: AutoFormItem[] = [
    { prop: 'name', label: 'system.sysconf.confItem', required: true },
    { prop: 'key', label: 'system.sysconf.confKey', required: true, disabled: (f) => f.id != null },
    { prop: 'permissionAccount', label: 'system.sysconf.permission', type: 'custom' },
    { prop: 'params', label: 'system.sysconf.confItem', type: 'custom' },
    { prop: 'remark', label: 'common.remark', type: 'textarea', rows: 2 },
];

/** 配置项表单定义（v1 JSON Schema，与表单字段并行维护，提交时序列化进 form.params） */
const params = ref({ version: 1, fields: [] } as AutoFormJsonSchema);

const accounts = ref<import('@/views/system/types').Account[]>([]);
/** 权限账号（与 form.permission 字符串互转的编辑态） */
const permissionAccount = ref<string[]>([]);

/** 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成） */
const editData = computed<AutoFormData | null>(() => {
    if (props.data) {
        return { ...(props.data as ConfigForm) } as unknown as AutoFormData;
    }
    return { id: null, name: '', key: '', params: '', value: '', remark: '', permission: 'all' } as unknown as AutoFormData;
});

const { isFetching: saveBtnLoading, execute: saveConfigExec } = configApi.save.useApi();

// 抽屉打开时解析入参 schema 与权限账号
watch(visible, () => {
    if (!visible.value) {
        return;
    }
    const data = props.data as ConfigForm | boolean | null;
    const form = (data && typeof data === 'object' ? data : {}) as ConfigForm;
    if (form.params) {
        try {
            const parsed = JSON.parse(form.params);
            params.value = isJsonFormSchema(parsed) ? parsed : { version: 1, fields: [] };
        } catch {
            params.value = { version: 1, fields: [] };
        }
    } else {
        params.value = { version: 1, fields: [] };
    }

    const permission = form.permission ?? '';
    if (permission != 'all') {
        const accountArr = permission.split(',');
        permissionAccount.value = accountArr.slice(0, accountArr.length - 1);
    } else {
        permissionAccount.value = [];
    }
});

const getAccount = (username: string) => {
    if (username) {
        accountApi.list.request({ username }).then((res) => {
            accounts.value = res.list;
        });
    }
};

const onConfirm = async (rawForm: AutoFormData) => {
    const form = rawForm as ConfigForm;
    await useI18nFormValidate(drawerRef);
    if (params.value) {
        form.params = JSON.stringify(params.value);
    }
    if (permissionAccount.value.length > 0) {
        form.permission = permissionAccount.value.join(',') + ',';
    } else {
        form.permission = 'all';
    }

    await saveConfigExec(form);
    emit('val-change', form);
    visible.value = false;
};
</script>
<style lang="scss"></style>
