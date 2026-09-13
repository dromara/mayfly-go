<template>
    <div class="layout-breadcrumb-seting">
        <el-drawer :title="$t('layout.config.configTitle')" v-model="themeConfig.isDrawer" direction="rtl" destroy-on-close size="320px" @close="onDrawerClose">
            <el-scrollbar class="layout-breadcrumb-seting-bar">
                <!-- ssh终端主题 -->
                <el-divider content-position="left">{{ $t('layout.config.terminalTheme') }}</el-divider>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">{{ $t('layout.config.theme') }}</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-select v-model="themeConfig.terminalTheme" size="small" style="width: 140px">
                            <el-option v-for="(_, k) in themes" :key="k" :label="k" :value="k"> </el-option>
                            <el-option :label="$t('layout.config.custom')" value="custom"> </el-option>
                        </el-select>
                    </div>
                </div>
                <template v-if="themeConfig.terminalTheme == 'custom'">
                    <div class="layout-breadcrumb-seting-bar-flex mt-2!">
                        <div class="layout-breadcrumb-seting-bar-flex-label">{{ $t('layout.config.fontColor') }}</div>
                        <div class="layout-breadcrumb-seting-bar-flex-value">
                            <el-color-picker v-model="themeConfig.terminalForeground" size="small" @change="onColorPickerChange('terminalForeground')">
                            </el-color-picker>
                        </div>
                    </div>
                    <div class="layout-breadcrumb-seting-bar-flex">
                        <div class="layout-breadcrumb-seting-bar-flex-label">{{ $t('layout.config.backgroundColor') }}</div>
                        <div class="layout-breadcrumb-seting-bar-flex-value">
                            <el-color-picker v-model="themeConfig.terminalBackground" size="small" @change="onColorPickerChange('terminalBackground')">
                            </el-color-picker>
                        </div>
                    </div>
                    <div class="layout-breadcrumb-seting-bar-flex">
                        <div class="layout-breadcrumb-seting-bar-flex-label">{{ $t('layout.config.cursorColor') }}</div>
                        <div class="layout-breadcrumb-seting-bar-flex-value">
                            <el-color-picker v-model="themeConfig.terminalCursor" size="small" @change="onColorPickerChange('terminalCursor')">
                            </el-color-picker>
                        </div>
                    </div>
                </template>

                <div class="layout-breadcrumb-seting-bar-flex mt-2!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">{{ $t('layout.config.fontSize') }}</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-input-number v-model="themeConfig.terminalFontSize" controls-position="right" :min="12" :max="24" size="small" style="width: 90px">
                        </el-input-number>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt-2!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">{{ $t('layout.config.fontWeight') }}</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-select v-model="themeConfig.terminalFontWeight" size="small" style="width: 90px">
                            <el-option label="normal" value="normal"> </el-option>
                            <el-option label="bold" value="bold"> </el-option>
                        </el-select>
                    </div>
                </div>

                <el-divider content-position="left">{{ $t('layout.config.editorSetting') }}</el-divider>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">{{ $t('layout.config.theme') }}</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-select v-model="themeConfig.editorTheme" size="small" style="width: 130px">
                            <el-option label="vs" value="vs"> </el-option>
                            <el-option label="vs-dark" value="vs-dark"> </el-option>
                            <el-option label="SolarizedLight" value="SolarizedLight"> </el-option>
                            <el-option label="SolarizedDark" value="SolarizedDark"> </el-option>
                        </el-select>
                    </div>
                </div>

                <!-- 全局设置 -->
                <el-divider content-position="left">{{ $t('layout.config.globalSetting') }}</el-divider>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">{{ $t('layout.config.pagesize') }}</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-input-number
                            v-model="themeConfig.defaultListPageSize"
                            controls-position="right"
                            :min="10"
                            :max="50"
                            size="small"
                            style="width: 90px"
                        >
                        </el-input-number>
                    </div>
                </div>

                <!-- 全局主题 -->
                <el-divider content-position="left">{{ $t('layout.config.globalTheme') }}</el-divider>
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

                <!-- 菜单 -->
                <el-divider content-position="left">{{ $t('layout.config.menuSetting') }}</el-divider>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">{{ $t('layout.config.menuBar') }}</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.menuBar" size="small" @change="onBgColorPickerChange('menuBar')"> </el-color-picker>
                    </div>
                </div>

                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">
                        {{ $t('layout.config.menuBarFontColor') }}
                    </div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.menuBarColor" size="small" @change="onBgColorPickerChange('menuBarColor')"> </el-color-picker>
                    </div>
                </div>

                <div class="layout-breadcrumb-seting-bar-flex mt14">
                    <div class="layout-breadcrumb-seting-bar-flex-label">
                        {{ $t('layout.config.isMenuBarColorGradual') }}
                    </div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isMenuBarColorGradual" @change="onMenuBarGradualChange"></el-switch>
                    </div>
                </div>

                <!-- <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">顶栏背景</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.topBar" size="small"
                            @change="onBgColorPickerChange('topBar')">
                        </el-color-picker>
                    </div>
                </div>

                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">分栏菜单背景</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.columnsMenuBar" size="small"
                            @change="onBgColorPickerChange('columnsMenuBar')"> </el-color-picker>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">顶栏默认字体颜色</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.topBarColor" size="small"
                            @change="onBgColorPickerChange('topBarColor')">
                        </el-color-picker>
                    </div>
                </div>

                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">分栏菜单默认字体颜色</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-color-picker v-model="themeConfig.columnsMenuBarColor" size="small"
                            @change="onBgColorPickerChange('columnsMenuBarColor')">
                        </el-color-picker>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt-2!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">顶栏背景渐变</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isTopBarColorGradual"
                            @change="onTopBarGradualChange"></el-switch>
                    </div>
                </div>

                <div class="layout-breadcrumb-seting-bar-flex mt14">
                    <div class="layout-breadcrumb-seting-bar-flex-label">分栏菜单背景渐变</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isColumnsMenuBarColorGradual"
                            @change="onColumnsMenuBarGradualChange"></el-switch>
                    </div>
                </div> -->

                <!-- 界面设置 -->
                <el-divider content-position="left">{{ $t('layout.config.interfaceSetting') }}</el-divider>
                <div class="layout-breadcrumb-seting-bar-flex">
                    <div class="layout-breadcrumb-seting-bar-flex-label">
                        {{ $t('layout.config.isCollapse') }}
                    </div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isCollapse" @change="onThemeConfigChange"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">
                        {{ $t('layout.config.isUniqueOpened') }}
                    </div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isUniqueOpened"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">
                        {{ $t('layout.config.isFixedHeader') }}
                    </div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isFixedHeader" @change="onIsFixedHeaderChange"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5!" :style="{ opacity: themeConfig.layout !== 'classic' ? 0.5 : 1 }">
                    <div class="layout-breadcrumb-seting-bar-flex-label">
                        {{ $t('layout.config.isClassicSplitMenu') }}
                    </div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isClassicSplitMenu" :disabled="themeConfig.layout !== 'classic'" @change="onClassicSplitMenuChange">
                        </el-switch>
                    </div>
                </div>

                <!-- 界面显示 -->
                <el-divider content-position="left">{{ $t('layout.config.interfaceDisplay') }}</el-divider>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">{{ $t('layout.config.isShowLogo') }}</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isShowLogo" @change="onIsShowLogoChange"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5!" :style="{ opacity: themeConfig.layout === 'transverse' ? 0.5 : 1 }">
                    <div class="layout-breadcrumb-seting-bar-flex-label">
                        {{ $t('layout.config.isBreadcrumb') }}
                    </div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch
                            v-model="themeConfig.isBreadcrumb"
                            :disabled="themeConfig.layout === 'transverse'"
                            @change="onIsBreadcrumbChange"
                        ></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">
                        {{ $t('layout.config.isBreadcrumbIcon') }}
                    </div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isBreadcrumbIcon"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">{{ $t('layout.config.isTagsview') }}</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isTagsview"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">
                        {{ $t('layout.config.isTagsviewIcon') }}
                    </div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isTagsviewIcon"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">
                        {{ $t('layout.config.isCacheTagsView') }}
                    </div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isCacheTagsView"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">
                        {{ $t('layout.config.isSortableTagsView') }}
                    </div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isSortableTagsView"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">{{ $t('layout.config.isFooter') }}</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isFooter"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">{{ $t('layout.config.isGrayscale') }}</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isGrayscale" @change="onAddFilterChange('grayscale')"></el-switch>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">{{ $t('layout.config.isInvert') }}</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isInvert" @change="onAddFilterChange('invert')"></el-switch>
                    </div>
                </div>

                <!-- 液态玻璃态 -->
                <el-divider content-position="left">{{ $t('layout.config.glassMode') }}</el-divider>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">{{ $t('layout.config.glassMode') }}</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-switch v-model="themeConfig.isGlassMode" @change="onGlassModeChange"></el-switch>
                    </div>
                </div>

                <!-- 玻璃壁纸（内置渐变预设 + 自定义上传，效果仅在液态玻璃态可见） -->
                <el-divider content-position="left">{{ $t('layout.config.wallpaper') }}</el-divider>
                <div class="wallpaper-grid">
                    <div v-for="wp in wallpapers" :key="wp.id" class="wallpaper-cell" @click="onWallpaperSelect(wp.id)">
                        <div
                            class="wallpaper-thumb"
                            :class="{ 'is-active': themeConfig.glassWallpaper === wp.id }"
                            :style="{ background: `var(${wp.cssVar})` }"
                            :title="$t(wp.nameKey)"
                        >
                            <span v-if="themeConfig.glassWallpaper === wp.id" class="wallpaper-thumb__check">✓</span>
                        </div>
                        <span class="wallpaper-name" :class="{ 'is-active': themeConfig.glassWallpaper === wp.id }">{{ $t(wp.nameKey) }}</span>
                    </div>
                    <div class="wallpaper-cell" @click="onCustomWallpaperClick">
                        <div
                            class="wallpaper-thumb is-custom"
                            :class="{ 'is-active': themeConfig.glassWallpaper === 'custom' }"
                            :style="themeConfig.bgImage ? { backgroundImage: `url(${themeConfig.bgImage})` } : undefined"
                            :title="$t('layout.config.wallpaperCustom')"
                        >
                            <span v-if="!themeConfig.bgImage" class="wallpaper-thumb__plus">+</span>
                            <span v-if="themeConfig.glassWallpaper === 'custom'" class="wallpaper-thumb__check">✓</span>
                        </div>
                        <span class="wallpaper-name" :class="{ 'is-active': themeConfig.glassWallpaper === 'custom' }">{{ $t('layout.config.wallpaperCustom') }}</span>
                    </div>
                </div>
                <!-- 隐藏上传器：由「自定义」瓦片 / 更换按钮程序化触发 -->
                <el-upload ref="bgUploadRef" class="wallpaper-upload-hidden" :auto-upload="false" :show-file-list="false" accept="image/*" :on-change="onBgImageUpload">
                    <span></span>
                </el-upload>
                <template v-if="themeConfig.glassWallpaper === 'custom' && themeConfig.bgImage">
                    <!-- 液态玻璃态下壁纸图层钉死"清晰+全不透明"(见 backdrop.scss), 模糊/不透明度滑杆已移除:
                         其缓存值曾把源图糊化/半透明化, 造成整屏底雾且下游任何材质调整都看不出效果 -->
                    <div class="wallpaper-custom-actions mt-3.5! mb-5.5!">
                        <el-button size="small" @click="triggerUpload">{{ $t('layout.config.bgImageReplace') }}</el-button>
                        <el-button size="small" text type="danger" @click="onBgImageClear">{{ $t('layout.config.bgImageClear') }}</el-button>
                    </div>
                </template>
                <div v-else class="mb-3.5!"></div>

                <!-- 磨砂程度：0=壁纸纹理完全清晰(默认零磨砂档)，调大则整屏壁纸纱层统一糊化(文字仍锐利) -->
                <div class="mb-5.5!">
                    <div class="mb-1! flex items-center justify-between">
                        <span class="text-13px">{{ $t('layout.config.glassFrost') }}</span>
                        <span class="text-12px opacity-50">
                            {{ themeConfig.glassFrost === 0 ? $t('layout.config.glassFrostClear') : `${themeConfig.glassFrost}px` }}
                        </span>
                    </div>
                    <el-slider v-model="themeConfig.glassFrost" :min="0" :max="40" :step="2" :show-tooltip="false" @change="onFrostChange" />
                </div>

                <!-- 其它设置 -->
                <el-divider content-position="left">{{ $t('layout.config.otherSetting') }}</el-divider>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">{{ $t('layout.config.animation') }}</div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-select v-model="themeConfig.animation" size="small" style="width: 90px">
                            <el-option label="slide-right" value="slide-right"></el-option>
                            <el-option label="slide-left" value="slide-left"></el-option>
                            <el-option label="opacitys" value="opacitys"></el-option>
                        </el-select>
                    </div>
                </div>
                <div class="layout-breadcrumb-seting-bar-flex mt-3.5! mb-5.5!">
                    <div class="layout-breadcrumb-seting-bar-flex-label">
                        {{ $t('layout.config.columnsAsideStyle') }}
                    </div>
                    <div class="layout-breadcrumb-seting-bar-flex-value">
                        <el-select v-model="themeConfig.columnsAsideStyle" placeholder="请选择" size="small" style="width: 90px">
                            <el-option label="圆角" value="columns-round"></el-option>
                            <el-option label="卡片" value="columns-card"></el-option>
                        </el-select>
                    </div>
                </div>

                <!-- 布局切换 -->
                <el-divider content-position="left">{{ $t('layout.config.layoutSwitch') }}</el-divider>
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
                                <p class="layout-tips-txt">{{ $t('layout.config.defaults') }}</p>
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
                                <p class="layout-tips-txt">{{ $t('layout.config.classic') }}</p>
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
                                <p class="layout-tips-txt">{{ $t('layout.config.transverse') }}</p>
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
                                <p class="layout-tips-txt">{{ $t('layout.config.columns') }}</p>
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
import { getLocal, setLocal } from '@/common/utils/storage';
import { getLightColor } from '@/common/utils/theme';
import { useThemeConfig } from '@/store/themeConfig';
import { storeToRefs } from 'pinia';
import { nextTick, onMounted, ref, watch } from 'vue';

