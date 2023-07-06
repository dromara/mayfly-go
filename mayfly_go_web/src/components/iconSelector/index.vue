<template>
    <div class="icon-selector w100 h100">
        <el-input
            v-model="state.fontIconSearch"
            :placeholder="state.fontIconPlaceholder"
            :clearable="clearable"
            :disabled="disabled"
            :size="size"
            ref="inputWidthRef"
            @clear="onClearFontIcon"
            @focus="onIconFocus"
            @blur="onIconBlur"
        >
            <template #prepend>
                <SvgIcon :name="state.fontIconPrefix === '' ? prepend : state.fontIconPrefix" class="font14" />
            </template>
        </el-input>
        <el-popover
            placement="bottom"
            :width="state.fontIconWidth"
            transition="el-zoom-in-top"
            popper-class="icon-selector-popper"
            trigger="click"
            :virtual-ref="inputWidthRef"
            virtual-triggering
        >
            <template #default>
                <div class="icon-selector-warp">
                    <div class="icon-selector-warp-title">{{ title }}</div>
                    <el-tabs v-model="state.fontIconTabActive" @tab-click="onIconClick">
                        <el-tab-pane lazy label="ele" name="ele">
                            <IconList :list="fontIconSheetsFilterList" :empty="emptyDescription" :prefix="state.fontIconPrefix" @get-icon="onColClick" />
                        </el-tab-pane>
                        <el-tab-pane lazy label="ali" name="ali">
                            <IconList :list="fontIconSheetsFilterList" :empty="emptyDescription" :prefix="state.fontIconPrefix" @get-icon="onColClick" />
                        </el-tab-pane>
                        <!-- <el-tab-pane lazy label="awe" name="awe">
							<IconList :list="fontIconSheetsFilterList" :empty="emptyDescription" :prefix="state.fontIconPrefix" @get-icon="onColClick" />
						</el-tab-pane> -->
                    </el-tabs>
                </div>
            </template>
        </el-popover>
    </div>
</template>

<script setup lang="ts" name="iconSelector">
import { defineAsyncComponent, ref, reactive, onMounted, nextTick, computed, watch } from 'vue';
import type { TabsPaneContext } from 'element-plus';
import initIconfont from '@/common/utils/svgIcons';
import '@/theme/iconSelector.scss';

// 定义父组件传过来的值
const props = defineProps({
    // 输入框前置内容
    prepend: {
        type: String,
        default: () => 'Pointer',
    },
    // 输入框占位文本
    placeholder: {
        type: String,
        default: () => '请输入内容搜索图标或者选择图标',
    },
    // 输入框占位文本
    size: {
        type: String,
        default: () => 'default',
    },
    // 弹窗标题
    title: {
        type: String,
        default: () => '请选择图标',
    },
    // 禁用
    disabled: {
        type: Boolean,
        default: () => false,
    },
    // 是否可清空
    clearable: {
        type: Boolean,
        default: () => true,
    },
    // 自定义空状态描述文字
    emptyDescription: {
        type: String,
        default: () => '无相关图标',
    },
    // 双向绑定值，默认为 modelValue，
    // 参考：https://v3.cn.vuejs.org/guide/migration/v-model.html#%E8%BF%81%E7%A7%BB%E7%AD%96%E7%95%A5
    // 参考：https://v3.cn.vuejs.org/guide/component-custom-events.html#%E5%A4%9A%E4%B8%AA-v-model-%E7%BB%91%E5%AE%9A
    modelValue: String,
});

// 定义子组件向父组件传值/事件
const emit = defineEmits(['update:modelValue', 'get', 'clear']);

// 引入组件
const IconList = defineAsyncComponent(() => import('@/components/iconSelector/list.vue'));

// 定义变量内容
const inputWidthRef = ref();
const state = reactive({
    fontIconPrefix: '',
    fontIconWidth: 0,
    fontIconSearch: '',
    fontIconPlaceholder: '',
    fontIconTabActive: 'ele',
    fontIconList: {
        ali: [],
        ele: [],
        awe: [],
    },
});

