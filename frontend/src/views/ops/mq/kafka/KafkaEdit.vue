<template>
    <div>
        <auto-form-drawer ref="drawerRef" v-model:visible="dialogVisible" :title="title" :items="items" :data="editData" size="40%" :confirm-api="onConfirm" @submitted="emit('cancel')" @cancel="emit('cancel')">
            <!-- 关联标签 -->
            <template #tagCodePaths="{ form }">
                <TagTreeSelect multiple :code="form.code" v-model="form.tagCodePaths" />
            </template>

            <!-- SSH 隧道 -->
            <template #sshTunnelMachineId="{ form }">
                <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
            </template>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="onTestConn" :loading="testConnBtnLoading" type="success">{{ $t('ac.testConn') }}</el-button>
                    <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="drawerRef?.submitting" @click="drawerRef?.submit()">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { mqApi } from '@/views/ops/mq/api';
import { computed, ref, useTemplateRef, type PropType } from 'vue';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import SshTunnelSelect from '../../component/SshTunnelSelect.vue';
import TagTreeSelect from '../../component/TagTreeSelect.vue';
import type { Kafka } from '@/views/ops/mq/types';

/** Kafka 编辑表单类型 */
interface KafkaForm extends Omit<Partial<Kafka>, 'id' | 'name' | 'sshTunnelMachineId'> {
    id?: number | null;
    name?: string | null;
    sshTunnelMachineId?: number | null;
    tagCodePaths?: string[];
}

const props = defineProps({
    kafka: {
        type: Object as PropType<Kafka | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const sasl_mechanism_options = [
    {
        label: 'PLAIN',
        value: 'PLAIN',
    },
    {
        label: 'SCRAM-SHA-256',
        value: 'SCRAM-SHA-256',
    },
    {
        label: 'SCRAM-SHA-512',
        value: 'SCRAM-SHA-512',
    },
];

const dialogVisible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；tagCodePaths/sshTunnel 走插槽） */
const items: AutoFormItem[] = [
    { prop: 'tagCodePaths', label: 'tag.relateTag', required: true },
    { prop: 'name', label: 'common.name', required: true },
    { prop: 'hosts', label: 'Hosts', type: 'textarea', rows: 2, required: true, placeholder: 'Kafka 连接地址，格式: host1:port1,host2:port2 或单个 broker' },
    { prop: 'saslMechanism', label: 'mq.kafka.sasl_mechanism', type: 'select', options: sasl_mechanism_options, placeholder: 'mq.kafka.sasl_mechanism_placeholder' },
    { prop: 'username', label: 'mq.kafka.username' },
    { prop: 'password', label: 'common.password', type: 'password' },
    { prop: 'sshTunnelMachineId', label: 'machine.sshTunnel' },
];

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => Promise<unknown>; submitting: boolean; submit: () => Promise<void> }>('drawerRef');

/** 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成） */
const editData = computed<AutoFormData>(() => {
    const kafka = props.kafka as KafkaForm | false | undefined;
    if (kafka) {
        return { ...kafka } as AutoFormData;
    }
    return { saslMechanism: 'PLAIN', tagCodePaths: [] } as AutoFormData;
});

/** 抽屉打开后暂存的内部表单引用（提交组装基于它） */
const internalForm = ref<AutoFormData>({});

const { isFetching: testConnBtnLoading, execute: testConnExec } = mqApi.KafkaTestConn.useApi();
const { execute: saveKafkaExec } = mqApi.kafkaSave.useApi();

const getReqForm = () => {
    const reqForm = { ...internalForm.value } as KafkaForm;
    if (!reqForm.sshTunnelMachineId || reqForm.sshTunnelMachineId <= 0) {
        reqForm.sshTunnelMachineId = -1;
    }
    return reqForm;
};

const onTestConn = async () => {
    // 校验失败内部已 toast（catch 吞掉 reject）
    const valid = await useI18nFormValidate(drawerRef).catch(() => false);
    if (valid === false) return;
    await testConnExec(getReqForm());
    Msg.success('ac.connSuccess');
};

// confirmApi 提交动作（组装内部表单为请求参数）；成功提示与关闭抽屉由组件内置逻辑处理
const onConfirm = async () => {
    await saveKafkaExec(getReqForm());
    emit('val-change', internalForm.value);
};
</script>
<style lang="scss"></style>