import themes from '@/components/terminal/themes';
import { useWindowSize } from '@vueuse/core';

const copyConfigBtnRef = ref();
const bgUploadRef = ref();
const { themeConfig } = storeToRefs(useThemeConfig());
const themeConfigStore = useThemeConfig();

// 获取窗口大小
const { width } = useWindowSize();

watch(width, () => {
    checkClientWidth();
});

onMounted(() => {
    nextTick(() => {
        checkClientWidth();
    });
});

// 主题颜色类配置键（均为字符串值）
type ThemeColorKey = 'primary' | 'success' | 'info' | 'warning' | 'danger' | 'terminalForeground' | 'terminalBackground' | 'terminalCursor';
// 背景颜色类配置键（均为字符串值）
type ThemeBgColorKey = 'menuBar' | 'menuBarColor' | 'topBar' | 'columnsMenuBar' | 'topBarColor' | 'columnsMenuBarColor';

// 1、全局主题
const onColorPickerChange = (color: ThemeColorKey) => {
    setPropertyFun(`--color-${color}`, themeConfig.value[color]);
    setDispatchThemeConfig();
};
// 1、全局主题设置函数
const setPropertyFun = (color: string, targetVal: string) => {
    document.documentElement.style.setProperty(color, targetVal);
    for (let i = 1; i <= 9; i++) {
        document.documentElement.style.setProperty(`${color}-light-${i}`, getLightColor(targetVal, i / 10));
    }
};
// 2、菜单 / 顶栏
const onBgColorPickerChange = (bg: ThemeBgColorKey) => {
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

        const elNavbars = document.querySelector('.layout-navbars-breadcrumb-index') as HTMLElement | null;
        const elAside = document.querySelector('.layout-container .el-aside') as HTMLElement | null;
        const elColumns = document.querySelector('.layout-container .layout-columns-aside') as HTMLElement | null;
        if (elNavbars) setLocal('navbarsBgStyle', elNavbars.style.cssText);
        if (elAside) setLocal('asideBgStyle', elAside.style.cssText);
        if (elColumns) setLocal('columnsBgStyle', elColumns.style.cssText);
    });
};

