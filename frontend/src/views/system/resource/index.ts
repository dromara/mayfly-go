import { ResourceTypeEnum } from '../enums';

export { default } from './ResourceList.vue';

/**
 * 获取menu icon
 * @param menu menu信息
 * @returns icon name
 */
export function getMenuIcon(menu: any) {
    if (menu.type == ResourceTypeEnum.Permission.value) {
        return 'icon menu/permission';
    }
    if (!menu.meta) {
        return '';
    }
    const meta = JSON.parse(menu.meta);
    if (meta) {
        return meta.icon;
    }
    return '';
}
