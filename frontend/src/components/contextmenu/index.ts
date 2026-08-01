import Contextmenu from './index.vue';

class ContextmenuItem {
    clickId: string | number;

    txt: string;

    icon: string;

    affix: boolean;

    permission: string;

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

export { Contextmenu, ContextmenuItem };
