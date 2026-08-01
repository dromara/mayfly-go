<template>
    <div style="width: 100%">
        <el-select
            @focus="getSshTunnelMachines"
            @change="change"
            style="width: 100%"
            v-model="sshTunnelMachineId"
            @clear="clear"
            placeholder="SSH tunnel machine"
            clearable
            filterable
        >
            <el-option v-for="item in sshTunnelMachineList" :key="item.id" :label="`${item.ip}:${item.port} [${item.name}]`" :value="item.id"> </el-option>
        </el-select>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, onMounted } from 'vue';
import { machineApi } from '../machine/api';
import { MachineProtocolEnum } from '../machine/enums';
import type { MachineVO } from '../machine/types';

const modelValue = defineModel<number | null>();

const state = reactive({
    // 单选则为id，多选为id数组
    sshTunnelMachineId: null as number | null,
    sshTunnelMachineList: [] as MachineVO[],
});

const { sshTunnelMachineId, sshTunnelMachineList } = toRefs(state);

onMounted(async () => {
    if (!modelValue.value || modelValue.value <= 0) {
        state.sshTunnelMachineId = null;
    } else {
        state.sshTunnelMachineId = modelValue.value;
    }
    await getSshTunnelMachines();
});

const getSshTunnelMachines = async () => {
    if (state.sshTunnelMachineList.length == 0) {
        const res = await machineApi.list.request({ pageNum: 1, pageSize: 100, protocol: MachineProtocolEnum.Ssh.value });
        state.sshTunnelMachineList = res.list;
    }
};

const clear = () => {
    state.sshTunnelMachineId = null;
    change();
};

const change = () => {
    modelValue.value = state.sshTunnelMachineId;
};
</script>
<style lang="scss"></style>
