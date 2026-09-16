<template>
    <el-select v-model="localValue" multiple :placeholder="$t('alert.selectNotifyChannel')" style="width: 100%">
        <el-option v-for="ch in channels" :key="ch.id" :label="ch.name" :value="ch.id" />
    </el-select>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue';
import Api from '@/common/Api';
import { Msg } from '@/hooks/useI18n';

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

const channels = ref<{ id: number; name: string }[]>([]);

// 获取消息渠道列表（复用 msg 模块 API）
const msgChannelApi = {
    list: Api.newGet<{ id: number; name: string }[]>('/msg/channels/simple'),
};

onMounted(async () => {
    try {
        const res = await msgChannelApi.list.request();
        channels.value = res ?? [];
    } catch {
        channels.value = [];
        // 该接口需要 msg:channel:base 权限，无权限时下拉恒为空；
        // 静默失败会让用户以为"没有渠道"，必须显式提示
        Msg.warning('alert.channelLoadFailed');
    }
});
</script>
