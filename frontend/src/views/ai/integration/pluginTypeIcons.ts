/**
 * 插件类型图标映射（图标名 → 组件）
 *
 * 元数据表（integration/types 的 PLUGIN_TYPE_META）只存图标「名字」——types 层不该依赖 UI 库；
 * 本文件是唯一的「名字 → 组件」适配点，插件卡片与类型选择器共用，不再各自写一份表。
 */
import type { Component } from 'vue';
import { Connection, FolderOpened, Menu } from '@element-plus/icons-vue';
import { PLUGIN_TYPE_META } from './types';

const PLUGIN_TYPE_ICONS: Record<string, Component> = {
    skill: FolderOpened,
    mcp: Connection,
};

/**
 * 取类型图标。
 *
 * 未登记的类型一律给中性 Menu 图标：不能走兜底元数据的 icon 值，那会让一个全新的类型
 * 在界面上长得像 MCP，用户无从分辨（图标也是「类型身份」的一部分）
 */
export function pluginTypeIcon(typeCode: string): Component {
    const meta = PLUGIN_TYPE_META[typeCode];
    return meta ? PLUGIN_TYPE_ICONS[meta.icon] ?? Menu : Menu;
}
