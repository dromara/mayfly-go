<template>
    <div v-if="items.length" class="search-form">
        <el-form ref="formRef" :model="searchParam" label-width="auto">
            <Grid ref="gridRef" :collapsed="collapsed" :gap="[20, 0]" :cols="searchCol">
                <GridItem v-for="(item, index) in items" :key="item.prop" v-bind="getResponsive(item)" :index="index">
                    <el-form-item>
                        <template #label>
                            <el-space :size="4">
                                <span>{{ `${item?.label}` }}</span>
                                <el-tooltip v-if="item.tooltip" :content="item?.tooltip" placement="top">
                                    <SvgIcon name="QuestionFilled" />
                                </el-tooltip>
                            </el-space>
                            <span>:</span>
                        </template>

                        <SearchFormItem @keyup.enter="handleItemKeyupEnter(item)" v-if="!item.slot" :item="item" v-model="searchParam[item.prop]" />

                        <slot v-else :name="item.slot"></slot>
                    </el-form-item>
                </GridItem>
                <GridItem suffix>
                    <div class="operation">
                        <el-button type="primary" :icon="Search" @click="search" plain> 搜索 </el-button>
                        <el-button :icon="Delete" @click="reset"> 重置 </el-button>
                        <el-button v-if="showCollapse" type="primary" link class="search-isOpen" @click="collapsed = !collapsed">
                            {{ collapsed ? '展开' : '合并' }}
                            <el-icon class="el-icon--right">
                                <component :is="collapsed ? ArrowDown : ArrowUp"></component>
                            </el-icon>
                        </el-button>
                    </div>
                </GridItem>
            </Grid>
        </el-form>
    </div>
</template>
<script setup lang="ts" name="SearchForm">
import { computed, ref } from 'vue';
import { BreakPoint } from '@/components/Grid/interface/index';
import { Delete, Search, ArrowDown, ArrowUp } from '@element-plus/icons-vue';
import SearchFormItem from './components/SearchFormItem.vue';
import Grid from '@/components/Grid/index.vue';
import GridItem from '@/components/Grid/components/GridItem.vue';
import SvgIcon from '@/components/svgIcon/index.vue';
import { SearchItem } from './index';

interface ProTableProps {
    items: SearchItem[]; // 搜索配置项
    searchCol: number | Record<BreakPoint, number>;
    search: (params: any) => void; // 搜索方法
    reset: (params: any) => void; // 重置方法
}

// 默认值
const props = withDefaults(defineProps<ProTableProps>(), {
    items: () => [],
    modelValue: () => ({}),
});

const searchParam: any = defineModel('modelValue');

// 获取响应式设置
const getResponsive = (item: SearchItem) => {
    return {
        span: item?.span,
        offset: item.offset ?? 0,
        // xs: item.search?.xs,
        // sm: item.search?.sm,
        // md: item.search?.md,
        // lg: item.search?.lg,
        // xl: item.search?.xl,
    };
};

// 是否默认折叠搜索项
const collapsed = ref(true);

// 获取响应式断点
const gridRef = ref();
const breakPoint = computed<BreakPoint>(() => gridRef.value?.breakPoint);

// 判断是否显示 展开/合并 按钮
const showCollapse = computed(() => {
    let show = false;
    props.items.reduce((prev, current) => {
        prev += (current![breakPoint.value]?.span ?? current?.span ?? 1) + (current![breakPoint.value]?.offset ?? current?.offset ?? 0);
        if (typeof props.searchCol !== 'number') {
            if (prev >= props.searchCol[breakPoint.value]) show = true;
        } else {
            if (prev >= props.searchCol) show = true;
        }
        return prev;
    }, 0);
    return show;
});

const handleItemKeyupEnter = (item: SearchItem) => {
    if (item.type == 'input') {
        props.search(searchParam);
    }
};
</script>
<style lang="scss">
.search-form {
    padding: 18px 18px 0;
    margin-bottom: 10px;

    box-sizing: border-box;
    overflow-x: hidden;
    background-color: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 6px;
    box-shadow: 0 0 12px rgb(0 0 0 / 5%);

    .el-form {
        .el-form-item__content > * {
            width: 100%;
        }

        // 去除时间选择器上下 padding
        .el-range-editor.el-input__wrapper {
            padding: 0 10px;
        }

        .el-form-item {
            margin-bottom: 18px !important;
        }
    }

    .operation {
        display: flex;
        align-items: center;
        justify-content: flex-end;
        margin-bottom: 18px;
    }
}
</style>
