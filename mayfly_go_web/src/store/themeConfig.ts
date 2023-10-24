import { defineStore } from 'pinia';
import { dateFormat2 } from '@/common/utils/date';
import { useUserInfo } from '@/store/userInfo';

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
            globalViceTitle: 'mayfly',
            // 默认初始语言，可选值"<zh-cn|en|zh-tw>"，默认 zh-cn
            globalI18n: 'zh-cn',
            // 默认全局组件大小，可选值"<|large|default|small>"，默认 ''
            globalComponentSize: '',
        },
    }),
    actions: {
        // 设置布局配置
        setThemeConfig(data: ThemeConfigState) {
            this.themeConfig = data.themeConfig;
        },
        // 切换暗模式
        switchDark(isDark: boolean) {
            this.themeConfig.isDark = isDark;
            const body = document.documentElement as HTMLElement;
            if (isDark) {
                body.setAttribute('class', 'dark');
                this.themeConfig.editorTheme = 'vs-dark';
            } else {
                body.setAttribute('class', '');
                this.themeConfig.editorTheme = 'SolarizedLight';
            }
        },
        // 设置水印配置信息
        setWatermarkConfig(useWatermarkConfig: any) {
            this.themeConfig.watermarkText = [];
            this.themeConfig.isWatermark = useWatermarkConfig.isUse;
            if (!useWatermarkConfig.isUse) {
                return;
            }
            // 索引2为用户自定义水印信息
            this.themeConfig.watermarkText[2] = useWatermarkConfig.content;
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
