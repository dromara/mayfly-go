<template>
    <div class="icon-selector">
        <el-popover :placement="placement" :width="650" v-model:visible="fontIconVisible" popper-class="icon-selector-popper">
            <template #reference>
                <el-input
                    v-model="modelValue"
                    placeholder="请点击选择图标"
                    clearable
                    size="small"
                    ref="inputWidthRef"
                    :prefix-icon="modelValue"
                    @clear="onClearFontIcon"
                ></el-input>
            </template>
            <transition name="el-zoom-in-top">
                <div class="icon-selector-warp" v-show="fontIconVisible">
                    <div class="icon-selector-warp-title">请选择一个图标</div>
                    <div v-if="isAllOn" class="icon-selector-all">
                        <el-input v-model="fontIconSearch" placeholder="请输入内容进行搜索" size="small"></el-input>
                        <div class="icon-selector-all-tabs">
                            <div
                                class="icon-selector-all-tabs-item"
                                v-for="(v, k) in fontIconTabsList"
                                :key="k"
                                @click="onFontIconTabsClick(v, k)"
                                :class="{ 'icon-selector-all-tabs-active': fontIconTabsIndex === k }"
                            >
                                <div class="label">{{ v.label }}</div>
                            </div>
                        </div>
                    </div>
                    <div class="icon-selector-warp-row">
                        <el-row :gutter="10">
                            <el-col
                                :xs="4"
                                :sm="4"
                                :md="2"
                                :lg="2"
                                :xl="1"
                                :class="`${fontIconTabsIcon}-col`"
                                @click="onColClick(v, k)"
                                v-for="(v, k) in fontIconSheetsFilterList"
                                :key="k"
                            >
                                <div class="icon-selector-warp-item" :class="{ 'icon-selector-active': fontIconIndex === k }">
                                    <div class="flex-margin">
                                        <div class="icon-selector-warp-item-value">
                                            <i :class="[fontIconTabsIcon, v]"></i>
                                        </div>
                                    </div>
                                </div>
                            </el-col>
                        </el-row>
                        <el-empty :image-size="100" v-if="fontIconSheetsFilterList.length <= 0"></el-empty>
                    </div>
                </div>
            </transition>
        </el-popover>
    </div>
</template>

<script lang="ts">
import { ref, toRefs, reactive, onMounted, computed } from 'vue';
import initIconfont from '@/common/utils/getStyleSheets.ts';
export default {
    name: 'iconSelector',
    props: {
        // 是否开启高级功能
        isAllOn: {
            type: Boolean,
            default: () => false,
        },
        // 出现位置
        placement: {
            type: String,
            default: () => 'bottom',
        },
        modelValue: {
            type: String,
        },
    },
    setup(props, { emit }) {
        const inputWidthRef = ref();
        const state: any = reactive({
            // fontIcon: '',
            // fontIconPrefix: '',
            fontIconVisible: false,
            fontIconWidth: 0,
            fontIconIndex: '',
            fontIconSearch: '',
            fontIconTabsIndex: 0,
            fontIconTabsIcon: 'iconfont ali',
            fontIconTabsList: [{ label: 'element' }, { label: 'iconfont' }],
            fontIconSheetsList: [],
            fontIconSheetsListAli: [],
            fontIconSheetsListEle: [],
            fontIconSheetsListAwe: [],
        });
        // 设置无数据时的空状态
        const fontIconSheetsFilterList = computed(() => {
            if (!state.fontIconSearch) return state.fontIconSheetsList;
            let search = state.fontIconSearch.trim().toLowerCase();
            return state.fontIconSheetsList.filter((item: any) => {
                if (item.toLowerCase().indexOf(search) !== -1) return item;
            });
        });

        // 获取 input 的宽度
        // const getInputWidth = () => {
        // 	nextTick(() => {
        // 		state.fontIconWidth = inputWidthRef.value.$el.offsetWidth;
        // 	});
        // };
        // 监听页面宽度改变
        // const initResize = () => {
        // 	window.addEventListener('resize', () => {
        // 		getInputWidth();
        // 	});
        // };

        // 初始化数据
        const initFontIconData = () => {
            initIconfont.ele().then((res: any) => {
                state.fontIconSheetsListEle = res;
                state.fontIconSheetsList = res;
            });
            // initIconfont.ali().then((res: any) => {
            // 	state.fontIconSheetsListAli = res;
            // });
        };
        // 当前项点击
        const onColClick = (v: any, k: number) => {
            state.fontIconIndex = k;
            // state.fontIcon = v;
            state.fontIconVisible = false;
            // if (state.fontIconTabsIndex === 0) state.fontIconPrefix = `iconfont ali ${v}`;
            // else if (state.fontIconTabsIndex === 1) state.fontIconPrefix = `${v}`;
            emit('update:modelValue', v);
        };
        // input 点击清除按钮时
        const onClearFontIcon = () => {
            state.fontIconIndex = '';
            // state.fontIconPrefix = '';
            emit('update:modelValue', state.fontIconPrefix);
        };
        // tabs 点击
        const onFontIconTabsClick = (v: any, k: number) => {
            state.fontIconTabsIndex = k;
            if (v.label === 'iconfont') state.fontIconSheetsList = state.fontIconSheetsListAli;
            else if (v.label === 'element') state.fontIconSheetsList = state.fontIconSheetsListEle;
            if (k === 0) state.fontIconTabsIcon = `iconfont ali`;
            else if (k === 1) state.fontIconTabsIcon = `ele`;
        };
        // 页面加载时
        onMounted(() => {
            initFontIconData();
            // initResize();
            // getInputWidth();
        });
        return {
            inputWidthRef,
            fontIconSheetsFilterList,
            onColClick,
            onClearFontIcon,
            onFontIconTabsClick,
            ...toRefs(state),
        };
    },
};
</script>
