declare interface UserInfoState {
    userInfo: UserInfo;
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
        isDark: boolean;
        isGrayscale: boolean;
        isInvert: boolean;
        isWatermark: boolean;
        watermarkText: Array<string>;
        animation: string;
        columnsAsideStyle: string;
        layout: string;
        isRequestRoutes: boolean;
        globalTitle: string;
        globalViceTitle: string;
        appSlogan: string;
        logoIcon: string;
        version: string;
        globalI18n: string;
        globalComponentSize: string;
        terminalTheme: string;
        terminalForeground: string;
        terminalBackground: string;
        terminalCursor: string;
        terminalFontSize: number;
        terminalFontWeight: string;
        editorTheme: string;

        defaultListPageSize: number;
    };
}

declare interface TagsView {
    /**
     * 路径
     */
    path: string;

    /**
     * 标题
     */
    title: string;

    /**
     * router name
     */
    name: string;

    /**
     * router query
     */
    query: Record<string, string | string[] | null>;

    /**
     * 图标
     */
    icon: string;

    isAffix: boolean;
    isKeepAlive: boolean;
    isHide?: boolean;
}

// TagsView 路由列表
declare interface TagsViewsState {
    tagsViews: TagsView[];
    currentRefreshPath: string; // 当前刷新的路由 path
}

// 路由项
declare interface RouteItem {
    path: string;
    name?: string;
    meta: Record<string, any>;
    redirect?: string;
    component?: unknown;
    children?: RouteItem[];
    [key: string]: unknown;
}

// 分栏/经典布局传送的当前子级菜单数据
declare interface SendChildrenResult {
    item: Array<RouteItem & { k: number }>;
    children: RouteItem[];
}

// 路由列表
declare interface RoutesListState {
    routesList: RouteItem[];
}

// 路由缓存列表
declare interface KeepAliveNamesState {
    keepAliveNames: string[];
    cachedViews: string[];
}

declare interface MilvusDb {
    id: string;
    name: string;
    [key: string]: unknown;
}

declare interface MilvusCollection {
    name: string;
    [key: string]: unknown;
}

declare interface MilvusState {
    dbs: MilvusDb[];
    selectedDb: string;
    selectedCollection: string;
    collections: MilvusCollection[];
    authCertName: string;
}
