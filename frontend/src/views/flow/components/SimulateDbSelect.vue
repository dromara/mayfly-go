<template>
    <db-select-tree v-model:db-id="dbId" @select-db="onSelectDb" />
</template>

<script lang="ts" setup>
import DbSelectTree from '@/views/ops/db/widgets/DbSelectTree.vue';
import type { DbNodeParams } from '@/views/ops/db/types';

/**
 * 试算面板的「目标库」控件：包一层库选择树。数字 id 与试算 raw 表字符串的
 * 换算由 defineModel 的 get/set 就地完成（raw 表各值都是字符串，数字组件会打不开接口）；
 * 选中库名经 pick 事件上抛，宿主按场景声明的 nameKey 回填对应输入项，
 * 库名输入框仍可手改——改的只是连接库名，选中的库资产不变；
 * 回显文本由选择树内部未绑定的 db-name 本地态自维护，包装层无需中转
 */
const emit = defineEmits<{ pick: [dbName: string] }>();

const dbId = defineModel<string, 'id', number | undefined, number | undefined>('id', {
    get: (value) => {
        const parsed = Number(value);
        return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined;
    },
    set: (value) => (value ? String(value) : ''),
});

const onSelectDb = (params: DbNodeParams) => emit('pick', params.db);
</script>
