---
trigger: always_on
---

# 前端开发规范

## 技术栈

Vue 3 (Composition API) + TypeScript 5.x + Vite 5.x + Element Plus + Tailwind CSS 3.x + Pinia

## 综合示例：列表页 + 编辑对话框

### 枚举定义

```typescript
// src/views/system/enums.ts
import { EnumValue } from '@/common/Enum';

export const AccountStatusEnum = {
    Enable: EnumValue.of(1, 'system.account.statusEnable').tagTypeSuccess(),
    Disable: EnumValue.of(-1, 'system.account.statusDisable').tagTypeDanger(),
};
```

### API 定义

```typescript
// src/views/system/api.ts
import Api from '@/common/Api';

export const accountApi = {
    list: Api.newGet('/sys/accounts'),
    save: Api.newPost('/sys/accounts'),
    update: Api.newPut('/sys/accounts/{id}'),
    del: Api.newDelete('/sys/accounts/{id}'),
};
```

### 列表页

```vue
<template>
    <data-table ref="dataTableRef" :page-api="accountApi.list" :search-items="searchItems" v-model:query-form="query" :columns="columns">
        <template #tableHeader>
            <el-button v-auth="'account:add'" type="primary" @click="onAdd">
                {{ $t('common.create') }}
            </el-button>
        </template>
        <template #action="{ data }">
            <el-button link v-auth="'account:edit'" @click="onEdit(data)">
                {{ $t('common.edit') }}
            </el-button>
        </template>
    </data-table>
    <AccountEdit v-model:visible="editVisible" :data="editData" @success="onEditSuccess" />
</template>

<script lang="ts" setup>
import { ref } from 'vue';
import { accountApi } from '../api';
import { AccountStatusEnum } from '../enums';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
import { TableColumn } from '@/components/data-table';
import AccountEdit from './AccountEdit.vue';

const dataTableRef = ref();
const editVisible = ref(false);
const editData = ref<any>(null);

const query = ref({ username: '', status: null as number | null, pageNum: 1, pageSize: 10 });

const searchItems = [SearchItem.input('username', 'common.username'), SearchItem.select('status', 'common.status', AccountStatusEnum)];
const columns = [
    TableColumn.new('username', 'common.username'),
    TableColumn.new('status', 'common.status').typeTag(AccountStatusEnum),
    TableColumn.new('action', 'common.operation').isSlot().fixedRight(),
];

const onAdd = () => { editData.value = null; editVisible.value = true; };
const onEdit = (row: any) => { editData.value = row; editVisible.value = true; };
const onEditSuccess = () => { editVisible.value = false; dataTableRef.value?.search(); };
</script>
```

### 编辑对话框

```vue
<template>
    <el-dialog v-model="visible" :title="dialogTitle" width="500px" @close="onDialogClose">
        <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
            <el-form-item :label="$t('common.username')" prop="username">
                <el-input v-model="form.username" :disabled="!!form.id" />
            </el-form-item>
        </el-form>
        <template #footer>
            <el-button @click="onCancel">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" :loading="submitting" @click="onSubmit">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
import { ref, reactive, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { accountApi } from '../api';
import { useI18nOperateSuccessMsg } from '@/hooks/useI18n';

const props = defineProps<{ visible?: boolean; data?: any }>();
const emit = defineEmits(['update:visible', 'success']);

const formRef = ref();
const submitting = ref(false);
const form = reactive({ id: undefined, username: '', name: '', status: 1 });
const visible = computed({ get: () => props.visible, set: (val) => emit('update:visible', val) });
const { t } = useI18n();
const dialogTitle = computed(() => (form.id ? t('system.account.editAccount') : t('system.account.addAccount')));

watch(() => props.data, (newVal) => { newVal ? Object.assign(form, newVal) : resetForm(); }, { immediate: true });

const resetForm = () => { form.id = undefined; form.username = ''; form.name = ''; form.status = 1; formRef.value?.clearValidate(); };
const onSubmit = async () => {
    await formRef.value?.validate();
    submitting.value = true;
    try {
        form.id ? await accountApi.update.request(form) : await accountApi.save.request(form);
        useI18nOperateSuccessMsg();
        visible.value = false;
        emit('success');
    } finally { submitting.value = false; }
};
const onCancel = () => { visible.value = false; };
const onDialogClose = () => { resetForm(); };
</script>
```
