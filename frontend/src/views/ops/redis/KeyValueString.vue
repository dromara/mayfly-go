<template>
    <div class="flex flex-col h-full">
        <format-viewer ref="formatViewerRef" height="280px" :content="string.value"></format-viewer>
        <div class="mt-1 flex justify-end">
            <el-button @click="saveValue" type="primary" v-auth="'redis:data:save'">{{ $t('common.save') }}</el-button>
        </div>
    </div>
</template>
<script lang="ts" setup>
import { notEmptyI18n } from '@/common/assert';
import { Msg } from '@/hooks/useI18n';
import { onMounted, reactive, toRefs, useTemplateRef, watch } from 'vue';
import { redisApi } from './api';
import FormatViewer from './FormatViewer.vue';
import { RedisInst } from './redis';

const props = defineProps({
    redis: {
        type: RedisInst,
        required: true,
    },
    keyInfo: {
        type: [Object],
    },
});

const formatViewerRef = useTemplateRef<{ getContent: () => string }>('formatViewerRef');

const state = reactive({
    key: '',
    keyInfo: {
        key: '',
        type: 'string',
        timed: -1,
    },
    string: {
        type: 'text',
        value: '',
    },
});

const { string } = toRefs(state);

onMounted(() => {
    setProps(props);
});

watch(props, (newVal) => {
    setProps(newVal);
});

const setProps = (val: { keyInfo?: { key?: string } }) => {
    state.key = val.keyInfo?.key ?? '';
    initData();
};

const initData = () => {
    getStringValue();
};

const getStringValue = async () => {
    if (state.key) {
        state.string.value = (await props.redis.runCmd<string | null>(['GET', state.key])) ?? '';
    }
};

const saveValue = async () => {
    state.string.value = formatViewerRef.value?.getContent() ?? '';
    notEmptyI18n(state.string.value, 'value');

    const ttl = await redisApi.keyTtl.request({
        id: props.redis.id,
        db: props.redis.db,
        key: state.key,
    });

    let tArr = [];
    if (ttl > 0) {
        tArr.push('EX', ttl);
    }

    await props.redis.runCmd(['SET', state.key, state.string.value, ...tArr]);
    Msg.saveSuccess();
};

defineExpose({ initData });
</script>
<style lang="scss"></style>
