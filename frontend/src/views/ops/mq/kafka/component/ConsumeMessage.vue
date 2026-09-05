<template>
    <div class="kafka-consume-message h-full card p-1! flex flex-col">
        <auto-form ref="consumeFormRef" v-model="form" :items="consumeItems" label-width="auto" size="small" class="flex-shrink-0">
            <template #topic>
                <el-select v-model="form.topic" filterable :placeholder="$t('mq.kafka.selectTopicPlaceholder')" clearable :teleported="false">
                    <el-option v-for="topic in topics" :key="topic" :label="topic" :value="topic" />
                </el-select>
            </template>
            <template #group>
                <el-select v-model="form.group" filterable :placeholder="$t('mq.kafka.consumerGroupPlaceholder')" clearable allow-create :teleported="false">
                    <el-option label="(auto generate)" value="" />
                    <el-option v-for="g in groups" :key="g.Group" :label="g.Group" :value="g.Group" />
                </el-select>
            </template>
        </auto-form>

        <div class="mb-2 flex flex-shrink-0 gap-2">
            <el-button @click="resetForm" icon="refresh">{{ $t('common.reset') }}</el-button>
            <el-button @click="consumeMessage" type="primary" icon="download" :loading="consuming">
                {{ $t('mq.kafka.consumeMessage') }}
            </el-button>
        </div>

        <el-table :data="messages" stripe v-loading="consuming">
            <el-table-column prop="offset" :label="$t('mq.kafka.offset')" min-width="100" />
            <el-table-column prop="partition" :label="$t('mq.kafka.partition')" min-width="80" />
            <el-table-column prop="key" :label="$t('mq.kafka.key')" min-width="150" />
            <el-table-column prop="timestamp" :label="$t('mq.kafka.timestamp')" min-width="180" />
            <el-table-column prop="value" :label="$t('mq.kafka.messageBody')" min-width="300">
                <template #default="{ row }">
                    <div class="flex items-center">
                        <el-input v-model="row.displayValue" type="textarea" :rows="1" size="small" class="flex-1" />
                        <SvgIcon
                            v-if="row.value && row.value.length > 50"
                            @click="viewMessageDetail(row)"
                            class="string-input-container-icon ml-1 cursor-pointer"
                            name="FullScreen"
                            :size="10"
                        />
                    </div>
                </template>
            </el-table-column>
            <el-table-column prop="headers" :label="$t('mq.kafka.headers')" min-width="150">
                <template #default="{ row }">
                    {{ JSON.stringify(row.headers) }}
                </template>
            </el-table-column>
        </el-table>
    </div>
</template>

<script lang="ts" setup>
import { computed, ref, reactive, toRefs, onMounted, defineAsyncComponent, watch } from 'vue';
import type { AutoFormItem } from '@/components/auto-form';
import { mqApi } from '../../api';
import { ElMessage } from 'element-plus';
import { useI18n } from 'vue-i18n';
import SvgIcon from '@/components/svg-icon/index.vue';
import MonacoEditorBox from '@/components/monaco/MonacoEditorBox';
import type { KafkaGroup } from '../../types';
import { randomUuid } from '@/common/utils/string';
import { Msg } from '@/hooks/useI18n';

const { t } = useI18n();

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
    groups: {
        type: Array as () => KafkaGroup[],
        default: () => [],
    },
});

const consumeFormRef = ref();
const consuming = ref(false);

const state = reactive({
    form: {
        topic: '',
        number: 10,
        group: '',
        pullTimeout: 10,
        decompression: '',
        decode: '',
        isolationLevel: 'read_uncommitted',
        commitOffset: false,
        earliest: true,
        startTime: '',
    },
    messages: [] as (import('@/views/ops/mq/types').KafkaMessage & { displayValue?: string })[],
});

const { form, messages } = toRefs(state);

