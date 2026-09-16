import { getServerConf, getSysStyleConfig } from '@/common/sysconfig';
import { formatDate } from '@/common/utils/format';
import { getLocal, getThemeConfig } from '@/common/utils/storage';
import { useUserInfo } from '@/store/userInfo';
import { defineStore } from 'pinia';

// 系统默认logo图标，对应于@/assets/image/logo.svg
const logoIcon =
    'data:image/svg+xml;charset=utf-8;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBzdGFuZGFsb25lPSJubyI/Pgo8IURPQ1RZUEUgc3ZnIFBVQkxJQyAiLS8vVzNDLy9EVEQgU1ZHIDEuMS8vRU4iICJodHRwOi8vd3d3LnczLm9yZy9HcmFwaGljcy9TVkcvMS4xL0RURC9zdmcxMS5kdGQiPgo8c3ZnIHQ9IjE2MjE4NTkwMDk2MDUiIGNsYXNzPSJpY29uIiB2aWV3Qm94PSIwIDAgMTAyNCAxMDI0IiB2ZXJzaW9uPSIxLjEiIAogICAgIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgcC1pZD0iOTcwOSIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiIAogICAgIHdpZHRoPSIyMDAiIGhlaWdodD0iMjAwIj4KICAgICA8ZGVmcz48c3R5bGUgdHlwZT0idGV4dC9jc3MiPjwvc3R5bGU+PC9kZWZzPgogICAgIDxwYXRoIGQ9Ik04MjAuMjAzOTIyIDgxMi4xNzI1NDlINjg0LjY3NDUxdi00NS4xNzY0NzFoMTEyLjQzOTIxNVYyNzkuMDkwMTk2SDYzMy40NzQ1MWwtODUuMzMzMzM0IDI3Ny4wODIzNTNjLTMuMDExNzY1IDEwLjAzOTIxNi0xMi4wNDcwNTkgMTYuMDYyNzQ1LTIyLjA4NjI3NCAxNi4wNjI3NDUtMTAuMDM5MjE2IDAtMTkuMDc0NTEtNy4wMjc0NTEtMjEuMDgyMzUzLTE3LjA2NjY2N2wtNzEuMjc4NDMxLTI4MC4wOTQxMTdoLTE4MC43MDU4ODNWNzYyLjk4MDM5MmgxMjAuNDcwNTg5djQ1LjE3NjQ3MUgyMjkuODk4MDM5Yy0xMi4wNDcwNTkgMC0yMi4wODYyNzUtMTAuMDM5MjE2LTIyLjA4NjI3NC0yMi4wODYyNzVWMjUyLjk4ODIzNWMwLTEyLjA0NzA1OSAxMC4wMzkyMTYtMjIuMDg2Mjc1IDIyLjA4NjI3NC0yMi4wODYyNzRINDUxLjc2NDcwNmMxMC4wMzkyMTYgMCAxOS4wNzQ1MSA3LjAyNzQ1MSAyMi4wODYyNzQgMTcuMDY2NjY2bDU1LjIxNTY4NyAyMTguODU0OTAyTDU5NS4zMjU0OSAyNTAuOTgwMzkyYzMuMDExNzY1LTkuMDM1Mjk0IDEyLjA0NzA1OS0xNi4wNjI3NDUgMjEuMDgyMzUzLTE2LjA2Mjc0NWgyMDIuNzkyMTU3YzEyLjA0NzA1OSAwIDIyLjA4NjI3NSAxMC4wMzkyMTYgMjIuMDg2Mjc1IDIyLjA4NjI3NXY1MzMuMDgyMzUzYzEuMDAzOTIyIDEyLjA0NzA1OS05LjAzNTI5NCAyMi4wODYyNzUtMjEuMDgyMzUzIDIyLjA4NjI3NHogbTAgMCIgZmlsbD0iI2UyNTgxMyIgcC1pZD0iOTcxMCIgc3Ryb2tlLXdpZHRoPSIzMCIgc3Ryb2tlPSIjZTI1ODEzIj48L3BhdGg+CiAgICAgPHBhdGggZD0iTTczMS44NTg4MjQgNDI1LjY2Mjc0NWM0LjAxNTY4Ni0xMi4wNDcwNTktMi4wMDc4NDMtMjUuMDk4MDM5LTE0LjA1NDkwMi0yOS4xMTM3MjUtMTIuMDQ3MDU5LTQuMDE1Njg2LTI1LjA5ODAzOSAyLjAwNzg0My0yOS4xMTM3MjYgMTQuMDU0OTAyTDU2My4yIDc2Ni45OTYwNzhoLTczLjI4NjI3NUwzNzEuNDUwOTggNDEwLjYwMzkyMmMtNC4wMTU2ODYtMTIuMDQ3MDU5LTE3LjA2NjY2Ny0xOC4wNzA1ODgtMjguMTA5ODA0LTE0LjA1NDkwMi0xMi4wNDcwNTkgNC4wMTU2ODYtMTguMDcwNTg4IDE3LjA2NjY2Ny0xNC4wNTQ5MDEgMjguMTA5ODA0bDEyMy40ODIzNTIgMzcxLjQ1MDk4YzMuMDExNzY1IDkuMDM1Mjk0IDEyLjA0NzA1OSAxNS4wNTg4MjQgMjEuMDgyMzUzIDE1LjA1ODgyM2g3Mi4yODIzNTNsLTUzLjIwNzg0MyAxNjAuNjI3NDUxIDQ2LjE4MDM5MiAyLjAwNzg0NCAxOTIuNzUyOTQyLTU0OC4xNDExNzd6IiBmaWxsPSIjMmMyYzJjIiBwLWlkPSI5NzExIiBzdHJva2Utd2lkdGg9IjMwIiBzdHJva2U9IiMyYzJjMmMiPjwvcGF0aD4KPC9zdmc+';