// 处理 input 获取焦点时，modelValue 有值时，改变 input 的 placeholder 值
const onIconFocus = () => {
    if (!props.modelValue) return false;
    state.fontIconSearch = '';
    state.fontIconPlaceholder = props.modelValue;
};
// 处理 input 失去焦点时，为空将清空 input 值，为点击选中图标时，将取原先值
const onIconBlur = () => {
    const list = fontIconTabNameList();
    setTimeout(() => {
        const icon = list.filter((icon: string) => icon === state.fontIconSearch);
        if (icon.length <= 0) state.fontIconSearch = '';
    }, 300);
};
// 图标搜索及图标数据显示
const fontIconSheetsFilterList = computed(() => {
    const list = fontIconTabNameList();
    if (!state.fontIconSearch) return list;
    let search = state.fontIconSearch.trim().toLowerCase();
    return list.filter((item: string) => {
        if (item.toLowerCase().indexOf(search) !== -1) return item;
    });
});
// 根据 tab name 类型设置图标
const fontIconTabNameList = () => {
    let iconList: any = [];
    if (state.fontIconTabActive === 'ali') iconList = state.fontIconList.ali;
    else if (state.fontIconTabActive === 'ele') iconList = state.fontIconList.ele;
    else if (state.fontIconTabActive === 'awe') iconList = state.fontIconList.awe;
    return iconList;
};
// 处理 icon 双向绑定数值回显
const initModeValueEcho = () => {
    if (props.modelValue === '') return ((<string | undefined>state.fontIconPlaceholder) = props.placeholder);
    (<string | undefined>state.fontIconPlaceholder) = props.modelValue;
    (<string | undefined>state.fontIconPrefix) = props.modelValue;
};
// 处理 icon 类型，用于回显时，tab 高亮与初始化数据
const initFontIconName = () => {
    let name = 'ele';
    if (props.modelValue!.indexOf('iconfont') > -1) {
        name = 'ali';
    } else {
        name = 'ele';
    }
    // else if (props.modelValue!.indexOf('ele-') > -1) name = 'ele';
    // else if (props.modelValue!.indexOf('fa') > -1) name = 'awe';
    // 初始化 tab 高亮回显
    state.fontIconTabActive = name;
    return name;
};
// 初始化数据
const initFontIconData = async (name: string) => {
    if (name === 'ali') {
        // 阿里字体图标使用 `iconfont xxx`
        if (state.fontIconList.ali.length > 0) return;
        const res: any = await initIconfont.ali();
        state.fontIconList.ali = res.map((i: string) => `iconfont ${i}`);
    } else if (name === 'ele') {
        // element plus 图标
        if (state.fontIconList.ele.length > 0) return;
        await initIconfont.ele().then((res: any) => {
            state.fontIconList.ele = res;
        });
    } else if (name === 'awe') {
        // fontawesome字体图标使用 `fa xxx`
        // if (state.fontIconList.awe.length > 0) return;
        // await initIconfont.awe().then((res: any) => {
        // 	state.fontIconList.awe = res.map((i: string) => `fa ${i}`);
        // });
    }
    // 初始化 input 的 placeholder
    // 参考（单项数据流）：https://cn.vuejs.org/v2/guide/components-props.html?#%E5%8D%95%E5%90%91%E6%95%B0%E6%8D%AE%E6%B5%81
    state.fontIconPlaceholder = props.placeholder;
    // 初始化双向绑定回显
    initModeValueEcho();
};
// 图标点击切换
const onIconClick = (pane: TabsPaneContext) => {
    initFontIconData(pane.paneName as string);
    inputWidthRef.value.focus();
};
// 获取当前点击的 icon 图标
const onColClick = (v: string) => {
    state.fontIconPlaceholder = v;
    state.fontIconPrefix = v;
    emit('get', state.fontIconPrefix);
    emit('update:modelValue', state.fontIconPrefix);
    inputWidthRef.value.focus();
};
// 清空当前点击的 icon 图标
const onClearFontIcon = () => {
    state.fontIconPrefix = '';
    emit('clear', state.fontIconPrefix);
    emit('update:modelValue', state.fontIconPrefix);
};
// 获取 input 的宽度
const getInputWidth = () => {
    nextTick(() => {
        state.fontIconWidth = inputWidthRef.value.$el.offsetWidth;
    });
};
// 监听页面宽度改变
const initResize = () => {
    window.addEventListener('resize', () => {
        getInputWidth();
    });
};
// 页面加载时
onMounted(() => {
    initFontIconData(initFontIconName());
    initResize();
    getInputWidth();
});
// 监听双向绑定 modelValue 的变化
watch(
    () => props.modelValue,
    () => {
        initModeValueEcho();
        initFontIconName();
    }
);
</script>