// 3、界面设置 --> 菜单水平折叠
const onThemeConfigChange = () => {
    setDispatchThemeConfig();
};
// 3、界面设置 --> 固定 Header
const onIsFixedHeaderChange = () => {
    themeConfig.value.isFixedHeaderChange = themeConfig.value.isFixedHeader ? false : true;
};
// 3、界面设置 --> 经典布局分割菜单
const onClassicSplitMenuChange = () => {
    themeConfig.value.isBreadcrumb = false;
};
// 4、界面显示 --> 侧边栏 Logo
const onIsShowLogoChange = () => {
    themeConfig.value.isShowLogoChange = themeConfig.value.isShowLogo ? false : true;
};
// 4、界面显示 --> 面包屑 Breadcrumb
const onIsBreadcrumbChange = () => {
    if (themeConfig.value.layout === 'classic') {
        themeConfig.value.isClassicSplitMenu = false;
    }
};

// 4、界面显示 --> 暗模式/灰色模式/色弱模式
const onAddFilterChange = (attr: string) => {
    if (attr === 'grayscale') {
        if (themeConfig.value.isGrayscale) themeConfig.value.isInvert = false;
    } else {
        if (themeConfig.value.isInvert) themeConfig.value.isGrayscale = false;
    }
    const cssAttr = attr === 'grayscale' ? `grayscale(${themeConfig.value.isGrayscale ? 1 : 0})` : `invert(${themeConfig.value.isInvert ? '80%' : '0%'})`;
    const appEle = document.querySelector('#app') as HTMLElement | null;
    if (!appEle) return;
    appEle.setAttribute('style', `filter: ${cssAttr}`);

    setLocal('appFilterStyle', appEle.style.cssText);
};

