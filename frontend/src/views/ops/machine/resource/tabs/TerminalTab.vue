<template>
    <div class="machine-terminal-tab h-full flex flex-col">
        <!-- Terminal body -->
        <div class="terminal-body flex-1 min-h-0">
            <!-- SSH 用默认窗格内容；RDP 等非 SSH 协议通过作用域插槽接入多窗格容器 -->
            <TerminalPanes
                ref="terminalRef"
                :mount-init="false"
                @status-change="onStatusChange"
                :socket-url="socketUrl"
                :machine-id="machineId"
                :auth-cert-name="authCertName"
                :file-id="0"
                :protocol="protocol"
            >
                <template v-if="protocol != MachineProtocolEnum.Ssh.value" #pane="{ register, setStatus }">
                    <MachineRdp :ref="register" :machine-id="machineId" :auth-cert="authCertName" :protocol="protocol" class="h-full w-full" @status-change="(status: TerminalStatus) => setStatus(status)" />
                </template>
            </TerminalPanes>
        </div>
    </div>
</template>

<script lang="ts" setup>
import MachineRdp from '@/components/terminal-rdp/MachineRdp.vue';
import TerminalPanes from '@/components/terminal/TerminalPanes.vue';
import { TerminalStatus, TerminalStatusEnum } from '@/components/terminal/common';
import { getMachineTerminalSocketUrl } from '@/views/ops/machine/api';
import { MachineProtocolEnum } from '@/views/ops/machine/enums';
import { updateTabComponentProps } from '@/views/ops/resource/resourceOp';
import { computed, nextTick, onMounted, ref, watch } from 'vue';

const props = defineProps<{
    tabKey?: string;
    machineId: number;
    authCertName: string;
    protocol: number; // Machine protocol
}>();

const terminalRef = ref();
const status = ref<TerminalStatus>(TerminalStatusEnum.Disconnected.value);

// 映射终端状态到 tab 标签状态
const getTabStatus = (terminalStatus: number): string => {
    switch (terminalStatus) {
        case TerminalStatusEnum.Connected.value:
            return 'connected';
        case TerminalStatusEnum.NoConnected.value:
            return 'disconnected';
        case TerminalStatusEnum.Error.value:
            return 'error';
        case TerminalStatusEnum.Disconnected.value:
        default:
            return 'disconnected';
    }
};

// Watch status changes and update tab component props
watch(status, (newStatus: number) => {
    if (props.tabKey) {
        updateTabComponentProps(props.tabKey, {
            status: getTabStatus(newStatus),
        });
    }
});

// Compute socket URL
const socketUrl = computed(() => {
    return getMachineTerminalSocketUrl(props.authCertName);
});

onMounted(() => {
    // Auto-connect terminal on mount
    nextTick(() => {
        handleReconnect();
        setTimeout(() => fitTerminal(), 300);
    });
});

const onStatusChange = (statusValue: TerminalStatus) => {
    status.value = statusValue;
};

const handleReconnect = () => {
    terminalRef.value?.init?.();
};

const fitTerminal = () => {
    terminalRef.value?.fitTerminal?.();
};

const close = () => {
    terminalRef.value?.close?.();
};

const focus = () => {
    terminalRef.value?.focus?.();
};

const blur = () => {
    terminalRef.value?.blur?.();
};

defineExpose({
    onRefresh: handleReconnect,
    onActivate: focus,
    onResize: fitTerminal,
    close,
    fitTerminal,
    focus,
    blur,
});
</script>

<style lang="scss">
.machine-terminal-tab {
    height: 100%;
    display: flex;
    flex-direction: column;
}

.terminal-body {
    overflow: hidden;
}
</style>
