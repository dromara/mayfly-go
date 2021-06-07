<template>
    <div v-if="getUserAuthBtnList">
        <slot />
    </div>
</template>

<script lang="ts">
import { computed } from 'vue';
import { useStore } from '/@/store/index.ts';
export default {
    name: 'auths',
    props: {
        value: {
            type: Array,
            default: () => [],
        },
    },
    setup(props) {
        const store = useStore();
        // 获取 vuex 中的用户权限
        const getUserAuthBtnList = computed(() => {
            let flag = false;
            store.state.userInfos.userInfos.authBtnList.map((val: any) => {
                props.value.map((v) => {
                    if (val === v) flag = true;
                });
            });
            return flag;
        });
        return {
            getUserAuthBtnList,
        };
    },
};
</script>
