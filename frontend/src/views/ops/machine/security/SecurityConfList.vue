<template>
    <div class="card h-full">
        <el-tabs v-model="activeName" @tab-change="handleTabChange">
            <el-tab-pane :label="$t('machine.cmdConfig')" :name="CmdConfTab">
                <CmdConfList />
            </el-tab-pane>

            <el-tab-pane :label="$t('machine.hostKeys')" :name="HostKeyTab">
                <HostKeyList />
            </el-tab-pane>
        </el-tabs>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, onMounted, defineAsyncComponent } from 'vue';
import { useRoute } from 'vue-router';

const CmdConfList = defineAsyncComponent(() => import('./CmdConfList.vue'));
const HostKeyList = defineAsyncComponent(() => import('./HostKeyList.vue'));

const CmdConfTab = 'cmdConf';
const HostKeyTab = 'hostKey';

const route = useRoute();

const state = reactive({
    activeName: CmdConfTab,
    cmdConfs: [],
});

const { activeName } = toRefs(state);

onMounted(async () => {
    // 支持从其他页面通过 ?tab=hostKey 直达指定标签页（如机器列表入口）
    const tab = route.query.tab;
    if (tab === HostKeyTab) {
        state.activeName = HostKeyTab;
    } else {
        state.activeName = CmdConfTab;
    }
});

const handleTabChange = (_tabName: string) => {
    // tab changed
};
</script>

<style scoped>
/* DataTable 依赖父级固定高度撑出 flex-1 区（裸 el-table 是自然高度不需要），故让 el-tabs 把 card 高度一路传到 tab-pane */
:deep(.el-tabs) {
    height: 100%;
    display: flex;
    flex-direction: column;
}

:deep(.el-tabs__content) {
    flex: 1;
    min-height: 0;
}

:deep(.el-tab-pane) {
    height: 100%;
}
</style>
