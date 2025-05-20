<template>
    <el-form-item :label="label">
        <el-select v-model="roleId" filterable v-bind="$attrs" :ref="(el: any) => props.focus && el?.focus()">
            <el-option v-for="item in roles" :key="item.id" :label="`${item.name} [${item.code}]`" :value="item.id"> </el-option>
        </el-select>
    </el-form-item>
</template>

<script lang="ts" setup>
import { onMounted, ref } from 'vue';
import { roleApi } from '../../api';

const props = defineProps({
    // 是否获取焦点
    focus: {
        type: Boolean,
        default: false,
    },
    label: {
        type: String,
        default: '角色',
    },
});

onMounted(() => {
    getRole();
});

const roleId = defineModel('modelValue');

const roles: any = ref([]);

const getRole = () => {
    roleApi.list.request().then((res) => {
        roles.value = res.list;
    });
};
</script>
