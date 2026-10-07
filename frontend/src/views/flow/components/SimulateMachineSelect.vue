<template>
    <machine-select-tree v-model:machine-id="machineId" />
</template>

<script lang="ts" setup>
import MachineSelectTree from '@/views/ops/machine/component/MachineSelectTree.vue';

/**
 * 试算面板的「目标机器」控件：机器选择树选中后只取 id（命令黑名单按该机器
 * 的标签定位），数字 id 与试算 raw 表字符串的换算由 defineModel 的 get/set
 * 就地完成；回显文本由选择树内部未绑定的 machine-name 本地态自维护，
 * 不参与试算求值
 */
const machineId = defineModel<string, 'id', number | undefined, number | undefined>('id', {
    get: (value) => {
        const parsed = Number(value);
        return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined;
    },
    set: (value) => (value ? String(value) : ''),
});
</script>
