<template>
    <el-form-item :label="props.label">
        <el-select style="width: 100%" v-model="procdefKey" filterable placeholder="绑定流程则开启对应审批流程" v-bind="$attrs" clearable>
            <el-option v-for="item in procdefs" :key="item.defKey" :label="`${item.defKey} [${item.name}]`" :value="item.defKey"> </el-option>
        </el-select>
    </el-form-item>
</template>

<script lang="ts" setup>
import { onMounted, ref } from 'vue';
import { procdefApi } from '../api';

const props = defineProps({
    label: {
        type: String,
        default: '工单流程',
    },
});

onMounted(() => {
    getProcdefs();
});

const procdefKey = defineModel('modelValue');

const procdefs: any = ref([]);

const getProcdefs = () => {
    procdefApi.list.request({ pageSize: 200 }).then((res) => {
        procdefs.value = res.list;
    });
};
</script>
