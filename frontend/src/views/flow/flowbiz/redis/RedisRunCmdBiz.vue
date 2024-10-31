<template>
    <div>
        <el-descriptions :column="3" border>
            <el-descriptions-item :span="3" label="标签"><TagCodePath :path="redis.codePaths" /></el-descriptions-item>

            <el-descriptions-item :span="2" label="编号">{{ redis?.code }}</el-descriptions-item>
            <el-descriptions-item :span="1" label="名称">{{ redis?.name }}</el-descriptions-item>

            <el-descriptions-item :span="1" label="主机">{{ `${redis?.host}` }}</el-descriptions-item>
            <el-descriptions-item :span="1" label="库">{{ state.db }}</el-descriptions-item>
            <el-descriptions-item :span="1" label="mode">
                {{ redis.mode }}
            </el-descriptions-item>

            <el-descriptions-item :span="3" label="执行Cmd">
                <el-input type="textarea" disabled v-model="cmd" rows="5" />
            </el-descriptions-item>
        </el-descriptions>

        <div v-if="runRes && runRes.length > 0">
            <el-divider content-position="left">处理结果</el-divider>
            <el-table :data="runRes" :max-height="400">
                <el-table-column prop="cmd" label="命令" show-overflow-tooltip />
                <el-table-column prop="res" label="执行结果" :min-width="50" show-overflow-tooltip> </el-table-column>
            </el-table>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, onMounted } from 'vue';
import { redisApi } from '@/views/ops/redis/api';
import TagCodePath from '@/views/ops/component/TagCodePath.vue';
import { tagApi } from '@/views/ops/tag/api';
import { TagResourceTypeEnum } from '@/common/commonEnum';

const props = defineProps({
    procinst: {
        type: [Object],
        default: () => {},
    },
});

const state = reactive({
    cmd: '',
    runRes: [],
    db: 0,
    redis: {} as any,
});

const { cmd, redis, runRes } = toRefs(state);

onMounted(() => {
    parseRunCmdForm(props.procinst.bizForm);
});

watch(
    () => props.procinst.bizForm,
    (newValue: any) => {
        parseRunCmdForm(newValue);
    }
);

const parseRunCmdForm = async (bizFormStr: string) => {
    if (props.procinst.bizHandleRes) {
        state.runRes = JSON.parse(props.procinst.bizHandleRes);
    } else {
        state.runRes = [];
    }

    if (!bizFormStr) {
        return;
    }
    const bizForm = JSON.parse(bizFormStr);
    state.cmd = bizForm.cmd;
    state.db = bizForm.db;

    const res = await redisApi.redisList.request({ id: bizForm.id });
    if (!res.list) {
        return;
    }
    state.redis = res.list?.[0];

    tagApi.listByQuery.request({ type: TagResourceTypeEnum.Redis.value, codes: state.redis.code }).then((res) => {
        state.redis.codePaths = res.map((item: any) => item.codePath);
    });
};
</script>
<style lang="scss"></style>
