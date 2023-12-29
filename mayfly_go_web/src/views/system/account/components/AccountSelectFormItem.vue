<template>
    <el-form-item label="账号">
        <el-select
            style="width: 100%"
            remote
            :remote-method="getAccount"
            v-model="accountId"
            filterable
            placeholder="请输入账号模糊搜索并选择"
            v-bind="$attrs"
            :ref="(el: any) => props.focus && el?.focus()"
        >
            <el-option v-for="item in accounts" :key="item.id" :label="`${item.username} [${item.name}]`" :value="item.id"> </el-option>
        </el-select>
    </el-form-item>
</template>

<script lang="ts" setup>
import { ref } from 'vue';
import { accountApi } from '../../api';

const props = defineProps({
    // 是否获取焦点
    focus: {
        type: Boolean,
        default: false,
    },
});

const accountId = defineModel('modelValue');

const accounts: any = ref([]);

const getAccount = (username: any) => {
    if (username) {
        accountApi.list.request({ username }).then((res) => {
            accounts.value = res.list;
        });
    } else {
        accounts.value = [];
    }
};
</script>
