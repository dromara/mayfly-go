<template>
    <div v-if="getUserAuthBtnList">
        <slot />
    </div>
</template>

<script lang="ts">
import { computed } from 'vue';
import { useUserInfo } from '@/store/userInfo';
import { judementSameArr } from '/@/utils/arrayOperation.ts';
export default {
    name: 'authAll',
    props: {
        value: {
            type: Array,
            default: () => [],
        },
    },
    setup(props) {
        // 获取 vuex 中的用户权限
        const getUserAuthBtnList = computed(() => {
            return judementSameArr(props.value, useUserInfo().userInfo.authBtnList);
        });
        return {
            getUserAuthBtnList,
        };
    },
};
</script>
