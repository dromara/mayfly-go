<template>
    <el-tag v-if="!isEmpty" :disable-transitions="true" v-bind="$attrs" :type="tagType" :color="tagColor" effect="plain">{{ $t(tagLabel) }}</el-tag>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import EnumValue from '@/common/Enum';

const props = defineProps<{
    enums: Record<string, EnumValue>;
    value?: string | number | boolean | null;
}>();

const defaultType = 'primary';

/**
 * 空值（未填/不适用）整格不渲染内容。
 *
 * 下面的「未命中回显原始值」是为「有值但前端未登记展示元数据」设计的，而空串也走那条回退：
 * `String('')` 得空文本，再叠上未命中的 danger 色，就会渲出一个**没有字的红色空心框**——
 * 比什么都不显示更糟，看着像数据损坏（真机实测：反馈列未反馈的每行都有一个 20×24 空红框）。
 * 只拦空串与 null/undefined，不能用 falsy 判：0 与 false 是合法枚举值
 */
const isEmpty = computed(() => props.value === '' || props.value == null);

const enumValue = computed(() => {
    if (props.value == null || typeof props.value === 'boolean') {
        return null;
    }
    return EnumValue.getEnumByValue(props.enums, props.value);
});

// 未命中枚举时回退显示原始值而不是「-」：枚举清单是前端登记的展示元数据，后端目录（如 Agent
// 档位、事件触发源）新增一项时，旧行为会把完全正常的数据显示成红色「-」，看起来像数据损坏；
// 保留 danger 色作为「尚未登记展示元数据」的信号，既不丢信息也不掩盖漂移
const tagLabel = computed(() => enumValue.value?.label || String(props.value ?? '-'));

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
