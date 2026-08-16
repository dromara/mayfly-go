<template>
    <div v-if="hasAuthAll">
        <slot />
    </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import { useUserInfo } from '@/store/userInfo';
import { judementSameArr } from '@/common/utils/arrayOperation';

const props = defineProps<{
    value: string[];
}>();

// 获取 Pinia 中的用户权限
const hasAuthAll = computed(() => {
    return judementSameArr(props.value, useUserInfo().userInfo.authBtnList);
});
</script>
