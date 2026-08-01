<template>
    <ResourceSelect
        v-bind="$attrs"
        v-model="selectNode"
        @change="changeNode"
        :resource-type="ResourceTypeEnum.Machine.value"
        :leaf-node-types="[NodeTypeAuthCert]"
    >
        <template #iconPrefix>
            <SvgIcon name="Monitor" :size="16" />
            <TagCodePath v-if="authCertName" :code="authCertName" />
        </template>
        <!-- <template #label>
            <TagCodePath v-if="authCertName" :code="authCertName" />
        </template> -->
    </ResourceSelect>
</template>

<script setup lang="ts">
import { ResourceTypeEnum } from '@/common/commonEnum';
import { TagTreeNode } from '@/views/ops/component/tag';
import TagCodePath from '@/views/ops/component/TagCodePath.vue';
import { NodeTypeAuthCert } from '@/views/ops/machine/resource';
import type { MachineNodeParams } from '@/views/ops/machine/resource';
import ResourceSelect from '@/views/ops/resource/ResourceSelect.vue';
import { watch } from 'vue';

const authCertName = defineModel<string>('authCertName');
const machineId = defineModel<number>('machineId');
const machineName = defineModel<string>('machineName');
const machineIp = defineModel<string>('machineIp');
const machinePort = defineModel<number>('machinePort');
const username = defineModel<string>('username');
const tagPath = defineModel<string>('tagPath');

const emits = defineEmits(['selectMachine']);

const selectNode = defineModel<string>('modelValue', {
    default: '',
});

// 监听内部字段变化，自动更新 selectNode
watch(
    [authCertName, tagPath, machineName, username],
    () => {
        selectNode.value = machineName.value || '';
    },
    { immediate: true }
);

const changeNode = (node: TagTreeNode) => {
    const params = node.params as MachineNodeParams;

    const selectAuthCert = params.selectAuthCert;
    authCertName.value = selectAuthCert?.name || '';
    machineId.value = params.id;
    machineName.value = params.name;
    machineIp.value = params.ip;
    machinePort.value = params.port;
    username.value = selectAuthCert?.username || params.username;
    tagPath.value = params.tagPath || '';

    emits('selectMachine', params);
};
</script>
