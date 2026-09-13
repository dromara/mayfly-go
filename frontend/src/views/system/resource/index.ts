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
    if (!menu.meta || typeof menu.meta !== 'string') {
        return menu.meta?.icon ?? '';
    }
    try {
        const meta = JSON.parse(menu.meta);
        return meta?.icon ?? '';
    } catch {
        return '';
    }
}