// 液态玻璃态模式切换
const onGlassModeChange = () => {
    themeConfigStore.toggleGlassMode(themeConfig.value.isGlassMode);
    setDispatchThemeConfig();
};

// 内置玻璃壁纸预设（渐变定义见 theme/glass/wallpaper.scss，缩略图直接复用同名 CSS 变量）
const wallpapers = [
    { id: 'none', nameKey: 'layout.config.wallpaperSpots', cssVar: '--wallpaper-spots' },
    { id: 'aurora', nameKey: 'layout.config.wallpaperAurora', cssVar: '--wallpaper-aurora' },
    { id: 'dusk', nameKey: 'layout.config.wallpaperDusk', cssVar: '--wallpaper-dusk' },
    { id: 'reef', nameKey: 'layout.config.wallpaperReef', cssVar: '--wallpaper-reef' },
    { id: 'bloom', nameKey: 'layout.config.wallpaperBloom', cssVar: '--wallpaper-bloom' },
    { id: 'nebula', nameKey: 'layout.config.wallpaperNebula', cssVar: '--wallpaper-nebula' },
];

// 选择内置壁纸
const onWallpaperSelect = (id: string) => {
    themeConfigStore.setGlassWallpaper(id);
    setDispatchThemeConfig();
};

// 磨砂程度滑杆: 0=纹理清晰(默认零磨砂), store 内联覆盖 --glass-fx-chrome/panel/content 档位
const onFrostChange = (v: number | number[]) => {
    themeConfigStore.setGlassFrost(Array.isArray(v) ? v[0] : v);
    setDispatchThemeConfig();
};
// 触发隐藏的上传器
const triggerUpload = () => {
    const input = (bgUploadRef.value?.$el as HTMLElement | undefined)?.querySelector('input[type="file"]') as HTMLInputElement | null;
    input?.click();
};
// 点击「自定义」瓦片：已有图片则直接应用，否则打开文件选择
const onCustomWallpaperClick = () => {
    if (themeConfig.value.bgImage) {
        themeConfigStore.setGlassWallpaper('custom');
        setDispatchThemeConfig();
    } else {
        triggerUpload();
    }
};

