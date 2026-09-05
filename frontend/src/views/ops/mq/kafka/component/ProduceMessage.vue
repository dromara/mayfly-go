<template>
    <div class="kafka-produce-message h-full card p-1!">
        <auto-form ref="produceFormRef" v-model="form" :items="produceItems" label-width="auto" size="small">
            <template #topic>
                <el-select v-model="form.topic" filterable :placeholder="$t('mq.kafka.selectTopicPlaceholder')">
                    <el-option v-for="topic in topics" :key="topic" :label="topic" :value="topic" />
                </el-select>
            </template>
            <template #value>
                <monaco-editor v-model="form.value" language="json" height="200px" :can-change-mode="true" />
            </template>
            <template #headers>
                <div class="w-full">
                    <el-button @click="addHeader" type="primary" size="small" icon="plus">
                        {{ $t('mq.kafka.addHeader') }}
                    </el-button>
                    <div class="mt-2" v-if="form.headers && form.headers.length > 0">
                        <div v-for="(header, index) in form.headers" :key="index" class="flex items-center mb-2">
                            <el-input v-model="header.key" :placeholder="$t('mq.kafka.headerKey')" size="small" class="w-60 mr-2" />
                            <el-input v-model="header.value" :placeholder="$t('mq.kafka.headerValue')" size="small" class="w-80 mr-2" />
                            <el-button @click="removeHeader(index)" type="danger" size="small" icon="delete" />
                        </div>
                    </div>
                </div>
            </template>
        </auto-form>

        <div class="mt-2 flex gap-2">
            <el-button @click="resetForm" icon="refresh">{{ $t('common.reset') }}</el-button>
            <el-button @click="sendMessage" type="primary" icon="upload" :loading="sending">
                {{ $t('mq.kafka.sendMessage') }}
            </el-button>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { Msg } from '@/hooks/useI18n';
import type { AutoFormItem } from '@/components/auto-form';
import { defineAsyncComponent, onMounted, reactive, ref, toRefs, watch } from 'vue';
import { mqApi } from '../../api';

const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

interface Header {
    key: string;
    value: string;
}

const props = defineProps({
    kafkaId: {
        type: Number,
        required: true,
    },
    defaultTopic: {
        type: String,
        default: '',
    },
    topics: {
        type: Array as () => string[],
        default: () => [],
    },
});

const produceFormRef = ref();
const sending = ref(false);

const state = reactive({
    form: {
        topic: '',
        key: '',
        value: '',
        partition: 0,
        headers: [] as Header[],
        times: 1,
        compression: '',
    },
});

const { form } = toRefs(state);

/** 发送消息表单声明（topic/messageBody/headers 为 custom 插槽） */
const produceItems: AutoFormItem[] = [
    { prop: 'topic', label: 'mq.kafka.selectTopic', type: 'custom', required: true, span: 8 },
    { prop: 'key', label: 'mq.kafka.messageKey', placeholder: 'mq.kafka.messageKeyPlaceholder', span: 8 },
    { prop: 'partition', label: 'mq.kafka.partition', type: 'number', min: 0, max: 100, tooltip: 'mq.kafka.partitionPlaceholder', span: 8 },
    { prop: 'value', label: 'mq.kafka.messageBody', type: 'custom', required: true },
    { prop: 'headers', label: 'mq.kafka.messageHeaders', type: 'custom' },
    { prop: 'times', label: 'mq.kafka.sendTimes', type: 'number', min: 1, max: 100, span: 6 },
    { prop: 'compression', label: 'mq.kafka.compression', type: 'select', span: 6, placeholder: 'mq.kafka.compressionPlaceholder', props: { teleported: false }, options: [{ label: 'none', value: '' }, { label: 'gzip', value: 'gzip' }, { label: 'lz4', value: 'lz4' }, { label: 'zstd', value: 'zstd' }, { label: 'snappy', value: 'snappy' }] },
];

onMounted(() => {
    if (props.defaultTopic) {
        state.form.topic = props.defaultTopic;
    }
});

watch(
    () => props.defaultTopic,
    (newTopic) => {
        state.form.topic = newTopic || '';
    }
);

const addHeader = () => {
    if (!state.form.headers) {
        state.form.headers = [];
    }
    state.form.headers.push({ key: '', value: '' });
};

const removeHeader = (index: number) => {
    state.form.headers.splice(index, 1);
};

const resetForm = () => {
    state.form = {
        topic: props.defaultTopic || '',
        key: '',
        value: '',
        partition: -1,
        headers: [] as Header[],
        times: 1,
        compression: '',
    };
};

const sendMessage = async () => {
    if (!produceFormRef.value) return;
    await produceFormRef.value?.validate();

    sending.value = true;
    try {
        const param = {
            id: props.kafkaId,
            topic: state.form.topic,
            key: state.form.key,
            value: state.form.value,
            partition: state.form.partition,
            headers: state.form.headers.filter((h: Header) => h.key || h.value),
            times: state.form.times,
            compression: state.form.compression,
        };

        await mqApi.kafkaTopicProduce.request(param);
        Msg.operateSuccess();
    } catch (error: unknown) {
        Msg.error(error instanceof Error ? error.message || 'common.requestFail' : 'common.requestFail');
    } finally {
        sending.value = false;
    }
};
</script>

<style lang="scss" scoped>
.kafka-produce-message {
    overflow: auto;
}
</style>
