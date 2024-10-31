<template>
    <div>
        <el-popover v-if="props.accountId" @show="getAccountInfo(props.accountId)" placement="top-start" title="账号信息" :width="400" trigger="click">
            <template #reference>
                <el-link type="primary">{{ props.username }}</el-link>
            </template>
            <el-descriptions v-loading="loading" :column="2" border>
                <el-descriptions-item label="username">{{ account.username }}</el-descriptions-item>
                <el-descriptions-item label="姓名">
                    {{ account.name }}
                </el-descriptions-item>

                <el-descriptions-item :span="2" label="角色">
                    <el-tag v-for="role in account.roles" :key="role.code" class="ml5">
                        {{ role.roleName }}
                    </el-tag>
                </el-descriptions-item>
            </el-descriptions>
        </el-popover>
        <span v-else>{{ props.username }}</span>
    </div>
</template>

<script lang="ts" setup>
import { reactive, toRefs } from 'vue';
import { accountApi } from '../../api';
const props = defineProps({
    accountId: {
        type: [Number],
    },
    username: {
        type: [String],
        required: true,
    },
});

const state = reactive({
    account: {} as any,
    loading: false,
});

const { account, loading } = toRefs(state);

const getAccountInfo = async (id: number) => {
    try {
        state.loading = true;
        state.account = await accountApi.getAccountDetail.request({ id });
    } finally {
        state.loading = false;
    }
};
</script>

<style lang="scss"></style>
