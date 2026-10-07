<template>
    <div class="dynamic-form-edit w-full!">
        <el-table :data="formItems" stripe class="w-full!">
            <el-table-column prop="name" :label="$t('components.af.modelField')" min-width="100px">
                <template #header>
                    <el-button class="ml0" type="primary" circle size="small" icon="Plus" @click="addItem()"> </el-button>
                    <span class="ml-2">{{ $t('components.af.modelField') }}</span>
                </template>
                <template #default="scope">
                    <el-input v-model="scope.row['model']" :placeholder="$t('components.af.fieldModelPlaceholder')" clearable> </el-input>
                </template>
            </el-table-column>

            <el-table-column prop="name" :label="$t('components.af.fieldLabel')" min-width="100px">
                <template #default="scope">
                    <el-input v-model="scope.row['name']" clearable> </el-input>
                </template>
            </el-table-column>

            <el-table-column prop="placeholder" :label="$t('components.af.fieldPlaceholder')" min-width="140px">
                <template #default="scope">
                    <el-input v-model="scope.row['placeholder']" clearable> </el-input>
                </template>
            </el-table-column>

            <el-table-column prop="options" :label="$t('components.af.optionalValues')" min-width="140px">
                <template #default="scope">
                    <el-input v-model="scope.row['options']" :placeholder="$t('components.af.optionalValuesPlaceholder')" clearable> </el-input>
                </template>
            </el-table-column>

            <el-table-column prop="required" :label="$t('components.af.required')" min-width="65px">
                <template #default="scope">
                    <el-checkbox v-model="scope.row['required']" />
                </template>
            </el-table-column>

            <el-table-column :label="$t('common.operation')" width="80px">
                <template #default="scope">
                    <el-button type="danger" @click="deleteItem(scope.$index)" icon="delete" plain></el-button>
                </template>
            </el-table-column>
        </el-table>
    </div>
</template>

<script lang="ts" setup>
interface FormItemConfig {
    model?: string;
    name?: string;
    placeholder?: string;
    options?: string;
    required?: boolean;
}

const formItems = defineModel<FormItemConfig[]>('modelValue', { default: () => [] });

const addItem = () => {
    formItems.value.push({});
};

const deleteItem = (index: number) => {
    formItems.value.splice(index, 1);
};
</script>
