<template>
    <div class="layout-breadcrumb-seting">
        <el-drawer title="布局设置" v-model="themeConfig.isDrawer" direction="rtl" destroy-on-close size="240px" @close="onDrawerClose">
            <el-scrollbar class="layout-breadcrumb-seting-bar">
                <!-- ssh终端主题 -->
                <el-divider content-position="left">终端主题</el-divider>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">字体颜色</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.terminalForeground" size="small" @change="onColorPickerChange('terminalForeground')">
                        </el-color-picker>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">背景颜色</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.terminalBackground" size="small" @change="onColorPickerChange('terminalBackground')">
                        </el-color-picker>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">cursor颜色</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.terminalCursor" size="small" @change="onColorPickerChange('terminalCursor')"> </el-color-picker>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">字体大小</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-input-number
                            v-model="themeConfig.terminalFontSize"
                            controls-position="right"
                            :min="12"
                            :max="24"
                            @change="setLocalThemeConfig"
                            size="small"
                            style="width: 90px"
                        >
                        </el-input-number>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">字体粗细</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-select @change="setLocalThemeConfig" v-model="themeConfig.terminalFontWeight" size="small" style="width: 90px">
                            <el-option label="normal" value="normal"> </el-option>
                            <el-option label="bold" value="bold"> </el-option>
                        </el-select>
                    </div>
                </div>

                <el-divider content-position="left">editor 设置</el-divider>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">主题</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-select @change="setLocalThemeConfig" v-model="themeConfig.editorTheme" size="small" style="width: 130px">
                            <el-option label="vs" value="vs"> </el-option>
                            <el-option label="vs-dark" value="vs-dark"> </el-option>
                            <el-option label="SolarizedLight" value="SolarizedLight"> </el-option>
                        </el-select>
                    </div>
                </div>

                <!-- 全局设置 -->
                <el-divider content-position="left">全局设置</el-divider>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">分页size</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-input-number
                            v-model="themeConfig.defaultListPageSize"
                            controls-position="right"
                            :min="10"
                            :max="50"
                            @change="setLocalThemeConfig"
                            size="small"
                            style="width: 90px"
                        >
                        </el-input-number>
                    </div>
                </div>

                <!-- 全局主题 -->
                <el-divider content-position="left">全局主题</el-divider>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">primary</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.primary" size="small" @change="onColorPickerChange('primary')"> </el-color-picker>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">success</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.success" size="small" @change="onColorPickerChange('success')"> </el-color-picker>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">info</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.info" size="small" @change="onColorPickerChange('info')"> </el-color-picker>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">warning</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.warning" size="small" @change="onColorPickerChange('warning')"> </el-color-picker>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">danger</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.danger" size="small" @change="onColorPickerChange('danger')"> </el-color-picker>
                    </div>
                </div>

                <!-- 菜单 / 顶栏 -->
                <el-divider content-position="left">菜单 / 顶栏</el-divider>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">顶栏背景</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.topBar" size="small" @change="onBgColorPickerChange('topBar')"> </el-color-picker>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">菜单背景</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.menuBar" size="small" @change="onBgColorPickerChange('menuBar')"> </el-color-picker>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">分栏菜单背景</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.columnsMenuBar" size="small" @change="onBgColorPickerChange('columnsMenuBar')"> </el-color-picker>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">顶栏默认字体颜色</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.topBarColor" size="small" @change="onBgColorPickerChange('topBarColor')"> </el-color-picker>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">菜单默认字体颜色</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.menuBarColor" size="small" @change="onBgColorPickerChange('menuBarColor')"> </el-color-picker>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">分栏菜单默认字体颜色</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.columnsMenuBarColor" size="small" @change="onBgColorPickerChange('columnsMenuBarColor')">
                        </el-color-picker>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt10">
                    <div class="layout-breadcrumb-seting-bar-flex-label">顶栏背景渐变</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isTopBarColorGradual" @change="onTopBarGradualChange"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt14">
                    <div class="layout-breadcrumb-seting-bar-flex-label">菜单背景渐变</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isMenuBarColorGradual" @change="onMenuBarGradualChange"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt14">
                    <div class="layout-breadcrumb-seting-bar-flex-label">分栏菜单背景渐变</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isColumnsMenuBarColorGradual" @change="onColumnsMenuBarGradualChange"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt14">
                    <div class="layout-breadcrumb-seting-bar-flex-label">菜单字体背景高亮</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isMenuBarColorHighlight" @change="onMenuBarHighlightChange"></el-switch>
                    </div>
                </div>

                <!-- 界面设置 -->
                <el-divider content-position="left">界面设置</el-divider>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">菜单水平折叠</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isCollapse" @change="onThemeConfigChange"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">菜单手风琴</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isUniqueOpened" @change="setLocalThemeConfig"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">固定 Header</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isFixedHeader" @change="onIsFixedHeaderChange"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15" :style="{ opacity: themeConfig.layout !== 'classic' ? 0.5 : 1 }">
                    <div class="layout-breadcrumb-seting-bar-flex-label">经典布局分割菜单</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isClassicSplitMenu" :disabled="themeConfig.layout !== 'classic'" @change="onClassicSplitMenuChange">
                        </el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">开启锁屏</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isLockScreen" @change="setLocalThemeConfig"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt11">
                    <div class="layout-breadcrumb-seting-bar-flex-label">自动锁屏(s/秒)</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-input-number
                            v-model="themeConfig.lockScreenTime"
                            controls-position="right"
                            :min="0"
                            :max="9999"
                            @change="setLocalThemeConfig"
                            size="small"
                            style="width: 90px"
                        >
                        </el-input-number>
                    </div>
                </div>

                <!-- 界面显示 -->
                <el-divider content-position="left">界面显示</el-divider>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">侧边栏 Logo</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isShowLogo" @change="onIsShowLogoChange"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15" :style="{ opacity: themeConfig.layout === 'transverse' ? 0.5 : 1 }">
                    <div class="layout-breadcrumb-seting-bar-flex-label">开启Breadcrumb</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch
                            v-model="themeConfig.isBreadcrumb"
                            :disabled="themeConfig.layout === 'transverse'"
                            @change="onIsBreadcrumbChange"
                        ></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">开启Breadcrumb图标</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isBreadcrumbIcon" @change="setLocalThemeConfig"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">开启 Tagsview</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isTagsview" @change="setLocalThemeConfig"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">开启 Tagsview图标</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isTagsviewIcon" @change="setLocalThemeConfig"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">开启 TagsView缓存</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isCacheTagsView" @change="setLocalThemeConfig"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">开启 TagsView拖拽</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isSortableTagsView" @change="onSortableTagsViewChange"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">开启 Footer</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isFooter" @change="setLocalThemeConfig"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">灰色模式</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isGrayscale" @change="onAddFilterChange('grayscale')"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">色弱模式</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isInvert" @change="onAddFilterChange('invert')"></el-switch>
                    </div>
                </div>

                <!-- 其它设置 -->
                <el-divider content-position="left">其他设置</el-divider>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">Tagsview 风格</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-select v-model="themeConfig.tagsStyle" placeholder="请选择" size="small" style="width: 90px" @change="setLocalThemeConfig">
                            <el-option label="风格1" value="tags-style-one"></el-option>
                            <el-option label="风格2" value="tags-style-two"></el-option>
                            <el-option label="风格3" value="tags-style-three"></el-option>
                        </el-select>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15">
                    <div class="layout-breadcrumb-seting-bar-flex-label">主页面切换动画</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-select v-model="themeConfig.animation" placeholder="请选择" size="small" style="width: 90px" @change="setLocalThemeConfig">
                            <el-option label="slide-right" value="slide-right"></el-option>
                            <el-option label="slide-left" value="slide-left"></el-option>
                            <el-option label="opacitys" value="opacitys"></el-option>
                        </el-select>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt15 mb28">
                    <div class="layout-breadcrumb-seting-bar-flex-label">分栏高亮风格</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-select v-model="themeConfig.columnsAsideStyle" placeholder="请选择" size="small" style="width: 90px" @change="setLocalThemeConfig">
                            <el-option label="圆角" value="columns-round"></el-option>
                            <el-option label="卡片" value="columns-card"></el-option>
                        </el-select>
                    </div>
                </div>

                <!-- 布局切换 -->
                <el-divider content-position="left">布局切换</el-divider>
                <div class="layout-drawer-content-flex">
                    <!-- defaults 布局 -->
                    <div class="layout-drawer-content-item" @click="onSetLayout('defaults')">
                        <section class="el-container el-circular" :class="{ 'drawer-layout-active': themeConfig.layout === 'defaults' }">
                            <aside class="el-aside" style="width: 20px"></aside>
                            <section class="el-container is-vertical">
                                <header class="el-header" style="height: 10px"></header>
                                <main class="el-main"></main>
                            </section>
                        </section>
                        <div class="layout-tips-warp" :class="{ 'layout-tips-warp-active': themeConfig.layout === 'defaults' }">
                            <div class="layout-tips-box">
                                <p class="layout-tips-txt">默认</p>
                            </div>
                        </div>
                    </div>
                    <!-- classic 布局 -->
                    <div class="layout-drawer-content-item" @click="onSetLayout('classic')">
                        <section class="el-container is-vertical el-circular" :class="{ 'drawer-layout-active': themeConfig.layout === 'classic' }">
                            <header class="el-header" style="height: 10px"></header>
                            <section class="el-container">
                                <aside class="el-aside" style="width: 20px"></aside>
                                <section class="el-container is-vertical">
                                    <main class="el-main"></main>
                                </section>
                            </section>
                        </section>
                        <div class="layout-tips-warp" :class="{ 'layout-tips-warp-active': themeConfig.layout === 'classic' }">
                            <div class="layout-tips-box">
                                <p class="layout-tips-txt">经典</p>
                            </div>
                        </div>
                    </div>
                    <!-- transverse 布局 -->
                    <div class="layout-drawer-content-item" @click="onSetLayout('transverse')">
                        <section class="el-container is-vertical el-circular" :class="{ 'drawer-layout-active': themeConfig.layout === 'transverse' }">
                            <header class="el-header" style="height: 10px"></header>
                            <section class="el-container">
                                <section class="el-container is-vertical">
                                    <main class="el-main"></main>
                                </section>
                            </section>
                        </section>
                        <div class="layout-tips-warp" :class="{ 'layout-tips-warp-active': themeConfig.layout === 'transverse' }">
                            <div class="layout-tips-box">
                                <p class="layout-tips-txt">横向</p>
                            </div>
                        </div>
                    </div>
                    <!-- columns 布局 -->
                    <div class="layout-drawer-content-item" @click="onSetLayout('columns')">
                        <section class="el-container el-circular" :class="{ 'drawer-layout-active': themeConfig.layout === 'columns' }">
                            <aside class="el-aside-dark" style="width: 10px"></aside>
                            <aside class="el-aside" style="width: 20px"></aside>
                            <section class="el-container is-vertical">
                                <header class="el-header" style="height: 10px"></header>
                                <main class="el-main"></main>
                            </section>
                        </section>
                        <div class="layout-tips-warp" :class="{ 'layout-tips-warp-active': themeConfig.layout === 'columns' }">
                            <div class="layout-tips-box">
                                <p class="layout-tips-txt">分栏</p>
                            </div>
                        </div>
                    </div>
                </div>
                <!-- <div class="copy-config">
                    <el-alert title="点击下方按钮，复制布局配置去 /src/store/modules/themeConfig.ts中修改" type="warning" :closable="false"> </el-alert>
                    <el-button
                        size="small"
                        class="copy-config-btn"
                        icon="el-icon-document-copy"
                        type="primary"
                        ref="copyConfigBtnRef"
                        @click="onCopyConfigClick($event.target)"
                        >一键复制配置
                    </el-button>
                </div> -->
            </el-scrollbar>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup name="layoutBreadcrumbSeting">
