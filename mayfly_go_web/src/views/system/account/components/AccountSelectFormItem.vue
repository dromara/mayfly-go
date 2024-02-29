<template>
    <el-form-item :label="label">
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
import { onMounted, ref } from 'vue';
import { accountApi } from '../../api';

const props = defineProps({
    // 是否获取焦点
    focus: {
        type: Boolean,
        default: false,
    },
    label: {
        type: String,
        default: '账号',
    },
});

onMounted(() => {
    // 如果初始化时有accountId，则需要获取对应用户信息，用于回显用户名等信息
    if (accountId.value) {
        accountApi.querySimple.request({ ids: accountId.value }).then((res) => {
            accounts.value = res.list;
        });
    }
});

const accountId = defineModel('modelValue');

const accounts: any = ref([]);

const getAccount = (username: any) => {
    if (username) {
        accountApi.querySimple.request({ username }).then((res) => {
            accounts.value = res.list;
        });
    } else {
        accounts.value = [];
    }
};
</script>
