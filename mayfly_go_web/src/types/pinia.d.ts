declare interface UserInfoState<T = any> {
    userInfo: any
}

declare interface ThemeConfigState {
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
        isShareTagsView: boolean;
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
        terminalFontWeight: string | any;
        editorTheme: string;
    };
}

// TagsView 路由列表
declare interface TagsViewRoutesState<T = any> {
	tagsViewRoutes: T[];
	isTagsViewCurrenFull: Boolean;
}

// 路由列表
declare interface RoutesListState {
    routesList: T[];
}

// 路由缓存列表
declare interface KeepAliveNamesState {
    keepAliveNames: string[];
    cachedViews: string[];
}