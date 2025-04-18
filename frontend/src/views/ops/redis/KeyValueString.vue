<template>
    <div>
        <el-form class="key-content-string" label-width="auto">
            <div>
                <format-viewer ref="formatViewerRef" height="250px" :content="string.value"></format-viewer>
            </div>
        </el-form>
        <div class="!mt-2 float-right">
            <el-button @click="saveValue" type="primary" v-auth="'redis:data:save'">{{ $t('common.save') }}</el-button>
        </div>
    </div>
</template>
<script lang="ts" setup>
import { ref, watch, reactive, toRefs, onMounted } from 'vue';
import { notEmptyI18n } from '@/common/assert';
import FormatViewer from './FormatViewer.vue';
import { RedisInst } from './redis';
import { useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { redisApi } from './api';

const props = defineProps({
    redis: {
        type: RedisInst,
        required: true,
    },
    keyInfo: {
        type: [Object],
    },
});

const formatViewerRef = ref(null) as any;

const state = reactive({
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
    setProps(props);
});

watch(props, (newVal) => {
    setProps(newVal);
});

const setProps = (val: any) => {
    state.key = val.keyInfo?.key;
    initData();
};

const initData = () => {
    getStringValue();
};

const getStringValue = async () => {
    if (state.key) {
        state.string.value = await props.redis.runCmd(['GET', state.key]);
    }
};

const saveValue = async () => {
    state.string.value = formatViewerRef.value.getContent();
    notEmptyI18n(state.string.value, 'value');

    const ttl = await redisApi.keyTtl.request({
        id: props.redis.id,
        db: props.redis.db,
        key: state.key,
    });

    let tArr = [];
    if (ttl > 0) {
        tArr.push('EX', ttl);
    }

    await props.redis.runCmd(['SET', state.key, state.string.value, ...tArr]);
    useI18nSaveSuccessMsg();
};

defineExpose({ initData });
</script>
<style lang="scss">
// .key-content-string .format-viewer-container {
//     min-height: calc(100vh - 253px);
// }

// /*text viewer box*/
// .key-content-string .el-textarea textarea {
//     font-size: 14px;
//     height: calc(100vh - 436px);
// }

// /*json in monaco editor*/
// .key-content-string .monaco-editor-content {
//     height: calc(100vh - 450px) !important;
// }
</style>
