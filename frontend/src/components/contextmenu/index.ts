import Contextmenu from './index.vue';
import { ContextmenuItem, filterVisibleItems } from './item';
import { MENU_CLOSE_ANIMATION_MS } from './constants';

// 对外 API 保持向后兼容：历史调用方从本目录导入 Contextmenu / ContextmenuItem
export { filterVisibleItems };
export { ContextmenuItem };
export { Contextmenu };
// 点击 handler 的派发延迟（= 菜单关闭动画时长），调用方若需等待 handler 执行完毕可据此对齐
export { MENU_CLOSE_ANIMATION_MS };
