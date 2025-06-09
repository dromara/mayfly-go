<template>
    <div>
        <el-popover
            v-if="props.username && props.username != '-'"
            @show="getAccountInfo(props.username)"
            placement="top-start"
            :title="$t('system.account.accountInfo')"
            :width="400"
            trigger="click"
        >
            <template #reference>
                <el-link type="primary">{{ props.username }}</el-link>
            </template>
            <el-descriptions v-loading="loading" :column="2" border>
                <el-descriptions-item label="username">{{ account.username }}</el-descriptions-item>
                <el-descriptions-item :label="$t('system.account.name')">
                    {{ account.name }}
                </el-descriptions-item>

                <el-descriptions-item :span="2" :label="$t('common.role')">
                    <el-tag v-for="role in account.roles" :key="role.code" class="ml-1">
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

const getAccountInfo = async (username: string) => {
    try {
        state.loading = true;
        state.account = await accountApi.getAccountDetail.request({ username });
    } finally {
        state.loading = false;
    }
};
</script>

<style lang="scss"></style>
