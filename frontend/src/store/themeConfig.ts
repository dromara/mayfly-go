import { defineStore } from 'pinia';
import { formatDate } from '@/common/utils/format';
import { useUserInfo } from '@/store/userInfo';
import { getServerConf, getSysStyleConfig } from '@/common/sysconfig';
import { getLocal, getThemeConfig } from '@/common/utils/storage';

// 系统默认logo图标，对应于@/assets/image/logo.svg
const logoIcon =
    'data:image/svg+xml;charset=utf-8;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBzdGFuZGFsb25lPSJubyI/Pgo8IURPQ1RZUEUgc3ZnIFBVQkxJQyAiLS8vVzNDLy9EVEQgU1ZHIDEuMS8vRU4iICJodHRwOi8vd3d3LnczLm9yZy9HcmFwaGljcy9TVkcvMS4xL0RURC9zdmcxMS5kdGQiPgo8c3ZnIHQ9IjE2MjE4NTkwMDk2MDUiIGNsYXNzPSJpY29uIiB2aWV3Qm94PSIwIDAgMTAyNCAxMDI0IiB2ZXJzaW9uPSIxLjEiIAogICAgIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgcC1pZD0iOTcwOSIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiIAogICAgIHdpZHRoPSIyMDAiIGhlaWdodD0iMjAwIj4KICAgICA8ZGVmcz48c3R5bGUgdHlwZT0idGV4dC9jc3MiPjwvc3R5bGU+PC9kZWZzPgogICAgIDxwYXRoIGQ9Ik04MjAuMjAzOTIyIDgxMi4xNzI1NDlINjg0LjY3NDUxdi00NS4xNzY0NzFoMTEyLjQzOTIxNVYyNzkuMDkwMTk2SDYzMy40NzQ1MWwtODUuMzMzMzM0IDI3Ny4wODIzNTNjLTMuMDExNzY1IDEwLjAzOTIxNi0xMi4wNDcwNTkgMTYuMDYyNzQ1LTIyLjA4NjI3NCAxNi4wNjI3NDUtMTAuMDM5MjE2IDAtMTkuMDc0NTEtNy4wMjc0NTEtMjEuMDgyMzUzLTE3LjA2NjY2N2wtNzEuMjc4NDMxLTI4MC4wOTQxMTdoLTE4MC43MDU4ODNWNzYyLjk4MDM5MmgxMjAuNDcwNTg5djQ1LjE3NjQ3MUgyMjkuODk4MDM5Yy0xMi4wNDcwNTkgMC0yMi4wODYyNzUtMTAuMDM5MjE2LTIyLjA4NjI3NC0yMi4wODYyNzVWMjUyLjk4ODIzNWMwLTEyLjA0NzA1OSAxMC4wMzkyMTYtMjIuMDg2Mjc1IDIyLjA4NjI3NC0yMi4wODYyNzRINDUxLjc2NDcwNmMxMC4wMzkyMTYgMCAxOS4wNzQ1MSA3LjAyNzQ1MSAyMi4wODYyNzQgMTcuMDY2NjY2bDU1LjIxNTY4NyAyMTguODU0OTAyTDU5NS4zMjU0OSAyNTAuOTgwMzkyYzMuMDExNzY1LTkuMDM1Mjk0IDEyLjA0NzA1OS0xNi4wNjI3NDUgMjEuMDgyMzUzLTE2LjA2Mjc0NWgyMDIuNzkyMTU3YzEyLjA0NzA1OSAwIDIyLjA4NjI3NSAxMC4wMzkyMTYgMjIuMDg2Mjc1IDIyLjA4NjI3NXY1MzMuMDgyMzUzYzEuMDAzOTIyIDEyLjA0NzA1OS05LjAzNTI5NCAyMi4wODYyNzUtMjEuMDgyMzUzIDIyLjA4NjI3NHogbTAgMCIgZmlsbD0iI2UyNTgxMyIgcC1pZD0iOTcxMCIgc3Ryb2tlLXdpZHRoPSIzMCIgc3Ryb2tlPSIjZTI1ODEzIj48L3BhdGg+CiAgICAgPHBhdGggZD0iTTczMS44NTg4MjQgNDI1LjY2Mjc0NWM0LjAxNTY4Ni0xMi4wNDcwNTktMi4wMDc4NDMtMjUuMDk4MDM5LTE0LjA1NDkwMi0yOS4xMTM3MjUtMTIuMDQ3MDU5LTQuMDE1Njg2LTI1LjA5ODAzOSAyLjAwNzg0My0yOS4xMTM3MjYgMTQuMDU0OTAyTDU2My4yIDc2Ni45OTYwNzhoLTczLjI4NjI3NUwzNzEuNDUwOTggNDEwLjYwMzkyMmMtNC4wMTU2ODYtMTIuMDQ3MDU5LTE3LjA2NjY2Ny0xOC4wNzA1ODgtMjguMTA5ODA0LTE0LjA1NDkwMi0xMi4wNDcwNTkgNC4wMTU2ODYtMTguMDcwNTg4IDE3LjA2NjY2Ny0xNC4wNTQ5MDEgMjguMTA5ODA0bDEyMy40ODIzNTIgMzcxLjQ1MDk4YzMuMDExNzY1IDkuMDM1Mjk0IDEyLjA0NzA1OSAxNS4wNTg4MjQgMjEuMDgyMzUzIDE1LjA1ODgyM2g3Mi4yODIzNTNsLTUzLjIwNzg0MyAxNjAuNjI3NDUxIDQ2LjE4MDM5MiAyLjAwNzg0NCAxOTIuNzUyOTQyLTU0OC4xNDExNzd6IiBmaWxsPSIjMmMyYzJjIiBwLWlkPSI5NzExIiBzdHJva2Utd2lkdGg9IjMwIiBzdHJva2U9IiMyYzJjMmMiPjwvcGF0aD4KPC9zdmc+';

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

            /* 其它设置
            ------------------------------- */
            // 默认 Tagsview 风格，可选 1、 tags-style-one 2、 tags-style-two 3、 tags-style-three
            tagsStyle: 'tags-style-three',
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
                this.themeConfig = tc;
                document.documentElement.style.cssText = getLocal('themeConfigStyle');
            } else {
                getServerConf().then((res) => {
                    this.themeConfig.globalI18n = res.i18n;
                });
            }

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
                this.themeConfig.isWatermark = res?.useWatermark;
                if (!res?.useWatermark) {
                    return;
                }
                // 索引2为用户自定义水印信息
                this.themeConfig.watermarkText[2] = res.watermarkContent;
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
        },
    },
});