import { nextTick, onUnmounted, onMounted, ref } from 'vue';
import { ElMessage } from 'element-plus';
import ClipboardJS from 'clipboard';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { getLightColor } from '@/common/utils/theme';
import { setLocal, getLocal, removeLocal } from '@/common/utils/storage';
import mittBus from '@/common/utils/mitt';

const copyConfigBtnRef = ref();
const { themeConfig } = storeToRefs(useThemeConfig());

// 1、全局主题
const onColorPickerChange = (color: string) => {
    setPropertyFun(`--color-${color}`, themeConfig.value[color]);
    setDispatchThemeConfig();
};
// 1、全局主题设置函数
const setPropertyFun = (color: string, targetVal: any) => {
    document.documentElement.style.setProperty(color, targetVal);
    for (let i = 1; i <= 9; i++) {
        document.documentElement.style.setProperty(`${color}-light-${i}`, getLightColor(targetVal, i / 10) as any);
    }
};
// 2、菜单 / 顶栏
const onBgColorPickerChange = (bg: string) => {
    document.documentElement.style.setProperty(`--bg-${bg}`, themeConfig.value[bg]);
    onTopBarGradualChange();
    onMenuBarGradualChange();
    onColumnsMenuBarGradualChange();
    setDispatchThemeConfig();
};
// 2、菜单 / 顶栏 --> 顶栏背景渐变
const onTopBarGradualChange = () => {
    setGraduaFun('.layout-navbars-breadcrumb-index', themeConfig.value.isTopBarColorGradual, themeConfig.value.topBar);
};
// 2、菜单 / 顶栏 --> 菜单背景渐变
const onMenuBarGradualChange = () => {
    setGraduaFun('.layout-container .el-aside', themeConfig.value.isMenuBarColorGradual, themeConfig.value.menuBar);
};
// 2、菜单 / 顶栏 --> 分栏菜单背景渐变
const onColumnsMenuBarGradualChange = () => {
    setGraduaFun('.layout-container .layout-columns-aside', themeConfig.value.isColumnsMenuBarColorGradual, themeConfig.value.columnsMenuBar);
};
// 2、菜单 / 顶栏 --> 背景渐变函数
const setGraduaFun = (el: string, bool: boolean, color: string) => {
    nextTick(() => {
        let els = document.querySelector(el);
        if (!els) return false;
        if (bool) els.setAttribute('style', `background-image:linear-gradient(to bottom left , ${color}, ${getLightColor(color, 0.6)})`);
        else els.setAttribute('style', `background-image:${color}`);
        setLocalThemeConfig();
        const elNavbars: any = document.querySelector('.layout-navbars-breadcrumb-index');
        const elAside: any = document.querySelector('.layout-container .el-aside');
        const elColumns: any = document.querySelector('.layout-container .layout-columns-aside');
        if (elNavbars) setLocal('navbarsBgStyle', elNavbars.style.cssText);
        if (elAside) setLocal('asideBgStyle', elAside.style.cssText);
        if (elColumns) setLocal('columnsBgStyle', elColumns.style.cssText);
    });
};
// 2、菜单 / 顶栏 --> 菜单字体背景高亮
const onMenuBarHighlightChange = () => {
    nextTick(() => {
        setTimeout(() => {
            let elsItems = document.querySelectorAll('.el-menu-item');
            let elActive = document.querySelector('.el-menu-item.is-active');
            if (!elActive) return false;
            if (themeConfig.value.isMenuBarColorHighlight) {
                elsItems.forEach((el: any) => el.setAttribute('id', ``));
                elActive.setAttribute('id', `add-is-active`);
                setLocal('menuBarHighlightId', elActive.getAttribute('id'));
            } else {
                elActive.setAttribute('id', ``);
            }
            setLocalThemeConfig();
        }, 0);
    });
};
// 3、界面设置 --> 菜单水平折叠
const onThemeConfigChange = () => {
    onMenuBarHighlightChange();
    setDispatchThemeConfig();
};
// 3、界面设置 --> 固定 Header
const onIsFixedHeaderChange = () => {
    themeConfig.value.isFixedHeaderChange = themeConfig.value.isFixedHeader ? false : true;
    setLocalThemeConfig();
};
// 3、界面设置 --> 经典布局分割菜单
const onClassicSplitMenuChange = () => {
    themeConfig.value.isBreadcrumb = false;
    setLocalThemeConfig();
    mittBus.emit('getBreadcrumbIndexSetFilterRoutes');
};
// 4、界面显示 --> 侧边栏 Logo
const onIsShowLogoChange = () => {
    themeConfig.value.isShowLogoChange = themeConfig.value.isShowLogo ? false : true;
    setLocalThemeConfig();
};
// 4、界面显示 --> 面包屑 Breadcrumb
const onIsBreadcrumbChange = () => {
    if (themeConfig.value.layout === 'classic') {
        themeConfig.value.isClassicSplitMenu = false;
    }
    setLocalThemeConfig();
};
// 4、界面显示 --> 开启 TagsView 拖拽
const onSortableTagsViewChange = () => {
    mittBus.emit('openOrCloseSortable');
    setLocalThemeConfig();
};
// 4、界面显示 --> 暗模式/灰色模式/色弱模式
const onAddFilterChange = (attr: string) => {
    if (attr === 'grayscale') {
        if (themeConfig.value.isGrayscale) themeConfig.value.isInvert = false;
    } else {
        if (themeConfig.value.isInvert) themeConfig.value.isGrayscale = false;
    }
    const cssAttr = attr === 'grayscale' ? `grayscale(${themeConfig.value.isGrayscale ? 1 : 0})` : `invert(${themeConfig.value.isInvert ? '80%' : '0%'})`;
    const appEle: any = document.querySelector('#app');
    appEle.setAttribute('style', `filter: ${cssAttr}`);
    setLocalThemeConfig();
    setLocal('appFilterStyle', appEle.style.cssText);
};
// 5、布局切换
const onSetLayout = (layout: string) => {
    setLocal('oldLayout', layout);
    if (themeConfig.value.layout === layout) return false;
    themeConfig.value.layout = layout;
    themeConfig.value.isDrawer = false;
    initSetLayoutChange();
    onMenuBarHighlightChange();
};
// 设置布局切换，重置主题样式
const initSetLayoutChange = () => {
    // themeConfig.value.menuBar = '#FFFFFF';
    // themeConfig.value.menuBarColor = '#606266';
    // themeConfig.value.topBar = '#ffffff';
    // themeConfig.value.topBarColor = '#606266';

    if (themeConfig.value.layout === 'classic') {
        themeConfig.value.isShowLogo = true;
        themeConfig.value.isBreadcrumb = true;
        themeConfig.value.isCollapse = false;
        themeConfig.value.isClassicSplitMenu = false;
    } else if (themeConfig.value.layout === 'transverse') {
        themeConfig.value.isShowLogo = true;
        themeConfig.value.isBreadcrumb = false;
        themeConfig.value.isCollapse = false;
        themeConfig.value.isTagsview = true;
        themeConfig.value.isClassicSplitMenu = false;
    } else if (themeConfig.value.layout === 'columns') {
        themeConfig.value.isShowLogo = true;
        themeConfig.value.isBreadcrumb = true;
        themeConfig.value.isCollapse = false;
        themeConfig.value.isTagsview = true;
        themeConfig.value.isClassicSplitMenu = false;
    } else {
        themeConfig.value.isShowLogo = false;
        themeConfig.value.isBreadcrumb = true;
        themeConfig.value.isCollapse = false;
        themeConfig.value.isTagsview = true;
        themeConfig.value.isClassicSplitMenu = false;
    }

    initLayoutChangeFun();
};
// 设置布局切换函数
const initLayoutChangeFun = () => {
    onBgColorPickerChange('menuBar');
    onBgColorPickerChange('menuBarColor');
    onBgColorPickerChange('topBar');
    onBgColorPickerChange('topBarColor');
};
// 关闭弹窗时，初始化变量。变量用于处理 proxy.$refs.layoutScrollbarRef.update()
const onDrawerClose = () => {
    themeConfig.value.isFixedHeaderChange = false;
    themeConfig.value.isShowLogoChange = false;
    themeConfig.value.isDrawer = false;
    setLocalThemeConfig();
};
// 布局配置弹窗打开
const openDrawer = () => {
    themeConfig.value.isDrawer = true;
    nextTick(() => {
        // 初始化复制功能，防止点击两次才可以复制
        onCopyConfigClick(copyConfigBtnRef.value?.$el);
    });
};
// 触发 store 布局配置更新
const setDispatchThemeConfig = () => {
    setLocalThemeConfig();
    setLocalThemeConfigStyle();
};
// 存储布局配置
const setLocalThemeConfig = () => {
    removeLocal('themeConfig');
    setLocal('themeConfig', themeConfig.value);
};
// 存储布局配置全局主题样式（html根标签）
const setLocalThemeConfigStyle = () => {
    setLocal('themeConfigStyle', document.documentElement.style.cssText);
};
// 一键复制配置
const onCopyConfigClick = (target: any) => {
    if (!target) {
        return;
    }
    let copyThemeConfig = getLocal('themeConfig');
    copyThemeConfig.isDrawer = false;
    const clipboard = new ClipboardJS(target, {
        text: () => JSON.stringify(copyThemeConfig),
    });
    clipboard.on('success', () => {
        themeConfig.value.isDrawer = false;
        ElMessage.success('复制成功');
        clipboard.destroy();
    });
    clipboard.on('error', () => {
        ElMessage.error('复制失败');
        clipboard.destroy();
    });
};
onMounted(() => {
    nextTick(() => {
        // 监听菜单点击，菜单字体背景高亮
        mittBus.on('onMenuClick', () => {
            onMenuBarHighlightChange();
        });
        // 监听窗口大小改变，非默认布局，设置成默认布局（适配移动端）
        mittBus.on('layoutMobileResize', (res: any) => {
            themeConfig.value.layout = res.layout;
            themeConfig.value.isDrawer = false;
            initSetLayoutChange();
            onMenuBarHighlightChange();
            themeConfig.value.isCollapse = false;
        });

        window.addEventListener('load', () => {
            // 刷新页面时，设置了值，直接取缓存中的值进行初始化
            setTimeout(() => {
                // 顶栏背景渐变
                if (getLocal('navbarsBgStyle') && themeConfig.value.isTopBarColorGradual) {
                    const breadcrumbIndexEl: any = document.querySelector('.layout-navbars-breadcrumb-index');
                    breadcrumbIndexEl.style.cssText = getLocal('navbarsBgStyle');
                }
                // 菜单背景渐变
                if (getLocal('asideBgStyle') && themeConfig.value.isMenuBarColorGradual) {
                    const asideEl: any = document.querySelector('.layout-container .el-aside');
                    asideEl.style.cssText = getLocal('asideBgStyle');
                }
                // 分栏菜单背景渐变
                if (getLocal('columnsBgStyle') && themeConfig.value.isColumnsMenuBarColorGradual) {
                    const asideEl: any = document.querySelector('.layout-container .layout-columns-aside');
                    asideEl.style.cssText = getLocal('columnsBgStyle');
                }
                // 菜单字体背景高亮
                if (getLocal('menuBarHighlightId') && themeConfig.value.isMenuBarColorHighlight) {
                    let els = document.querySelector('.el-menu-item.is-active');
                    if (!els) return false;
                    els.setAttribute('id', getLocal('menuBarHighlightId'));
                }
                // 灰色模式/色弱模式
                if (getLocal('appFilterStyle')) {
                    const appEl: any = document.querySelector('#app');
                    appEl.style.cssText = getLocal('appFilterStyle');
                }
                // // 语言国际化
                // if (getLocal('themeConfig')) proxy.$i18n.locale = getLocal('themeConfig').globalI18n;
            }, 100);
        });
    });
});
onUnmounted(() => {
    // 取消监听菜单点击，菜单字体背景高亮
    mittBus.off('onMenuClick');
    mittBus.off('layoutMobileResize');
});

