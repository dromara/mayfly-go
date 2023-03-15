<template>
    <div v-if="getUserAuthBtnList">
        <slot />
    </div>
</template>

<script lang="ts">
import { computed } from 'vue';
import { useUserInfo } from '@/store/userInfo';
export default {
    name: 'auth',
    props: {
        value: {
            type: String,
            default: () => '',
        },
    },
    setup(props) {
        // 获取 vuex 中的用户权限
        const getUserAuthBtnList = computed(() => {
            return useUserInfo().userInfo.authBtnList.some((v: any) => v === props.value);
        });
        return {
            getUserAuthBtnList,
        };
    },
};
</script>