// 壁纸 base64 字符预算：实测 Chrome 对内联自定义属性值有 ~2Mi(2,097,152)字符的静默丢弃
// 上限(setProperty 超限不报错、变量直接消失 —— 正是"替换壁纸完全没反应"的根因)，
// 同时留足 localStorage 配额余量(单键 5MB 总量)。1.4M 字符 ≈ 1MB 二进制，壁纸素材绰绰有余
const WALLPAPER_BUDGET = 1_400_000;

// 按给定尺寸/质量编码一次(优先 webp, 不支持则 jpeg)
const encodeWallpaper = (img: HTMLImageElement, maxEdge: number, quality: number): string => {
    const scale = Math.min(1, maxEdge / Math.max(img.width, img.height));
    const canvas = document.createElement('canvas');
    canvas.width = Math.max(1, Math.round(img.width * scale));
    canvas.height = Math.max(1, Math.round(img.height * scale));
    const ctx = canvas.getContext('2d');
    if (!ctx) return '';
    ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
    const webp = canvas.toDataURL('image/webp', quality);
    return webp.startsWith('data:image/webp') ? webp : canvas.toDataURL('image/jpeg', quality);
};

// 壁纸压缩(字节预算制)：高熵素材(噪点/细节密集照片)在固定 2560+q0.82 下仍可产出 2M+ 字符
// 触顶被丢弃, 必须逐级降分辨率+质量直到落进预算。玻璃壁纸经 blur 采样, 1280px 底档无明显画质损失
const compressWallpaper = (file: File): Promise<string> =>
    new Promise((resolve, reject) => {
        const objectUrl = URL.createObjectURL(file);
        const img = new Image();
        img.onload = () => {
            URL.revokeObjectURL(objectUrl);
            for (const [edge, q] of [[2560, 0.82], [1920, 0.72], [1440, 0.62], [1280, 0.5]] as const) {
                const dataUrl = encodeWallpaper(img, edge, q);
                if (dataUrl && dataUrl.length <= WALLPAPER_BUDGET) {
                    resolve(dataUrl);
                    return;
                }
            }
            // 终极兜底：最低档直接用(再小的编码产出也远超照片壁纸需求)
            const fallback = encodeWallpaper(img, 1080, 0.45);
            fallback ? resolve(fallback) : reject(new Error('wallpaper encode failed'));
        };
        img.onerror = () => {
            URL.revokeObjectURL(objectUrl);
            reject(new Error('image decode failed'));
        };
        img.src = objectUrl;
    });