defineExpose({ openDrawer });
</script>

<style scoped lang="scss">
.layout-breadcrumb-seting-bar {
    height: calc(100vh - 50px);
    padding: 0 15px;

    ::v-deep(.el-scrollbar__view) {
        overflow-x: hidden !important;
    }

    .layout-breadcrumb-seting-bar-flex {
        display: flex;
        align-items: center;

        &-label {
            flex: 1;
            color: #666666;
        }
    }

    .layout-drawer-content-flex {
        overflow: hidden;
        display: flex;
        flex-wrap: wrap;
        align-content: flex-start;
        margin: 0 -5px;

        .layout-drawer-content-item {
            width: 50%;
            height: 70px;
            cursor: pointer;
            border: 1px solid transparent;
            position: relative;
            padding: 5px;

            .el-container {
                height: 100%;

                .el-aside-dark {
                    background-color: #b3c0d1;
                }

                .el-aside {
                    background-color: #d3dce6;
                }

                .el-header {
                    background-color: #b3c0d1;
                }

                .el-main {
                    background-color: #e9eef3;
                }
            }

            .el-circular {
                border-radius: 2px;
                overflow: hidden;
                border: 1px solid transparent;
                transition: all 0.3s ease-in-out;
            }

            .drawer-layout-active {
                border: 1px solid;
                border-color: var(--el-color-primary);
            }

            .layout-tips-warp,
            .layout-tips-warp-active {
                transition: all 0.3s ease-in-out;
                position: absolute;
                left: 50%;
                top: 50%;
                transform: translate(-50%, -50%);
                border: 1px solid;
                border-color: var(--el-color-primary-light-4);
                border-radius: 100%;
                padding: 4px;

                .layout-tips-box {
                    transition: inherit;
                    width: 30px;
                    height: 30px;
                    z-index: 9;
                    border: 1px solid;
                    border-color: var(--el-color-primary-light-4);
                    border-radius: 100%;

                    .layout-tips-txt {
                        transition: inherit;
                        position: relative;
                        top: 5px;
                        font-size: 12px;
                        line-height: 1;
                        letter-spacing: 2px;
                        white-space: nowrap;
                        color: var(--el-color-primary-light-4);
                        text-align: center;
                        transform: rotate(30deg);
                        left: -1px;
                        background-color: #e9eef3;
                        width: 32px;
                        height: 17px;
                        line-height: 17px;
                    }
                }
            }

            .layout-tips-warp-active {
                border: 1px solid;
                border-color: var(--el-color-primary);

                .layout-tips-box {
                    border: 1px solid;
                    border-color: var(--el-color-primary);

                    .layout-tips-txt {
                        color: var(--el-color-primary) !important;
                        background-color: #e9eef3 !important;
                    }
                }
            }

            &:hover {
                .el-circular {
                    transition: all 0.3s ease-in-out;
                    border: 1px solid;
                    border-color: var(--el-color-primary);
                }

                .layout-tips-warp {
                    transition: all 0.3s ease-in-out;
                    border-color: var(--el-color-primary);

                    .layout-tips-box {
                        transition: inherit;
                        border-color: var(--el-color-primary);

                        .layout-tips-txt {
                            transition: inherit;
                            color: var(--el-color-primary) !important;
                            background-color: #e9eef3 !important;
                        }
                    }
                }
            }
        }
    }

    .copy-config {
        margin: 10px 0;

        .copy-config-btn {
            width: 100%;
            margin-top: 15px;
        }

        .copy-config-last-btn {
            margin: 10px 0 0;
        }
    }
}
</style>