// Chrome 对内联自定义属性值有 ~2Mi 字符的静默丢弃上限(实测 setProperty 超限不报错、变量直接消失)。
// 历史缓存的大图/兜底直读路径超限时显式不写壁纸变量, 行为可预期(重新上传会走预算压缩链路恢复)
const BACKDROP_DOM_LIMIT = 2_000_000;
function applyBackdropImageVar(url: string) {
    const value = url && url.length <= BACKDROP_DOM_LIMIT ? `url("${url}")` : 'none';
    document.documentElement.style.setProperty('--backdrop-image', value);
}

export const useThemeConfig = defineStore('themeConfig', {
    state: (): ThemeConfigState => ({
        themeConfig: {
            // 是否开启布局配置抽屉
            isDrawer: false,

            /* 全局主题
            ------------------------------- */
            // 默认 primary 颜色，请注意：需要同时修改 `/@/theme/common/var.scss` 对应的值
            primary: '#409eff',
            // 默认 success 颜色，请注意：需要同时修改 `/@/theme/common/var.scss` 对应的值
            success: '#67c23a',
            // 默认 info 颜色，请注意：需要同时修改 `/@/theme/common/var.scss` 对应的值
            info: '#909399',
            // 默认 warning 颜色，请注意：需要同时修改 `/@/theme/common/var.scss` 对应的值
            warning: '#e6a23c',
            // 默认 danger 颜色，请注意：需要同时修改 `/@/theme/common/var.scss` 对应的值
            danger: '#f56c6c',

            /* 菜单 / 顶栏
            ------------------------------- */
            // 默认顶栏导航背景颜色，请注意：需要同时修改 `/@/theme/common/var.scss` 对应的值
            topBar: '#ffffff',
            // 默认菜单导航背景颜色，请注意：需要同时修改 `/@/theme/common/var.scss` 对应的值
            menuBar: '#FFFFFF',
            // 默认分栏菜单背景颜色，请注意：需要同时修改 `/@/theme/common/var.scss` 对应的值
            columnsMenuBar: '#545c64',
            // 默认顶栏导航字体颜色，请注意：需要同时修改 `/@/theme/common/var.scss` 对应的值
            topBarColor: '#606266',
            // 默认菜单导航字体颜色，请注意：需要同时修改 `/@/theme/common/var.scss` 对应的值
            menuBarColor: '#606266',
            // 默认分栏菜单字体颜色，请注意：需要同时修改 `/@/theme/common/var.scss` 对应的值
            columnsMenuBarColor: '#e6e6e6',
            // 是否开启顶栏背景颜色渐变
            isTopBarColorGradual: false,
            // 是否开启菜单背景颜色渐变
            isMenuBarColorGradual: false,
            // 是否开启分栏菜单背景颜色渐变
            isColumnsMenuBarColorGradual: false,
            // 是否开启菜单字体背景高亮
            isMenuBarColorHighlight: false,
            // 是否开启菜单字体背景高亮

            /* 界面设置
            ------------------------------- */
            // 是否开启菜单水平折叠效果
            isCollapse: false,
            // 是否开启菜单手风琴效果
            isUniqueOpened: false,
            // 是否开启固定 Header
            isFixedHeader: false,
            // 初始化变量，用于更新菜单 el-scrollbar 的高度，请勿删除
            isFixedHeaderChange: false,
            // 是否开启经典布局分割菜单（仅经典布局生效）
            isClassicSplitMenu: false,

            /* 界面显示
            ------------------------------- */
            // 是否开启侧边栏 Logo
            isShowLogo: true,
            // 初始化变量，用于 el-scrollbar 的高度更新，请勿删除
            isShowLogoChange: true,
            // 是否开启 Breadcrumb
            isBreadcrumb: true,
            // 是否开启 Tagsview
            isTagsview: true,
            isShareTagsView: false,
            // 是否开启 Breadcrumb 图标
            isBreadcrumbIcon: true,
            // 是否开启 Tagsview 图标
            isTagsviewIcon: true,
            // 是否开启 TagsView 缓存
            isCacheTagsView: true,
            // 是否开启 TagsView 拖拽
            isSortableTagsView: true,
            // 是否开启 Footer 底部版权信息
            isFooter: false,
            // 是否暗模式
            isDark: false,
            // 是否开启灰色模式
            isGrayscale: false,
            // 是否开启色弱模式
            isInvert: false,
            // 是否开启水印
            isWatermark: false,
            // 水印文案数组，0->用户信息  1->当前时间 2->额外信息
            watermarkText: ['', '', ''],

            /* 液态玻璃态
            ------------------------------- */
            // 是否开启液态玻璃态模式
            isGlassMode: false,
            // 玻璃壁纸：none(默认动态光斑) | aurora | dusk | reef | bloom | nebula | custom(上传图)
            glassWallpaper: 'none',
            // 磨砂程度(blur px, 0-40): 0=纹理完全清晰(默认), 驱动 chrome/panel/content 三档 blur
            glassFrost: 0,
            // 背景图 URL 或 base64（glassWallpaper='custom' 时生效）
            bgImage: '',
            // 背景图模糊程度 (0-20px)
            bgImageBlur: 0,
            // 背景图不透明度 (0-100)
            bgImageOpacity: 100,

            /* 其它设置
            ------------------------------- */
            // 默认主页面切换动画，可选 1、 slide-right 2、 slide-left 3、 opacitys
            animation: 'slide-right',
            // 默认分栏高亮风格，可选 1、 圆角 columns-round 2、 卡片 columns-card
            columnsAsideStyle: 'columns-round',

            /* 布局切换
            ------------------------------- */
            // 默认布局，可选 1、默认 defaults 2、经典 classic 3、横向 transverse 4、分栏 columns
            layout: 'transverse',

            terminalTheme: 'light',
            // ssh终端字体颜色
            terminalForeground: '#C5C8C6',
            // ssh终端背景色
            terminalBackground: '#121212',
            // ssh终端cursor色
            terminalCursor: '#F0CC09',
            terminalFontSize: 14,
            terminalFontWeight: 'bold',

            // 编辑器主题
            editorTheme: 'vs',

            /* 后端控制路由
            ------------------------------- */
            // 是否开启后端控制路由
            isRequestRoutes: true,

            /* 全局网站标题 / 副标题
            ------------------------------- */
            // 网站主标题（菜单导航、浏览器当前网页标题）
            globalTitle: 'mayfly',
            // 网站副标题（登录页顶部文字）
            globalViceTitle: 'mayfly-go',
            appSlogan: 'common.appSlogan',
            // 网站logo icon, base64编码内容
            logoIcon: logoIcon,
            version: 'latest',
            // 默认初始语言，可选值"<zh-cn|en|zh-tw>"，默认 zh-cn
            globalI18n: 'zh-cn',
            // 默认全局组件大小，可选值"<|large|default|small>"，默认 ''
            globalComponentSize: '',

            /** 全局设置 */
            // 默认列表页的分页大小
            defaultListPageSize: 10,
        },
    }),
    actions: {
        initThemeConfig() {
            // 获取缓存中的布局配置
            const tc = getThemeConfig();

            if (tc) {
                // 与默认值合并而非整体替换：缓存是历史快照，新增的配置项在旧缓存里缺席，
                // 直接赋值会让这些字段变成 undefined（曾导致 ElSwitch 的 model-value 校验告警）
                // isDrawer 强制为 false：缓存中可能残留上次会话的 drawer 开启状态，导致 mount 时短暂打开再关闭触发告警
                this.themeConfig = { ...this.themeConfig, ...tc, isDrawer: false };
                document.documentElement.style.cssText = getLocal('themeConfigStyle') || '';

                // 恢复液态玻璃态模式
                if (this.themeConfig.isGlassMode) {
                    document.body.classList.add('glass-mode');
                    this._applyGlassVariables();
                }

                // 恢复背景图（玻璃态壁纸图层钉死清晰全不透明, 见 backdrop.scss; 不再恢复 blur/opacity 滑杆值）
                if (this.themeConfig.bgImage) {
                    applyBackdropImageVar(this.themeConfig.bgImage);
                }

                // 恢复玻璃壁纸（旧缓存无该字段时视为 none，向后兼容）
                const wallpaper = this.themeConfig.glassWallpaper || 'none';
                if (wallpaper !== 'none') {
                    document.documentElement.setAttribute('data-wallpaper', wallpaper);
                }
            }

            getServerConf().then((res) => {
                this.themeConfig.globalI18n = res.i18n;
                this.themeConfig.version = res.version;
            });

            this.themeConfig.defaultListPageSize = calculatePageSizeByScreenHeight();

            // 根据后台系统配置初始化
            getSysStyleConfig().then((res) => {
                if (res?.title) {
                    this.themeConfig.globalTitle = res.title;
                }
                if (res?.viceTitle) {
                    this.themeConfig.globalViceTitle = res.viceTitle;
                }
                if (res?.logoIcon) {
                    this.themeConfig.logoIcon = res.logoIcon;
                }

                this.themeConfig.watermarkText = [];
                this.themeConfig.isWatermark = res?.useWatermark ?? false;
                if (!res?.useWatermark) {
                    return;
                }
                // 索引2为用户自定义水印信息
                this.themeConfig.watermarkText[2] = res.watermarkContent || '';
            });
        },
        // 设置水印用户信息
        setWatermarkUser(del: boolean = false) {
            const userinfo = useUserInfo().userInfo;
            let desc = '';
            if (!del && userinfo && userinfo.username) {
                desc = `${userinfo.username}(${userinfo.name})`;
            }
            this.themeConfig.watermarkText[0] = desc;
        },
        // 设置水印时间为当前时间
        setWatermarkNowTime() {
            this.themeConfig.watermarkText[1] = formatDate(new Date());
        },
        // 切换暗黑模式
        switchDark(isDark: boolean) {
            this.themeConfig.isDark = isDark;
            // 切换编辑器主题
            if (isDark) {
                this.themeConfig.editorTheme = 'vs-dark';
            } else {
                this.themeConfig.editorTheme = 'vs';
            }
            // 如果终端主题不是自定义主题，则切换主题
            if (this.themeConfig.terminalTheme != 'custom') {
                if (isDark) {
                    this.themeConfig.terminalTheme = 'dark';
                } else {
                    this.themeConfig.terminalTheme = 'light';
                }
            }
            // 玻璃模式：切换明暗后重新计算玻璃变量（亮/暗基底色不同）
            if (this.themeConfig.isGlassMode) {
                this._applyGlassVariables();
            }
        },
        // 切换液态玻璃态模式
        toggleGlassMode(enabled: boolean) {
            this.themeConfig.isGlassMode = enabled;
            if (enabled) {
                document.body.classList.add('glass-mode');
                this._applyGlassVariables();
            } else {
                document.body.classList.remove('glass-mode');
                this._removeGlassVariables();
            }
        },
        // 玻璃模式：覆盖 CSS 变量为半透明值（解决 scoped 样式优先级问题）
        // 亮/暗模式使用不同基底色：暗模式必须用深色半透明面板，否则白面板会盖住暗背景并与浅色文字冲突
        _applyGlassVariables() {
            const el = document.documentElement;
            const set = (k: string, v: string) => el.style.setProperty(k, v);

            // 磨砂程度滑杆(glassFrost, 0-40px): 驱动 .app-backdrop__frost 全局纱层 ——
            // 整屏统一糊壁纸(资源树等"裸铺"页同样生效), 面板在纱之上、文字永远锐利;
            // 不覆盖 chrome/panel/content 档 fx(那会造成纱+面板双重叠糊)。0 档移除变量回零成本
            const frost = Number(this.themeConfig.glassFrost) || 0;
            if (frost > 0) {
                set('--glass-frost-fx', `blur(${frost}px) saturate(1.5)`);
                // 白纱随 blur 提浓(糊而不灰: blur 抹掉细节后需底色维持明度); 暗态用深纱
                const veil = Math.min(0.18 + frost * 0.008, 0.5);
                set('--glass-frost-veil', this.themeConfig.isDark ? `rgba(30, 32, 48, ${veil.toFixed(2)})` : `rgba(255, 255, 255, ${veil.toFixed(2)})`);
            } else {
                ['--glass-frost-fx', '--glass-frost-veil'].forEach((v) => el.style.removeProperty(v));
            }

            if (this.themeConfig.isDark) {
                // 暗色玻璃：深灰蓝半透明面板 + 浅色高光（Element 暗模式文字为浅色，配深面板才可读）
                // 清透实验档同步降底(0.6→0.46), 浮层 0.72; 回退参考亮色块注释
                set('--bg-main-color', 'rgba(30, 32, 48, 0.46)');
                set('--bg-color', 'rgba(30, 32, 48, 0.46)');
                set('--bg-menuBar', 'rgba(30, 32, 48, 0.46)');
                set('--bg-topBar', 'rgba(30, 32, 48, 0.46)');
                set('--el-bg-color', 'rgba(30, 32, 48, 0.46)');
                set('--el-bg-color-page', 'transparent');
                set('--el-bg-color-overlay', 'rgba(40, 44, 60, 0.72)');
                set('--el-fill-color-blank', 'rgba(255, 255, 255, 0.05)');
                set('--el-fill-color-light', 'rgba(255, 255, 255, 0.09)');
                set('--el-fill-color', 'rgba(255, 255, 255, 0.11)');
                set('--el-fill-color-lighter', 'rgba(255, 255, 255, 0.06)');
                set('--el-fill-color-extra-light', 'rgba(255, 255, 255, 0.03)');
                set('--el-table-bg-color', 'transparent');
                set('--el-table-tr-bg-color', 'rgba(30, 32, 48, 0.25)');
                set('--el-table-header-bg-color', 'rgba(30, 32, 48, 0.18)');
                set('--el-table-row-hover-bg-color', 'rgba(255, 255, 255, 0.06)');
                return;
            }

            // 亮色玻璃：清透实验档(与 tokens.scss --glass-alpha-* 单一对齐)
            // 面板白底大幅下调(0.48→0.26)、铬层 0.3→0.22 —— 让清晰壁纸细节主导画面(资源树观感)，
            // 浮层仍保 0.72 可读底;文字墨色随之更重要(壁纸直透后对比依赖墨色加深)。
            // 回重磨砂档: 恢复 menuBar/topBar 0.3 / bg 0.48 / overlay 0.76 / tr 0.08 / header 0.24 / hover 0.2
            set('--bg-main-color', 'transparent');
            set('--bg-color', 'transparent');
            set('--bg-menuBar', 'rgba(255, 255, 255, 0.22)');
            set('--bg-topBar', 'rgba(255, 255, 255, 0.22)');
            set('--el-bg-color', 'rgba(255, 255, 255, 0.26)');
            set('--el-bg-color-page', 'transparent');
            set('--el-bg-color-overlay', 'rgba(255, 255, 255, 0.72)');
            set('--el-fill-color-blank', 'rgba(255, 255, 255, 0.22)');
            set('--el-fill-color-light', 'rgba(255, 255, 255, 0.14)');
            set('--el-fill-color', 'rgba(255, 255, 255, 0.1)');
            set('--el-fill-color-lighter', 'rgba(255, 255, 255, 0.07)');
            set('--el-fill-color-extra-light', 'rgba(255, 255, 255, 0.04)');
            set('--el-table-bg-color', 'transparent');
            set('--el-table-tr-bg-color', 'rgba(255, 255, 255, 0.04)');
            set('--el-table-header-bg-color', 'rgba(255, 255, 255, 0.14)');
            set('--el-table-row-hover-bg-color', 'rgba(255, 255, 255, 0.12)');
            // 文本墨色: EP 默认 secondary #909399 / placeholder #a8abb2 在浅色玻璃上仅 ~2.8:1,
            // 达不到 4.5:1 → 观感"发灰不清晰"。玻璃态统一加深到 slate 档。
            // primary/regular 同样下压: 面板白底清透(0.22~0.26)、壁纸直透鲜艳, 正文/导航是主要
            // 承载文字, 默认 #303133/#606266 在亮壁纸上对比不足; 加深墨色换可读(不动 alpha/磨砂,
            // 保住用户定调的通透鲜艳观感)。
            set('--el-text-color-primary', '#1f2937');
            set('--el-text-color-regular', '#374151');
            set('--el-text-color-secondary', '#64748b');
            set('--el-text-color-placeholder', '#6b7280');
        },
        // 移除玻璃模式变量（恢复原值）
        _removeGlassVariables() {
            const el = document.documentElement;
            const vars = [
                '--bg-main-color', '--bg-color', '--bg-menuBar', '--bg-topBar',
                '--el-bg-color', '--el-bg-color-page', '--el-bg-color-overlay',
                '--el-fill-color-blank', '--el-fill-color-light', '--el-fill-color',
                '--el-fill-color-lighter', '--el-fill-color-extra-light',
                '--el-table-bg-color', '--el-table-tr-bg-color',
                '--el-table-header-bg-color', '--el-table-row-hover-bg-color',
                '--el-text-color-secondary', '--el-text-color-placeholder',
                '--el-text-color-primary', '--el-text-color-regular',
                '--glass-frost-fx', '--glass-frost-veil',
            ];
            vars.forEach(v => el.style.removeProperty(v));
            // 恢复缓存中的原始样式
            const savedStyle = getLocal('themeConfigStyle');
            if (savedStyle) {
                // 重新解析原始变量
                const parser = new DOMParser();
                const doc = parser.parseFromString(`<html style="${savedStyle}"></html>`, 'text/html');
                const origStyle = doc.documentElement.getAttribute('style') || '';
                origStyle.split(';').forEach(pair => {
                    const [key] = pair.split(':').map(s => s.trim());
                    if (key && vars.includes(key)) {
                        el.style.removeProperty(key);
                    }
                });
            }
        },
        // 设置玻璃壁纸：切换 <html data-wallpaper> 属性（'none' 移除属性，回退默认动态光斑）
        // 内置预设的渐变由 wallpaper.scss 依据该属性选择器应用；'custom' 走 --backdrop-image 图层
        setGlassWallpaper(id: string) {
            this.themeConfig.glassWallpaper = id;
            const root = document.documentElement;
            if (id === 'none') {
                root.removeAttribute('data-wallpaper');
            } else {
                root.setAttribute('data-wallpaper', id);
            }
        },
        // 设置磨砂程度(0-40 blur px, 0=纹理清晰默认档): 立即重算 fx 档位变量
        setGlassFrost(level: number) {
            this.themeConfig.glassFrost = level;
            this._applyGlassVariables();
        },
        // 设置背景图
        setBgImage(url: string) {
            // 先写 DOM 变量再更新状态: 当前页效果立即可见, 不受后续任何链路异常影响
            applyBackdropImageVar(url);
            this.themeConfig.bgImage = url;
        },
        // 设置背景图模糊
        setBgImageBlur(blur: number) {
            this.themeConfig.bgImageBlur = blur;
        },
        // 设置背景图不透明度（玻璃态下壁纸图层钉死, 该值仅保留兼容, 不参与渲染）
        setBgImageOpacity(opacity: number) {
            this.themeConfig.bgImageOpacity = opacity;
        },
    },
});

