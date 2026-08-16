<template>
    <el-tag :disable-transitions="true" v-bind="$attrs" :type="tagType" :color="tagColor" effect="plain">{{ $t(tagLabel) }}</el-tag>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import EnumValue from '@/common/Enum';

const props = defineProps<{
    enums: Record<string, EnumValue>;
    value?: string | number | boolean | null;
}>();

const defaultType = 'primary';

const enumValue = computed(() => {
    if (props.value == null || typeof props.value === 'boolean') {
        return null;
    }
    return EnumValue.getEnumByValue(props.enums, props.value);
});

const tagLabel = computed(() => enumValue.value?.label || '-');

const tagType = computed(() => {
    if (!enumValue.value) {
        return 'danger';
    }
    return enumValue.value.tag?.type || defaultType;
});

const tagColor = computed(() => {
    if (!enumValue.value) {
        return '';
    }
    return enumValue.value.tag?.color || '';
});
</script>
