/**
 * 右键菜单项模型与可见性过滤（纯逻辑，不依赖组件）：
 * - children 非空时渲染为子菜单（二级/三级...递归），子项全部隐藏时本项自动隐藏
 * - txt 为 i18n key；hideFunc/onClickFunc 的入参为 openContextmenu(item) 传入的业务数据
 */
export class ContextmenuItem {
    clickId: string | number;

    txt: string;

    icon: string;

    affix: boolean;

    permission: string;

    children?: ContextmenuItem[];

    /**
     * 是否隐藏回调函数
     */
    hideFunc: (data: any) => boolean;

    onClickFunc: (data: any) => void;

    constructor(clickId: string | number, txt: string) {
        this.clickId = clickId;
        this.txt = txt;
    }

    withIcon(icon: string) {
        this.icon = icon;
        return this;
    }

    withPermission(permission: string) {
        this.permission = permission;
        return this;
    }

    withHideFunc(func: (data: any) => boolean) {
        this.hideFunc = func;
        return this;
    }

    withOnClick(func: (data: any) => void) {
        this.onClickFunc = func;
        return this;
    }

    withChildren(children: ContextmenuItem[]) {
        this.children = children;
        return this;
    }

    /**
     * 是否隐藏
     * @param data 点击数据项
     * @returns
     */
    isHide(data: unknown) {
        if (this.hideFunc) {
            return this.hideFunc(data);
        }
        return false;
    }
}

/**
 * 递归过滤出可见菜单项：剔除 affix 项与 hideFunc 命中的项；
 * 声明了 children 但子项全部不可见的项一并剔除（避免空壳子菜单）
 */
export const filterVisibleItems = (items: ContextmenuItem[], payload: unknown): ContextmenuItem[] => {
    return items
        .filter((item) => !item.affix && !item.isHide(payload))
        .map((item) => {
            if (!item.children?.length) {
                return item;
            }
            // 拷贝实例以保留原型方法，避免污染调用方持有的原项
            const copy = new ContextmenuItem(item.clickId, item.txt);
            Object.assign(copy, item, { children: filterVisibleItems(item.children, payload) });
            return copy;
        })
        .filter((item) => !item.children || item.children.length > 0);
};
