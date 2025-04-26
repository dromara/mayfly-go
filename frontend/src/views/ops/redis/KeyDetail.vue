<template>
    <div class="flex flex-col h-full">
        <!-- key info -->
        <key-header :redis="props.redis" :key-info="state.keyInfo" @refresh-content="refreshContent" @del-key="delKey" @change-key="changeKey"> </key-header>

        <!-- key content , 暂不懂为啥要用h-0或者其他随便设个高度？，h-full就是不行会导致loadMore按钮不显示 -->
        <component class="h-0 flex-1" ref="keyValueRef" :is="components[componentName]" :redis="props.redis" :key-info="keyInfo"> </component>
    </div>
</template>
<script lang="ts" setup>
import { defineAsyncComponent, watch, ref, shallowReactive, reactive, computed, onMounted } from 'vue';
import { ElMessage } from 'element-plus';
import KeyHeader from './KeyHeader.vue';
import { RedisInst } from './redis';

const KeyValueString = defineAsyncComponent(() => import('./KeyValueString.vue'));
const KeyValueHash = defineAsyncComponent(() => import('./KeyValueHash.vue'));
const KeyValueSet = defineAsyncComponent(() => import('./KeyValueSet.vue'));
const KeyValueList = defineAsyncComponent(() => import('./KeyValueList.vue'));
const KeyValueZset = defineAsyncComponent(() => import('./KeyValueZset.vue'));

const components: any = shallowReactive({
    KeyValueString,
    KeyValueHash,
    KeyValueSet,
    KeyValueList,
    KeyValueZset,
});

const keyValueRef = ref(null) as any;

const props = defineProps({
    redis: {
        type: RedisInst,
        required: true,
    },
    keyInfo: {
        type: [Object],
    },
});

const emit = defineEmits(['update:visible', 'changeKey', 'delKey']);

const state = reactive({
    keyInfo: {} as any,
});

const componentMap: any = {
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
.key-content-container {
    margin-top: 15px;
}

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
