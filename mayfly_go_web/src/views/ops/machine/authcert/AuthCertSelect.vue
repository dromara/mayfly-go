<template>
    <div style="width: 100%">
        <el-select @change="changeValue" v-model="id" filterable placeholder="请选择授权凭证,可前往[机器管理->授权凭证]添加" style="width: 100%">
            <el-option v-for="ac in acs" :key="ac.id" :value="ac.id" :label="ac.name">
                <el-tag v-if="ac.authMethod == 1" type="success" size="small">密码</el-tag>
                <el-tag v-if="ac.authMethod == 2" size="small">密钥</el-tag>

                <el-divider direction="vertical" border-style="dashed" />
                {{ ac.name }}

                <el-divider direction="vertical" border-style="dashed" />
                {{ ac.remark }}
            </el-option>
        </el-select>
    </div>
</template>

<script lang="ts" setup>
import { reactive, toRefs, onMounted } from 'vue';
import { authCertApi } from '../api';

//定义事件
const emit = defineEmits(['update:modelValue', 'change']);

const props = defineProps({
    modelValue: {
        type: [Number],
        required: true,
    },
});

const state = reactive({
    acs: [] as any,
    id: null as any,
});

const { acs, id } = toRefs(state);

onMounted(async () => {
    await getAcs();
    if (props.modelValue) {
        state.id = props.modelValue;
    }
});

const changeValue = (val: any) => {
    emit('update:modelValue', val);
    emit('change', val);
};

const getAcs = async () => {
    const acs = await authCertApi.baseList.request({ pageSize: 100, type: 2 });
    state.acs = acs.list;
};
</script>

<style lang="scss"></style>
