import { defineStore } from 'pinia';
import { dateFormat2 } from '@/common/utils/date';
import { useUserInfo } from '@/store/userInfo';
import { getSysStyleConfig } from '@/common/sysconfig';
import { getLocal, getThemeConfig } from '@/common/utils/storage';

// 系统默认logo图标，对应于@/assets/image/logo.svg
const logoIcon =
    'data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBzdGFuZGFsb25lPSJubyI/PjwhRE9DVFlQRSBzdmcgUFVCTElDICItLy9XM0MvL0RURCBTVkcgMS4xLy9FTiIgImh0dHA6Ly93d3cudzMub3JnL0dyYXBoaWNzL1NWRy8xLjEvRFREL3N2ZzExLmR0ZCI+PHN2ZyB0PSIxNjIxODU5MDA5NjA1IiBjbGFzcz0iaWNvbiIgdmlld0JveD0iMCAwIDEwMjQgMTAyNCIgdmVyc2lvbj0iMS4xIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHAtaWQ9Ijk3MDkiIHhtbG5zOnhsaW5rPSJodHRwOi8vd3d3LnczLm9yZy8xOTk5L3hsaW5rIiB3aWR0aD0iMjAwIiBoZWlnaHQ9IjIwMCI+PGRlZnM+PHN0eWxlIHR5cGU9InRleHQvY3NzIj48L3N0eWxlPjwvZGVmcz48cGF0aCBkPSJNODIwLjIwMzkyMiA4MTIuMTcyNTQ5SDY4NC42NzQ1MXYtNDUuMTc2NDcxaDExMi40MzkyMTVWMjc5LjA5MDE5Nkg2MzMuNDc0NTFsLTg1LjMzMzMzNCAyNzcuMDgyMzUzYy0zLjAxMTc2NSAxMC4wMzkyMTYtMTIuMDQ3MDU5IDE2LjA2Mjc0NS0yMi4wODYyNzQgMTYuMDYyNzQ1LTEwLjAzOTIxNiAwLTE5LjA3NDUxLTcuMDI3NDUxLTIxLjA4MjM1My0xNy4wNjY2NjdsLTcxLjI3ODQzMS0yODAuMDk0MTE3aC0xODAuNzA1ODgzVjc2Mi45ODAzOTJoMTIwLjQ3MDU4OXY0NS4xNzY0NzFIMjI5Ljg5ODAzOWMtMTIuMDQ3MDU5IDAtMjIuMDg2Mjc1LTEwLjAzOTIxNi0yMi4wODYyNzQtMjIuMDg2Mjc1VjI1Mi45ODgyMzVjMC0xMi4wNDcwNTkgMTAuMDM5MjE2LTIyLjA4NjI3NSAyMi4wODYyNzQtMjIuMDg2Mjc0SDQ1MS43NjQ3MDZjMTAuMDM5MjE2IDAgMTkuMDc0NTEgNy4wMjc0NTEgMjIuMDg2Mjc0IDE3LjA2NjY2Nmw1NS4yMTU2ODcgMjE4Ljg1NDkwMkw1OTUuMzI1NDkgMjUwLjk4MDM5MmMzLjAxMTc2NS05LjAzNTI5NCAxMi4wNDcwNTktMTYuMDYyNzQ1IDIxLjA4MjM1My0xNi4wNjI3NDVoMjAyLjc5MjE1N2MxMi4wNDcwNTkgMCAyMi4wODYyNzUgMTAuMDM5MjE2IDIyLjA4NjI3NSAyMi4wODYyNzV2NTMzLjA4MjM1M2MxLjAwMzkyMiAxMi4wNDcwNTktOS4wMzUyOTQgMjIuMDg2Mjc1LTIxLjA4MjM1MyAyMi4wODYyNzR6IG0wIDAiIGZpbGw9IiNlMjU4MTMiIHAtaWQ9Ijk3MTAiPjwvcGF0aD48cGF0aCBkPSJNNzMxLjg1ODgyNCA0MjUuNjYyNzQ1YzQuMDE1Njg2LTEyLjA0NzA1OS0yLjAwNzg0My0yNS4wOTgwMzktMTQuMDU0OTAyLTI5LjExMzcyNS0xMi4wNDcwNTktNC4wMTU2ODYtMjUuMDk4MDM5IDIuMDA3ODQzLTI5LjExMzcyNiAxNC4wNTQ5MDJMNTYzLjIgNzY2Ljk5NjA3OGgtNzMuMjg2Mjc1TDM3MS40NTA5OCA0MTAuNjAzOTIyYy00LjAxNTY4Ni0xMi4wNDcwNTktMTcuMDY2NjY3LTE4LjA3MDU4OC0yOC4xMDk4MDQtMTQuMDU0OTAyLTEyLjA0NzA1OSA0LjAxNTY4Ni0xOC4wNzA1ODggMTcuMDY2NjY3LTE0LjA1NDkwMSAyOC4xMDk4MDRsMTIzLjQ4MjM1MiAzNzEuNDUwOThjMy4wMTE3NjUgOS4wMzUyOTQgMTIuMDQ3MDU5IDE1LjA1ODgyNCAyMS4wODIzNTMgMTUuMDU4ODIzaDcyLjI4MjM1M2wtNTMuMjA3ODQzIDE2MC42Mjc0NTEgNDYuMTgwMzkyIDIuMDA3ODQ0IDE5Mi43NTI5NDItNTQ4LjE0MTE3N3oiIGZpbGw9IiMyYzJjMmMiIHAtaWQ9Ijk3MTEiPjwvcGF0aD48L3N2Zz4=';

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
            // 是否开启自动锁屏
            isLockScreen: false,
            // 开启自动锁屏倒计时(s/秒)
            lockScreenTime: 30,

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
            layout: 'classic',

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
            this.themeConfig.watermarkText[1] = dateFormat2('yyyy-MM-dd HH:mm:ss', new Date());
        },
    },
});