// 计算每页显示数量的方法
const calculatePageSizeByScreenHeight = (): number => {
    const windowHeight = window.innerHeight || document.documentElement.clientHeight;

    // 计算页面其他部分的高度（这是一个大概的估算）
    // 包括顶部导航、面包屑、搜索区域、分页控件等
    const headerHeight = 60; // 页面顶部导航高度
    const subHeaderHeight = 50; // 子页面头部或其他内容高度
    const searchFormHeight = 100; // 搜索表单高度，如果显示的话
    const tableHeaderHeight = 44; // 表格头部高度
    const paginationHeight = 40; // 分页控件高度
    const paddingMarginHeight = 30; // 额外的内外边距

    // 计算可用于表格内容的高度
    const availableContentHeight =
        windowHeight - headerHeight - subHeaderHeight - searchFormHeight - tableHeaderHeight - paginationHeight - paddingMarginHeight;

    // 根据表格尺寸确定行高
    const rowHeight = 40;

    // 计算理论上的行数
    const calculatedRows = Math.floor(availableContentHeight / rowHeight);

    // 设置限制范围
    const minPageSize = 10;
    const maxPageSize = 30;

    // 确保返回值在合理范围内，且至少有基本的行数
    return Math.max(minPageSize, Math.min(maxPageSize, calculatedRows));
};
