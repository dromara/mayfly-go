<template>
    <template v-for="(seg, si) in segments" :key="`seg-${si}`">
        <!-- 分组容器（对齐 tokhub GroupContainer）：标题 + 描述 + 带边框内层，跨全宽；组内字段全部隐藏时不渲染空壳 -->
        <el-col v-if="seg.group && hasVisibleItem(seg)" :span="24">
            <div class="mb-1 text-sm font-medium">{{ seg.group.label ? $t(seg.group.label) : '' }}</div>
            <div v-if="seg.group.groupDescription" class="mb-2 text-xs text-gray-400 leading-5">{{ $t(seg.group.groupDescription) }}</div>
            <div class="mb-2 rounded-lg border p-3" style="border-color: var(--el-border-color-lighter)">
                <el-row :gutter="16">
                    <AutoFormFieldCol v-for="item in seg.items" :key="item.prop ?? item.label ?? ''" :item="item" :form="form" :default-span="defaultSpan" :readonly="readonly">
                        <template v-for="(_, name) in $slots" :key="name" #[name]="slotProps">
                            <slot :name="name" v-bind="slotProps ?? {}" />
                        </template>
                    </AutoFormFieldCol>
                </el-row>
            </div>
        </el-col>

        <!-- 无分组字段平铺（el-row 包裹以支持 span 并排布局） -->
        <el-row v-else :gutter="16">
            <AutoFormFieldCol v-for="item in seg.items" :key="item.prop ?? item.label ?? ''" :item="item" :form="form" :default-span="defaultSpan" :readonly="readonly">
                <template v-for="(_, name) in $slots" :key="name" #[name]="slotProps">
                    <slot :name="name" v-bind="slotProps ?? {}" />
                </template>
            </AutoFormFieldCol>
        </el-row>
    </template>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import AutoFormFieldCol from './AutoFormFieldCol.vue';
import type { AutoFormData, AutoFormItem } from './types';

const props = defineProps<{
    items: AutoFormItem[];
    /** 表单数据对象（共享引用） */
    form: AutoFormData;
    /** 栅格列数（默认 1） */
    cols?: number;
    /** 只读 */
    readonly?: boolean;
}>();

/** 每个字段默认占据的栅格跨度 */
const defaultSpan = computed(() => Math.floor(24 / (props.cols ?? 1)));

/** 分组内是否存在可见字段（动态 when 隐藏全部字段时整组不渲染，避免残留空壳容器） */
const hasVisibleItem = (seg: { group: AutoFormItem | null; items: AutoFormItem[] }): boolean =>
    !seg.group || seg.items.some((item) => !item.hidden && (!item.when || item.when(props.form)));

/** 按 group 类型分段：group 项包裹其后续字段，直到下一个 group（对齐 tokhub renderGroupedFields） */
const segments = computed(() => {
    const segs: { group: AutoFormItem | null; items: AutoFormItem[] }[] = [];
    let cur: { group: AutoFormItem | null; items: AutoFormItem[] } = { group: null, items: [] };
    for (const item of props.items) {
        if (item.type == 'group') {
            if (cur.group || cur.items.length > 0) {
                segs.push(cur);
            }
            cur = { group: item, items: [] };
        } else {
            cur.items.push(item);
        }
    }
    if (cur.group || cur.items.length > 0) {
        segs.push(cur);
    }
    return segs;
});
</script>
