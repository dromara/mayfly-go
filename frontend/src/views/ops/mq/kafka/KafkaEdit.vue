<template>
    <div>
        <el-drawer :append-to-body="false" :title="title" v-model="dialogVisible" :before-close="onCancel" :close-on-click-modal="false" size="40%" :destroy-on-close="true">
            <el-form :model="form" ref="kafkaFormRef" :rules="rules" label-width="auto">
                <el-form-item prop="tagCodePaths" :label="$t('tag.relateTag')" required>
                    <TagTreeSelect multiple :code="form.code" v-model="form.tagCodePaths" />
                </el-form-item>

                <el-form-item prop="name" :label="$t('common.name')" required>
                    <el-input v-model.trim="form.name" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="hosts" label="Hosts" required>
                    <el-input
                        type="textarea"
                        :rows="2"
                        v-model.trim="form.hosts"
                        placeholder="Kafka 连接地址，格式: host1:port1,host2:port2 或单个 broker"
                        auto-complete="off"
                    />
                </el-form-item>
                <el-form-item prop="saslMechanism" :label="$t('mq.kafka.sasl_mechanism')">
                    <el-select
                        v-model="form.saslMechanism"
                        :options="sasl_mechanism_options"
                        :placeholder="t('mq.kafka.sasl_mechanism_placeholder')"
                        filterable
                        clearable
                    />
                </el-form-item>
                <el-form-item prop="username" :label="$t('mq.kafka.username')">
                    <el-input v-model.trim="form.username" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="password" :label="$t('common.password')">
                    <el-input v-model.trim="form.password" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="sshTunnelMachineId" :label="$t('machine.sshTunnel')">
                    <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="onTestConn" :loading="testConnBtnLoading" type="success">{{ $t('ac.testConn') }}</el-button>
                    <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { Rules } from '@/common/rule';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { mqApi } from '@/views/ops/mq/api';
import { reactive, toRefs, useTemplateRef, watchEffect, type PropType } from 'vue';
import type { FormInstance } from 'element-plus';
import { useI18n } from 'vue-i18n';
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

const { t } = useI18n();

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

const rules = {
    tagCodePaths: [Rules.requiredSelect('tag.relateTag')],
    name: [Rules.requiredInput('common.name')],
    uri: [Rules.requiredInput('kafka.connUrl')],
};

const kafkaFormRef = useTemplateRef<FormInstance>('kafkaFormRef');

const state = reactive({
    form: {
        id: null,
        code: '',
        name: null,
        hosts: '',
        username: '',
        saslMechanism: 'PLAIN',
        password: '',
        sshTunnelMachineId: null as number | null,
        tagCodePaths: [],
    } as KafkaForm,
});

const { form } = toRefs(state);

const { isFetching: testConnBtnLoading, execute: testConnExec } = mqApi.KafkaTestConn.useApi();
const { isFetching: saveBtnLoading, execute: saveKafkaExec } = mqApi.kafkaSave.useApi();

watchEffect(() => {
    if (!dialogVisible.value) {
        return;
    }
    const kafka = props.kafka as import('@/views/ops/mq/types').Kafka | false | undefined;
    if (kafka) {
        state.form = { ...kafka } as KafkaForm;
    } else {
        state.form = { saslMechanism: 'PLAIN', tagCodePaths: [] } as KafkaForm;
    }
});

const getReqForm = () => {
    const reqForm = { ...state.form };
    if (!state.form.sshTunnelMachineId || state.form.sshTunnelMachineId <= 0) {
        reqForm.sshTunnelMachineId = -1;
    }
    return reqForm;
};

const onTestConn = async () => {
    await useI18nFormValidate(kafkaFormRef);
    await testConnExec(getReqForm());
    Msg.success('ac.connSuccess');
};

const onConfirm = async () => {
    await useI18nFormValidate(kafkaFormRef);
    await saveKafkaExec(getReqForm());
    Msg.saveSuccess();
    emit('val-change', state.form);
    onCancel();
};

const onCancel = () => {
    dialogVisible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
