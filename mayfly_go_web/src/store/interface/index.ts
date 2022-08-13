// 接口类型声明

// 布局配置
export interface ThemeConfigState {
    themeConfig: {
        isDrawer: boolean;
        primary: string;
        success: string;
        info: string;
        warning: string;
        danger: string;
        topBar: string;
        menuBar: string;
        columnsMenuBar: string;
        topBarColor: string;
        menuBarColor: string;
        columnsMenuBarColor: string;
        isTopBarColorGradual: boolean;
        isMenuBarColorGradual: boolean;
        isColumnsMenuBarColorGradual: boolean;
        isMenuBarColorHighlight: boolean;
        isCollapse: boolean;
        isUniqueOpened: boolean;
        isFixedHeader: boolean;
        isFixedHeaderChange: boolean;
        isClassicSplitMenu: boolean;
        isLockScreen: boolean;
        lockScreenTime: number;
        isShowLogo: boolean;
        isShowLogoChange: boolean;
        isBreadcrumb: boolean;
        isTagsview: boolean;
        isBreadcrumbIcon: boolean;
        isTagsviewIcon: boolean;
        isCacheTagsView: boolean;
        isSortableTagsView: boolean;
        isFooter: boolean;
        isGrayscale: boolean;
        isInvert: boolean;
        isWartermark: boolean;
        wartermarkText: string;
        tagsStyle: string;
        animation: string;
        columnsAsideStyle: string;
        layout: string;
        isRequestRoutes: boolean;
        globalTitle: string;
        globalViceTitle: string;
        globalI18n: string;
        globalComponentSize: string;
        terminalForeground: string;
        terminalBackground: string;
        terminalCursor: string;
        terminalFontSize: number;
        terminalFontWeight: string;
    };
}

// 路由列表
export interface RoutesListState {
    routesList: Array<object>;
}

// 路由缓存列表
export interface KeepAliveNamesState {
    keepAliveNames: Array<string>;
}

// 用户信息
export interface UserInfosState {
    userInfos: object;
}

// 后端返回原始路由(未处理时)
// export interface RequestOldRoutesState {
//     requestOldRoutes: Array<object>;
// }

// 主接口(顶级类型声明)
export interface RootStateTypes {
    themeConfig: ThemeConfigState;
    routesList: RoutesListState;
    keepAliveNames: KeepAliveNamesState;
    userInfos: UserInfosState;
    // requestOldRoutes: RequestOldRoutesState;
}
