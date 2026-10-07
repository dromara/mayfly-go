<template>
    <div>
        <el-descriptions :column="3" border>
            <!-- 资源信息可能取不到：工单候选人不必是该资源的可见用户，实例也可能早已被删除 -->
            <template v-if="redis">
                <el-descriptions-item :span="3" :label="$t('common.tag')"><TagCodePath :path="redis.codePaths" /></el-descriptions-item>
                <el-descriptions-item :span="2" :label="$t('common.code')">{{ redis.code }}</el-descriptions-item>
                <el-descriptions-item :span="1" :label="$t('common.name')">{{ redis.name }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="Host">{{ redis.host }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="mode">{{ redis.mode }}</el-descriptions-item>
            </template>
            <el-descriptions-item v-else :span="3" :label="$t('common.name')">
                <el-text type="warning">{{ $t('flow.resourceUnavailable') }}</el-text>
            </el-descriptions-item>

            <el-descriptions-item :span="1" label="DB">{{ state.db }}</el-descriptions-item>

            <el-descriptions-item :span="3" :label="$t('flow.runCmd')">
                <el-input type="textarea" disabled v-model="cmd" rows="5" />
            </el-descriptions-item>
        </el-descriptions>

        <div v-if="runRes && runRes.length > 0">
            <el-divider content-position="left">{{ $t('flow.handleResult') }}</el-divider>
            <el-table :data="runRes" :max-height="400">
                <el-table-column prop="cmd" label="Cmd" show-overflow-tooltip />
                <el-table-column prop="res" :label="$t('flow.runResult')" :min-width="50" show-overflow-tooltip> </el-table-column>
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
    // 取不到资源信息时必须保持为 null，面板才能落到「不可用」提示而不是渲染一串空值
    redis: null as any,
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
    state.redis = res?.list?.[0] ?? null;
    if (!state.redis) {
        // 取不到就不查标签路径了：原来这里直接读 state.redis.code，会抛 TypeError 把面板打断
        return;
    }

    tagApi.listByQuery.request({ type: TagResourceTypeEnum.Redis.value, codes: state.redis.code }).then((tags) => {
        state.redis.codePaths = tags.map((item: any) => item.codePath);
    });
};
</script>
<style lang="scss"></style>
