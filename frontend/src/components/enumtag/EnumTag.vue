<template>
    <el-tag :disable-transitions="true" v-bind="$attrs" :type="type" :color="color" effect="plain">{{ $t(enumLabel) }}</el-tag>
</template>

<script lang="ts" setup>
import { toRefs, watch, reactive, onMounted } from 'vue';
import EnumValue from '@/common/Enum';

const props = defineProps({
    enums: {
        type: Object, // 需要为EnumValue类型
        required: true,
    },
    value: {
        type: [Object, String, Number, null, Boolean],
        required: true,
        default: () => null,
    },
});

const defaultType = 'primary';

const state = reactive({
    type: defaultType,
    color: '',
    enumLabel: '',
});

const { type, color, enumLabel } = toRefs(state);

// 监听该值是否改变，改变则需要将其枚举值与标签进行调整
watch(
    () => props.value,
    (newValue: any) => {
        convert(newValue);
    }
);

onMounted(() => {
    convert(props.value);
});

const convert = (value: any) => {
    const enumValue = EnumValue.getEnumByValue(props.enums, value);
    if (!enumValue) {
        state.enumLabel = '-';
        state.type = 'danger';
        state.color = '';
        return;
    }

    state.enumLabel = enumValue?.label || '';
    if (enumValue.tag) {
        state.color = enumValue.tag.color || '';
        state.type = enumValue.tag.type || defaultType;
    } else {
        state.type = defaultType;
    }
};
</script>
<style scoped lang="scss"></style>
