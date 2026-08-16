<template>
    <div class="kafka-produce-message h-full card p-1!">
        <el-form ref="produceFormRef" :model="form" label-width="auto" size="small">
            <el-row :gutter="10">
                <el-col :span="8">
                    <el-form-item :label="$t('mq.kafka.selectTopic')" required>
                        <template #label>
                            <el-space>
                                <span>{{ $t('mq.kafka.selectTopic') }}</span>
                                <el-button icon="refresh" link />
                            </el-space>
                        </template>

                        <el-select v-model="form.topic" filterable :placeholder="$t('mq.kafka.selectTopicPlaceholder')">
                            <el-option v-for="topic in topics" :key="topic" :label="topic" :value="topic" />
                        </el-select>
                    </el-form-item>
                </el-col>
                <el-col :span="8">
                    <el-form-item :label="$t('mq.kafka.messageKey')">
                        <el-input v-model="form.key" :placeholder="$t('mq.kafka.messageKeyPlaceholder')" />
                    </el-form-item>
                </el-col>
                <el-col :span="6">
                    <el-form-item :label="$t('mq.kafka.partition')">
                        <el-tooltip :content="$t('mq.kafka.partitionPlaceholder')">
                            <el-input-number v-model="form.partition" :min="0" :max="100" />
                        </el-tooltip>
                    </el-form-item>
                </el-col>
            </el-row>

            <el-form-item :label="$t('mq.kafka.messageBody')" required>
                <monaco-editor v-model="form.value" language="json" height="200px" :can-change-mode="true" />
            </el-form-item>

            <el-form-item :label="$t('mq.kafka.messageHeaders')">
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
            </el-form-item>

            <el-row :gutter="10">
                <el-col :span="6">
                    <el-form-item :label="$t('mq.kafka.sendTimes')">
                        <el-input-number v-model="form.times" :min="1" :max="100" />
                    </el-form-item>
                </el-col>
                <el-col :span="6">
                    <el-form-item :label="$t('mq.kafka.compression')">
                        <el-select v-model="form.compression" :placeholder="$t('mq.kafka.compressionPlaceholder')" :teleported="false">
                            <el-option label="none" value="" />
                            <el-option label="gzip" value="gzip" />
                            <el-option label="lz4" value="lz4" />
                            <el-option label="zstd" value="zstd" />
                            <el-option label="snappy" value="snappy" />
                        </el-select>
                    </el-form-item>
                </el-col>
            </el-row>

            <el-form-item>
                <el-button @click="resetForm" icon="refresh">{{ $t('common.reset') }}</el-button>
                <el-button @click="sendMessage" type="primary" icon="upload" :loading="sending">
                    {{ $t('mq.kafka.sendMessage') }}
                </el-button>
            </el-form-item>
        </el-form>
    </div>
</template>

<script lang="ts" setup>
import { Msg } from '@/hooks/useI18n';
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
