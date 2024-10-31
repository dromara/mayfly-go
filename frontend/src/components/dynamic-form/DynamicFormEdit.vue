<template>
    <div class="dynamic-form-edit w100">
        <el-table :data="formItems" stripe class="w100" empty-text="暂无表单项">
            <el-table-column prop="name" label="model" min-width="100px">
                <template #header>
                    <el-button class="ml0" type="primary" circle size="small" icon="Plus" @click="addItem()"> </el-button>
                    <span class="ml10">model</span>
                </template>
                <template #default="scope">
                    <el-input v-model="scope.row['model']" placeholder="字段model" clearable> </el-input>
                </template>
            </el-table-column>

            <el-table-column prop="name" label="label" min-width="100px">
                <template #default="scope">
                    <el-input v-model="scope.row['name']" placeholder="字段title" clearable> </el-input>
                </template>
            </el-table-column>

            <el-table-column prop="placeholder" label="字段说明" min-width="140px">
                <template #default="scope">
                    <el-input v-model="scope.row['placeholder']" placeholder="字段说明" clearable> </el-input>
                </template>
            </el-table-column>

            <el-table-column prop="options" label="可选值" min-width="140px">
                <template #default="scope">
                    <el-input v-model="scope.row['options']" placeholder="可选值 ,分割" clearable> </el-input>
                </template>
            </el-table-column>

            <el-table-column prop="required" label="必填" min-width="40px">
                <template #default="scope">
                    <el-checkbox v-model="scope.row['required']" />
                </template>
            </el-table-column>

            <el-table-column label="操作" wdith="20px">
                <template #default="scope">
                    <el-button type="danger" @click="deleteItem(scope.$index)" icon="delete" plain></el-button>
                </template>
            </el-table-column>
        </el-table>
    </div>
</template>

<script lang="ts" setup>
const formItems: any = defineModel('modelValue');

const addItem = () => {
    formItems.value.push({});
};

const deleteItem = (index: any) => {
    formItems.value.splice(index, 1);
};
</script>
<style lang="scss"></style>
