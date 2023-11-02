<template>
    <div>
        <el-container direction="vertical" class="key-tab-container">
            <!-- key info -->
            <key-header
                ref="keyHeader"
                :redis-id="redisId"
                :db="db"
                :key-info="state.keyInfo"
                @refresh-content="refreshContent"
                @del-key="delKey"
                @change-key="changeKey"
                class="key-header-info"
            >
            </key-header>

            <!-- key content -->
            <component ref="keyValueRef" :is="components[componentName]" :redis-id="redisId" :db="db" :key-info="keyInfo"> </component>
        </el-container>
    </div>
</template>
<script lang="ts" setup>
import { defineAsyncComponent, watch, ref, shallowReactive, reactive, computed, onMounted } from 'vue';
import { ElMessage } from 'element-plus';
import KeyHeader from './KeyHeader.vue';

const KeyValueString = defineAsyncComponent(() => import('./KeyValueString.vue'));
const KeyValueHash = defineAsyncComponent(() => import('./KeyValueHash.vue'));
const KeyValueSet = defineAsyncComponent(() => import('./KeyValueSet.vue'));
const KeyValueList = defineAsyncComponent(() => import('./KeyValueList.vue'));
const KeyValueZset = defineAsyncComponent(() => import('./KeyValueZset.vue'));

const components = shallowReactive({
    KeyValueString,
    KeyValueHash,
    KeyValueSet,
    KeyValueList,
    KeyValueZset,
});

const keyValueRef = ref(null) as any;

const props = defineProps({
    redisId: {
        type: Number,
    },
    db: {
        type: Number,
    },
    keyInfo: {
        type: [Object],
    },
});

const emit = defineEmits(['update:visible', 'changeKey', 'delKey']);

const state = reactive({
    redisId: 0,
    keyInfo: {} as any,
});

const componentMap = {
    string: 'KeyValueString',
    hash: 'KeyValueHash',
    zset: 'KeyValueZset',
    set: 'KeyValueSet',
    list: 'KeyValueList',
};

const componentName = computed(() => {
    const component = componentMap[props.keyInfo?.type];
    if (!component) {
        ElMessage.error('暂不支持该类型');
        return '';
    }
    return component;
});

const refreshContent = () => {
    keyValueRef.value?.initData();
};

const delKey = () => {
    emit('delKey', state.keyInfo.key);
};

const changeKey = () => {
    emit('changeKey');
};

const setKeyInfo = (val: any) => {
    state.keyInfo.timed = val.timed;
    state.keyInfo.key = val.key;
    state.keyInfo.type = val.type;
};

watch(
    () => props.keyInfo,
    (val) => {
        setKeyInfo(val);
    },
    {
        deep: true,
    }
);

onMounted(() => {
    setKeyInfo(props.keyInfo);
});
</script>
<style lang="scss">
.key-tab-container {
    /*padding-left: 5px;*/
}

.key-header-info {
    // margin-top: 15px;
}

.key-content-container {
    margin-top: 15px;
}

// .key-detail-filter-value {
//     width: 90%;
//     height: 24px;
//     padding: 0 5px;
// }

/*tooltip in table width limit*/
.el-tooltip__popper {
    max-width: 50%;
}

.content-more-container {
    text-align: center;
    margin-top: 10px;
}

.content-more-container .content-more-btn {
    width: 95%;
    padding-top: 5px;
    padding-bottom: 5px;
}

/*data table list styles*/
.key-content-container .el-table {
    border-radius: 3px;
}

/*table list height*/
.key-content-container .el-table .el-table__body td {
    padding: 0px 0px;
}

/*table list border*/
.key-content-container .el-table--border td,
.key-content-container .el-table--border th {
    border-right-width: 0;
}
</style>