/** 消费消息表单声明（topic/group 为 custom 插槽；原表单未调用 validate，保持仅星号提示） */
const consumeItems = computed<AutoFormItem[]>(() => [
    { prop: 'topic', label: 'mq.kafka.selectTopic', type: 'custom', required: true, span: 10 },
    { prop: 'number', label: 'mq.kafka.messageNumber', type: 'number', min: 1, max: 1000, required: true, span: 5 },
    { prop: 'group', label: 'mq.kafka.consumerGroup', type: 'custom', span: 9 },
    { prop: 'pullTimeout', label: 'mq.kafka.pullTimeout', type: 'number', min: 1, max: 100, span: 5 },
    { prop: 'decompression', label: 'mq.kafka.decompression', type: 'select', span: 5, placeholder: 'mq.kafka.decompressionPlaceholder', props: { clearable: true, teleported: false }, options: [{ label: 'none', value: '' }, { label: 'gzip', value: 'gzip' }, { label: 'lz4', value: 'lz4' }, { label: 'zstd', value: 'zstd' }, { label: 'snappy', value: 'snappy' }] },
    { prop: 'decode', label: 'mq.kafka.decode', type: 'select', span: 5, placeholder: 'mq.kafka.decodePlaceholder', props: { clearable: true, teleported: false }, options: [{ label: 'None', value: '' }, { label: 'Base64', value: 'base64' }] },
    { prop: 'isolationLevel', label: 'mq.kafka.isolationLevel', type: 'select', span: 5, placeholder: 'mq.kafka.isolationLevelPlaceholder', props: { teleported: false }, options: [{ label: 'mq.kafka.readUncommitted', value: 'read_uncommitted' }, { label: 'mq.kafka.readCommitted', value: 'read_committed' }] },
    { prop: 'commitOffset', label: 'mq.kafka.commitOffset', type: 'switch', span: 5 },
    { prop: 'earliest', label: 'mq.kafka.defaultConsumePosition', type: 'switch', span: 7, tooltip: 'mq.kafka.consumerOnlyTip', props: { activeText: t('mq.kafka.earliest'), inactiveText: t('mq.kafka.latest') } },
    { prop: 'startTime', label: 'mq.kafka.defaultConsumeStartTime', type: 'date', span: 8, tooltip: 'mq.kafka.consumerOnlyTip', placeholder: 'mq.kafka.selectDateTime', props: { type: 'datetime', valueFormat: 'YYYY-MM-DD HH:mm:ss', size: 'small' } },
]);

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

const resetForm = () => {
    state.form = {
        topic: props.defaultTopic || '',
        number: 10,
        group: '',
        pullTimeout: 10,
        decompression: '',
        decode: '',
        isolationLevel: 'read_uncommitted',
        commitOffset: false,
        earliest: true,
        startTime: '',
    };
    state.messages = [];
};

const consumeMessage = async () => {
    if (!consumeFormRef.value) return;
    consuming.value = true;
    if (!state.form.group) {
        state.form.group = '__mayfly-server__' + randomUuid();
    }

    try {
        const param = {
            id: props.kafkaId,
            ...state.form,
        };

        const res = await mqApi.kafkaTopicConsume.request(param);
        state.messages = (res || []).map((msg: import('@/views/ops/mq/types').KafkaMessage, index: number) => ({
            ...msg,
            displayValue: typeof msg.value === 'object' ? JSON.stringify(msg.value, null, 2) : String(msg.value),
        }));
    } catch (error: unknown) {
        Msg.error((error instanceof Error ? error.message : String(error)) || 'common.requestFail');
    } finally {
        consuming.value = false;
    }
};

const viewMessageDetail = (row: import('@/views/ops/mq/types').KafkaMessage) => {
    const value = typeof row.value === 'object' ? JSON.stringify(row.value, null, 2) : String(row.value);
    const editorLang = getEditorLangByValue(value);

    MonacoEditorBox({
        content: value,
        title: `${t('mq.kafka.messageBody')} - Offset ${row.offset}`,
        language: editorLang,
        showConfirmButton: false,
        closeFn: () => {},
    });
};

const getEditorLangByValue = (value: string) => {
    try {
        if (typeof JSON.parse(value) === 'object') {
            return 'json';
        }
    } catch (e) {
        /* empty */
    }

    try {
        const doc = new DOMParser().parseFromString(value, 'text/html');
        if (Array.from(doc.body.childNodes).some((node) => node.nodeType === 1)) {
            return 'html';
        }
    } catch (e) {
        /* empty */
    }

    return 'text';
};
</script>

<style lang="scss" scoped>
.kafka-consume-message {
    overflow: hidden;

    > .el-table {
        flex: 1;
        min-height: 0;
    }

    .string-input-container-icon {
        color: var(--el-color-primary);
        &:hover {
            color: var(--el-color-success);
        }
    }
}
</style>
