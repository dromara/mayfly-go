<template>
    <div>
        <el-form class="key-content-string" label-width="auto">
            <div>
                <format-viewer ref="formatViewerRef" :content="string.value"></format-viewer>
            </div>
        </el-form>
        <div class="mt10 fr">
            <el-button @click="saveValue" type="primary" v-auth="'redis:data:save'">保 存</el-button>
        </div>
    </div>
</template>
<script lang="ts" setup>
import { ref, reactive, toRefs, onMounted } from 'vue';
import { redisApi } from './api';
import { ElMessage } from 'element-plus';
import { notEmpty } from '@/common/assert';
import FormatViewer from './FormatViewer.vue';

const props = defineProps({
    redisId: {
        type: [Number],
        require: true,
        default: 0,
    },
    db: {
        type: [Number],
        require: true,
        default: 0,
    },
    keyInfo: {
        type: [Object],
    },
});

const formatViewerRef = ref(null) as any;

const state = reactive({
    redisId: 0,
    db: 0,
    key: '',
    keyInfo: {
        key: '',
        type: 'string',
        timed: -1,
    },
    string: {
        type: 'text',
        value: '',
    },
});

const { string } = toRefs(state);

onMounted(() => {
    state.redisId = props.redisId;
    state.db = props.db;
    state.key = props.keyInfo?.key;
    initData();
});

const initData = () => {
    getStringValue();
};

const getStringValue = async () => {
    if (state.key) {
        state.string.value = await redisApi.getString.request(getBaseReqParam());
    }
};

const saveValue = async () => {
    state.string.value = formatViewerRef.value.getContent();
    notEmpty(state.string.value, 'value不能为空');

    await redisApi.setString.request({
        ...getBaseReqParam(),
        value: state.string.value,
    });
    ElMessage.success('数据保存成功');
};

const getBaseReqParam = () => {
    return {
        id: state.redisId,
        db: state.db,
        key: state.key,
    };
};

defineExpose({ initData });
</script>
<style lang="scss">
.key-content-string .format-viewer-container {
    min-height: calc(100vh - 453px);
}

/*text viewer box*/
.key-content-string .el-textarea textarea {
    font-size: 14px;
    height: calc(100vh - 436px);
}

/*json in monaco editor*/
.key-content-string .monaco-editor-content {
    height: calc(100vh - 450px) !important;
}
</style>