const readFileAsDataUrl = (file: File): Promise<string> =>
    new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = (e) => resolve(e.target?.result as string);
        reader.onerror = () => reject(reader.error);
        reader.readAsDataURL(file);
    });

// 背景图上传：统一走字节预算编码链路；gif/svg 在预算内直存(保留动画/矢量)，超预算同样重编码
const onBgImageUpload = async (uploadFile: any) => {
    const file = uploadFile.raw;
    if (!file) return;
    let dataUrl = '';
    try {
        const raw = await readFileAsDataUrl(file);
        const keepNative = (file.type === 'image/gif' || file.type === 'image/svg+xml') && raw.length <= WALLPAPER_BUDGET;
        dataUrl = keepNative ? raw : await compressWallpaper(file);
    } catch {
        // 兜底：canvas 链路异常(gif 解码失败/画布不可用等)时回退原文件直读，能否生效取决于体积
        try {
            dataUrl = await readFileAsDataUrl(file);
        } catch {
            return;
        }
    }
    themeConfigStore.setBgImage(dataUrl);
    themeConfigStore.setGlassWallpaper('custom');
    setDispatchThemeConfig();
};
const onBgImageClear = () => {
    themeConfigStore.setBgImage('');
    themeConfigStore.setGlassWallpaper('none');
    setDispatchThemeConfig();
};

// 5、布局切换
const onSetLayout = (layout: string) => {
    setLocal('oldLayout', layout);
    if (themeConfig.value.layout === layout) {
        return;
    }
    themeConfig.value.layout = layout;
    themeConfig.value.isDrawer = false;
    initSetLayoutChange();
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
        themeConfig.value.isClassicSplitMenu = false;
    } else if (themeConfig.value.layout === 'columns') {
        themeConfig.value.isShowLogo = true;
        themeConfig.value.isBreadcrumb = true;
        themeConfig.value.isCollapse = false;
        themeConfig.value.isClassicSplitMenu = false;
    } else {
        themeConfig.value.isShowLogo = false;
        themeConfig.value.isBreadcrumb = true;
        themeConfig.value.isCollapse = false;
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
};

