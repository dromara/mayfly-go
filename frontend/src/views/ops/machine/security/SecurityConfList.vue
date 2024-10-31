<template>
    <div class="card">
        <el-tabs v-model="activeName" class="demo-tabs" @tab-change="handleTabChange">
            <el-tab-pane label="命令配置" :name="CmdConfTab">
                <CmdConfList />
            </el-tab-pane>
        </el-tabs>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, onMounted, defineAsyncComponent } from 'vue';

const CmdConfList = defineAsyncComponent(() => import('./CmdConfList.vue'));

const CmdConfTab = 'cmdConf';

const state = reactive({
    activeName: CmdConfTab,
    cmdConfs: [],
});

const { activeName } = toRefs(state);

onMounted(async () => {
    state.activeName = CmdConfTab;
});

const handleTabChange = (tabName: any) => {
    if (tabName == CmdConfTab) {
        console.log('get cmd confs');
    }
    console.log(tabName);
};
</script>

<style></style>
