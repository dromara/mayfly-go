<template>
    <el-select
        v-model="localValue"
        multiple
        filterable
        remote
        :reserve-keyword="false"
        :remote-method="searchAccounts"
        :loading="loading"
        :placeholder="$t('alert.notifyReceiverPlaceholder')"
        style="width: 100%"
    >
        <el-option v-for="item in options" :key="item.id" :label="`${item.username} [${item.name}]`" :value="item.id" />
    </el-select>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue';
import { accountApi } from '@/views/system/api';
import type { Account } from '@/views/system/types';

const props = defineProps<{
    modelValue: number[];
}>();

const emit = defineEmits<{
    (e: 'update:modelValue', value: number[]): void;
}>();

const localValue = computed({
    get: () => props.modelValue || [],
    set: (val) => emit('update:modelValue', val),
});

const options = ref<Account[]>([]);
const loading = ref(false);

const queryAccounts = async (params: Record<string, unknown>) => {
    loading.value = true;
    try {
        const res = await accountApi.querySimple.request(params);
        options.value = res?.list ?? [];
    } finally {
        loading.value = false;
    }
};

const searchAccounts = (username: string) => {
    if (!username) {
        queryAccounts({ pageSize: 20 });
        return;
    }
    queryAccounts({ username, pageSize: 20 });
};

// 回显已选接收人：远程搜索模式下 options 初始为空，未选中项会只显示 ID，需按 id 拉取一次
watch(
    () => props.modelValue,
    (ids) => {
        const pending = (ids || []).filter((id) => !options.value.some((item) => item.id === id));
        if (!pending.length) {
            return;
        }
        queryAccounts({ ids: pending.join(','), pageSize: pending.length });
    },
    { immediate: true }
);
</script>