// 触发 store 布局配置更新
const setDispatchThemeConfig = () => {
    setLocalThemeConfigStyle();
};

// 存储布局配置全局主题样式（html根标签）
const setLocalThemeConfigStyle = () => {
    // 剔除 --backdrop-image：壁纸 base64 与 themeConfig.bgImage 存的是同一份数据，
    // 双份直塞会把该键撑爆配额导致整个写入失败；init 会从 bgImage 重新注入 DOM 变量
    const cssText = document.documentElement.style.cssText.replace(/--backdrop-image:\s*(?:url\("[^"]*"\)|none)\s*;?/g, '');
    setLocal('themeConfigStyle', cssText);
};
// 一键复制配置
const onCopyConfigClick = () => {
    let copyThemeConfig = getLocal('themeConfig');
    copyThemeConfig.isDrawer = false;
};

const checkClientWidth = () => {
    let oldLayout = getLocal('oldLayout');
    if (!oldLayout) {
        oldLayout = themeConfig.value.layout;
        setLocal('oldLayout', themeConfig.value.layout);
    }
    if (width.value < 1000) {
        themeConfig.value.isCollapse = false;
        themeConfig.value.layout = 'defaults';
    } else {
        themeConfig.value.layout = oldLayout ? oldLayout : 'defaults';
    }

    themeConfig.value.isDrawer = false;
    initSetLayoutChange();
    themeConfig.value.isCollapse = false;
};
</script>

<style scoped lang="scss">
::v-deep(.el-drawer) {
    --el-drawer-padding-primary: unset !important;

    .el-drawer__header {
        padding: 0 15px !important;
        height: 50px;
        display: flex;
        align-items: center;
        margin-bottom: 0 !important;
        border-bottom: 1px solid var(--el-border-color);
    }

    .el-drawer__body {
        width: 100%;
        height: 100%;
        overflow: auto;
    }
}

.layout-breadcrumb-seting-bar {
    height: calc(100vh - 50px);

    ::v-deep(.el-scrollbar__view) {
        // padding 必须挂在 view 上(而非 bar 根): overflow-x 裁剪发生在 view 的 padding box 边缘,
        // 挂根上时裁剪线贴着内容左缘, 滑杆按钮(左探 10px)/激活缩略图高亮环会被裁掉一半
        padding: 0 15px;
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

    .wallpaper-grid {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 8px;
        margin-top: 4px;
    }

    .wallpaper-cell {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 4px;
        cursor: pointer;
    }

    .wallpaper-thumb {
        position: relative;
        width: 100%;
        height: 44px;
        border-radius: 8px;
        background-size: cover;
        background-position: center;
        border: 1px solid var(--el-border-color);
        transition:
            box-shadow 0.2s ease,
            transform 0.2s ease;

        &.is-custom {
            display: flex;
            align-items: center;
            justify-content: center;
            color: var(--el-text-color-secondary);
            background-color: var(--el-fill-color-lighter);
            border-style: dashed;
        }

        &.is-active {
            box-shadow: 0 0 0 2px var(--el-color-primary);
            transform: scale(1.04);
        }

        &__check {
            position: absolute;
            top: -6px;
            right: -6px;
            width: 16px;
            height: 16px;
            font-size: 10px;
            line-height: 16px;
            color: #fff;
            text-align: center;
            background: var(--el-color-primary);
            border-radius: 50%;
        }

        &__plus {
            font-size: 18px;
            line-height: 1;
        }
    }

    .wallpaper-name {
        font-size: 11px;
        line-height: 1;
        color: var(--el-text-color-secondary);

        &.is-active {
            font-weight: 500;
            color: var(--el-color-primary);
        }
    }

    .wallpaper-upload-hidden {
        display: none;
    }

    .wallpaper-custom-actions {
        display: flex;
        gap: 8px;
        align-items: center;
    }
}
</style>
